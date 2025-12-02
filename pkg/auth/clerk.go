// Package auth provides authentication utilities for QL-RF services.
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ClerkClaims represents the claims in a Clerk JWT.
type ClerkClaims struct {
	jwt.RegisteredClaims
	// Clerk-specific claims
	SessionID    string                 `json:"sid,omitempty"`
	ActorID      string                 `json:"act,omitempty"`
	OrgID        string                 `json:"org_id,omitempty"`
	OrgRole      string                 `json:"org_role,omitempty"`
	OrgSlug      string                 `json:"org_slug,omitempty"`
	OrgPerms     []string               `json:"org_permissions,omitempty"`
	UserMetadata map[string]interface{} `json:"user_metadata,omitempty"`

	// User info from Clerk
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	ImageURL      string `json:"image_url,omitempty"`
}

// ClerkVerifier verifies Clerk JWTs using JWKS.
type ClerkVerifier struct {
	issuer     string
	jwksURL    string
	httpClient *http.Client

	mu     sync.RWMutex
	keys   map[string]*rsa.PublicKey
	expiry time.Time
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key.
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// NewClerkVerifier creates a new Clerk JWT verifier.
// clerkDomain should be like "fluent-glowworm-43.clerk.accounts.dev" (from publishable key)
func NewClerkVerifier(clerkDomain string) *ClerkVerifier {
	// Extract domain from publishable key if provided
	domain := clerkDomain
	if strings.HasPrefix(clerkDomain, "pk_") {
		// pk_test_Zmx1ZW50LWdsb3d3b3JtLTQzLmNsZXJrLmFjY291bnRzLmRldiQ
		// Decode base64 after pk_test_ or pk_live_
		parts := strings.SplitN(clerkDomain, "_", 3)
		if len(parts) == 3 {
			decoded, err := base64.RawStdEncoding.DecodeString(parts[2])
			if err == nil {
				domain = strings.TrimSuffix(string(decoded), "$")
			}
		}
	}

	return &ClerkVerifier{
		issuer:     fmt.Sprintf("https://%s", domain),
		jwksURL:    fmt.Sprintf("https://%s/.well-known/jwks.json", domain),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]*rsa.PublicKey),
	}
}

// Verify verifies a Clerk JWT and returns the claims.
func (v *ClerkVerifier) Verify(ctx context.Context, tokenString string) (*ClerkClaims, error) {
	// Parse the token without verification to get the key ID
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, &ClerkClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Get the key ID from header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("missing kid in token header")
	}

	// Get the public key
	key, err := v.getKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Parse and verify the token
	claims := &ClerkClaims{}
	token, err = jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return key, nil
	}, jwt.WithIssuer(v.issuer), jwt.WithExpirationRequired())

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// getKey retrieves a public key by key ID, fetching from JWKS if needed.
func (v *ClerkVerifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Check cache first
	v.mu.RLock()
	key, ok := v.keys[kid]
	expired := time.Now().After(v.expiry)
	v.mu.RUnlock()

	if ok && !expired {
		return key, nil
	}

	// Fetch JWKS
	if err := v.fetchJWKS(ctx); err != nil {
		return nil, err
	}

	// Check again after fetch
	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("key not found: %s", kid)
	}

	return key, nil
}

// fetchJWKS fetches the JWKS from Clerk.
func (v *ClerkVerifier) fetchJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", v.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS fetch returned status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue
		}

		key, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			continue // Skip invalid keys
		}

		v.keys[jwk.Kid] = key
	}

	// Cache for 1 hour
	v.expiry = time.Now().Add(1 * time.Hour)

	return nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key.
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode N (modulus)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode N: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode E (exponent)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode E: %w", err)
	}
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

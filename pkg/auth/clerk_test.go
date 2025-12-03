// Package auth provides authentication utilities for QL-RF services.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewClerkVerifier(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedIssuer string
	}{
		{
			name:           "plain domain",
			input:          "example.clerk.accounts.dev",
			expectedIssuer: "https://example.clerk.accounts.dev",
		},
		{
			name:           "publishable key format",
			input:          "pk_test_" + base64.RawStdEncoding.EncodeToString([]byte("fluent-glowworm-43.clerk.accounts.dev$")),
			expectedIssuer: "https://fluent-glowworm-43.clerk.accounts.dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewClerkVerifier(tt.input)
			if v.issuer != tt.expectedIssuer {
				t.Errorf("expected issuer %q, got %q", tt.expectedIssuer, v.issuer)
			}
		})
	}
}

func TestClerkVerifier_Verify(t *testing.T) {
	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	// Create a test JWKS server
	kid := "test-key-id"
	jwks := JWKS{
		Keys: []JWK{
			{
				Kid: kid,
				Kty: "RSA",
				Alg: "RS256",
				Use: "sig",
				N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(jwks)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	// Create verifier with test server
	domain := server.URL[7:] // Remove "http://"
	v := &ClerkVerifier{
		issuer:     "https://" + domain,
		jwksURL:    server.URL + "/.well-known/jwks.json",
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]*rsa.PublicKey),
	}

	t.Run("valid token", func(t *testing.T) {
		claims := &ClerkClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    v.issuer,
				Subject:   "user_123",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			OrgID:   "org_456",
			OrgRole: "admin",
			Email:   "test@example.com",
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = kid
		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		result, err := v.Verify(context.Background(), tokenString)
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}

		if result.Subject != "user_123" {
			t.Errorf("expected subject user_123, got %s", result.Subject)
		}
		if result.OrgID != "org_456" {
			t.Errorf("expected org_id org_456, got %s", result.OrgID)
		}
		if result.Email != "test@example.com" {
			t.Errorf("expected email test@example.com, got %s", result.Email)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		claims := &ClerkClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    v.issuer,
				Subject:   "user_123",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = kid
		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = v.Verify(context.Background(), tokenString)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("wrong issuer", func(t *testing.T) {
		claims := &ClerkClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "https://wrong-issuer.example.com",
				Subject:   "user_123",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = kid
		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = v.Verify(context.Background(), tokenString)
		if err == nil {
			t.Error("expected error for wrong issuer")
		}
	})

	t.Run("missing kid", func(t *testing.T) {
		claims := &ClerkClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    v.issuer,
				Subject:   "user_123",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		// Not setting kid header
		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = v.Verify(context.Background(), tokenString)
		if err == nil {
			t.Error("expected error for missing kid")
		}
	})

	t.Run("unknown kid", func(t *testing.T) {
		claims := &ClerkClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    v.issuer,
				Subject:   "user_123",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = "unknown-key-id"
		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = v.Verify(context.Background(), tokenString)
		if err == nil {
			t.Error("expected error for unknown kid")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := v.Verify(context.Background(), "not-a-valid-token")
		if err == nil {
			t.Error("expected error for malformed token")
		}
	})
}

func TestJwkToRSAPublicKey(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	t.Run("valid JWK", func(t *testing.T) {
		jwk := JWK{
			Kid: "test",
			Kty: "RSA",
			Alg: "RS256",
			Use: "sig",
			N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
		}

		result, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			t.Fatalf("jwkToRSAPublicKey failed: %v", err)
		}

		if result.N.Cmp(publicKey.N) != 0 {
			t.Error("N values don't match")
		}
		if result.E != publicKey.E {
			t.Errorf("E values don't match: got %d, want %d", result.E, publicKey.E)
		}
	})

	t.Run("invalid N encoding", func(t *testing.T) {
		jwk := JWK{
			Kid: "test",
			Kty: "RSA",
			N:   "not-valid-base64!!",
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
		}

		_, err := jwkToRSAPublicKey(jwk)
		if err == nil {
			t.Error("expected error for invalid N encoding")
		}
	})

	t.Run("invalid E encoding", func(t *testing.T) {
		jwk := JWK{
			Kid: "test",
			Kty: "RSA",
			N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
			E:   "not-valid-base64!!",
		}

		_, err := jwkToRSAPublicKey(jwk)
		if err == nil {
			t.Error("expected error for invalid E encoding")
		}
	})
}

func TestClerkVerifier_KeyCaching(t *testing.T) {
	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	kid := "test-key-id"
	jwks := JWKS{
		Keys: []JWK{
			{
				Kid: kid,
				Kty: "RSA",
				Alg: "RS256",
				Use: "sig",
				N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
			},
		},
	}

	fetchCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			fetchCount++
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(jwks)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	v := &ClerkVerifier{
		issuer:     "https://" + server.URL[7:],
		jwksURL:    server.URL + "/.well-known/jwks.json",
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]*rsa.PublicKey),
	}

	// Fetch the key twice
	ctx := context.Background()
	_, err = v.getKey(ctx, kid)
	if err != nil {
		t.Fatalf("first getKey failed: %v", err)
	}

	_, err = v.getKey(ctx, kid)
	if err != nil {
		t.Fatalf("second getKey failed: %v", err)
	}

	// Should only have fetched once due to caching
	if fetchCount != 1 {
		t.Errorf("expected 1 fetch, got %d", fetchCount)
	}
}

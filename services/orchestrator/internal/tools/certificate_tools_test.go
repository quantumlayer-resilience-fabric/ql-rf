package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ListCertificatesTool Tests
// =============================================================================

func TestListCertificatesTool_Metadata(t *testing.T) {
	tool := &ListCertificatesTool{}

	assert.Equal(t, "list_certificates", tool.Name())
	assert.Contains(t, tool.Description(), "certificates")
	assert.Equal(t, RiskReadOnly, tool.Risk())
	assert.Equal(t, ScopeOrganization, tool.Scope())
	assert.True(t, tool.Idempotent())
	assert.False(t, tool.RequiresApproval())
}

func TestListCertificatesTool_Parameters(t *testing.T) {
	tool := &ListCertificatesTool{}
	params := tool.Parameters()

	require.NotNil(t, params)
	assert.Equal(t, "object", params["type"])

	props, ok := params["properties"].(map[string]interface{})
	require.True(t, ok)

	// Verify expected parameters exist
	expectedParams := []string{"status", "platform", "expiring_within_days", "common_name", "limit"}
	for _, param := range expectedParams {
		_, exists := props[param]
		assert.True(t, exists, "expected parameter %s to exist", param)
	}

	// Verify status enum
	statusProp := props["status"].(map[string]interface{})
	statusEnum := statusProp["enum"].([]string)
	assert.Contains(t, statusEnum, "active")
	assert.Contains(t, statusEnum, "expiring_soon")
	assert.Contains(t, statusEnum, "expired")
	assert.Contains(t, statusEnum, "revoked")

	// Verify platform enum
	platformProp := props["platform"].(map[string]interface{})
	platformEnum := platformProp["enum"].([]string)
	assert.Contains(t, platformEnum, "aws")
	assert.Contains(t, platformEnum, "azure")
	assert.Contains(t, platformEnum, "gcp")
	assert.Contains(t, platformEnum, "k8s")
	assert.Contains(t, platformEnum, "vsphere")
}

// Note: Execute with nil DB would panic - these tools require a valid database connection.
// Integration tests with a real/mock database should be used for Execute tests.

// =============================================================================
// GetCertificateDetailsTool Tests
// =============================================================================

func TestGetCertificateDetailsTool_Metadata(t *testing.T) {
	tool := &GetCertificateDetailsTool{}

	assert.Equal(t, "get_certificate_details", tool.Name())
	assert.Contains(t, tool.Description(), "certificate")
	assert.Equal(t, RiskReadOnly, tool.Risk())
	assert.Equal(t, ScopeAsset, tool.Scope())
	assert.True(t, tool.Idempotent())
	assert.False(t, tool.RequiresApproval())
}

func TestGetCertificateDetailsTool_Parameters(t *testing.T) {
	tool := &GetCertificateDetailsTool{}
	params := tool.Parameters()

	require.NotNil(t, params)
	assert.Equal(t, "object", params["type"])

	props, ok := params["properties"].(map[string]interface{})
	require.True(t, ok)

	// Verify expected parameters exist
	_, hasCertID := props["certificate_id"]
	_, hasCommonName := props["common_name"]
	assert.True(t, hasCertID, "expected certificate_id parameter")
	assert.True(t, hasCommonName, "expected common_name parameter")
}

func TestGetCertificateDetailsTool_Execute_MissingParams(t *testing.T) {
	tool := &GetCertificateDetailsTool{db: nil}

	// Test with empty params - should require either certificate_id or common_name
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate_id or common_name is required")
}

// =============================================================================
// MapCertificateUsageTool Tests
// =============================================================================

func TestMapCertificateUsageTool_Metadata(t *testing.T) {
	tool := &MapCertificateUsageTool{}

	assert.Equal(t, "map_certificate_usage", tool.Name())
	assert.Contains(t, tool.Description(), "blast radius")
	assert.Equal(t, RiskReadOnly, tool.Risk())
	assert.Equal(t, ScopeOrganization, tool.Scope())
	assert.True(t, tool.Idempotent())
	assert.False(t, tool.RequiresApproval())
}

func TestMapCertificateUsageTool_Parameters(t *testing.T) {
	tool := &MapCertificateUsageTool{}
	params := tool.Parameters()

	require.NotNil(t, params)
	assert.Equal(t, "object", params["type"])

	props, ok := params["properties"].(map[string]interface{})
	require.True(t, ok)

	// Verify expected parameters exist
	_, hasCertID := props["certificate_id"]
	_, hasCommonName := props["common_name"]
	assert.True(t, hasCertID, "expected certificate_id parameter")
	assert.True(t, hasCommonName, "expected common_name parameter")
}

func TestMapCertificateUsageTool_Execute_MissingParams(t *testing.T) {
	tool := &MapCertificateUsageTool{db: nil}

	// Test with empty params - should require either certificate_id or common_name
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate_id or common_name is required")
}

// =============================================================================
// GenerateCertRenewalPlanTool Tests
// =============================================================================

func TestGenerateCertRenewalPlanTool_Metadata(t *testing.T) {
	tool := &GenerateCertRenewalPlanTool{}

	assert.Equal(t, "generate_cert_renewal_plan", tool.Name())
	assert.Contains(t, tool.Description(), "renewal")
	assert.Equal(t, RiskPlanOnly, tool.Risk())
	assert.Equal(t, ScopeOrganization, tool.Scope())
	assert.True(t, tool.Idempotent())
	assert.False(t, tool.RequiresApproval())
}

func TestGenerateCertRenewalPlanTool_Parameters(t *testing.T) {
	tool := &GenerateCertRenewalPlanTool{}
	params := tool.Parameters()

	require.NotNil(t, params)
	assert.Equal(t, "object", params["type"])

	props, ok := params["properties"].(map[string]interface{})
	require.True(t, ok)

	// Verify expected parameters exist
	_, hasCertID := props["certificate_id"]
	_, hasRenewalType := props["renewal_type"]
	_, hasStrategy := props["strategy"]
	assert.True(t, hasCertID, "expected certificate_id parameter")
	assert.True(t, hasRenewalType, "expected renewal_type parameter")
	assert.True(t, hasStrategy, "expected strategy parameter")

	// Verify renewal_type enum
	renewalTypeProp := props["renewal_type"].(map[string]interface{})
	renewalTypeEnum := renewalTypeProp["enum"].([]string)
	assert.Contains(t, renewalTypeEnum, "auto")
	assert.Contains(t, renewalTypeEnum, "manual")
	assert.Contains(t, renewalTypeEnum, "emergency")

	// Verify strategy enum
	strategyProp := props["strategy"].(map[string]interface{})
	strategyEnum := strategyProp["enum"].([]string)
	assert.Contains(t, strategyEnum, "rolling")
	assert.Contains(t, strategyEnum, "blue_green")
	assert.Contains(t, strategyEnum, "immediate")

	// Verify required fields
	required := params["required"].([]string)
	assert.Contains(t, required, "certificate_id")
}

func TestGenerateCertRenewalPlanTool_Execute_MissingCertID(t *testing.T) {
	tool := &GenerateCertRenewalPlanTool{db: nil}

	// Test with empty params - should require certificate_id
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate_id is required")

	// Test with empty string certificate_id
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"certificate_id": "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate_id is required")
}

// =============================================================================
// ProposeCertRotationTool Tests
// =============================================================================

func TestProposeCertRotationTool_Metadata(t *testing.T) {
	tool := &ProposeCertRotationTool{}

	assert.Equal(t, "propose_cert_rotation", tool.Name())
	assert.Contains(t, tool.Description(), "rotation")
	assert.Equal(t, RiskStateChangeProd, tool.Risk())
	assert.Equal(t, ScopeOrganization, tool.Scope())
	assert.False(t, tool.Idempotent())
	assert.True(t, tool.RequiresApproval())
}

func TestProposeCertRotationTool_Parameters(t *testing.T) {
	tool := &ProposeCertRotationTool{}
	params := tool.Parameters()

	require.NotNil(t, params)
	assert.Equal(t, "object", params["type"])

	props, ok := params["properties"].(map[string]interface{})
	require.True(t, ok)

	// Verify expected parameters exist
	_, hasCertID := props["certificate_id"]
	_, hasRotationType := props["rotation_type"]
	_, hasPlan := props["plan"]
	_, hasReason := props["reason"]
	assert.True(t, hasCertID, "expected certificate_id parameter")
	assert.True(t, hasRotationType, "expected rotation_type parameter")
	assert.True(t, hasPlan, "expected plan parameter")
	assert.True(t, hasReason, "expected reason parameter")

	// Verify rotation_type enum
	rotationTypeProp := props["rotation_type"].(map[string]interface{})
	rotationTypeEnum := rotationTypeProp["enum"].([]string)
	assert.Contains(t, rotationTypeEnum, "renewal")
	assert.Contains(t, rotationTypeEnum, "replacement")
	assert.Contains(t, rotationTypeEnum, "emergency")
	assert.Contains(t, rotationTypeEnum, "scheduled")

	// Verify required fields
	required := params["required"].([]string)
	assert.Contains(t, required, "certificate_id")
	assert.Contains(t, required, "rotation_type")
}

func TestProposeCertRotationTool_Execute_MissingCertID(t *testing.T) {
	tool := &ProposeCertRotationTool{db: nil}

	// Test with empty params - should require certificate_id
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate_id is required")
}

// =============================================================================
// ValidateTLSHandshakeTool Tests
// =============================================================================

func TestValidateTLSHandshakeTool_Metadata(t *testing.T) {
	tool := &ValidateTLSHandshakeTool{}

	assert.Equal(t, "validate_tls_handshake", tool.Name())
	assert.Contains(t, tool.Description(), "TLS")
	assert.Equal(t, RiskReadOnly, tool.Risk())
	assert.Equal(t, ScopeAsset, tool.Scope())
	assert.True(t, tool.Idempotent())
	assert.False(t, tool.RequiresApproval())
}

func TestValidateTLSHandshakeTool_Parameters(t *testing.T) {
	tool := &ValidateTLSHandshakeTool{}
	params := tool.Parameters()

	require.NotNil(t, params)
	assert.Equal(t, "object", params["type"])

	props, ok := params["properties"].(map[string]interface{})
	require.True(t, ok)

	// Verify expected parameters exist
	_, hasEndpoint := props["endpoint"]
	_, hasExpectedCN := props["expected_cn"]
	_, hasExpectedSAN := props["expected_san"]
	_, hasMinTLSVersion := props["min_tls_version"]
	assert.True(t, hasEndpoint, "expected endpoint parameter")
	assert.True(t, hasExpectedCN, "expected expected_cn parameter")
	assert.True(t, hasExpectedSAN, "expected expected_san parameter")
	assert.True(t, hasMinTLSVersion, "expected min_tls_version parameter")

	// Verify min_tls_version enum
	tlsVersionProp := props["min_tls_version"].(map[string]interface{})
	tlsVersionEnum := tlsVersionProp["enum"].([]string)
	assert.Contains(t, tlsVersionEnum, "TLS1.2")
	assert.Contains(t, tlsVersionEnum, "TLS1.3")

	// Verify required fields
	required := params["required"].([]string)
	assert.Contains(t, required, "endpoint")
}

func TestValidateTLSHandshakeTool_Execute_MissingEndpoint(t *testing.T) {
	tool := &ValidateTLSHandshakeTool{db: nil}

	// Test with empty params - should require endpoint
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")

	// Test with empty string endpoint
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"endpoint": "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")
}

func TestValidateTLSHandshakeTool_Execute_Success(t *testing.T) {
	tool := &ValidateTLSHandshakeTool{db: nil}

	// The tool returns simulated response
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"endpoint": "api.example.com:443",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "api.example.com:443", resultMap["endpoint"])
	assert.Equal(t, "success", resultMap["status"])
	assert.Equal(t, "TLS1.3", resultMap["tls_version"])

	// Verify certificate info is present
	cert := resultMap["certificate"].(map[string]interface{})
	assert.NotEmpty(t, cert["common_name"])
	assert.NotEmpty(t, cert["issuer"])
	assert.True(t, cert["chain_valid"].(bool))

	// Verify validation info
	validation := resultMap["validation"].(map[string]interface{})
	assert.True(t, validation["handshake_success"].(bool))
	assert.True(t, validation["chain_valid"].(bool))
	assert.True(t, validation["hostname_match"].(bool))
	assert.True(t, validation["not_expired"].(bool))
}

// =============================================================================
// Tool Interface Tests
// =============================================================================

func TestCertificateTools_ImplementToolInterface(t *testing.T) {
	// Verify all certificate tools implement the Tool interface
	var _ Tool = (*ListCertificatesTool)(nil)
	var _ Tool = (*GetCertificateDetailsTool)(nil)
	var _ Tool = (*MapCertificateUsageTool)(nil)
	var _ Tool = (*GenerateCertRenewalPlanTool)(nil)
	var _ Tool = (*ProposeCertRotationTool)(nil)
	var _ Tool = (*ValidateTLSHandshakeTool)(nil)
}

func TestCertificateTools_RiskLevels(t *testing.T) {
	tools := []struct {
		tool         Tool
		expectedRisk RiskLevel
	}{
		{&ListCertificatesTool{}, RiskReadOnly},
		{&GetCertificateDetailsTool{}, RiskReadOnly},
		{&MapCertificateUsageTool{}, RiskReadOnly},
		{&GenerateCertRenewalPlanTool{}, RiskPlanOnly},
		{&ProposeCertRotationTool{}, RiskStateChangeProd},
		{&ValidateTLSHandshakeTool{}, RiskReadOnly},
	}

	for _, tt := range tools {
		t.Run(tt.tool.Name(), func(t *testing.T) {
			assert.Equal(t, tt.expectedRisk, tt.tool.Risk())
		})
	}
}

func TestCertificateTools_ApprovalRequirements(t *testing.T) {
	// Only ProposeCertRotationTool should require approval
	tools := []struct {
		tool            Tool
		requiresApproval bool
	}{
		{&ListCertificatesTool{}, false},
		{&GetCertificateDetailsTool{}, false},
		{&MapCertificateUsageTool{}, false},
		{&GenerateCertRenewalPlanTool{}, false},
		{&ProposeCertRotationTool{}, true},
		{&ValidateTLSHandshakeTool{}, false},
	}

	for _, tt := range tools {
		t.Run(tt.tool.Name(), func(t *testing.T) {
			assert.Equal(t, tt.requiresApproval, tt.tool.RequiresApproval())
		})
	}
}

func TestCertificateTools_Idempotency(t *testing.T) {
	// Only ProposeCertRotationTool is not idempotent
	tools := []struct {
		tool        Tool
		idempotent  bool
	}{
		{&ListCertificatesTool{}, true},
		{&GetCertificateDetailsTool{}, true},
		{&MapCertificateUsageTool{}, true},
		{&GenerateCertRenewalPlanTool{}, true},
		{&ProposeCertRotationTool{}, false},
		{&ValidateTLSHandshakeTool{}, true},
	}

	for _, tt := range tools {
		t.Run(tt.tool.Name(), func(t *testing.T) {
			assert.Equal(t, tt.idempotent, tt.tool.Idempotent())
		})
	}
}

func TestCertificateTools_Scopes(t *testing.T) {
	tools := []struct {
		tool          Tool
		expectedScope Scope
	}{
		{&ListCertificatesTool{}, ScopeOrganization},
		{&GetCertificateDetailsTool{}, ScopeAsset},
		{&MapCertificateUsageTool{}, ScopeOrganization},
		{&GenerateCertRenewalPlanTool{}, ScopeOrganization},
		{&ProposeCertRotationTool{}, ScopeOrganization},
		{&ValidateTLSHandshakeTool{}, ScopeAsset},
	}

	for _, tt := range tools {
		t.Run(tt.tool.Name(), func(t *testing.T) {
			assert.Equal(t, tt.expectedScope, tt.tool.Scope())
		})
	}
}

// =============================================================================
// Tool Names Uniqueness Tests
// =============================================================================

func TestCertificateTools_UniqueNames(t *testing.T) {
	tools := []Tool{
		&ListCertificatesTool{},
		&GetCertificateDetailsTool{},
		&MapCertificateUsageTool{},
		&GenerateCertRenewalPlanTool{},
		&ProposeCertRotationTool{},
		&ValidateTLSHandshakeTool{},
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		name := tool.Name()
		if names[name] {
			t.Errorf("duplicate tool name: %s", name)
		}
		names[name] = true
	}

	assert.Len(t, names, 6, "expected 6 unique certificate tools")
}

// =============================================================================
// Tool Descriptions Tests
// =============================================================================

func TestCertificateTools_DescriptionsNotEmpty(t *testing.T) {
	tools := []Tool{
		&ListCertificatesTool{},
		&GetCertificateDetailsTool{},
		&MapCertificateUsageTool{},
		&GenerateCertRenewalPlanTool{},
		&ProposeCertRotationTool{},
		&ValidateTLSHandshakeTool{},
	}

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			assert.NotEmpty(t, tool.Description())
		})
	}
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

func TestValidateTLSHandshakeTool_Execute_WithCancelledContext(t *testing.T) {
	tool := &ValidateTLSHandshakeTool{db: nil}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The simulated tool doesn't check context, but this tests the pattern
	result, err := tool.Execute(ctx, map[string]interface{}{
		"endpoint": "api.example.com:443",
	})

	// Current implementation returns simulated data regardless of context
	// In a real implementation, this would return ctx.Err()
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// =============================================================================
// Parameter Validation Tests
// =============================================================================

func TestCertificateTools_ParametersHaveTypeDefinitions(t *testing.T) {
	tools := []Tool{
		&ListCertificatesTool{},
		&GetCertificateDetailsTool{},
		&MapCertificateUsageTool{},
		&GenerateCertRenewalPlanTool{},
		&ProposeCertRotationTool{},
		&ValidateTLSHandshakeTool{},
	}

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			params := tool.Parameters()
			require.NotNil(t, params)

			// All tools should have type=object
			assert.Equal(t, "object", params["type"])

			// All tools should have properties
			props, ok := params["properties"].(map[string]interface{})
			assert.True(t, ok, "parameters should have properties")
			assert.NotEmpty(t, props, "parameters should not be empty")
		})
	}
}

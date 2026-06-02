// PR #28 / CONN-008 — LIVE Azure Run Command client.
//
// SAFETY (READ THIS BEFORE EDITING):
// This is THE single file in the tools package allowed to construct
// `armcompute.NewVirtualMachineRunCommandsClient`. The structural safety
// test in no_azure_runcommand_sdk_import_test.go grants this file an
// allowlist exception by name; the complementary positive test
// TestLiveAzureRunCommandClient_IsTheOnlyFileReferencingSDKConstructor
// asserts (positive direction) that this file DOES reference the
// constructor. Both tests run on every push.
//
// Why this discipline matters: PR #27 (dry-run only) made the
// state-change Azure Run Command call structurally unreachable from the
// tools package. PR #28 re-introduces reachability deliberately and ONLY
// through this file, so auditors can grep for the constructor name and
// see exactly one match.
//
// Live mode is gated at four layers, every layer independently enforced:
//
//  1. Boot env opt-in: RF_CONNECTORS_AZURE_ALLOW_LIVE_RUN_COMMAND=true.
//     Without it, registerAzureLiveRunCommandTools in main.go is a no-op
//     and this client is never constructed.
//  2. Mock-conflict refusal: if FallbackToMock is also true, main.go
//     exits 1 at boot. The combination is incoherent.
//  3. Per-VM whitelist: parsed from
//     RF_CONNECTORS_AZURE_LIVE_RUN_COMMAND_WHITELIST_VMS at boot.
//     Format: "rg/vm-name" pairs. Empty + ALLOW_LIVE_RUN_COMMAND=true
//     also fails boot.
//  4. Two-approver workflow: enforced by OPA policy
//     (tool_authorization.rego from PR #21) AND by the coApproveTask
//     handler. The live tool's Execute method also re-checks the
//     whitelist before calling the SDK, as defense in depth.
package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"

	pkgconfig "github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// LiveAzureRunCommandClient SENDS run commands to Azure VMs. Distinct
// from AzureRunCommandClient (PR #27) which only BUILDS plans. A
// type-level separation so a caller expecting dry-run shape can't
// accidentally call SendRunCommand.
type LiveAzureRunCommandClient interface {
	// SendRunCommand fires the Azure state-change Run Command path
	// against the plan's VM. Returns the AWS-style "operation name"
	// (Azure's long-running-operation identifier) on success. Validates
	// against the whitelist (set at construction) before calling the SDK.
	SendRunCommand(ctx context.Context, plan *AzureRunCommandPlan) (operationName string, err error)
}

// liveAzureWhitelistEnvVar is the env name parsed at boot for the
// per-VM allowlist. Comma-separated "rg/vm-name" pairs. Exported as a
// const so the boot path and the docs/README reference the same string.
const liveAzureWhitelistEnvVar = "RF_CONNECTORS_AZURE_LIVE_RUN_COMMAND_WHITELIST_VMS"

// realLiveAzureRunCommandClient is the production client. The SDK
// constructor invocation in this file is the only one in the package;
// this struct holds the live state-change client.
type realLiveAzureRunCommandClient struct {
	sdk       *armcompute.VirtualMachineRunCommandsClient
	whitelist []string
	log       *logger.Logger
}

// NewLiveAzureRunCommandClient builds a real, credential-validated live
// Azure Run Command client. Reuses the same azidentity flow as PR #26's
// NewRealAzureClient, plus a cheap subscription check, so credential
// misconfigurations fail at boot, not on first SendRunCommand.
func NewLiveAzureRunCommandClient(ctx context.Context, cfg pkgconfig.AzureConfig, whitelist []string, log *logger.Logger) (LiveAzureRunCommandClient, error) {
	if len(whitelist) == 0 {
		return nil, fmt.Errorf("NewLiveAzureRunCommandClient: refuse to construct with empty whitelist")
	}
	if cfg.TenantID == "" || cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.SubscriptionID == "" {
		return nil, fmt.Errorf("azure credentials not configured: TenantID, ClientID, ClientSecret, and SubscriptionID are required")
	}

	cred, err := azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("create azure credentials: %w", err)
	}

	// THE allowlisted SDK constructor — this is the ONLY place in the
	// tools package that creates the state-change Run Command client.
	rcClient, err := armcompute.NewVirtualMachineRunCommandsClient(cfg.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create azure run-command client: %w", err)
	}

	// Boot-time credential validation: list the supported run-command
	// documents for a representative location. The list-by-location call
	// surfaces auth failure without firing any actual command.
	bootCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pager := rcClient.NewListPager("eastus", nil)
	if _, err := pager.NextPage(bootCtx); err != nil {
		// A "NotFound" or "ResourceNotFound" for the dummy location is
		// fine — the call reached the Azure plane, which proves auth.
		// Anything else means the credential is broken.
		if !strings.Contains(strings.ToLower(err.Error()), "notfound") &&
			!strings.Contains(strings.ToLower(err.Error()), "resourcenotfound") {
			return nil, fmt.Errorf("azure live-mode credential validation failed: %w", err)
		}
	}

	return &realLiveAzureRunCommandClient{
		sdk:       rcClient,
		whitelist: append([]string(nil), whitelist...),
		log:       log.WithComponent("azure-runcommand-live"),
	}, nil
}

// SendRunCommand fires the live Azure Run Command path. Validates the
// target VM against the whitelist (defense in depth — the tool's Execute
// validates first), then begins the long-running operation. Returns the
// operation name verbatim. The actual command execution is asynchronous;
// callers that want completion status should poll via the returned name.
func (c *realLiveAzureRunCommandClient) SendRunCommand(ctx context.Context, plan *AzureRunCommandPlan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("SendRunCommand: nil plan")
	}
	target := plan.ResourceGroup + "/" + plan.VMName
	if !slices.Contains(c.whitelist, target) {
		return "", fmt.Errorf("vm %q is not on the live-run-command whitelist (have %d allowed)", target, len(c.whitelist))
	}

	// Translate our plan into the SDK's input shape. Keep this confined
	// here — the rest of the codebase never sees the SDK type.
	rcInput := armcompute.VirtualMachineRunCommand{
		Location: &plan.ResourceGroup, // location is RG-scoped; SDK will pick from VM if unset
		Properties: &armcompute.VirtualMachineRunCommandProperties{
			Source: &armcompute.VirtualMachineRunCommandScriptSource{
				Script: &plan.Script,
			},
			TimeoutInSeconds: &plan.TimeoutSeconds,
		},
	}

	op, err := c.sdk.BeginCreateOrUpdate(ctx, plan.ResourceGroup, plan.VMName, plan.CommandID, rcInput, nil)
	if err != nil {
		return "", fmt.Errorf("azure BeginCreateOrUpdate rg=%s vm=%s: %w", plan.ResourceGroup, plan.VMName, err)
	}
	// BeginCreateOrUpdate returns an *runtime.Poller — the operation
	// name lives in its private state. Use the resume token as a stable
	// public identifier. (Azure's SDK doesn't expose op name directly.)
	tok, tokErr := op.ResumeToken()
	if tokErr != nil {
		return "", fmt.Errorf("azure poller resume token: %w", tokErr)
	}
	c.log.Info("live azure run-command fired",
		"resource_group", plan.ResourceGroup,
		"vm_name", plan.VMName,
		"command_id", plan.CommandID,
	)
	return tok, nil
}

// mockLiveAzureRunCommandClient is the test/CI variant. Lives in the
// same file as the real client so a single place defines "what does
// live mode look like." Validates the whitelist (mirroring the real
// client's defense in depth) but never touches the Azure SDK — the
// response is a deterministic `op-azmock-<hex>` id so audit-log
// consumers can recognize mock origin.
type mockLiveAzureRunCommandClient struct {
	whitelist []string
}

// NewMockLiveAzureRunCommandClient constructs a mock that respects the
// same whitelist contract as the real client. Used by tests and by
// local smoke when RF_CONNECTORS_AZURE_LIVE_RUN_COMMAND_CLIENT_MODE=mock.
func NewMockLiveAzureRunCommandClient(whitelist []string) LiveAzureRunCommandClient {
	return &mockLiveAzureRunCommandClient{whitelist: append([]string(nil), whitelist...)}
}

// SendRunCommand returns op-azmock-<random hex> without calling the SDK.
// The whitelist check runs first so tests against the mock catch
// whitelist regressions just as the real client would.
func (m *mockLiveAzureRunCommandClient) SendRunCommand(_ context.Context, plan *AzureRunCommandPlan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("SendRunCommand: nil plan")
	}
	target := plan.ResourceGroup + "/" + plan.VMName
	if !slices.Contains(m.whitelist, target) {
		return "", fmt.Errorf("vm %q is not on the live-run-command whitelist (have %d allowed)", target, len(m.whitelist))
	}
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("mock SendRunCommand: read entropy: %w", err)
	}
	return "op-azmock-" + hex.EncodeToString(buf), nil
}

// LoadAzureLiveWhitelistFromEnv parses the comma-separated whitelist
// from the env var. Returns nil for an unset or empty var. Exposed so
// the boot path can produce the same parsed result the tests use.
func LoadAzureLiveWhitelistFromEnv() []string {
	v := os.Getenv(liveAzureWhitelistEnvVar)
	return parseAzureLiveWhitelistCSV(v)
}

// parseAzureLiveWhitelistCSV is the pure helper underneath
// LoadAzureLiveWhitelistFromEnv. Each entry is a "rg/vm" pair; the
// helper strips whitespace and skips empty entries but doesn't enforce
// the slash shape — the caller does that when comparing against
// `plan.ResourceGroup + "/" + plan.VMName`.
func parseAzureLiveWhitelistCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// PR #26 / CONN-006 — Azure client interface for real cloud tool invocations.
//
// Mirrors aws_client.go from PR #19 / CONN-001: a narrow interface (just
// ListVMs for now) with a real implementation backed by the Azure SDK and
// a deterministic mock for unit tests + CI.
//
// Credential model: service principal (TenantID + ClientID + ClientSecret
// + SubscriptionID), the same shape the existing connectors service uses
// (services/connectors/internal/azure/client.go). Boot validates the
// credential by listing the subscription's locations — a cheap call that
// surfaces auth misconfiguration loudly rather than waiting for first
// real invocation.
//
// PR #27 will introduce an Azure dry-run tool (`azure_run_command`) and
// PR #28 the live variant (`azure_run_command_live`) following the SSM
// arc pattern (PR #20 + #21). This file's `AzureClient` interface stays
// narrow for read-only operations; live-mutation clients live in their
// own files for structural-safety isolation, the same way live_ssm_client.go
// is the sole importer of the SSM SDK in PR #21.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// safeStr returns *p or "" — the Azure SDK uses *string for most fields
// to distinguish empty from absent, and dereferencing nil panics. This
// is the canonical projection.
func safeStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// AzureClient is the narrow interface the Azure tools call. Keeping it
// small means the mock is trivial and the surface to audit stays tight.
// Each method is read-only by API contract for PR #26.
type AzureClient interface {
	// ListVMs lists virtual machines in the configured subscription. The
	// returned slice is a redacted projection — security-sensitive fields
	// (VM extensions, network interface IDs, disk encryption keys) are
	// deliberately omitted from the audit-log-bound shape.
	ListVMs(ctx context.Context) ([]AzureVM, error)
}

// AzureVM is the redacted projection of an armcompute.VirtualMachine we
// surface to the audit log and the UI. Fields are deliberately boring —
// no extensions, no disk URIs, no network identity material — so an
// audit log leak is low-risk.
type AzureVM struct {
	Name          string            `json:"name"`
	ResourceGroup string            `json:"resource_group"`
	Location      string            `json:"location"`
	Size          string            `json:"size,omitempty"`
	State         string            `json:"state,omitempty"`
	PowerState    string            `json:"power_state,omitempty"`
	OSType        string            `json:"os_type,omitempty"`
	OSDiskName    string            `json:"os_disk_name,omitempty"`
	ProvisionedAt string            `json:"provisioned_at,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

// realAzureClient wraps armcompute.VirtualMachinesClient. Constructed
// once at boot; the SubscriptionID is fixed (Azure resource hierarchy
// is per-subscription) so a single client services all listing calls.
type realAzureClient struct {
	vms *armcompute.VirtualMachinesClient
	log *logger.Logger
}

// NewRealAzureClient builds a real Azure VM-listing client. Validates
// credentials at construction via a cheap subscription-locations list,
// returning an error if credentials are missing or unusable. The caller
// decides whether to fall back to a mock client or skip tool registration.
//
// The validation call is one Resource Manager request per orchestrator
// boot — negligible cost, and the difference between "boots cleanly but
// tool silently broken" and "boots cleanly and tool actually works" at
// demo time.
func NewRealAzureClient(ctx context.Context, cfg config.AzureConfig, log *logger.Logger) (AzureClient, error) {
	if cfg.TenantID == "" || cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.SubscriptionID == "" {
		return nil, fmt.Errorf("azure credentials not configured: TenantID, ClientID, ClientSecret, and SubscriptionID are required")
	}

	cred, err := azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("create azure credentials: %w", err)
	}

	vmClient, err := armcompute.NewVirtualMachinesClient(cfg.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create azure vm client: %w", err)
	}

	// Boot-time credential validation. NewListAllPager doesn't make a
	// network call by itself; we have to advance the pager once to
	// confirm the credential works. Doing this with a short timeout
	// keeps a slow / blackholed network from blocking orchestrator boot.
	bootCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pager := vmClient.NewListAllPager(nil)
	if _, err := pager.NextPage(bootCtx); err != nil {
		return nil, fmt.Errorf("azure credential validation (list-vms) failed: %w", err)
	}

	return &realAzureClient{
		vms: vmClient,
		log: log.WithComponent("azure-tools"),
	}, nil
}

// ListVMs pages through every VM in the subscription. The orchestrator's
// audit log holds the redacted projection — see AzureVM for which fields
// are intentionally dropped.
func (c *realAzureClient) ListVMs(ctx context.Context) ([]AzureVM, error) {
	var results []AzureVM
	pager := c.vms.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("azure list vms page: %w", err)
		}
		for _, vm := range page.Value {
			if vm == nil {
				continue
			}
			results = append(results, normalizeAzureVM(vm))
		}
	}
	return results, nil
}

// normalizeAzureVM projects the (large) armcompute.VirtualMachine type
// into the small AzureVM shape we surface. Drops fields with any security
// relevance — extensions, disk URIs, NIC IDs — so the audit log doesn't
// carry them.
func normalizeAzureVM(vm *armcompute.VirtualMachine) AzureVM {
	out := AzureVM{
		Name: safeStr(vm.Name),
	}
	if vm.Location != nil {
		out.Location = *vm.Location
	}
	// Resource group is encoded in the ID:
	// /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/virtualMachines/{vm}
	out.ResourceGroup = parseResourceGroupFromID(safeStr(vm.ID))
	if vm.Properties != nil {
		if vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
			out.Size = string(*vm.Properties.HardwareProfile.VMSize)
		}
		if vm.Properties.ProvisioningState != nil {
			out.State = *vm.Properties.ProvisioningState
		}
		if vm.Properties.StorageProfile != nil {
			if osDisk := vm.Properties.StorageProfile.OSDisk; osDisk != nil {
				if osDisk.Name != nil {
					out.OSDiskName = *osDisk.Name
				}
				if osDisk.OSType != nil {
					out.OSType = string(*osDisk.OSType)
				}
			}
		}
		if vm.Properties.TimeCreated != nil {
			out.ProvisionedAt = vm.Properties.TimeCreated.UTC().Format(time.RFC3339)
		}
	}
	if len(vm.Tags) > 0 {
		out.Tags = make(map[string]string, len(vm.Tags))
		for k, v := range vm.Tags {
			out.Tags[k] = safeStr(v)
		}
	}
	return out
}

// parseResourceGroupFromID extracts the resource group name from an
// Azure resource ID. Returns "" if the ID is malformed — the tool's
// output shape carries an empty string rather than failing the call.
func parseResourceGroupFromID(id string) string {
	const marker = "/resourceGroups/"
	idx := indexOfFold(id, marker)
	if idx < 0 {
		return ""
	}
	rest := id[idx+len(marker):]
	end := indexOf(rest, "/")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// indexOfFold is a tiny strings.Index wrapper that's case-insensitive on
// "resourceGroups" (Azure's canonical casing is `resourceGroups` but the
// SDK is robust to either; we match either to avoid a subtle parse miss).
func indexOfFold(s, sub string) int {
	// Fast path: exact case match.
	if idx := indexOf(s, sub); idx >= 0 {
		return idx
	}
	// Fallback: lowercase compare.
	ls := toLower(s)
	lsub := toLower(sub)
	return indexOf(ls, lsub)
}

func indexOf(s, sub string) int {
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

// mockAzureClient returns a deterministic two-VM fixture. Used by unit
// tests and by CI (RF_CONNECTORS_AZURE_FALLBACK_TO_MOCK=true). The mock
// emits the same canonical resource-group path shape so audit-log
// consumers see realistic-looking data without an Azure account.
type mockAzureClient struct{}

// NewMockAzureClient constructs the deterministic fixture client.
func NewMockAzureClient() AzureClient {
	return &mockAzureClient{}
}

// ListVMs returns a fixed pair of mock VMs regardless of context. The
// names contain `mock-` so the audit log's origin is obvious. Tags
// include a `mock_origin` marker for SQL queries that need to filter.
func (m *mockAzureClient) ListVMs(_ context.Context) ([]AzureVM, error) {
	return []AzureVM{
		{
			Name:          "mock-vm-prod-01",
			ResourceGroup: "rg-mock-prod",
			Location:      "eastus",
			Size:          "Standard_D2s_v5",
			State:         "Succeeded",
			OSType:        "Linux",
			OSDiskName:    "mock-vm-prod-01-osdisk",
			ProvisionedAt: "2026-04-10T11:23:00Z",
			Tags: map[string]string{
				"env":         "prod",
				"team":        "platform",
				"mock_origin": "ql-rf-test",
			},
		},
		{
			Name:          "mock-vm-stage-02",
			ResourceGroup: "rg-mock-stage",
			Location:      "westeurope",
			Size:          "Standard_B2s",
			State:         "Succeeded",
			OSType:        "Windows",
			OSDiskName:    "mock-vm-stage-02-osdisk",
			ProvisionedAt: "2026-05-02T09:11:00Z",
			Tags: map[string]string{
				"env":         "stage",
				"team":        "platform",
				"mock_origin": "ql-rf-test",
			},
		},
	}, nil
}

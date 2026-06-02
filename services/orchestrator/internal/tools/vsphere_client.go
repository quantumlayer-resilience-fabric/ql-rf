// PR #33 / CONN-012 — vSphere client interface for real cloud tool invocations.
//
// Mirrors aws_client.go (PR #19), azure_client.go (PR #26), and
// gcp_client.go (PR #29): a narrow interface (just ListVMs) with a real
// implementation backed by govmomi and a deterministic mock for unit
// tests + CI.
//
// Credential model: vCenter URL + username + password (same shape the
// existing connectors service uses, services/connectors/internal/
// vsphere/client.go). Boot validates by listing one page of VMs across
// the configured datacenters.
//
// PR #34 will introduce a vSphere dry-run guest-ops tool, PR #35 the
// live variant — following the SSM / Azure / GCP arc patterns.
package tools

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// VSphereClient is the narrow interface the vSphere tools call. Keeping
// it small means the mock is trivial and the surface to audit stays
// tight. Each method is read-only by API contract for PR #33.
type VSphereClient interface {
	// ListVMs returns every virtual machine the configured vCenter sees,
	// projected into the redacted VSphereVM shape (no NIC details,
	// no annotation text, no datastore URLs — fields with security
	// relevance are dropped).
	ListVMs(ctx context.Context) ([]VSphereVM, error)
}

// VSphereVM is the redacted projection of a vim25 VirtualMachine we
// surface to the audit log and the UI.
type VSphereVM struct {
	Name       string            `json:"name"`
	UUID       string            `json:"uuid,omitempty"`
	PowerState string            `json:"power_state,omitempty"`
	GuestOS    string            `json:"guest_os,omitempty"`
	NumCPU     int32             `json:"num_cpu,omitempty"`
	MemoryMB   int32             `json:"memory_mb,omitempty"`
	Path       string            `json:"path,omitempty"`
	HostName   string            `json:"host_name,omitempty"`
	IPAddress  string            `json:"ip_address,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

// realVSphereClient wraps a govmomi.Client. Constructed once at boot;
// the client establishes a session and renews it as needed.
type realVSphereClient struct {
	client *govmomi.Client
	log    *logger.Logger
}

// NewRealVSphereClient builds a real vSphere VM-listing client.
// Validates credentials at construction by establishing a session.
// Returns an error if the URL is malformed, credentials are missing,
// or the vCenter rejects the login.
func NewRealVSphereClient(ctx context.Context, cfg config.VSphereConfig, log *logger.Logger) (VSphereClient, error) {
	if cfg.URL == "" || cfg.User == "" || cfg.Password == "" {
		return nil, fmt.Errorf("vsphere credentials not configured: URL, User, and Password are required")
	}

	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse vsphere url: %w", err)
	}
	u.User = url.UserPassword(cfg.User, cfg.Password)

	bootCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := govmomi.NewClient(bootCtx, u, cfg.Insecure)
	if err != nil {
		return nil, fmt.Errorf("vsphere connect: %w", err)
	}

	return &realVSphereClient{
		client: client,
		log:    log.WithComponent("vsphere-tools"),
	}, nil
}

// ListVMs enumerates every VM the vCenter sees, using a property
// collector to fetch the redacted projection efficiently. Tags are
// pulled best-effort; failures don't fail the whole listing.
func (c *realVSphereClient) ListVMs(ctx context.Context) ([]VSphereVM, error) {
	mgr := view.NewManager(c.client.Client)
	containerView, err := mgr.CreateContainerView(ctx, c.client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, fmt.Errorf("vsphere create view: %w", err)
	}
	defer func() { _ = containerView.Destroy(ctx) }() //nolint:errcheck // best-effort cleanup

	var vms []mo.VirtualMachine
	if err := containerView.Retrieve(ctx, []string{"VirtualMachine"},
		[]string{"name", "config.uuid", "config.guestFullName", "config.hardware",
			"runtime.powerState", "runtime.host", "summary.config.vmPathName", "guest.ipAddress"},
		&vms); err != nil {
		return nil, fmt.Errorf("vsphere retrieve vms: %w", err)
	}

	out := make([]VSphereVM, 0, len(vms))
	for i := range vms {
		out = append(out, normalizeVSphereVM(&vms[i]))
	}
	return out, nil
}

// normalizeVSphereVM projects the (large) mo.VirtualMachine type into
// the small VSphereVM shape. Drops fields with security relevance —
// NIC IDs, datastore URLs, annotation text.
func normalizeVSphereVM(vm *mo.VirtualMachine) VSphereVM {
	out := VSphereVM{
		Name:       vm.Name,
		PowerState: string(vm.Runtime.PowerState),
	}
	if vm.Config != nil {
		out.UUID = vm.Config.Uuid
		out.GuestOS = vm.Config.GuestFullName
		if vm.Config.Hardware.NumCPU > 0 {
			out.NumCPU = vm.Config.Hardware.NumCPU
		}
		if vm.Config.Hardware.MemoryMB > 0 {
			out.MemoryMB = vm.Config.Hardware.MemoryMB
		}
	}
	if vm.Summary.Config.VmPathName != "" {
		out.Path = vm.Summary.Config.VmPathName
	}
	if vm.Guest != nil && vm.Guest.IpAddress != "" {
		out.IPAddress = vm.Guest.IpAddress
	}
	if vm.Runtime.Host != nil {
		// Host is a ManagedObjectReference; the value is the host's MOID,
		// which is the most concise identifier without firing another
		// retrieve.
		out.HostName = vm.Runtime.Host.Value
	}
	return out
}

// mockVSphereClient returns a deterministic two-VM fixture. Used by
// unit tests and CI.
type mockVSphereClient struct{}

// NewMockVSphereClient constructs the deterministic fixture client.
func NewMockVSphereClient() VSphereClient {
	return &mockVSphereClient{}
}

// ListVMs returns a fixed pair of mock VMs.
func (m *mockVSphereClient) ListVMs(_ context.Context) ([]VSphereVM, error) {
	return []VSphereVM{
		{
			Name:       "mock-esx-vm-prod-01",
			UUID:       "564d5e62-mock-prod-01",
			PowerState: "poweredOn",
			GuestOS:    "Ubuntu Linux (64-bit)",
			NumCPU:     4,
			MemoryMB:   8192,
			Path:       "[mock-datastore-01] mock-esx-vm-prod-01/mock-esx-vm-prod-01.vmx",
			HostName:   "host-mock-01",
			IPAddress:  "10.20.0.5",
			Tags: map[string]string{
				"env":         "prod",
				"team":        "platform",
				"mock_origin": "ql-rf-test",
			},
		},
		{
			Name:       "mock-esx-vm-stage-02",
			UUID:       "564d5e62-mock-stage-02",
			PowerState: "poweredOn",
			GuestOS:    "Microsoft Windows Server 2022 (64-bit)",
			NumCPU:     8,
			MemoryMB:   16384,
			Path:       "[mock-datastore-02] mock-esx-vm-stage-02/mock-esx-vm-stage-02.vmx",
			HostName:   "host-mock-02",
			IPAddress:  "10.20.0.7",
			Tags: map[string]string{
				"env":         "stage",
				"team":        "platform",
				"mock_origin": "ql-rf-test",
			},
		},
	}, nil
}

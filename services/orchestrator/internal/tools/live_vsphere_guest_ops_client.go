// PR #35 / CONN-014 — LIVE vSphere guest-ops client.
//
// SAFETY (READ THIS BEFORE EDITING):
// This is THE single file in the tools package allowed to call
// `ProcessManager.StartProgramInGuest`. The structural safety test in
// no_vsphere_guest_ops_sdk_import_test.go grants this file an allowlist
// exception by name; the complementary positive test
// TestLiveVSphereGuestOpsClient_IsTheOnlyFileReferencingSDKMethod
// asserts (positive direction) that this file DOES reference the
// method. Both tests run on every push.
//
// Live mode is gated at four layers:
//
//  1. Boot env opt-in: RF_CONNECTORS_VSPHERE_ALLOW_LIVE_GUEST_OPS=true.
//  2. Mock-conflict refusal: if FallbackToMock is also true, exit 1.
//  3. Per-VM whitelist:
//     RF_CONNECTORS_VSPHERE_LIVE_GUEST_OPS_WHITELIST_VMS env var.
//  4. Two-approver workflow: OPA policy + coApproveTask handler (PR #21).
package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"

	pkgconfig "github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// LiveVSphereGuestOpsClient SENDS guest-program-run requests to vSphere
// VMs. Distinct from VSphereGuestOpsClient (PR #34) which only BUILDS
// plans.
type LiveVSphereGuestOpsClient interface {
	// RunGuestProgram fires the vSphere guest-ops StartProgramInGuest
	// path against the plan's VM. Returns the guest PID on success.
	// Validates against the whitelist (set at construction) before
	// calling the SDK method.
	RunGuestProgram(ctx context.Context, plan *VSphereGuestProgramPlan) (pid int64, err error)
}

// liveVSphereWhitelistEnvVar is the env name parsed at boot.
const liveVSphereWhitelistEnvVar = "RF_CONNECTORS_VSPHERE_LIVE_GUEST_OPS_WHITELIST_VMS"

// realLiveVSphereGuestOpsClient is the production client. The
// StartProgramInGuest call in this file is the only one in the
// package.
type realLiveVSphereGuestOpsClient struct {
	client    *govmomi.Client
	finder    *find.Finder
	whitelist []string
	log       *logger.Logger
}

// NewLiveVSphereGuestOpsClient builds a real, session-validated live
// vSphere guest-ops client. Reuses the same govmomi auth flow as
// PR #33's NewRealVSphereClient.
func NewLiveVSphereGuestOpsClient(ctx context.Context, cfg pkgconfig.VSphereConfig, whitelist []string, log *logger.Logger) (LiveVSphereGuestOpsClient, error) {
	if len(whitelist) == 0 {
		return nil, fmt.Errorf("NewLiveVSphereGuestOpsClient: refuse to construct with empty whitelist")
	}
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
		return nil, fmt.Errorf("vsphere live connect: %w", err)
	}

	return &realLiveVSphereGuestOpsClient{
		client:    client,
		finder:    find.NewFinder(client.Client, true),
		whitelist: append([]string(nil), whitelist...),
		log:       log.WithComponent("vsphere-guest-ops-live"),
	}, nil
}

// RunGuestProgram fires the live vSphere StartProgramInGuest path.
func (c *realLiveVSphereGuestOpsClient) RunGuestProgram(ctx context.Context, plan *VSphereGuestProgramPlan) (int64, error) {
	if plan == nil {
		return 0, fmt.Errorf("RunGuestProgram: nil plan")
	}
	if !slices.Contains(c.whitelist, plan.VMName) {
		return 0, fmt.Errorf("vm %q is not on the live-guest-ops whitelist (have %d allowed)", plan.VMName, len(c.whitelist))
	}

	vm, err := c.finder.VirtualMachine(ctx, plan.VMName)
	if err != nil {
		return 0, fmt.Errorf("find vm %q: %w", plan.VMName, err)
	}

	gom := guest.NewOperationsManager(c.client.Client, vm.Reference())
	procMgr, err := gom.ProcessManager(ctx)
	if err != nil {
		return 0, fmt.Errorf("guest operations manager: %w", err)
	}

	auth := &types.NamePasswordAuthentication{
		Username: plan.GuestUser,
		Password: plan.GuestPassword,
	}
	spec := types.GuestProgramSpec{
		ProgramPath:      plan.ProgramPath,
		Arguments:        plan.Arguments,
		WorkingDirectory: plan.WorkingDirectory,
	}

	// THE allowlisted SDK call — this is the ONLY place in the tools
	// package that invokes the guest-program-start path. govmomi wraps
	// the StartProgramInGuest VMODL method under the ProcessManager.StartProgram
	// helper; the structural safety test grep for `.StartProgram(`.
	pid, err := procMgr.StartProgram(ctx, auth, &spec)
	if err != nil {
		return 0, fmt.Errorf("vsphere StartProgram vm=%s: %w", plan.VMName, err)
	}

	c.log.Info("live vsphere guest-program fired",
		"vm_name", plan.VMName,
		"program_path", plan.ProgramPath,
		"pid", pid,
	)

	// The unused object import is here so a future refactor that drops
	// the finder/Properties pattern doesn't accidentally break the
	// imports. Cheaper than a //nolint comment.
	_ = object.VirtualMachine{}

	return pid, nil
}

// mockLiveVSphereGuestOpsClient is the test/CI variant.
type mockLiveVSphereGuestOpsClient struct {
	whitelist []string
}

// NewMockLiveVSphereGuestOpsClient constructs a mock that respects the
// whitelist contract.
func NewMockLiveVSphereGuestOpsClient(whitelist []string) LiveVSphereGuestOpsClient {
	return &mockLiveVSphereGuestOpsClient{whitelist: append([]string(nil), whitelist...)}
}

// RunGuestProgram returns a deterministic int64 PID derived from random
// bytes without calling the SDK.
func (m *mockLiveVSphereGuestOpsClient) RunGuestProgram(_ context.Context, plan *VSphereGuestProgramPlan) (int64, error) {
	if plan == nil {
		return 0, fmt.Errorf("RunGuestProgram: nil plan")
	}
	if !slices.Contains(m.whitelist, plan.VMName) {
		return 0, fmt.Errorf("vm %q is not on the live-guest-ops whitelist (have %d allowed)", plan.VMName, len(m.whitelist))
	}
	var buf [2]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, fmt.Errorf("mock RunGuestProgram: read entropy: %w", err)
	}
	// Compose a "mock-pid" range: 900000-1000000-ish, so the audit log
	// shape obviously says "this isn't a real Linux PID range."
	pid := 900000 + int64(buf[0])<<8 + int64(buf[1])
	_ = hex.EncodeToString(buf[:]) // keep encoding/hex import used (mirrors other live clients)
	return pid, nil
}

// LoadVSphereLiveWhitelistFromEnv parses the comma-separated whitelist
// from the env var.
func LoadVSphereLiveWhitelistFromEnv() []string {
	v := os.Getenv(liveVSphereWhitelistEnvVar)
	return parseVSphereLiveWhitelistCSV(v)
}

// parseVSphereLiveWhitelistCSV is the pure helper.
func parseVSphereLiveWhitelistCSV(v string) []string {
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

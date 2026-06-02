// PR #40 / CONN-017 — unit tests for K8sApplyLiveTool + live client +
// whitelist helpers.
package tools

import (
	"context"
	"strings"
	"testing"

	pkgconfig "github.com/quantumlayerhq/ql-rf/pkg/config"
)

func TestParseK8sLiveWhitelistCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{",,,", nil},
		{"prod/Deployment", []string{"prod/Deployment"}},
		{" prod/Deployment , stage/ConfigMap ", []string{"prod/Deployment", "stage/ConfigMap"}},
	}
	for _, c := range cases {
		got := parseK8sLiveWhitelistCSV(c.in)
		if !stringSliceEq(got, c.want) {
			t.Errorf("parseK8sLiveWhitelistCSV(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestMockLiveK8sApplyClient_AcceptsWhitelistedTarget(t *testing.T) {
	c := NewMockLiveK8sApplyClient([]string{"prod/Deployment"})
	plan := &K8sApplyPlan{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Namespace:  "prod",
		Name:       "web",
	}
	res, err := c.Apply(context.Background(), plan, `{"apiVersion":"apps/v1","kind":"Deployment"}`)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !strings.HasPrefix(res.UID, "mock-") {
		t.Errorf("UID = %q, want mock-* prefix", res.UID)
	}
}

func TestMockLiveK8sApplyClient_RejectsNonWhitelistedTarget(t *testing.T) {
	c := NewMockLiveK8sApplyClient([]string{"prod/Deployment"})
	plan := &K8sApplyPlan{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "stage", Name: "web"}
	if _, err := c.Apply(context.Background(), plan, "{}"); err == nil {
		t.Fatal("expected whitelist rejection, got nil")
	}
}

func TestMockLiveK8sApplyClient_RejectsUnsupportedKind(t *testing.T) {
	c := NewMockLiveK8sApplyClient([]string{"prod/CustomThing"})
	plan := &K8sApplyPlan{APIVersion: "v1", Kind: "CustomThing", Namespace: "prod", Name: "x"}
	if _, err := c.Apply(context.Background(), plan, "{}"); err == nil {
		t.Fatal("expected unsupported-kind rejection, got nil")
	}
}

func TestMockLiveK8sApplyClient_RejectsNilPlan(t *testing.T) {
	c := NewMockLiveK8sApplyClient([]string{"prod/Deployment"})
	if _, err := c.Apply(context.Background(), nil, "{}"); err == nil {
		t.Fatal("expected nil-plan rejection, got nil")
	}
}

func TestNewLiveK8sApplyClient_RefusesEmptyWhitelist(t *testing.T) {
	_, err := NewLiveK8sApplyClient(context.Background(), pkgconfig.K8sConfig{}, nil, testLoggerForK8s())
	if err == nil {
		t.Fatal("expected refusal for empty whitelist, got nil")
	}
	if !strings.Contains(err.Error(), "whitelist") {
		t.Errorf("error should mention whitelist; got %q", err.Error())
	}
}

func TestK8sApplyLiveTool_FiresMockAndReturnsLiveEnvelope(t *testing.T) {
	whitelist := []string{"prod/Deployment"}
	tool := NewK8sApplyLiveTool(
		NewRealK8sApplyClient(testLoggerForK8s()),
		NewMockLiveK8sApplyClient(whitelist),
	)
	got, err := tool.Execute(context.Background(), map[string]any{
		"namespace":     "prod",
		"manifest":      `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"web"},"spec":{"replicas":3}}`,
		"field_manager": "ql-rf",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	envelope, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got = %T, want map[string]any", got)
	}
	if envelope["dry_run"] != false {
		t.Errorf("dry_run = %v, want false", envelope["dry_run"])
	}
	if envelope["real_changes"] != true {
		t.Errorf("real_changes = %v, want true", envelope["real_changes"])
	}
	uid, _ := envelope["applied_uid"].(string)
	if !strings.HasPrefix(uid, "mock-") {
		t.Errorf("applied_uid = %q, want mock-* prefix", uid)
	}
}

func TestK8sApplyLiveTool_RejectsNonWhitelistedTarget(t *testing.T) {
	tool := NewK8sApplyLiveTool(
		NewRealK8sApplyClient(testLoggerForK8s()),
		NewMockLiveK8sApplyClient([]string{"prod/Deployment"}),
	)
	_, err := tool.Execute(context.Background(), map[string]any{
		"namespace":     "stage",
		"manifest":      `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"web"}}`,
		"field_manager": "ql-rf",
	})
	if err == nil {
		t.Fatal("expected whitelist rejection, got nil")
	}
}

func TestK8sApplyLiveTool_RisksAndApprovalShape(t *testing.T) {
	tool := NewK8sApplyLiveTool(
		NewRealK8sApplyClient(testLoggerForK8s()),
		NewMockLiveK8sApplyClient([]string{"prod/Deployment"}),
	)
	if tool.Risk() != RiskStateChangeProd {
		t.Errorf("Risk = %v, want RiskStateChangeProd", tool.Risk())
	}
	if !tool.RequiresApproval() {
		t.Error("RequiresApproval should be true")
	}
	if tool.Idempotent() {
		t.Error("Idempotent should be false")
	}
	if tool.Name() != "k8s_apply_live" {
		t.Errorf("Name = %q, want k8s_apply_live", tool.Name())
	}
}

func TestRegisterK8sLiveApplyTools_NoOpOnNilClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForK8s())
	r.RegisterK8sLiveApplyTools(nil, NewMockLiveK8sApplyClient([]string{"prod/Deployment"}))
	if _, ok := r.Get("k8s_apply_live"); ok {
		t.Error("should not register with nil dry client")
	}
	r.RegisterK8sLiveApplyTools(NewRealK8sApplyClient(testLoggerForK8s()), nil)
	if _, ok := r.Get("k8s_apply_live"); ok {
		t.Error("should not register with nil live client")
	}
}

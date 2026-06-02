// PR #39 / CONN-016 — unit tests for the K8s server-side-apply dry-run tool.
package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

func testLoggerForK8s() *logger.Logger {
	return logger.New("error", "json")
}

func TestK8sApplyDryRunTool_Metadata(t *testing.T) {
	tool := NewK8sApplyDryRunTool(NewMockK8sApplyClient())
	if tool.Name() != "k8s_apply" {
		t.Errorf("Name = %q, want k8s_apply", tool.Name())
	}
	if tool.Risk() != RiskStateChangeProd {
		t.Errorf("Risk = %v, want RiskStateChangeProd", tool.Risk())
	}
	if !tool.RequiresApproval() {
		t.Error("RequiresApproval should be true for state-change tool")
	}
}

func TestK8sApplyDryRunTool_ExecuteBuildsPlan(t *testing.T) {
	tool := NewK8sApplyDryRunTool(NewRealK8sApplyClient(testLoggerForK8s()))

	manifest := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"web","namespace":"prod"},"spec":{"replicas":3}}`

	got, err := tool.Execute(context.Background(), map[string]any{
		"namespace":     "prod",
		"manifest":      manifest,
		"field_manager": "ql-rf",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	envelope, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("envelope type = %T, want map[string]any", got)
	}
	if envelope["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", envelope["dry_run"])
	}
	if envelope["real_changes"] != false {
		t.Errorf("real_changes = %v, want false", envelope["real_changes"])
	}

	plan, ok := envelope["apply_plan"].(*K8sApplyPlan)
	if !ok {
		t.Fatalf("apply_plan type = %T, want *K8sApplyPlan", envelope["apply_plan"])
	}
	if plan.Kind != "Deployment" {
		t.Errorf("plan.Kind = %q, want Deployment", plan.Kind)
	}
	if plan.Name != "web" {
		t.Errorf("plan.Name = %q, want web", plan.Name)
	}
	if plan.Namespace != "prod" {
		t.Errorf("plan.Namespace = %q, want prod", plan.Namespace)
	}
	if !plan.DryRun || plan.RealChanges {
		t.Error("plan must have DryRun=true, RealChanges=false")
	}
}

func TestK8sApplyDryRunTool_RejectsMissingFields(t *testing.T) {
	tool := NewK8sApplyDryRunTool(NewRealK8sApplyClient(testLoggerForK8s()))
	cases := []struct {
		name   string
		params map[string]any
		want   string
	}{
		{"missing namespace", map[string]any{"manifest": "{}", "field_manager": "ql-rf"}, "namespace"},
		{"missing manifest", map[string]any{"namespace": "prod", "field_manager": "ql-rf"}, "manifest"},
		{"missing field_manager", map[string]any{"namespace": "prod", "manifest": "{}"}, "field_manager"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), c.params)
			if err == nil {
				t.Fatalf("expected error mentioning %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error should mention %q; got %q", c.want, err.Error())
			}
		})
	}
}

func TestK8sApplyClient_RejectsInvalidNamespace(t *testing.T) {
	c := NewRealK8sApplyClient(testLoggerForK8s())
	_, err := c.BuildApplyPlan(context.Background(), K8sApplyRequest{
		Namespace:    "Prod-Caps", // uppercase violates DNS-1123 label
		Manifest:     `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"x"}}`,
		FieldManager: "ql-rf",
	})
	if err == nil {
		t.Fatal("expected error for invalid namespace")
	}
	if !strings.Contains(err.Error(), "namespace") {
		t.Errorf("error should mention namespace; got %q", err.Error())
	}
}

func TestK8sApplyClient_RejectsManifestNamespaceMismatch(t *testing.T) {
	c := NewRealK8sApplyClient(testLoggerForK8s())
	_, err := c.BuildApplyPlan(context.Background(), K8sApplyRequest{
		Namespace:    "prod",
		Manifest:     `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"x","namespace":"staging"}}`,
		FieldManager: "ql-rf",
	})
	if err == nil {
		t.Fatal("expected error for cross-namespace mismatch")
	}
	if !strings.Contains(err.Error(), "namespace") {
		t.Errorf("error should mention namespace mismatch; got %q", err.Error())
	}
}

func TestK8sApplyClient_RejectsInvalidJSON(t *testing.T) {
	c := NewRealK8sApplyClient(testLoggerForK8s())
	_, err := c.BuildApplyPlan(context.Background(), K8sApplyRequest{
		Namespace:    "prod",
		Manifest:     `not-valid-json`,
		FieldManager: "ql-rf",
	})
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
}

func TestK8sApplyClient_RequiresAPIVersionAndKind(t *testing.T) {
	c := NewRealK8sApplyClient(testLoggerForK8s())
	_, err := c.BuildApplyPlan(context.Background(), K8sApplyRequest{
		Namespace:    "prod",
		Manifest:     `{"metadata":{"name":"web"}}`,
		FieldManager: "ql-rf",
	})
	if err == nil {
		t.Fatal("expected error for missing apiVersion/kind")
	}
}

func TestK8sApplyClient_RequiresName(t *testing.T) {
	c := NewRealK8sApplyClient(testLoggerForK8s())
	_, err := c.BuildApplyPlan(context.Background(), K8sApplyRequest{
		Namespace:    "prod",
		Manifest:     `{"apiVersion":"v1","kind":"ConfigMap","metadata":{}}`,
		FieldManager: "ql-rf",
	})
	if err == nil {
		t.Fatal("expected error for missing metadata.name")
	}
}

func TestMockK8sApplyClient_ReturnsDeterministicPlan(t *testing.T) {
	m := NewMockK8sApplyClient()
	plan, err := m.BuildApplyPlan(context.Background(), K8sApplyRequest{Namespace: "prod"})
	if err != nil {
		t.Fatalf("BuildApplyPlan: %v", err)
	}
	if plan.Name != "mock-app" || plan.Kind != "Deployment" {
		t.Errorf("mock plan unexpected: name=%q kind=%q", plan.Name, plan.Kind)
	}
	if !plan.DryRun {
		t.Error("mock plan must be dry-run")
	}
}

func TestRegisterK8sApplyDryRunTools_NoOpOnNilClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForK8s())
	r.RegisterK8sApplyDryRunTools(nil)
	if _, ok := r.Get("k8s_apply"); ok {
		t.Error("k8s_apply should NOT be registered when client is nil")
	}
}

func TestRegisterK8sApplyDryRunTools_RegistersWithClient(t *testing.T) {
	r := NewRegistry(nil, testLoggerForK8s())
	r.RegisterK8sApplyDryRunTools(NewMockK8sApplyClient())
	if _, ok := r.Get("k8s_apply"); !ok {
		t.Error("k8s_apply should be registered with non-nil client")
	}
}

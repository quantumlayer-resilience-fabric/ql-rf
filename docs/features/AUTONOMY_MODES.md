# Autonomy Modes

> Configuration for AI-assisted automation levels

**Location:** `services/orchestrator/internal/autonomy/modes.go`

---

## Overview

Autonomy modes control how much human intervention is required for AI-generated operations. Each organization can configure their preferred autonomy level based on risk tolerance and compliance requirements.

---

## Modes

| Mode | Description | Auto-Execute | Human Approval |
|------|-------------|--------------|----------------|
| `plan_only` | AI generates plans, humans execute manually | Never | Always required |
| `approve_all` | Human approval required for all operations | Never | Always required |
| `canary_only` | Auto-execute canary phases, approve full rollout | Canary only | Full rollout |
| `risk_based` | Auto-execute based on risk score | Low/Medium risk | High/Critical risk |
| `full_auto` | Full automation with guardrails | All operations | Alerts only |

---

## Configuration

```go
type AutonomyConfig struct {
    Mode              AutonomyMode  // plan_only, approve_all, canary_only, risk_based, full_auto
    MaxRiskLevel      string        // For risk_based: maximum auto-approve level
    RequireCanary     bool          // For full_auto: require canary deployment first
    NotifyOnAuto      bool          // Send notifications on auto-execution
    AllowedHours      []int         // Time windows for auto-execution (0-23)
    ExcludedEnvs      []string      // Environments always requiring approval
}
```

### Environment Variables

```bash
RF_AUTONOMY_MODE=risk_based
RF_AUTONOMY_MAX_RISK=medium
RF_AUTONOMY_REQUIRE_CANARY=true
RF_AUTONOMY_NOTIFY=true
RF_AUTONOMY_ALLOWED_HOURS=0,1,2,3,4,5  # Off-peak hours
RF_AUTONOMY_EXCLUDED_ENVS=production,dr
```

---

## Decision Flow

```
Task Created → Risk Assessment → Autonomy Check → Decision
                                       │
                    ┌──────────────────┼──────────────────┐
                    ▼                  ▼                  ▼
              AutoApprove       RequireApproval        Block
              (proceed)         (wait for human)    (escalate)
```

### Decision Logic

```go
func (e *Engine) ShouldAutoApprove(task *Task, riskLevel string) Decision {
    switch e.config.Mode {
    case PlanOnly:
        return RequireApproval
    case ApproveAll:
        return RequireApproval
    case CanaryOnly:
        if task.Phase == "canary" {
            return AutoApprove
        }
        return RequireApproval
    case RiskBased:
        if riskLevel <= e.config.MaxRiskLevel {
            return AutoApprove
        }
        return RequireApproval
    case FullAuto:
        if e.config.RequireCanary && !task.HasCanary {
            return RequireApproval
        }
        return AutoApprove
    }
}
```

---

## Usage

### Get Current Mode

```bash
curl http://localhost:8083/api/v1/ai/settings \
  -H "Authorization: Bearer $TOKEN"
```

### Update Autonomy Settings

```bash
curl -X PUT http://localhost:8083/api/v1/ai/settings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "autonomy_mode": "risk_based",
    "max_risk_level": "medium",
    "require_canary": true
  }'
```

---

## Recommended Configurations

### Enterprise (Conservative)
```yaml
mode: approve_all
excluded_envs: [production, dr, staging]
```

### Startup (Aggressive)
```yaml
mode: risk_based
max_risk_level: medium
require_canary: true
allowed_hours: [0, 1, 2, 3, 4, 5]
```

### Mature DevOps (Balanced)
```yaml
mode: canary_only
notify_on_auto: true
excluded_envs: [production]
```

---

## See Also

- [Risk Scoring](RISK_SCORING.md) - How risk levels are calculated
- [Canary Analysis](CANARY_ANALYSIS.md) - Canary deployment validation
- [ARCHITECTURE.md](../ARCHITECTURE.md) - Full system architecture

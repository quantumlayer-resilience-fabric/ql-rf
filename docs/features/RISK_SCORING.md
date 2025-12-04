# Risk Scoring

> AI-powered risk assessment for infrastructure operations

**Location:** `services/orchestrator/internal/risk/scorer.go`

---

## Overview

The Risk Scoring service calculates operation risk to inform autonomy decisions, batch sizing, and rollout strategies. Risk scores combine multiple weighted factors to produce a 0-100 score.

---

## Risk Formula

```
Risk Score = Σ(Factor Weight × Factor Score) × Environment Multiplier
```

---

## Risk Factors

| Factor | Weight | Description | Scoring |
|--------|--------|-------------|---------|
| Asset Criticality | 20% | Environment importance | Prod: 100, DR: 70, Staging: 40, Dev: 20 |
| Change Type | 20% | Operation severity | Reimage: 100, Patch: 60, Config: 40, Status: 20 |
| Blast Radius | 15% | Percentage of environment affected | Linear 0-100 |
| Time of Day | 10% | Business hours risk | Business: 100, Off-hours: 30 |
| Historical Failure | 15% | Past failure rate for similar ops | 0-100 based on history |
| Rollback Complexity | 10% | Difficulty of reverting | Easy: 20, Medium: 50, Hard: 100 |
| Dependencies | 5% | Count of dependent services | 10 points per dependency |
| Compliance Impact | 5% | Affects compliance controls | Yes: 100, No: 0 |

---

## Environment Multipliers

| Environment | Multiplier | Rationale |
|-------------|------------|-----------|
| Production | 1.5x | Highest business impact |
| DR | 1.2x | Critical for recovery |
| Staging | 1.0x | Pre-production validation |
| Development | 0.5x | Low business impact |

---

## Risk Levels

| Level | Score Range | Action | Color |
|-------|-------------|--------|-------|
| **Low** | 0-24 | Safe for automation | Green |
| **Medium** | 25-49 | Automation with monitoring | Yellow |
| **High** | 50-74 | Requires approval | Orange |
| **Critical** | 75-100 | Escalation required | Red |

---

## Batch Size Recommendations

Based on risk level, the system recommends batch sizes for rollout:

| Risk Level | Batch Size | Wait Time |
|------------|------------|-----------|
| Low | 25% | 5 minutes |
| Medium | 10% | 10 minutes |
| High | 5% | 15 minutes |
| Critical | 1 asset | 30 minutes |

---

## API Usage

### Get Risk Summary

```bash
curl http://localhost:8080/api/v1/risk/summary \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "overall_score": 42,
  "level": "medium",
  "asset_count": 156,
  "critical_count": 3,
  "high_count": 12,
  "medium_count": 45,
  "low_count": 96
}
```

### Get Top Risks

```bash
curl "http://localhost:8080/api/v1/risk/top?limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "assets": [
    {
      "asset_id": "uuid",
      "name": "prod-web-01",
      "risk_score": 78,
      "risk_level": "critical",
      "factors": {
        "drift_age_days": 45,
        "vulnerability_count": 12,
        "critical_vulns": 3,
        "environment": "production"
      }
    }
  ]
}
```

### Calculate Task Risk

```bash
curl -X POST http://localhost:8083/api/v1/ai/risk/calculate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "uuid",
    "change_type": "patch",
    "target_assets": ["asset-1", "asset-2"],
    "environment": "production"
  }'
```

---

## Configuration

```go
type RiskConfig struct {
    Weights       map[string]float64  // Factor weights
    Multipliers   map[string]float64  // Environment multipliers
    Thresholds    RiskThresholds      // Level boundaries
    BatchSizes    map[string]float64  // Batch size recommendations
}

type RiskThresholds struct {
    Low      int  // 0-24
    Medium   int  // 25-49
    High     int  // 50-74
    Critical int  // 75-100
}
```

### Environment Variables

```bash
RF_RISK_WEIGHT_CRITICALITY=0.20
RF_RISK_WEIGHT_CHANGE_TYPE=0.20
RF_RISK_WEIGHT_BLAST_RADIUS=0.15
RF_RISK_WEIGHT_TIME_OF_DAY=0.10
RF_RISK_WEIGHT_HISTORICAL=0.15
RF_RISK_WEIGHT_ROLLBACK=0.10
RF_RISK_WEIGHT_DEPENDENCIES=0.05
RF_RISK_WEIGHT_COMPLIANCE=0.05
```

---

## Integration with Autonomy

The risk score directly feeds into autonomy decisions:

```go
riskLevel := scorer.Calculate(task)
decision := autonomy.ShouldAutoApprove(task, riskLevel)

switch decision {
case AutoApprove:
    executor.Start(task)
case RequireApproval:
    notifier.RequestApproval(task)
case Block:
    notifier.Escalate(task)
}
```

---

## See Also

- [Autonomy Modes](AUTONOMY_MODES.md) - How risk informs automation
- [Canary Analysis](CANARY_ANALYSIS.md) - Risk mitigation through canary
- [ARCHITECTURE.md](../ARCHITECTURE.md) - Full system architecture

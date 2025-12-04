package autonomy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/risk"
)

func TestMode_Constants(t *testing.T) {
	assert.Equal(t, Mode("plan_only"), ModePlanOnly)
	assert.Equal(t, Mode("approve_all"), ModeApproveAll)
	assert.Equal(t, Mode("canary_only"), ModeCanaryOnly)
	assert.Equal(t, Mode("risk_based"), ModeRiskBased)
	assert.Equal(t, Mode("full_auto"), ModeFullAuto)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, ModeApproveAll, cfg.Mode)
	assert.Equal(t, float64(30), cfg.RiskThreshold)
	assert.Equal(t, 10, cfg.CanaryPercentage)
	assert.Equal(t, 100, cfg.MaxAssetsPerExecution)
	assert.Equal(t, 5, cfg.MaxCriticalAssets)
	assert.True(t, cfg.RequireCanarySuccess)
	assert.True(t, cfg.NotifyOnAutoExecution)
	assert.True(t, cfg.AllowRollback)
	assert.Contains(t, cfg.AllowedPlatforms, "aws")
	assert.Contains(t, cfg.AllowedPlatforms, "azure")
	assert.Contains(t, cfg.AllowedPlatforms, "gcp")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid mode",
			cfg: Config{
				Mode:              Mode("invalid"),
				RiskThreshold:     30,
				CanaryPercentage:  10,
				MaxAssetsPerExecution: 100,
			},
			wantErr: true,
			errMsg:  "invalid autonomy mode",
		},
		{
			name: "risk threshold too high",
			cfg: Config{
				Mode:              ModeApproveAll,
				RiskThreshold:     150,
				CanaryPercentage:  10,
				MaxAssetsPerExecution: 100,
			},
			wantErr: true,
			errMsg:  "risk_threshold must be between 0 and 100",
		},
		{
			name: "risk threshold negative",
			cfg: Config{
				Mode:              ModeApproveAll,
				RiskThreshold:     -10,
				CanaryPercentage:  10,
				MaxAssetsPerExecution: 100,
			},
			wantErr: true,
			errMsg:  "risk_threshold must be between 0 and 100",
		},
		{
			name: "canary percentage too low",
			cfg: Config{
				Mode:              ModeApproveAll,
				RiskThreshold:     30,
				CanaryPercentage:  0,
				MaxAssetsPerExecution: 100,
			},
			wantErr: true,
			errMsg:  "canary_percentage must be between 1 and 50",
		},
		{
			name: "canary percentage too high",
			cfg: Config{
				Mode:              ModeApproveAll,
				RiskThreshold:     30,
				CanaryPercentage:  75,
				MaxAssetsPerExecution: 100,
			},
			wantErr: true,
			errMsg:  "canary_percentage must be between 1 and 50",
		},
		{
			name: "max assets zero",
			cfg: Config{
				Mode:              ModeApproveAll,
				RiskThreshold:     30,
				CanaryPercentage:  10,
				MaxAssetsPerExecution: 0,
			},
			wantErr: true,
			errMsg:  "max_assets_per_execution must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewController(t *testing.T) {
	log := logger.New("error", "text")
	cfg := DefaultConfig()

	ctrl := NewController(log, cfg)

	require.NotNil(t, ctrl)
	assert.Equal(t, cfg.Mode, ctrl.GetConfig().Mode)
}

func TestController_SetConfig(t *testing.T) {
	log := logger.New("error", "text")
	ctrl := NewController(log, DefaultConfig())

	newCfg := Config{
		Mode:              ModeFullAuto,
		RiskThreshold:     50,
		CanaryPercentage:  5,
		MaxAssetsPerExecution: 50,
	}

	ctrl.SetConfig(newCfg)

	assert.Equal(t, ModeFullAuto, ctrl.GetConfig().Mode)
	assert.Equal(t, float64(50), ctrl.GetConfig().RiskThreshold)
}

func TestController_Decide_PlanOnly(t *testing.T) {
	log := logger.New("error", "text")
	cfg := DefaultConfig()
	cfg.Mode = ModePlanOnly

	ctrl := NewController(log, cfg)

	decision, err := ctrl.Decide(context.Background(), OperationContext{
		OperationType: "patch",
		Environment:   "staging",
		Platform:      "aws",
		AssetCount:    10,
	})

	require.NoError(t, err)
	assert.False(t, decision.CanAutoExecute)
	assert.True(t, decision.RequiresApproval)
	assert.Contains(t, decision.Reason, "Plan-only mode")
}

func TestController_Decide_ApproveAll(t *testing.T) {
	log := logger.New("error", "text")
	cfg := DefaultConfig()
	cfg.Mode = ModeApproveAll

	ctrl := NewController(log, cfg)

	decision, err := ctrl.Decide(context.Background(), OperationContext{
		OperationType: "drift-fix",
		Environment:   "staging",
		Platform:      "aws",
		AssetCount:    10,
	})

	require.NoError(t, err)
	assert.False(t, decision.CanAutoExecute)
	assert.True(t, decision.RequiresApproval)
	assert.Contains(t, decision.Reason, "Approve-all mode")
}

func TestController_Decide_CanaryOnly(t *testing.T) {
	log := logger.New("error", "text")
	cfg := DefaultConfig()
	cfg.Mode = ModeCanaryOnly
	cfg.CanaryPercentage = 10

	ctrl := NewController(log, cfg)

	decision, err := ctrl.Decide(context.Background(), OperationContext{
		OperationType: "patch",
		Environment:   "staging",
		Platform:      "aws",
		AssetCount:    100,
	})

	require.NoError(t, err)
	assert.True(t, decision.CanAutoExecute)
	assert.False(t, decision.RequiresApproval) // Canary auto-executes
	assert.True(t, decision.RequiresCanary)
	assert.Equal(t, 10, decision.CanaryPercentage)
	assert.Equal(t, 10, decision.MaxBatchSize) // 10% of 100
}

func TestController_Decide_RiskBased(t *testing.T) {
	log := logger.New("error", "text")
	cfg := DefaultConfig()
	cfg.Mode = ModeRiskBased
	cfg.RiskThreshold = 40

	ctrl := NewController(log, cfg)

	t.Run("low risk - auto execute", func(t *testing.T) {
		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType: "query",
			Environment:   "staging",
			Platform:      "aws",
			AssetCount:    10,
			RiskScore:     &risk.RiskScore{OverallScore: 20, Level: risk.RiskLevelLow},
		})

		require.NoError(t, err)
		assert.True(t, decision.CanAutoExecute)
		assert.False(t, decision.RequiresApproval)
	})

	t.Run("high risk - requires approval", func(t *testing.T) {
		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType: "patch",
			Environment:   "production",
			Platform:      "aws",
			AssetCount:    50,
			RiskScore:     &risk.RiskScore{OverallScore: 75, Level: risk.RiskLevelHigh},
		})

		require.NoError(t, err)
		assert.False(t, decision.CanAutoExecute)
		assert.True(t, decision.RequiresApproval)
		assert.True(t, decision.RequiresCanary)
	})

	t.Run("no risk score - defaults to approval", func(t *testing.T) {
		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType: "patch",
			Environment:   "staging",
			Platform:      "aws",
			AssetCount:    10,
		})

		require.NoError(t, err)
		assert.False(t, decision.CanAutoExecute)
		assert.True(t, decision.RequiresApproval)
	})
}

func TestController_Decide_FullAuto(t *testing.T) {
	log := logger.New("error", "text")
	cfg := DefaultConfig()
	cfg.Mode = ModeFullAuto

	ctrl := NewController(log, cfg)

	t.Run("staging - full auto", func(t *testing.T) {
		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType: "patch",
			Environment:   "staging",
			Platform:      "aws",
			AssetCount:    10,
		})

		require.NoError(t, err)
		assert.True(t, decision.CanAutoExecute)
		assert.False(t, decision.RequiresApproval)
	})

	t.Run("production - with canary", func(t *testing.T) {
		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType: "patch",
			Environment:   "production",
			Platform:      "aws",
			AssetCount:    50,
		})

		require.NoError(t, err)
		assert.True(t, decision.CanAutoExecute)
		assert.True(t, decision.RequiresCanary)
	})

	t.Run("critical risk - blocks full auto", func(t *testing.T) {
		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType: "patch",
			Environment:   "production",
			Platform:      "aws",
			AssetCount:    100,
			RiskScore:     &risk.RiskScore{OverallScore: 95, Level: risk.RiskLevelCritical},
		})

		require.NoError(t, err)
		assert.False(t, decision.CanAutoExecute)
		assert.True(t, decision.RequiresApproval)
	})
}

func TestController_Decide_Blockers(t *testing.T) {
	log := logger.New("error", "text")
	cfg := DefaultConfig()
	cfg.Mode = ModeFullAuto
	cfg.RequireApprovalFor = []string{"terminate", "delete"}
	cfg.MaxAssetsPerExecution = 50
	cfg.MaxCriticalAssets = 5
	cfg.AllowedPlatforms = []string{"aws", "azure"}

	ctrl := NewController(log, cfg)

	t.Run("blocked operation type", func(t *testing.T) {
		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType: "terminate",
			Environment:   "staging",
			Platform:      "aws",
			AssetCount:    10,
		})

		require.NoError(t, err)
		assert.False(t, decision.CanAutoExecute)
		assert.Len(t, decision.BlockedReasons, 1)
		assert.Contains(t, decision.BlockedReasons[0], "always requires approval")
	})

	t.Run("asset count exceeded", func(t *testing.T) {
		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType: "patch",
			Environment:   "staging",
			Platform:      "aws",
			AssetCount:    100,
		})

		require.NoError(t, err)
		assert.False(t, decision.CanAutoExecute)
		assert.Contains(t, decision.BlockedReasons[0], "exceeds limit")
	})

	t.Run("critical assets exceeded", func(t *testing.T) {
		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType:  "patch",
			Environment:    "staging",
			Platform:       "aws",
			AssetCount:     10,
			CriticalAssets: 10,
		})

		require.NoError(t, err)
		assert.False(t, decision.CanAutoExecute)
		assert.Contains(t, decision.BlockedReasons[0], "Critical assets")
	})

	t.Run("platform not allowed", func(t *testing.T) {
		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType: "patch",
			Environment:   "staging",
			Platform:      "vsphere",
			AssetCount:    10,
		})

		require.NoError(t, err)
		assert.False(t, decision.CanAutoExecute)
		assert.Contains(t, decision.BlockedReasons[0], "not in allowed list")
	})

	t.Run("cooldown period", func(t *testing.T) {
		lastExec := time.Now().Add(-1 * time.Minute)
		cfg.CooldownPeriod = 5 * time.Minute

		decision, err := ctrl.Decide(context.Background(), OperationContext{
			OperationType:     "patch",
			Environment:       "staging",
			Platform:          "aws",
			AssetCount:        10,
			LastExecutionTime: &lastExec,
		})

		require.NoError(t, err)
		assert.False(t, decision.CanAutoExecute)
		assert.Contains(t, decision.BlockedReasons[0], "Cooldown period")
	})
}

func TestTimeWindow(t *testing.T) {
	window := TimeWindow{
		Start:    "09:00",
		End:      "17:00",
		Days:     []string{"Monday", "Tuesday"},
		Timezone: "UTC",
	}

	assert.Equal(t, "09:00", window.Start)
	assert.Equal(t, "17:00", window.End)
	assert.Len(t, window.Days, 2)
}

func TestDecision_ToJSON(t *testing.T) {
	decision := &Decision{
		CanAutoExecute:   true,
		RequiresApproval: false,
		RequiresCanary:   true,
		CanaryPercentage: 10,
		MaxBatchSize:     50,
		Reason:           "Test decision",
	}

	data, err := decision.ToJSON()
	require.NoError(t, err)

	var parsed Decision
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, decision.CanAutoExecute, parsed.CanAutoExecute)
	assert.Equal(t, decision.RequiresCanary, parsed.RequiresCanary)
}

func TestModeDescriptions(t *testing.T) {
	descriptions := ModeDescriptions()

	assert.Contains(t, descriptions, ModePlanOnly)
	assert.Contains(t, descriptions, ModeApproveAll)
	assert.Contains(t, descriptions, ModeCanaryOnly)
	assert.Contains(t, descriptions, ModeRiskBased)
	assert.Contains(t, descriptions, ModeFullAuto)

	assert.Contains(t, descriptions[ModePlanOnly], "plans only")
	assert.Contains(t, descriptions[ModeFullAuto], "automatically")
}

func TestOperationContext(t *testing.T) {
	ctx := OperationContext{
		OperationType:  "patch",
		Environment:    "production",
		Platform:       "aws",
		AssetCount:     100,
		CriticalAssets: 5,
		RiskScore:      &risk.RiskScore{OverallScore: 50},
		ScheduledTime:  time.Now(),
	}

	assert.Equal(t, "patch", ctx.OperationType)
	assert.Equal(t, "production", ctx.Environment)
	assert.Equal(t, 100, ctx.AssetCount)
	assert.NotNil(t, ctx.RiskScore)
}

func TestConfig_Serialization(t *testing.T) {
	cfg := DefaultConfig()

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var parsed Config
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, cfg.Mode, parsed.Mode)
	assert.Equal(t, cfg.RiskThreshold, parsed.RiskThreshold)
}

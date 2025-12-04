// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// AdapterAgent handles cross-cloud infrastructure code generation.
type AdapterAgent struct {
	BaseAgent
}

// NewAdapterAgent creates a new adapter agent.
func NewAdapterAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *AdapterAgent {
	return &AdapterAgent{
		BaseAgent: BaseAgent{
			name:        "adapter_agent",
			description: "Generates cross-cloud Terraform modules from requirements",
			tasks:       []TaskType{TaskTypeTerraformGeneration},
			tools: []string{
				"query_assets",
				"get_golden_image",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("adapter-agent"),
		},
	}
}

// Execute runs the adapter agent.
func (a *AdapterAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing adapter agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Parse requirements
	requirements, tokensUsed, err := a.parseTerraformRequirements(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse terraform requirements: %w", err)
	}

	// Step 2: Generate Terraform modules for each platform
	platforms := []string{"aws"}
	if p, ok := requirements["platforms"].([]interface{}); ok {
		platforms = make([]string, len(p))
		for i, platform := range p {
			platforms[i] = platform.(string)
		}
	}

	modules := make(map[string]interface{})
	for _, platform := range platforms {
		module := a.generateTerraformModule(platform, requirements)
		modules[platform] = module
	}

	// Generate the provisioning contract
	provisioningContract := a.generateProvisioningContract(requirements, modules)

	plan := map[string]interface{}{
		"summary":               fmt.Sprintf("Generated Terraform modules for %d platforms", len(platforms)),
		"provisioning_contract": provisioningContract,
		"terraform_modules":     modules,
		"platforms":             platforms,
		"affected_assets":       0,
		"phases": []map[string]interface{}{
			{
				"name":        "Module Validation",
				"description": "Validate Terraform syntax and policies",
			},
			{
				"name":        "Plan Generation",
				"description": "Generate terraform plan for each platform",
			},
			{
				"name":        "Infrastructure Provisioning",
				"description": "Apply Terraform changes",
				"wait_time":   "15m",
				"rollback_if": "apply_failure",
			},
		},
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        fmt.Sprintf("Generated Terraform modules for %v", platforms),
		AffectedAssets: 0,
		RiskLevel:      task.RiskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Apply", Description: "Approve and apply infrastructure changes"},
			{Type: "modify", Label: "Modify Modules", Description: "Edit Terraform modules before applying"},
			{Type: "reject", Label: "Reject", Description: "Reject and discard"},
		},
		Evidence: map[string]interface{}{
			"provisioning_contract": provisioningContract,
			"terraform_modules":     modules,
			"platforms":             platforms,
		},
	}, nil
}

func (a *AdapterAgent) parseTerraformRequirements(ctx context.Context, task *TaskSpec) (map[string]interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Adapter Agent. Parse the user's infrastructure request.

## User Request
%s

## Available Platforms
- aws, azure, gcp

## Your Task
Extract:
1. Resource type (compute, database, network, storage)
2. Target platforms
3. Instance specifications
4. Networking requirements
5. Tags and metadata

Output ONLY valid JSON:
{
  "resource_type": "compute|database|network|storage",
  "platforms": ["aws", "azure"],
  "instance_type": "string",
  "count": 1,
  "network": {
    "vpc": "existing|new",
    "subnets": "private|public"
  },
  "image_family": "string",
  "tags": {"key": "value"}
}`, task.UserIntent)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are an infrastructure expert. Parse requirements into structured JSON. Output ONLY valid JSON.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	var requirements map[string]interface{}
	content := resp.Content

	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &requirements); err != nil {
		requirements = map[string]interface{}{
			"resource_type": "compute",
			"platforms":     []interface{}{"aws"},
			"instance_type": "t3.medium",
			"count":         1,
			"tags":          map[string]interface{}{},
		}
	}

	return requirements, resp.Usage.TotalTokens, nil
}

func (a *AdapterAgent) generateTerraformModule(platform string, requirements map[string]interface{}) map[string]interface{} {
	resourceType := "compute"
	if rt, ok := requirements["resource_type"].(string); ok {
		resourceType = rt
	}

	instanceType := "t3.medium"
	if it, ok := requirements["instance_type"].(string); ok {
		instanceType = it
	}

	count := 1
	if c, ok := requirements["count"].(float64); ok {
		count = int(c)
	}

	switch platform {
	case "aws":
		return a.generateAWSTerraform(resourceType, instanceType, count, requirements)
	case "azure":
		return a.generateAzureTerraform(resourceType, instanceType, count, requirements)
	case "gcp":
		return a.generateGCPTerraform(resourceType, instanceType, count, requirements)
	default:
		return map[string]interface{}{"error": "unsupported platform"}
	}
}

func (a *AdapterAgent) generateAWSTerraform(resourceType, instanceType string, count int, reqs map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"format":   "hcl2",
		"platform": "aws",
		"terraform": map[string]interface{}{
			"required_version": ">= 1.5.0",
			"required_providers": map[string]interface{}{
				"aws": map[string]interface{}{
					"source":  "hashicorp/aws",
					"version": "~> 5.0",
				},
			},
		},
		"provider": map[string]interface{}{
			"aws": map[string]interface{}{
				"region": "${var.aws_region}",
			},
		},
		"resource": map[string]interface{}{
			"aws_instance": map[string]interface{}{
				"main": map[string]interface{}{
					"count":         count,
					"ami":           "${data.aws_ami.golden.id}",
					"instance_type": instanceType,
					"tags": map[string]interface{}{
						"Name":      "${var.name_prefix}-${count.index}",
						"ManagedBy": "quantumlayer",
					},
				},
			},
		},
		"data": map[string]interface{}{
			"aws_ami": map[string]interface{}{
				"golden": map[string]interface{}{
					"most_recent": true,
					"owners":      []string{"self"},
					"filter": map[string]interface{}{
						"name":   "tag:Family",
						"values": []string{"${var.image_family}"},
					},
				},
			},
		},
		"variable": map[string]interface{}{
			"aws_region": map[string]interface{}{
				"type":    "string",
				"default": "us-east-1",
			},
			"name_prefix": map[string]interface{}{
				"type": "string",
			},
			"image_family": map[string]interface{}{
				"type": "string",
			},
		},
		"output": map[string]interface{}{
			"instance_ids": map[string]interface{}{
				"value": "${aws_instance.main[*].id}",
			},
		},
	}
}

func (a *AdapterAgent) generateAzureTerraform(resourceType, instanceType string, count int, reqs map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"format":   "hcl2",
		"platform": "azure",
		"terraform": map[string]interface{}{
			"required_version": ">= 1.5.0",
			"required_providers": map[string]interface{}{
				"azurerm": map[string]interface{}{
					"source":  "hashicorp/azurerm",
					"version": "~> 3.0",
				},
			},
		},
		"provider": map[string]interface{}{
			"azurerm": map[string]interface{}{
				"features": map[string]interface{}{},
			},
		},
		"resource": map[string]interface{}{
			"azurerm_linux_virtual_machine": map[string]interface{}{
				"main": map[string]interface{}{
					"count":               count,
					"name":                "${var.name_prefix}-${count.index}",
					"resource_group_name": "${var.resource_group_name}",
					"location":            "${var.location}",
					"size":                instanceType,
					"source_image_id":     "${data.azurerm_image.golden.id}",
					"tags": map[string]interface{}{
						"ManagedBy": "quantumlayer",
					},
				},
			},
		},
		"variable": map[string]interface{}{
			"location": map[string]interface{}{
				"type":    "string",
				"default": "eastus",
			},
			"resource_group_name": map[string]interface{}{
				"type": "string",
			},
			"name_prefix": map[string]interface{}{
				"type": "string",
			},
		},
	}
}

func (a *AdapterAgent) generateGCPTerraform(resourceType, instanceType string, count int, reqs map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"format":   "hcl2",
		"platform": "gcp",
		"terraform": map[string]interface{}{
			"required_version": ">= 1.5.0",
			"required_providers": map[string]interface{}{
				"google": map[string]interface{}{
					"source":  "hashicorp/google",
					"version": "~> 5.0",
				},
			},
		},
		"provider": map[string]interface{}{
			"google": map[string]interface{}{
				"project": "${var.project_id}",
				"region":  "${var.region}",
			},
		},
		"resource": map[string]interface{}{
			"google_compute_instance": map[string]interface{}{
				"main": map[string]interface{}{
					"count":        count,
					"name":         "${var.name_prefix}-${count.index}",
					"machine_type": instanceType,
					"zone":         "${var.zone}",
					"boot_disk": map[string]interface{}{
						"initialize_params": map[string]interface{}{
							"image": "${data.google_compute_image.golden.self_link}",
						},
					},
					"labels": map[string]interface{}{
						"managed-by": "quantumlayer",
					},
				},
			},
		},
		"variable": map[string]interface{}{
			"project_id": map[string]interface{}{
				"type": "string",
			},
			"region": map[string]interface{}{
				"type":    "string",
				"default": "us-central1",
			},
			"zone": map[string]interface{}{
				"type":    "string",
				"default": "us-central1-a",
			},
			"name_prefix": map[string]interface{}{
				"type": "string",
			},
		},
	}
}

func (a *AdapterAgent) generateProvisioningContract(requirements map[string]interface{}, modules map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"version":       "1.0",
		"resource_type": requirements["resource_type"],
		"platforms":     requirements["platforms"],
		"modules":       modules,
		"invariants": map[string]interface{}{
			"tags_required":     []string{"ManagedBy", "Environment"},
			"naming_convention": "${prefix}-${env}-${resource_type}-${index}",
			"encryption":        true,
		},
		"validation": map[string]interface{}{
			"terraform_fmt":      true,
			"terraform_validate": true,
			"tfsec":              true,
			"checkov":            true,
		},
	}
}

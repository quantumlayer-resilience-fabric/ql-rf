// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
)

// ImageAgent handles golden image lifecycle management.
type ImageAgent struct {
	BaseAgent
}

// NewImageAgent creates a new image agent.
func NewImageAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *ImageAgent {
	return &ImageAgent{
		BaseAgent: BaseAgent{
			name:        "image_agent",
			description: "Creates cloud-agnostic golden images with CIS hardening and multi-platform support",
			tasks:       []TaskType{TaskTypeImageManagement},
			tools: []string{
				"get_golden_image",
				"list_image_versions",
				"generate_image_contract",
				"generate_packer_template",
				"generate_ansible_playbook",
				"build_image",
				"promote_image",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("image-agent"),
		},
	}
}

// Execute runs the image agent.
func (a *ImageAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing image agent", "task_id", task.ID, "goal", task.Goal)

	// Determine what operation is requested based on goal/intent
	operationType := a.determineOperation(task.Goal, task.UserIntent)

	switch operationType {
	case "create":
		return a.executeCreate(ctx, task)
	case "promote":
		return a.executePromote(ctx, task)
	case "list":
		return a.executeList(ctx, task)
	default:
		return a.executeCreate(ctx, task) // Default to create
	}
}

func (a *ImageAgent) determineOperation(goal, intent string) string {
	lowerGoal := goal + " " + intent
	if contains(lowerGoal, "promote") || contains(lowerGoal, "publish") {
		return "promote"
	}
	if contains(lowerGoal, "list") || contains(lowerGoal, "show") || contains(lowerGoal, "versions") {
		return "list"
	}
	return "create"
}

func (a *ImageAgent) executeCreate(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("creating golden image", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Parse requirements from user intent using LLM
	requirements, tokensUsed, err := a.parseImageRequirements(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image requirements: %w", err)
	}

	// Step 2: Generate the ImageContract using the tool
	contract, err := a.executeTool(ctx, "generate_image_contract", requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image contract: %w", err)
	}

	contractResult := contract.(map[string]interface{})
	imageContract := contractResult["contract"].(map[string]interface{})

	// Step 3: Generate Packer templates for each platform
	platforms := []string{"aws"}
	if p, ok := requirements["platforms"].([]interface{}); ok {
		platforms = make([]string, len(p))
		for i, platform := range p {
			platforms[i] = platform.(string)
		}
	}

	packerTemplates := make(map[string]interface{})
	for _, platform := range platforms {
		template, err := a.executeTool(ctx, "generate_packer_template", map[string]interface{}{
			"contract": imageContract,
			"platform": platform,
		})
		if err != nil {
			a.log.Warn("failed to generate packer template", "platform", platform, "error", err)
			continue
		}
		packerTemplates[platform] = template
	}

	// Step 4: Generate Ansible playbook
	playbook, err := a.executeTool(ctx, "generate_ansible_playbook", map[string]interface{}{
		"contract": imageContract,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate ansible playbook: %w", err)
	}

	// Step 5: Create the comprehensive plan
	plan := map[string]interface{}{
		"summary":          fmt.Sprintf("Create golden image: %s", imageContract["name"]),
		"image_contract":   imageContract,
		"packer_templates": packerTemplates,
		"ansible_playbook": playbook,
		"platforms":        platforms,
		"compliance":       imageContract["compliance"],
		"security":         imageContract["security"],
		"build_config":     imageContract["build"],
		"affected_assets":  0, // New image, no assets affected yet
		"phases": []map[string]interface{}{
			{
				"name":        "Contract Validation",
				"description": "Validate ImageContract against organization policies",
				"assets":      0,
			},
			{
				"name":        "Template Generation",
				"description": "Generate Packer templates for all target platforms",
				"assets":      len(platforms),
			},
			{
				"name":        "Image Build",
				"description": "Build images across platforms",
				"wait_time":   "30m",
				"rollback_if": "build_failure",
			},
			{
				"name":        "Compliance Testing",
				"description": "Run CIS benchmark and InSpec tests",
				"wait_time":   "15m",
			},
			{
				"name":        "SBOM Generation",
				"description": "Generate Software Bill of Materials",
			},
			{
				"name":        "Image Signing",
				"description": "Sign images with Cosign",
			},
		},
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        fmt.Sprintf("Generated ImageContract for %s with %d platform targets", imageContract["name"], len(platforms)),
		AffectedAssets: 0,
		RiskLevel:      task.RiskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Build", Description: "Approve the image contract and start building"},
			{Type: "modify", Label: "Modify Contract", Description: "Edit the image contract before building"},
			{Type: "reject", Label: "Reject", Description: "Reject and cancel the image creation"},
		},
		Evidence: map[string]interface{}{
			"image_contract":   imageContract,
			"packer_templates": packerTemplates,
			"ansible_playbook": playbook,
			"platforms":        platforms,
		},
	}, nil
}

func (a *ImageAgent) parseImageRequirements(ctx context.Context, task *TaskSpec) (map[string]interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Image Agent. Parse the user's request into structured image requirements.

## User Request
%s

## Available Options
- OS: ubuntu, rhel, amazon-linux, windows
- CIS Levels: 1 (basic), 2 (stricter)
- Platforms: aws, azure, gcp, docker, vsphere
- Runtimes: python:3.11, nodejs:20, java:17, go:1.21, docker

## Your Task
Extract:
1. Image name/family
2. Base OS and version
3. Purpose (web-server, database, k8s-node, etc.)
4. CIS hardening level
5. Target cloud platforms
6. Required runtimes
7. Additional packages

Output ONLY valid JSON (no markdown, no explanation):
{
  "name": "string",
  "os": "ubuntu|rhel|amazon-linux|windows",
  "os_version": "string (e.g. 22.04, 8.9)",
  "purpose": "string",
  "cis_level": 1|2,
  "platforms": ["aws", "azure", ...],
  "runtimes": ["python:3.11", ...],
  "packages": ["nginx", "curl", ...],
  "compliance": ["CIS", "SLSA", ...]
}`, task.UserIntent)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are an infrastructure image specification expert. Parse requirements into structured JSON. Output ONLY valid JSON.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	// Parse the JSON response
	var requirements map[string]interface{}
	content := resp.Content

	// Try to extract JSON if wrapped in markdown
	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &requirements); err != nil {
		// Fall back to defaults based on keywords
		requirements = a.fallbackRequirements(task.UserIntent)
	}

	return requirements, resp.Usage.TotalTokens, nil
}

func (a *ImageAgent) fallbackRequirements(intent string) map[string]interface{} {
	// Extract basic info from intent
	os := "ubuntu"
	osVersion := "22.04"
	purpose := "base"
	cisLevel := 1

	if contains(intent, "rhel") || contains(intent, "redhat") {
		os = "rhel"
		osVersion = "8.9"
	}
	if contains(intent, "amazon") {
		os = "amazon-linux"
		osVersion = "2"
	}
	if contains(intent, "web") {
		purpose = "web-server"
	}
	if contains(intent, "database") || contains(intent, "db") {
		purpose = "database"
	}
	if contains(intent, "kubernetes") || contains(intent, "k8s") {
		purpose = "k8s-node"
	}
	if contains(intent, "cis-2") || contains(intent, "level 2") || contains(intent, "strict") {
		cisLevel = 2
	}

	return map[string]interface{}{
		"name":       fmt.Sprintf("%s-%s-base", os, purpose),
		"os":         os,
		"os_version": osVersion,
		"purpose":    purpose,
		"cis_level":  cisLevel,
		"platforms":  []interface{}{"aws"},
		"runtimes":   []interface{}{},
		"packages":   []interface{}{},
		"compliance": []interface{}{"CIS"},
	}
}

func (a *ImageAgent) executePromote(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("promoting golden image", "task_id", task.ID)

	// Extract image info from metadata
	family := ""
	version := ""
	if m := task.Context.Metadata; m != nil {
		if f, ok := m["image_family"].(string); ok {
			family = f
		}
		if v, ok := m["image_version"].(string); ok {
			version = v
		}
	}

	result, err := a.executeTool(ctx, "promote_image", map[string]interface{}{
		"family":  family,
		"version": version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to promote image: %w", err)
	}

	return &AgentResult{
		TaskID:    task.ID,
		AgentName: a.name,
		Status:    AgentStatusCompleted,
		Summary:   fmt.Sprintf("Promoted image %s:%s to published", family, version),
		Evidence: map[string]interface{}{
			"promotion_result": result,
		},
	}, nil
}

func (a *ImageAgent) executeList(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("listing golden images", "task_id", task.ID)

	family := ""
	if m := task.Context.Metadata; m != nil {
		if f, ok := m["image_family"].(string); ok {
			family = f
		}
	}

	result, err := a.executeTool(ctx, "list_image_versions", map[string]interface{}{
		"family": family,
		"limit":  20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	return &AgentResult{
		TaskID:    task.ID,
		AgentName: a.name,
		Status:    AgentStatusCompleted,
		Summary:   "Listed available golden images",
		Evidence: map[string]interface{}{
			"images": result,
		},
	}, nil
}

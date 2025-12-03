// Package tools provides image-related tools for the AI agent.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// Image Generation Tools
// =============================================================================

// GenerateImageContractTool generates an ImageContract from requirements.
type GenerateImageContractTool struct {
	db *pgxpool.Pool
}

func (t *GenerateImageContractTool) Name() string        { return "generate_image_contract" }
func (t *GenerateImageContractTool) Description() string {
	return "Generate a cloud-agnostic ImageContract specification from natural language requirements"
}
func (t *GenerateImageContractTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *GenerateImageContractTool) Scope() Scope        { return ScopeOrganization }
func (t *GenerateImageContractTool) Idempotent() bool    { return true }
func (t *GenerateImageContractTool) RequiresApproval() bool { return false }
func (t *GenerateImageContractTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":        map[string]interface{}{"type": "string", "description": "Image name/family"},
			"os":          map[string]interface{}{"type": "string", "enum": []string{"ubuntu", "rhel", "amazon-linux", "windows"}},
			"os_version":  map[string]interface{}{"type": "string"},
			"purpose":     map[string]interface{}{"type": "string", "description": "What this image is for (web-server, database, k8s-node, etc.)"},
			"cis_level":   map[string]interface{}{"type": "integer", "enum": []int{1, 2}},
			"platforms":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"runtimes":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"packages":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"compliance":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
		},
		"required": []string{"name", "os", "purpose"},
	}
}

func (t *GenerateImageContractTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required and must be a string")
	}

	os, ok := params["os"].(string)
	if !ok || os == "" {
		return nil, fmt.Errorf("os is required and must be a string")
	}

	purpose, ok := params["purpose"].(string)
	if !ok || purpose == "" {
		return nil, fmt.Errorf("purpose is required and must be a string")
	}

	osVersion := "latest"
	if v, ok := params["os_version"].(string); ok {
		osVersion = v
	}

	cisLevel := 1
	if level, ok := params["cis_level"].(float64); ok {
		cisLevel = int(level)
	}

	// Build platforms list
	platforms := []map[string]interface{}{}
	if p, ok := params["platforms"].([]interface{}); ok {
		for _, platform := range p {
			platformStr, ok := platform.(string)
			if !ok {
				continue // skip invalid platform entries
			}
			platforms = append(platforms, map[string]interface{}{
				"platform": platformStr,
				"regions":  []string{"us-east-1", "us-west-2", "eu-west-1"},
			})
		}
	}
	if len(platforms) == 0 {
		// Default to AWS
		platforms = append(platforms, map[string]interface{}{
			"platform": "aws",
			"regions":  []string{"us-east-1", "us-west-2"},
		})
	}

	// Build runtimes
	runtimes := []map[string]interface{}{}
	if r, ok := params["runtimes"].([]interface{}); ok {
		for _, runtime := range r {
			runtimeStr, ok := runtime.(string)
			if !ok {
				continue // skip invalid runtime entries
			}
			parts := strings.Split(runtimeStr, ":")
			rt := map[string]interface{}{"name": parts[0]}
			if len(parts) > 1 {
				rt["version"] = parts[1]
			}
			runtimes = append(runtimes, rt)
		}
	}

	// Build packages
	packages := []string{}
	if p, ok := params["packages"].([]interface{}); ok {
		for _, pkg := range p {
			pkgStr, ok := pkg.(string)
			if !ok {
				continue // skip invalid package entries
			}
			packages = append(packages, pkgStr)
		}
	}

	// Build compliance frameworks
	frameworks := []string{"CIS"}
	if c, ok := params["compliance"].([]interface{}); ok {
		frameworks = []string{}
		for _, framework := range c {
			frameworkStr, ok := framework.(string)
			if !ok {
				continue // skip invalid framework entries
			}
			frameworks = append(frameworks, frameworkStr)
		}
		if len(frameworks) == 0 {
			frameworks = []string{"CIS"} // restore default if no valid entries
		}
	}

	// Generate the contract
	contract := map[string]interface{}{
		"name":        name,
		"version":     time.Now().Format("2006.01.02"),
		"family":      name,
		"description": fmt.Sprintf("Golden image for %s workloads", purpose),
		"tags": map[string]string{
			"purpose":    purpose,
			"managed-by": "quantumlayer",
		},
		"base": map[string]interface{}{
			"os":           os,
			"version":      osVersion,
			"architecture": "amd64",
		},
		"system": map[string]interface{}{
			"timezone": "UTC",
			"locale":   "en_US.UTF-8",
			"ntp": map[string]interface{}{
				"enabled": true,
				"servers": []string{"time.aws.com", "time.google.com"},
			},
			"ssh": map[string]interface{}{
				"permit_root_login":        "no",
				"password_authentication":  false,
				"pubkey_authentication":    true,
				"x11_forwarding":           false,
				"max_auth_tries":           3,
				"client_alive_interval":    300,
				"client_alive_count_max":   2,
			},
		},
		"security": map[string]interface{}{
			"cis_level": cisLevel,
			"firewall": map[string]interface{}{
				"enabled":      true,
				"default_deny": true,
				"allowed_ports": []map[string]interface{}{
					{"port": 22, "protocol": "tcp", "source": "10.0.0.0/8"},
				},
			},
			"password_policy": map[string]interface{}{
				"min_length":      14,
				"max_age":         90,
				"history_size":    5,
				"require_upper":   true,
				"require_lower":   true,
				"require_digit":   true,
				"require_special": true,
			},
			"secrets": map[string]interface{}{
				"remove_ssh_host_keys": true,
				"clear_machine_id":     true,
				"clear_history":        true,
			},
		},
		"packages": map[string]interface{}{
			"update":   true,
			"upgrade":  true,
			"install":  append([]string{"curl", "wget", "vim", "htop", "unzip"}, packages...),
			"remove":   []string{"telnet", "rsh-client", "rsh-server"},
			"runtimes": runtimes,
			"services": []map[string]interface{}{
				{"name": "sshd", "enabled": true, "state": "started"},
				{"name": "cron", "enabled": true, "state": "started"},
			},
		},
		"compliance": map[string]interface{}{
			"frameworks": frameworks,
			"slsa_level": 2,
			"signing": map[string]interface{}{
				"enabled":  true,
				"provider": "cosign",
			},
			"sbom": map[string]interface{}{
				"enabled": true,
				"format":  "spdx",
			},
		},
		"platforms": platforms,
		"tests": map[string]interface{}{
			"inspec": []map[string]interface{}{
				{"name": "cis-benchmark", "profile": fmt.Sprintf("cis-%s", os)},
			},
			"goss": []map[string]interface{}{
				{"name": "base-validation", "path": "/etc/goss.yaml"},
			},
		},
		"build": map[string]interface{}{
			"timeout":      "45m",
			"parallel":     true,
			"provisioners": []string{"ansible", "shell"},
		},
	}

	return map[string]interface{}{
		"contract": contract,
		"status":   "generated",
		"message":  fmt.Sprintf("Generated ImageContract for %s (%s %s)", name, os, osVersion),
	}, nil
}

// GeneratePackerTemplateTool generates Packer templates from ImageContract.
type GeneratePackerTemplateTool struct {
	db *pgxpool.Pool
}

func (t *GeneratePackerTemplateTool) Name() string        { return "generate_packer_template" }
func (t *GeneratePackerTemplateTool) Description() string {
	return "Generate platform-specific Packer templates from an ImageContract"
}
func (t *GeneratePackerTemplateTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *GeneratePackerTemplateTool) Scope() Scope        { return ScopeOrganization }
func (t *GeneratePackerTemplateTool) Idempotent() bool    { return true }
func (t *GeneratePackerTemplateTool) RequiresApproval() bool { return false }
func (t *GeneratePackerTemplateTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"contract": map[string]interface{}{"type": "object", "description": "ImageContract specification"},
			"platform": map[string]interface{}{"type": "string", "enum": []string{"aws", "azure", "gcp", "vsphere", "docker"}},
		},
		"required": []string{"contract", "platform"},
	}
}

func (t *GeneratePackerTemplateTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	contract, ok := params["contract"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("contract is required and must be an object")
	}

	platform := "aws"
	if p, ok := params["platform"].(string); ok {
		platform = p
	}

	name, ok := contract["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("contract.name is required and must be a string")
	}

	base, ok := contract["base"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("contract.base is required and must be an object")
	}

	osType, ok := base["os"].(string)
	if !ok || osType == "" {
		return nil, fmt.Errorf("contract.base.os is required and must be a string")
	}

	osVersion, ok := base["version"].(string)
	if !ok || osVersion == "" {
		return nil, fmt.Errorf("contract.base.version is required and must be a string")
	}

	arch := "amd64"
	if a, ok := base["architecture"].(string); ok {
		arch = a
	}

	var template map[string]interface{}

	switch platform {
	case "aws":
		template = generateAWSPackerTemplate(name, osType, osVersion, arch, contract)
	case "azure":
		template = generateAzurePackerTemplate(name, osType, osVersion, arch, contract)
	case "gcp":
		template = generateGCPPackerTemplate(name, osType, osVersion, arch, contract)
	case "docker":
		template = generateDockerPackerTemplate(name, osType, osVersion, contract)
	default:
		template = generateAWSPackerTemplate(name, osType, osVersion, arch, contract)
	}

	return map[string]interface{}{
		"template": template,
		"platform": platform,
		"format":   "hcl2",
		"status":   "generated",
	}, nil
}

func generateAWSPackerTemplate(name, osType, osVersion, arch string, contract map[string]interface{}) map[string]interface{} {
	// Get source AMI filter based on OS
	sourceAMI := map[string]interface{}{
		"filters": map[string]interface{}{
			"name":                getAMINameFilter(osType, osVersion),
			"root-device-type":    "ebs",
			"virtualization-type": "hvm",
			"architecture":        arch,
		},
		"owners":      []string{"amazon", "099720109477"}, // Amazon and Canonical
		"most_recent": true,
	}

	return map[string]interface{}{
		"packer": map[string]interface{}{
			"required_plugins": map[string]interface{}{
				"amazon": map[string]interface{}{
					"version": ">= 1.2.0",
					"source":  "github.com/hashicorp/amazon",
				},
				"ansible": map[string]interface{}{
					"version": ">= 1.1.0",
					"source":  "github.com/hashicorp/ansible",
				},
			},
		},
		"variable": map[string]interface{}{
			"aws_region": map[string]interface{}{
				"type":    "string",
				"default": "us-east-1",
			},
			"instance_type": map[string]interface{}{
				"type":    "string",
				"default": "t3.medium",
			},
			"image_version": map[string]interface{}{
				"type":    "string",
				"default": time.Now().Format("2006.01.02"),
			},
		},
		"source": map[string]interface{}{
			"amazon-ebs": map[string]interface{}{
				name: map[string]interface{}{
					"ami_name":                fmt.Sprintf("%s-{{var.image_version}}", name),
					"instance_type":           "{{var.instance_type}}",
					"region":                  "{{var.aws_region}}",
					"source_ami_filter":       sourceAMI,
					"ssh_username":            getSSHUsername(osType),
					"ami_description":         fmt.Sprintf("QuantumLayer Golden Image: %s", name),
					"encrypt_boot":            true,
					"force_deregister":        true,
					"force_delete_snapshot":   true,
					"tags": map[string]string{
						"Name":       fmt.Sprintf("%s-{{var.image_version}}", name),
						"Family":     name,
						"OS":         osType,
						"OSVersion":  osVersion,
						"ManagedBy":  "quantumlayer",
						"BuildTime":  "{{timestamp}}",
					},
				},
			},
		},
		"build": map[string]interface{}{
			"sources": []string{fmt.Sprintf("source.amazon-ebs.%s", name)},
			"provisioner": []map[string]interface{}{
				{
					"type": "shell",
					"inline": []string{
						"sudo apt-get update || sudo yum update -y",
						"sudo apt-get install -y python3 python3-pip || sudo yum install -y python3 python3-pip",
					},
				},
				{
					"type":          "ansible",
					"playbook_file": "./ansible/main.yml",
					"extra_arguments": []string{
						"--extra-vars", fmt.Sprintf("image_name=%s", name),
					},
				},
				{
					"type": "inspec",
					"profile": fmt.Sprintf("./inspec/cis-%s", osType),
				},
				{
					"type": "shell",
					"inline": []string{
						"sudo rm -rf /tmp/*",
						"sudo rm -rf /var/tmp/*",
						"sudo truncate -s 0 /var/log/*.log",
						"sudo rm -f /root/.bash_history",
						"sudo rm -f /home/*/.bash_history",
					},
				},
			},
			"post-processor": []map[string]interface{}{
				{
					"type":   "manifest",
					"output": "manifest.json",
				},
			},
		},
	}
}

func generateAzurePackerTemplate(name, osType, osVersion, arch string, contract map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"packer": map[string]interface{}{
			"required_plugins": map[string]interface{}{
				"azure": map[string]interface{}{
					"version": ">= 2.0.0",
					"source":  "github.com/hashicorp/azure",
				},
			},
		},
		"variable": map[string]interface{}{
			"subscription_id": map[string]interface{}{"type": "string"},
			"location":        map[string]interface{}{"type": "string", "default": "eastus"},
		},
		"source": map[string]interface{}{
			"azure-arm": map[string]interface{}{
				name: map[string]interface{}{
					"subscription_id":                "{{var.subscription_id}}",
					"managed_image_name":             fmt.Sprintf("%s-{{timestamp}}", name),
					"managed_image_resource_group_name": "golden-images-rg",
					"os_type":                        "Linux",
					"image_publisher":                getAzureImagePublisher(osType),
					"image_offer":                    getAzureImageOffer(osType),
					"image_sku":                      osVersion,
					"location":                       "{{var.location}}",
					"vm_size":                        "Standard_D2s_v3",
				},
			},
		},
		"build": map[string]interface{}{
			"sources": []string{fmt.Sprintf("source.azure-arm.%s", name)},
		},
	}
}

func generateGCPPackerTemplate(name, osType, osVersion, arch string, contract map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"packer": map[string]interface{}{
			"required_plugins": map[string]interface{}{
				"googlecompute": map[string]interface{}{
					"version": ">= 1.1.0",
					"source":  "github.com/hashicorp/googlecompute",
				},
			},
		},
		"variable": map[string]interface{}{
			"project_id": map[string]interface{}{"type": "string"},
			"zone":       map[string]interface{}{"type": "string", "default": "us-central1-a"},
		},
		"source": map[string]interface{}{
			"googlecompute": map[string]interface{}{
				name: map[string]interface{}{
					"project_id":          "{{var.project_id}}",
					"source_image_family": getGCPImageFamily(osType, osVersion),
					"zone":                "{{var.zone}}",
					"image_name":          fmt.Sprintf("%s-{{timestamp}}", name),
					"image_family":        name,
					"machine_type":        "e2-medium",
					"ssh_username":        getSSHUsername(osType),
				},
			},
		},
		"build": map[string]interface{}{
			"sources": []string{fmt.Sprintf("source.googlecompute.%s", name)},
		},
	}
}

func generateDockerPackerTemplate(name, osType, osVersion string, contract map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"packer": map[string]interface{}{
			"required_plugins": map[string]interface{}{
				"docker": map[string]interface{}{
					"version": ">= 1.0.0",
					"source":  "github.com/hashicorp/docker",
				},
			},
		},
		"source": map[string]interface{}{
			"docker": map[string]interface{}{
				name: map[string]interface{}{
					"image":  fmt.Sprintf("%s:%s", osType, osVersion),
					"commit": true,
				},
			},
		},
		"build": map[string]interface{}{
			"sources": []string{fmt.Sprintf("source.docker.%s", name)},
			"post-processor": []map[string]interface{}{
				{
					"type":       "docker-tag",
					"repository": fmt.Sprintf("quantumlayer/%s", name),
					"tags":       []string{"latest", "{{timestamp}}"},
				},
			},
		},
	}
}

// Helper functions
func getAMINameFilter(osType, osVersion string) string {
	switch osType {
	case "ubuntu":
		return fmt.Sprintf("ubuntu/images/hvm-ssd/ubuntu-*-%s*-amd64-server-*", osVersion)
	case "rhel":
		return fmt.Sprintf("RHEL-%s*-x86_64-*", osVersion)
	case "amazon-linux":
		return "amzn2-ami-hvm-*-x86_64-gp2"
	default:
		return "ubuntu/images/hvm-ssd/ubuntu-*-22.04*-amd64-server-*"
	}
}

func getSSHUsername(osType string) string {
	switch osType {
	case "ubuntu":
		return "ubuntu"
	case "rhel", "amazon-linux":
		return "ec2-user"
	default:
		return "ubuntu"
	}
}

func getAzureImagePublisher(osType string) string {
	switch osType {
	case "ubuntu":
		return "Canonical"
	case "rhel":
		return "RedHat"
	default:
		return "Canonical"
	}
}

func getAzureImageOffer(osType string) string {
	switch osType {
	case "ubuntu":
		return "0001-com-ubuntu-server-jammy"
	case "rhel":
		return "RHEL"
	default:
		return "0001-com-ubuntu-server-jammy"
	}
}

func getGCPImageFamily(osType, osVersion string) string {
	switch osType {
	case "ubuntu":
		return fmt.Sprintf("ubuntu-%s-lts", strings.Replace(osVersion, ".", "", 1))
	case "rhel":
		return fmt.Sprintf("rhel-%s", osVersion)
	default:
		return "ubuntu-2204-lts"
	}
}

// GenerateAnsiblePlaybookTool generates Ansible playbooks for image hardening.
type GenerateAnsiblePlaybookTool struct {
	db *pgxpool.Pool
}

func (t *GenerateAnsiblePlaybookTool) Name() string        { return "generate_ansible_playbook" }
func (t *GenerateAnsiblePlaybookTool) Description() string {
	return "Generate Ansible playbooks for image configuration and hardening"
}
func (t *GenerateAnsiblePlaybookTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *GenerateAnsiblePlaybookTool) Scope() Scope        { return ScopeOrganization }
func (t *GenerateAnsiblePlaybookTool) Idempotent() bool    { return true }
func (t *GenerateAnsiblePlaybookTool) RequiresApproval() bool { return false }
func (t *GenerateAnsiblePlaybookTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"contract": map[string]interface{}{"type": "object"},
		},
		"required": []string{"contract"},
	}
}

func (t *GenerateAnsiblePlaybookTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	contract, ok := params["contract"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("contract is required")
	}

	name := contract["name"].(string)
	security := contract["security"].(map[string]interface{})
	packages := contract["packages"].(map[string]interface{})

	cisLevel := 1
	if level, ok := security["cis_level"].(float64); ok {
		cisLevel = int(level)
	}

	// Build install packages list
	installPackages := []string{}
	if pkgs, ok := packages["install"].([]interface{}); ok {
		for _, pkg := range pkgs {
			installPackages = append(installPackages, pkg.(string))
		}
	}

	playbook := map[string]interface{}{
		"name":  fmt.Sprintf("Configure %s Golden Image", name),
		"hosts": "all",
		"become": true,
		"vars": map[string]interface{}{
			"image_name":       name,
			"cis_level":        cisLevel,
			"hardening_enabled": true,
		},
		"roles": []string{
			"common",
			"security-hardening",
			fmt.Sprintf("cis-level-%d", cisLevel),
		},
		"tasks": []map[string]interface{}{
			{
				"name": "Update package cache",
				"apt": map[string]interface{}{
					"update_cache": true,
					"cache_valid_time": 3600,
				},
				"when": "ansible_os_family == 'Debian'",
			},
			{
				"name": "Install required packages",
				"package": map[string]interface{}{
					"name":  installPackages,
					"state": "present",
				},
			},
			{
				"name": "Configure SSH hardening",
				"template": map[string]interface{}{
					"src":   "sshd_config.j2",
					"dest":  "/etc/ssh/sshd_config",
					"owner": "root",
					"mode":  "0600",
				},
				"notify": "restart sshd",
			},
			{
				"name": "Configure firewall",
				"ufw": map[string]interface{}{
					"state":   "enabled",
					"policy":  "deny",
					"logging": "on",
				},
				"when": "ansible_os_family == 'Debian'",
			},
			{
				"name": "Allow SSH through firewall",
				"ufw": map[string]interface{}{
					"rule":   "allow",
					"port":   "22",
					"proto":  "tcp",
					"from_ip": "10.0.0.0/8",
				},
				"when": "ansible_os_family == 'Debian'",
			},
			{
				"name": "Set password policies",
				"template": map[string]interface{}{
					"src":  "login.defs.j2",
					"dest": "/etc/login.defs",
				},
			},
			{
				"name": "Configure auditd",
				"package": map[string]interface{}{
					"name":  "auditd",
					"state": "present",
				},
			},
			{
				"name": "Enable auditd service",
				"service": map[string]interface{}{
					"name":    "auditd",
					"enabled": true,
					"state":   "started",
				},
			},
		},
		"handlers": []map[string]interface{}{
			{
				"name": "restart sshd",
				"service": map[string]interface{}{
					"name":  "sshd",
					"state": "restarted",
				},
			},
		},
	}

	return map[string]interface{}{
		"playbook": playbook,
		"format":   "yaml",
		"status":   "generated",
	}, nil
}

// BuildImageTool triggers image build (requires approval).
type BuildImageTool struct {
	db *pgxpool.Pool
}

func (t *BuildImageTool) Name() string        { return "build_image" }
func (t *BuildImageTool) Description() string { return "Trigger golden image build pipeline" }
func (t *BuildImageTool) Risk() RiskLevel     { return RiskStateChangeProd }
func (t *BuildImageTool) Scope() Scope        { return ScopeOrganization }
func (t *BuildImageTool) Idempotent() bool    { return false }
func (t *BuildImageTool) RequiresApproval() bool { return true }
func (t *BuildImageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"contract":  map[string]interface{}{"type": "object"},
			"platforms": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"dry_run":   map[string]interface{}{"type": "boolean", "default": false},
		},
		"required": []string{"contract"},
	}
}

func (t *BuildImageTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	contract := params["contract"].(map[string]interface{})
	dryRun := false
	if d, ok := params["dry_run"].(bool); ok {
		dryRun = d
	}

	buildID := uuid.New().String()
	name := contract["name"].(string)

	// In production, this would trigger the actual Packer build
	// For now, create a build record

	if !dryRun {
		// Store build request in database
		query := `
			INSERT INTO image_builds (id, image_name, contract, status, created_at)
			VALUES ($1, $2, $3, 'pending', NOW())
		`
		contractJSON, _ := json.Marshal(contract)
		_, err := t.db.Exec(ctx, query, buildID, name, contractJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to create build record: %w", err)
		}
	}

	return map[string]interface{}{
		"build_id":   buildID,
		"image_name": name,
		"status":     "pending",
		"dry_run":    dryRun,
		"message":    "Image build queued for execution",
	}, nil
}

// ListImageVersionsTool lists available versions of a golden image.
type ListImageVersionsTool struct {
	db *pgxpool.Pool
}

func (t *ListImageVersionsTool) Name() string        { return "list_image_versions" }
func (t *ListImageVersionsTool) Description() string { return "List all versions of a golden image family" }
func (t *ListImageVersionsTool) Risk() RiskLevel     { return RiskReadOnly }
func (t *ListImageVersionsTool) Scope() Scope        { return ScopeOrganization }
func (t *ListImageVersionsTool) Idempotent() bool    { return true }
func (t *ListImageVersionsTool) RequiresApproval() bool { return false }
func (t *ListImageVersionsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"family": map[string]interface{}{"type": "string"},
			"status": map[string]interface{}{"type": "string", "enum": []string{"draft", "testing", "published", "deprecated"}},
			"limit":  map[string]interface{}{"type": "integer", "default": 10},
		},
	}
}

func (t *ListImageVersionsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query := `
		SELECT id, family, version, os_name, os_version, cis_level, status, signed, created_at
		FROM images
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if family, ok := params["family"].(string); ok && family != "" {
		query += fmt.Sprintf(" AND family = $%d", argIdx)
		args = append(args, family)
		argIdx++
	}

	if status, ok := params["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	limit := 10
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := t.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	images := []map[string]interface{}{}
	for rows.Next() {
		var id, family, version, osName, osVersion, status string
		var cisLevel *int
		var signed bool
		var createdAt time.Time

		if err := rows.Scan(&id, &family, &version, &osName, &osVersion, &cisLevel, &status, &signed, &createdAt); err != nil {
			continue
		}

		img := map[string]interface{}{
			"id":         id,
			"family":     family,
			"version":    version,
			"os_name":    osName,
			"os_version": osVersion,
			"status":     status,
			"signed":     signed,
			"created_at": createdAt,
		}
		if cisLevel != nil {
			img["cis_level"] = *cisLevel
		}
		images = append(images, img)
	}

	return map[string]interface{}{
		"images": images,
		"total":  len(images),
	}, nil
}

// PromoteImageTool promotes an image to production status.
type PromoteImageTool struct {
	db *pgxpool.Pool
}

func (t *PromoteImageTool) Name() string        { return "promote_image" }
func (t *PromoteImageTool) Description() string { return "Promote a golden image from testing to published" }
func (t *PromoteImageTool) Risk() RiskLevel     { return RiskStateChangeProd }
func (t *PromoteImageTool) Scope() Scope        { return ScopeOrganization }
func (t *PromoteImageTool) Idempotent() bool    { return true }
func (t *PromoteImageTool) RequiresApproval() bool { return true }
func (t *PromoteImageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"image_id": map[string]interface{}{"type": "string"},
			"family":   map[string]interface{}{"type": "string"},
			"version":  map[string]interface{}{"type": "string"},
		},
	}
}

func (t *PromoteImageTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	imageID := ""
	if id, ok := params["image_id"].(string); ok {
		imageID = id
	}

	family := ""
	version := ""
	if f, ok := params["family"].(string); ok {
		family = f
	}
	if v, ok := params["version"].(string); ok {
		version = v
	}

	var query string
	var args []interface{}

	if imageID != "" {
		query = `UPDATE images SET status = 'published', updated_at = NOW() WHERE id = $1 RETURNING id, family, version`
		args = []interface{}{imageID}
	} else if family != "" && version != "" {
		query = `UPDATE images SET status = 'published', updated_at = NOW() WHERE family = $1 AND version = $2 RETURNING id, family, version`
		args = []interface{}{family, version}
	} else {
		return nil, fmt.Errorf("image_id or (family + version) required")
	}

	var id, fam, ver string
	err := t.db.QueryRow(ctx, query, args...).Scan(&id, &fam, &ver)
	if err != nil {
		return map[string]interface{}{
			"status": "failed",
			"error":  "Image not found",
		}, nil
	}

	return map[string]interface{}{
		"status":   "promoted",
		"image_id": id,
		"family":   fam,
		"version":  ver,
		"message":  fmt.Sprintf("Image %s:%s promoted to published", fam, ver),
	}, nil
}

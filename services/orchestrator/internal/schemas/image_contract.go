// Package schemas defines structured output schemas for AI-generated artifacts.
package schemas

// ImageContract represents a cloud-agnostic golden image specification.
// This is the canonical format that LLM generates, which then gets
// transformed into platform-specific artifacts (Packer, Ansible, etc.)
type ImageContract struct {
	// Metadata
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version" yaml:"version"`
	Family      string            `json:"family" yaml:"family"`
	Description string            `json:"description" yaml:"description"`
	Tags        map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Base Image
	Base BaseImage `json:"base" yaml:"base"`

	// System Configuration
	System SystemConfig `json:"system" yaml:"system"`

	// Security & Hardening
	Security SecurityConfig `json:"security" yaml:"security"`

	// Packages & Software
	Packages PackageConfig `json:"packages" yaml:"packages"`

	// Compliance Requirements
	Compliance ComplianceConfig `json:"compliance" yaml:"compliance"`

	// Platform Targets
	Platforms []PlatformTarget `json:"platforms" yaml:"platforms"`

	// Testing
	Tests TestConfig `json:"tests" yaml:"tests"`

	// Build Configuration
	Build BuildConfig `json:"build" yaml:"build"`
}

// BaseImage defines the source image to build from.
type BaseImage struct {
	OS           string `json:"os" yaml:"os"`                         // ubuntu, rhel, windows, amazon-linux
	Version      string `json:"version" yaml:"version"`               // 22.04, 8.9, 2022
	Architecture string `json:"architecture" yaml:"architecture"`     // amd64, arm64
	Source       string `json:"source,omitempty" yaml:"source,omitempty"` // specific AMI/image ID if needed
}

// SystemConfig defines system-level configuration.
type SystemConfig struct {
	Hostname    string            `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Timezone    string            `json:"timezone" yaml:"timezone"`
	Locale      string            `json:"locale" yaml:"locale"`
	NTP         NTPConfig         `json:"ntp" yaml:"ntp"`
	DNS         DNSConfig         `json:"dns,omitempty" yaml:"dns,omitempty"`
	Sysctl      map[string]string `json:"sysctl,omitempty" yaml:"sysctl,omitempty"`
	Limits      []SystemLimit     `json:"limits,omitempty" yaml:"limits,omitempty"`
	Users       []SystemUser      `json:"users,omitempty" yaml:"users,omitempty"`
	SSHConfig   SSHConfig         `json:"ssh" yaml:"ssh"`
}

// NTPConfig defines NTP configuration.
type NTPConfig struct {
	Enabled bool     `json:"enabled" yaml:"enabled"`
	Servers []string `json:"servers,omitempty" yaml:"servers,omitempty"`
}

// DNSConfig defines DNS configuration.
type DNSConfig struct {
	Nameservers []string `json:"nameservers,omitempty" yaml:"nameservers,omitempty"`
	Search      []string `json:"search,omitempty" yaml:"search,omitempty"`
}

// SystemLimit defines resource limits.
type SystemLimit struct {
	Domain string `json:"domain" yaml:"domain"`
	Type   string `json:"type" yaml:"type"`
	Item   string `json:"item" yaml:"item"`
	Value  string `json:"value" yaml:"value"`
}

// SystemUser defines a system user.
type SystemUser struct {
	Name       string   `json:"name" yaml:"name"`
	UID        int      `json:"uid,omitempty" yaml:"uid,omitempty"`
	Groups     []string `json:"groups,omitempty" yaml:"groups,omitempty"`
	Shell      string   `json:"shell,omitempty" yaml:"shell,omitempty"`
	SSHKeys    []string `json:"ssh_keys,omitempty" yaml:"ssh_keys,omitempty"`
	Sudo       bool     `json:"sudo,omitempty" yaml:"sudo,omitempty"`
	NoPassword bool     `json:"no_password,omitempty" yaml:"no_password,omitempty"`
}

// SSHConfig defines SSH hardening configuration.
type SSHConfig struct {
	PermitRootLogin       string `json:"permit_root_login" yaml:"permit_root_login"`
	PasswordAuthentication bool   `json:"password_authentication" yaml:"password_authentication"`
	PubkeyAuthentication  bool   `json:"pubkey_authentication" yaml:"pubkey_authentication"`
	X11Forwarding         bool   `json:"x11_forwarding" yaml:"x11_forwarding"`
	MaxAuthTries          int    `json:"max_auth_tries" yaml:"max_auth_tries"`
	ClientAliveInterval   int    `json:"client_alive_interval" yaml:"client_alive_interval"`
	ClientAliveCountMax   int    `json:"client_alive_count_max" yaml:"client_alive_count_max"`
}

// SecurityConfig defines security and hardening settings.
type SecurityConfig struct {
	CISLevel      int              `json:"cis_level" yaml:"cis_level"` // 1 or 2
	STIGCompliant bool             `json:"stig_compliant,omitempty" yaml:"stig_compliant,omitempty"`
	SELinux       string           `json:"selinux,omitempty" yaml:"selinux,omitempty"` // enforcing, permissive, disabled
	AppArmor      bool             `json:"apparmor,omitempty" yaml:"apparmor,omitempty"`
	Firewall      FirewallConfig   `json:"firewall" yaml:"firewall"`
	AuditRules    []string         `json:"audit_rules,omitempty" yaml:"audit_rules,omitempty"`
	PasswordPolicy PasswordPolicy  `json:"password_policy" yaml:"password_policy"`
	FileIntegrity FileIntegrity    `json:"file_integrity,omitempty" yaml:"file_integrity,omitempty"`
	Secrets       SecretsConfig    `json:"secrets,omitempty" yaml:"secrets,omitempty"`
}

// FirewallConfig defines firewall rules.
type FirewallConfig struct {
	Enabled      bool           `json:"enabled" yaml:"enabled"`
	DefaultDeny  bool           `json:"default_deny" yaml:"default_deny"`
	AllowedPorts []FirewallRule `json:"allowed_ports,omitempty" yaml:"allowed_ports,omitempty"`
}

// FirewallRule defines a single firewall rule.
type FirewallRule struct {
	Port     int    `json:"port" yaml:"port"`
	Protocol string `json:"protocol" yaml:"protocol"` // tcp, udp
	Source   string `json:"source,omitempty" yaml:"source,omitempty"`
}

// PasswordPolicy defines password requirements.
type PasswordPolicy struct {
	MinLength      int `json:"min_length" yaml:"min_length"`
	MaxAge         int `json:"max_age" yaml:"max_age"`
	MinAge         int `json:"min_age" yaml:"min_age"`
	WarnAge        int `json:"warn_age" yaml:"warn_age"`
	HistorySize    int `json:"history_size" yaml:"history_size"`
	RequireUpper   bool `json:"require_upper" yaml:"require_upper"`
	RequireLower   bool `json:"require_lower" yaml:"require_lower"`
	RequireDigit   bool `json:"require_digit" yaml:"require_digit"`
	RequireSpecial bool `json:"require_special" yaml:"require_special"`
}

// FileIntegrity defines file integrity monitoring.
type FileIntegrity struct {
	Enabled  bool     `json:"enabled" yaml:"enabled"`
	Tool     string   `json:"tool" yaml:"tool"` // aide, tripwire
	Paths    []string `json:"paths,omitempty" yaml:"paths,omitempty"`
}

// SecretsConfig defines secrets management.
type SecretsConfig struct {
	RemoveSSHHostKeys bool `json:"remove_ssh_host_keys" yaml:"remove_ssh_host_keys"`
	ClearMachineID    bool `json:"clear_machine_id" yaml:"clear_machine_id"`
	ClearHistory      bool `json:"clear_history" yaml:"clear_history"`
}

// PackageConfig defines software packages.
type PackageConfig struct {
	Update   bool              `json:"update" yaml:"update"`
	Upgrade  bool              `json:"upgrade" yaml:"upgrade"`
	Install  []string          `json:"install,omitempty" yaml:"install,omitempty"`
	Remove   []string          `json:"remove,omitempty" yaml:"remove,omitempty"`
	Repos    []PackageRepo     `json:"repos,omitempty" yaml:"repos,omitempty"`
	Runtimes []RuntimeConfig   `json:"runtimes,omitempty" yaml:"runtimes,omitempty"`
	Services []ServiceConfig   `json:"services,omitempty" yaml:"services,omitempty"`
}

// PackageRepo defines a package repository.
type PackageRepo struct {
	Name    string `json:"name" yaml:"name"`
	URL     string `json:"url" yaml:"url"`
	GPGKey  string `json:"gpg_key,omitempty" yaml:"gpg_key,omitempty"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
}

// RuntimeConfig defines runtime installations.
type RuntimeConfig struct {
	Name    string `json:"name" yaml:"name"`       // python, nodejs, java, go, docker
	Version string `json:"version" yaml:"version"` // 3.11, 20, 17, 1.21
}

// ServiceConfig defines service configurations.
type ServiceConfig struct {
	Name    string `json:"name" yaml:"name"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
	State   string `json:"state" yaml:"state"` // started, stopped
}

// ComplianceConfig defines compliance requirements.
type ComplianceConfig struct {
	Frameworks []string `json:"frameworks" yaml:"frameworks"` // CIS, STIG, SOC2, HIPAA, PCI
	SLSALevel  int      `json:"slsa_level" yaml:"slsa_level"` // 1-4
	Signing    SigningConfig `json:"signing" yaml:"signing"`
	SBOM       SBOMConfig    `json:"sbom" yaml:"sbom"`
}

// SigningConfig defines image signing configuration.
type SigningConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	Provider  string `json:"provider" yaml:"provider"` // cosign, notation
	KeyRef    string `json:"key_ref,omitempty" yaml:"key_ref,omitempty"`
}

// SBOMConfig defines SBOM generation configuration.
type SBOMConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Format  string `json:"format" yaml:"format"` // spdx, cyclonedx
}

// PlatformTarget defines a cloud/platform target.
type PlatformTarget struct {
	Platform     string            `json:"platform" yaml:"platform"` // aws, azure, gcp, vsphere, docker
	Regions      []string          `json:"regions,omitempty" yaml:"regions,omitempty"`
	InstanceType string            `json:"instance_type,omitempty" yaml:"instance_type,omitempty"`
	StorageType  string            `json:"storage_type,omitempty" yaml:"storage_type,omitempty"`
	StorageSize  int               `json:"storage_size,omitempty" yaml:"storage_size,omitempty"` // GB
	Tags         map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// TestConfig defines testing configuration.
type TestConfig struct {
	InSpec  []InSpecTest  `json:"inspec,omitempty" yaml:"inspec,omitempty"`
	Goss    []GossTest    `json:"goss,omitempty" yaml:"goss,omitempty"`
	Custom  []CustomTest  `json:"custom,omitempty" yaml:"custom,omitempty"`
}

// InSpecTest defines an InSpec test.
type InSpecTest struct {
	Name    string `json:"name" yaml:"name"`
	Profile string `json:"profile" yaml:"profile"` // cis-ubuntu22.04, custom
}

// GossTest defines a Goss test.
type GossTest struct {
	Name string `json:"name" yaml:"name"`
	Path string `json:"path" yaml:"path"`
}

// CustomTest defines a custom test.
type CustomTest struct {
	Name    string `json:"name" yaml:"name"`
	Command string `json:"command" yaml:"command"`
	Expect  string `json:"expect" yaml:"expect"`
}

// BuildConfig defines build configuration.
type BuildConfig struct {
	Timeout      string            `json:"timeout" yaml:"timeout"` // 30m, 1h
	Parallel     bool              `json:"parallel" yaml:"parallel"`
	Variables    map[string]string `json:"variables,omitempty" yaml:"variables,omitempty"`
	Provisioners []string          `json:"provisioners" yaml:"provisioners"` // ansible, shell, powershell
}

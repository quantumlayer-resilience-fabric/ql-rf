package sbom

import (
	"testing"

	"github.com/google/uuid"
)

func TestFormatIsValid(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		want   bool
	}{
		{
			name:   "valid spdx",
			format: FormatSPDX,
			want:   true,
		},
		{
			name:   "valid cyclonedx",
			format: FormatCycloneDX,
			want:   true,
		},
		{
			name:   "invalid format",
			format: Format("invalid"),
			want:   false,
		},
		{
			name:   "empty format",
			format: Format(""),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format.IsValid(); got != tt.want {
				t.Errorf("Format.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatString(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		want   string
	}{
		{
			name:   "spdx",
			format: FormatSPDX,
			want:   "spdx",
		},
		{
			name:   "cyclonedx",
			format: FormatCycloneDX,
			want:   "cyclonedx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("Format.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeneratePURL(t *testing.T) {
	tests := []struct {
		name       string
		pkgType    string
		pkgName    string
		pkgVersion string
		wantPrefix string
	}{
		{
			name:       "npm package",
			pkgType:    "npm",
			pkgName:    "express",
			pkgVersion: "4.18.2",
			wantPrefix: "pkg:npm/express@4.18.2",
		},
		{
			name:       "pip package",
			pkgType:    "pip",
			pkgName:    "django",
			pkgVersion: "4.2.0",
			wantPrefix: "pkg:pypi/django@4.2.0",
		},
		{
			name:       "go package",
			pkgType:    "go",
			pkgName:    "github.com/gin-gonic/gin",
			pkgVersion: "v1.9.1",
			wantPrefix: "pkg:golang/github.com/gin-gonic/gin@v1.9.1",
		},
		{
			name:       "deb package",
			pkgType:    "deb",
			pkgName:    "nginx",
			pkgVersion: "1.18.0",
			wantPrefix: "pkg:deb/nginx@1.18.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generatePURL(tt.pkgType, tt.pkgName, tt.pkgVersion)
			if got == "" {
				t.Errorf("generatePURL() returned empty string")
			}
			// Just check it starts with expected prefix
			if len(got) < len(tt.wantPrefix) {
				t.Errorf("generatePURL() = %v, want prefix %v", got, tt.wantPrefix)
			}
		})
	}
}

func TestGetFormatVersion(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		want   string
	}{
		{
			name:   "spdx",
			format: FormatSPDX,
			want:   "SPDX-2.3",
		},
		{
			name:   "cyclonedx",
			format: FormatCycloneDX,
			want:   "CycloneDX-1.5",
		},
		{
			name:   "unknown",
			format: Format("unknown"),
			want:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getFormatVersion(tt.format); got != tt.want {
				t.Errorf("getFormatVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapPackageTypeToOSVEcosystem(t *testing.T) {
	tests := []struct {
		name    string
		pkgType string
		want    string
	}{
		{
			name:    "npm",
			pkgType: "npm",
			want:    "npm",
		},
		{
			name:    "pip",
			pkgType: "pip",
			want:    "PyPI",
		},
		{
			name:    "pypi",
			pkgType: "pypi",
			want:    "PyPI",
		},
		{
			name:    "go",
			pkgType: "go",
			want:    "Go",
		},
		{
			name:    "golang",
			pkgType: "golang",
			want:    "Go",
		},
		{
			name:    "maven",
			pkgType: "maven",
			want:    "Maven",
		},
		{
			name:    "nuget",
			pkgType: "nuget",
			want:    "NuGet",
		},
		{
			name:    "unknown",
			pkgType: "unknown",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapPackageTypeToOSVEcosystem(tt.pkgType); got != tt.want {
				t.Errorf("mapPackageTypeToOSVEcosystem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCVSSScoreToSeverity(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  string
	}{
		{
			name:  "critical",
			score: 9.5,
			want:  "critical",
		},
		{
			name:  "high",
			score: 7.8,
			want:  "high",
		},
		{
			name:  "medium",
			score: 5.2,
			want:  "medium",
		},
		{
			name:  "low",
			score: 2.1,
			want:  "low",
		},
		{
			name:  "unknown",
			score: 0.0,
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cvssScoreToSeverity(tt.score); got != tt.want {
				t.Errorf("cvssScoreToSeverity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapPackageTypeToCycloneDX(t *testing.T) {
	tests := []struct {
		name    string
		pkgType string
		want    string
	}{
		{
			name:    "deb package",
			pkgType: "deb",
			want:    "library",
		},
		{
			name:    "npm package",
			pkgType: "npm",
			want:    "library",
		},
		{
			name:    "container",
			pkgType: "container",
			want:    "container",
		},
		{
			name:    "os",
			pkgType: "os",
			want:    "operating-system",
		},
		{
			name:    "application",
			pkgType: "application",
			want:    "application",
		},
		{
			name:    "unknown",
			pkgType: "unknown",
			want:    "library",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapPackageTypeToCycloneDX(tt.pkgType); got != tt.want {
				t.Errorf("mapPackageTypeToCycloneDX() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractDebPackages(t *testing.T) {
	g := &Generator{}

	tests := []struct {
		name string
		cmd  string
		want []string
	}{
		{
			name: "simple install",
			cmd:  "apt-get install -y nginx",
			want: []string{"nginx"},
		},
		{
			name: "multiple packages",
			cmd:  "apt-get install -y nginx curl wget",
			want: []string{"nginx", "curl", "wget"},
		},
		{
			name: "with version",
			cmd:  "apt-get install -y nginx=1.18.0-1",
			want: []string{"nginx"},
		},
		{
			name: "with flags",
			cmd:  "apt-get install -y --no-install-recommends nginx",
			want: []string{"nginx"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.extractDebPackages(tt.cmd)
			if len(got) != len(tt.want) {
				t.Errorf("extractDebPackages() returned %d packages, want %d", len(got), len(tt.want))
				return
			}
			for i, pkg := range got {
				if pkg != tt.want[i] {
					t.Errorf("extractDebPackages()[%d] = %v, want %v", i, pkg, tt.want[i])
				}
			}
		})
	}
}

func TestParseNPM(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name        string
		content     string
		wantErr     bool
		wantPkgName string
		wantDepLen  int
	}{
		{
			name: "valid package.json",
			content: `{
				"name": "test-app",
				"version": "1.0.0",
				"dependencies": {
					"express": "^4.18.2",
					"lodash": "^4.17.21"
				},
				"devDependencies": {
					"jest": "^29.0.0"
				}
			}`,
			wantErr:     false,
			wantPkgName: "test-app",
			wantDepLen:  3,
		},
		{
			name:    "invalid json",
			content: `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := parser.parseNPM(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNPM() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if manifest.Metadata["name"] != tt.wantPkgName {
				t.Errorf("parseNPM() package name = %v, want %v", manifest.Metadata["name"], tt.wantPkgName)
			}
			if len(manifest.Packages) != tt.wantDepLen {
				t.Errorf("parseNPM() package count = %v, want %v", len(manifest.Packages), tt.wantDepLen)
			}
		})
	}
}

func TestParsePip(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name       string
		content    string
		wantErr    bool
		wantPkgLen int
	}{
		{
			name: "valid requirements.txt",
			content: `django==4.2.0
flask>=2.3.0
requests
# comment
pytest==7.3.1`,
			wantErr:    false,
			wantPkgLen: 4,
		},
		{
			name:       "empty file",
			content:    "",
			wantErr:    false,
			wantPkgLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := parser.parsePip(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePip() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if len(manifest.Packages) != tt.wantPkgLen {
				t.Errorf("parsePip() package count = %v, want %v", len(manifest.Packages), tt.wantPkgLen)
			}
		})
	}
}

func TestParseGoMod(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name       string
		content    string
		wantErr    bool
		wantModule string
		wantPkgLen int
	}{
		{
			name: "valid go.mod",
			content: `module github.com/example/app

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.3.0
)`,
			wantErr:    false,
			wantModule: "github.com/example/app",
			wantPkgLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := parser.parseGoMod(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGoMod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if manifest.Metadata["module"] != tt.wantModule {
				t.Errorf("parseGoMod() module = %v, want %v", manifest.Metadata["module"], tt.wantModule)
			}
			if len(manifest.Packages) != tt.wantPkgLen {
				t.Errorf("parseGoMod() package count = %v, want %v", len(manifest.Packages), tt.wantPkgLen)
			}
		})
	}
}

func TestExtractXMLValue(t *testing.T) {
	tests := []struct {
		name string
		line string
		tag  string
		want string
	}{
		{
			name: "simple tag",
			line: "<artifactId>spring-boot</artifactId>",
			tag:  "artifactId",
			want: "spring-boot",
		},
		{
			name: "with whitespace",
			line: "  <version>  2.7.0  </version>  ",
			tag:  "version",
			want: "2.7.0",
		},
		{
			name: "not found",
			line: "<artifactId>spring-boot</artifactId>",
			tag:  "version",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractXMLValue(tt.line, tt.tag); got != tt.want {
				t.Errorf("extractXMLValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractAttribute(t *testing.T) {
	tests := []struct {
		name string
		line string
		attr string
		want string
	}{
		{
			name: "simple attribute",
			line: `<package id="Newtonsoft.Json" version="13.0.1" />`,
			attr: "id",
			want: "Newtonsoft.Json",
		},
		{
			name: "version attribute",
			line: `<package id="Newtonsoft.Json" version="13.0.1" />`,
			attr: "version",
			want: "13.0.1",
		},
		{
			name: "not found",
			line: `<package id="Newtonsoft.Json" version="13.0.1" />`,
			attr: "missing",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractAttribute(tt.line, tt.attr); got != tt.want {
				t.Errorf("extractAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateChecksum(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    int // length of hash
	}{
		{
			name:    "simple content",
			content: []byte("hello world"),
			want:    64, // SHA256 hex string length
		},
		{
			name:    "empty content",
			content: []byte(""),
			want:    64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateChecksum(tt.content)
			if len(got) != tt.want {
				t.Errorf("generateChecksum() length = %v, want %v", len(got), tt.want)
			}
		})
	}
}

func TestSBOMStruct(t *testing.T) {
	sbom := &SBOM{
		ID:      uuid.New(),
		ImageID: uuid.New(),
		OrgID:   uuid.New(),
		Format:  FormatSPDX,
		Version: "SPDX-2.3",
		Content: map[string]interface{}{
			"spdxVersion": "SPDX-2.3",
			"name":        "Test SBOM",
		},
		PackageCount: 10,
		VulnCount:    2,
	}

	if sbom.ID == uuid.Nil {
		t.Error("SBOM ID should not be nil")
	}
	if sbom.Format != FormatSPDX {
		t.Errorf("SBOM Format = %v, want %v", sbom.Format, FormatSPDX)
	}
	if sbom.PackageCount != 10 {
		t.Errorf("SBOM PackageCount = %v, want 10", sbom.PackageCount)
	}
}

func TestPackageStruct(t *testing.T) {
	pkg := Package{
		ID:      uuid.New(),
		SBOMID:  uuid.New(),
		Name:    "express",
		Version: "4.18.2",
		Type:    "npm",
		PURL:    "pkg:npm/express@4.18.2",
		License: "MIT",
	}

	if pkg.Name != "express" {
		t.Errorf("Package Name = %v, want express", pkg.Name)
	}
	if pkg.Type != "npm" {
		t.Errorf("Package Type = %v, want npm", pkg.Type)
	}
}

func TestVulnerabilityStruct(t *testing.T) {
	cvssScore := 9.1
	vuln := Vulnerability{
		ID:               uuid.New(),
		SBOMID:           uuid.New(),
		PackageID:        uuid.New(),
		CVEID:            "CVE-2024-1234",
		Severity:         "critical",
		CVSSScore:        &cvssScore,
		ExploitAvailable: true,
	}

	if vuln.CVEID != "CVE-2024-1234" {
		t.Errorf("Vulnerability CVEID = %v, want CVE-2024-1234", vuln.CVEID)
	}
	if vuln.Severity != "critical" {
		t.Errorf("Vulnerability Severity = %v, want critical", vuln.Severity)
	}
	if *vuln.CVSSScore != 9.1 {
		t.Errorf("Vulnerability CVSSScore = %v, want 9.1", *vuln.CVSSScore)
	}
}

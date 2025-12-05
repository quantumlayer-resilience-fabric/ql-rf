# ADR-014: Compliance Framework Integration with Pre-Populated Controls

## Status
Accepted

## Context

Enterprise customers operating in regulated industries (finance, healthcare, government) require:

1. **Audit Readiness**: Continuous compliance evidence for regulatory audits
2. **Multiple Framework Support**: Simultaneous compliance with CIS, SOC 2, NIST, ISO 27001, PCI-DSS, HIPAA
3. **Evidence Collection**: Automated gathering and storage of compliance artifacts
4. **Control Mapping**: Understanding how controls across frameworks relate to each other
5. **Assessment Automation**: Ability to assess compliance status across thousands of assets
6. **Exemption Management**: Documenting and tracking control exceptions with compensating controls

Current challenges:
- **Manual Compliance**: Most organizations track compliance in spreadsheets
- **Evidence Fragmentation**: Evidence scattered across multiple tools and systems
- **Cross-Framework Duplication**: Implementing the same control multiple times for different frameworks
- **Audit Burden**: High overhead for quarterly/annual audits
- **Inconsistent Interpretation**: Different teams interpret controls differently

## Decision

We implement a **compliance framework management system** with pre-populated controls and cross-framework mappings:

### 1. Pre-Populated Compliance Frameworks

Six major frameworks included out-of-the-box:

| Framework | Version | Controls | Automation Level |
|-----------|---------|----------|------------------|
| CIS Benchmarks | Linux L1/L2, K8s | 500+ | 70% automated |
| SOC 2 | Type I/II | 64 controls (5 trust principles) | 40% automated |
| NIST 800-53 | Rev 5 | 1,084 controls | 50% automated |
| ISO 27001 | 2013 | 114 controls (Annex A) | 45% automated |
| PCI-DSS | v4.0 | 300+ requirements | 60% automated |
| HIPAA Security Rule | Final Rule | 45 controls | 35% automated |

**Framework Schema**:
```go
type Framework struct {
    ID             uuid.UUID
    Name           string     // "CIS Benchmark Linux Level 1"
    Category       string     // "infrastructure", "application", "data"
    Version        string     // "v8.0"
    RegulatoryBody string     // "Center for Internet Security"
    EffectiveDate  *time.Time
}
```

### 2. Control Definitions

Each control includes comprehensive metadata:

```go
type Control struct {
    ID                     uuid.UUID
    FrameworkID            uuid.UUID
    ControlID              string        // "CIS-1.1.1", "SOC2-CC6.1"
    Name                   string
    Description            string
    Severity               Severity      // critical, high, medium, low
    ControlFamily          string        // "Access Control", "Logging"
    ImplementationGuidance string        // How to implement
    AssessmentProcedure    string        // How to test
    AutomationSupport      string        // automated, hybrid, manual
    Priority               string        // P0, P1, P2, P3
}
```

**Severity Levels**:
- **Critical**: Must be implemented, direct regulatory requirement
- **High**: Should be implemented, significant risk if missing
- **Medium**: Recommended, moderate risk
- **Low**: Optional, best practice

**Automation Support**:
- **Automated**: QL-RF can check compliance automatically (e.g., SSH disabled password auth)
- **Hybrid**: Partially automated, requires manual validation
- **Manual**: Requires human review (e.g., policies, procedures)

### 3. Cross-Framework Control Mappings

Controls across frameworks are mapped to enable evidence reuse:

```go
type ControlMapping struct {
    SourceControlID uuid.UUID  // CIS-1.1.1
    TargetControlID uuid.UUID  // NIST-AC-2
    MappingType     string      // equivalent, partial, related
    ConfidenceScore float64     // 0.0 to 1.0
    Notes           string
}
```

**Mapping Types**:
- **Equivalent**: Controls address the same requirement (90%+ overlap)
- **Partial**: Significant overlap but not identical (50-90%)
- **Related**: Related controls that support each other (<50%)

**Example Mapping**:
```
CIS-1.1.1 "Ensure mounting of cramfs filesystems is disabled"
  → NIST-CM-7 "Least Functionality" (partial, 0.7)
  → ISO-A.12.5.1 "Installation of software" (related, 0.5)
```

**Benefits**:
- Evidence for CIS-1.1.1 can be reused for NIST-CM-7
- Implementing one control addresses multiple framework requirements
- Audit preparation is faster (cross-reference evidence)

### 4. Assessment Lifecycle

**Assessment Creation**:
```go
type Assessment struct {
    OrgID           uuid.UUID
    FrameworkID     uuid.UUID
    AssessmentType  string        // self, external, continuous
    ScopeSites      []uuid.UUID   // Which sites to assess
    ScopeAssets     []uuid.UUID   // Which assets to assess
    Status          string        // pending, in_progress, completed
}
```

**Assessment Workflow**:
1. **Create**: Define scope (framework, sites, assets)
2. **Start**: Initiate automated checks and manual reviews
3. **Evaluate**: Run automated checks, collect evidence
4. **Record Results**: passed, failed, not_applicable, manual_review
5. **Complete**: Calculate score and generate report

**Control Results**:
```go
type AssessmentResult struct {
    AssessmentID        uuid.UUID
    ControlID           uuid.UUID
    Status              string      // passed, failed, not_applicable, manual_review
    Score               float64     // 0-100
    Findings            string      // What was found
    RemediationGuidance string      // How to fix
    EvidenceIDs         []uuid.UUID // Links to evidence
    CheckOutput         map[string]any  // Raw check results
}
```

### 5. Evidence Management

**Evidence Types**:
- **Screenshot**: UI screenshots showing configuration
- **Log**: Log file excerpts proving behavior
- **Config**: Configuration file dumps
- **Report**: Scanner output (Nessus, Qualys, etc.)
- **Attestation**: Signed statements from authorized personnel

**Evidence Schema**:
```go
type Evidence struct {
    OrgID            uuid.UUID
    ControlID        uuid.UUID
    EvidenceType     string
    Title            string
    StorageType      string      // s3, azure_blob, gcs, local
    StoragePath      string
    ContentHash      string      // SHA-256 for integrity
    FileSizeBytes    int64
    CollectedAt      time.Time
    CollectedBy      string      // user_id or "system"
    CollectionMethod string      // manual, automated, scanner
    ValidFrom        time.Time
    ValidUntil       *time.Time  // Evidence expiration
    IsCurrent        bool
}
```

**Evidence Storage**:
- **Cloud Storage**: S3, Azure Blob, GCS for production
- **Local Filesystem**: Development/testing only
- **Compression**: GZIP compression for large files
- **Encryption**: AES-256 encryption at rest

**Evidence Lifecycle**:
1. **Collection**: Automatic (scanners, QL-RF agents) or manual upload
2. **Validation**: Hash verification, format validation
3. **Storage**: Encrypted storage with access controls
4. **Review**: Optional human review workflow
5. **Expiration**: Automatic archival when `valid_until` passes
6. **Export**: PDF/ZIP evidence packs for auditors

### 6. Control Exemptions

**Exemption Schema**:
```go
type Exemption struct {
    OrgID                uuid.UUID
    ControlID            uuid.UUID
    AssetID              *uuid.UUID  // Exemption for specific asset
    SiteID               *uuid.UUID  // Or entire site
    Reason               string
    RiskAcceptance       string      // Risk statement
    CompensatingControls string      // Alternative controls
    ApprovedBy           string      // user_id
    ExpiresAt            time.Time   // Time-bounded
    ReviewFrequencyDays  int         // Re-review interval
    Status               string      // active, expired, revoked
}
```

**Exemption Workflow**:
1. **Request**: Engineer requests exemption with justification
2. **Risk Acceptance**: Document residual risk
3. **Compensating Controls**: Define alternative controls
4. **Approval**: Security admin or higher approves
5. **Time-Bounded**: Exemption expires after N days
6. **Periodic Review**: Re-review at defined interval (90 days)
7. **Auto-Reminder**: Alerts at T-7d, T-1d, T-0 (expiration)

### 7. Database Implementation

**Core Tables**:
- `compliance_frameworks` - Framework definitions
- `compliance_controls` - Control catalog
- `control_mappings` - Cross-framework mappings
- `compliance_assessments` - Assessment runs
- `compliance_assessment_results` - Per-control results
- `compliance_evidence` - Evidence artifacts
- `compliance_exemptions` - Control exemptions

**Key Queries**:
```sql
-- Get all controls for a framework with automation level
SELECT * FROM compliance_controls
WHERE framework_id = $1
AND automation_support = 'automated'
ORDER BY severity DESC, control_id;

-- Get controls mapped to a specific control
SELECT target.* FROM compliance_controls target
JOIN control_mappings cm ON target.id = cm.target_control_id
WHERE cm.source_control_id = $1
AND cm.mapping_type IN ('equivalent', 'partial');

-- Get compliance score for organization
SELECT
  COUNT(*) FILTER (WHERE status = 'passed') as passed,
  COUNT(*) FILTER (WHERE status = 'failed') as failed,
  COUNT(*) FILTER (WHERE status = 'not_applicable') as na
FROM compliance_assessment_results car
JOIN compliance_assessments ca ON car.assessment_id = ca.id
WHERE ca.org_id = $org_id AND ca.status = 'completed';
```

## Consequences

### Positive

1. **Audit Readiness**: Continuous compliance evidence always available
2. **Reduced Duplication**: Cross-framework mappings reduce redundant work
3. **Time Savings**: Pre-populated frameworks save months of initial setup
4. **Consistency**: Standard control definitions reduce interpretation variance
5. **Evidence Centralization**: All compliance artifacts in one location
6. **Automation**: 40-70% of controls can be checked automatically
7. **Exemption Tracking**: Clear audit trail for all exceptions

### Negative

1. **Framework Maintenance**: Must update frameworks as standards evolve
2. **Mapping Complexity**: Control mappings require security expertise
3. **Storage Costs**: Evidence storage can grow large (GBs per org)
4. **Manual Reviews**: 30-60% of controls still require human validation
5. **Framework Coverage**: Cannot cover every niche framework

### Mitigations

1. **Quarterly Framework Updates**: Scheduled review of framework changes
2. **Expert Review**: Security team validates all mappings before release
3. **Storage Lifecycle**: Auto-archive evidence older than 7 years
4. **Hybrid Approach**: Automated checks feed into manual review workflows
5. **Custom Framework Support**: Organizations can define custom frameworks

## Implementation Notes

### Pre-Population Script

Frameworks and controls are pre-populated via migration 000012:

```sql
-- Insert frameworks
INSERT INTO compliance_frameworks (name, category, version, regulatory_body) VALUES
('CIS Benchmark - Ubuntu Linux 22.04 LTS', 'infrastructure', 'v1.0.0', 'Center for Internet Security'),
('SOC 2 Type II', 'governance', '2017', 'AICPA'),
...;

-- Insert controls (500+ total)
INSERT INTO compliance_controls (framework_id, control_id, name, severity, ...) VALUES
((SELECT id FROM compliance_frameworks WHERE name = 'CIS Benchmark - Ubuntu Linux 22.04 LTS'),
 'CIS-1.1.1', 'Ensure mounting of cramfs filesystems is disabled', 'high', ...),
...;

-- Insert mappings (200+ cross-references)
INSERT INTO control_mappings (source_control_id, target_control_id, mapping_type, confidence_score) VALUES
((SELECT id FROM compliance_controls WHERE control_id = 'CIS-1.1.1'),
 (SELECT id FROM compliance_controls WHERE control_id = 'NIST-CM-7'), 'partial', 0.7),
...;
```

### Automated Assessment Flow

```go
1. Create assessment with scope
2. For each control in framework:
   a. Check automation_support level
   b. If "automated":
      - Run automated check (InSpec, custom scripts)
      - Collect output as evidence
      - Determine pass/fail
   c. If "hybrid" or "manual":
      - Create task for human reviewer
      - Provide assessment procedure
3. Calculate compliance score
4. Generate PDF report
5. Notify stakeholders
```

### Evidence Collection Integration

**Scanner Integration**:
```go
// Import Nessus/Qualys scan results
POST /api/v1/compliance/evidence/import
{
  "control_id": "uuid",
  "scanner": "nessus",
  "scan_file": "base64-encoded-xml"
}
```

**Automated Evidence**:
- InSpec scan results (JSON)
- Configuration snapshots (YAML/JSON)
- Audit log exports
- Screenshot captures (PNG)

### Compliance Dashboard

**Key Metrics**:
- Overall compliance score (% passed)
- Controls by status (passed, failed, not_applicable)
- Evidence completeness (% controls with evidence)
- Exemptions count and expiration timeline
- Trend: score over time (last 90 days)

**Drill-Down**:
- Framework → Control → Evidence → Assets

## Migration Path

**Phase 1** (Completed):
- Create compliance tables
- Pre-populate 6 frameworks with 500+ controls
- Create 200+ cross-framework mappings
- Implement Go service layer (`pkg/compliance`)

**Phase 2** (In Progress):
- Implement assessment workflow
- Build evidence collection API
- Create automated check runners (InSpec integration)

**Phase 3** (Planned):
- Build compliance dashboard UI
- Implement PDF report generation
- Add scanner integrations (Nessus, Qualys, Wiz)
- Create exemption workflow UI

## References

- Migration 000012: Compliance frameworks and controls
- `pkg/compliance/compliance.go`: Core compliance service
- PRD Section 14: Compliance and Evidence Packs
- CIS Benchmarks: https://www.cisecurity.org/cis-benchmarks
- SOC 2 Framework: https://www.aicpa.org/soc2
- NIST 800-53: https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final

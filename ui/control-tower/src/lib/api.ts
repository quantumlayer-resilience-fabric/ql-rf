/**
 * API Client for QL-RF Control Tower
 *
 * Provides typed API client for communicating with the backend services.
 * Includes Clerk authentication token handling.
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

// Token getter function - set by the auth provider (Clerk)
// This will be initialized by the AuthProvider component
let getAuthToken: (() => Promise<string | null>) | null = null;

// Promise that resolves when auth is ready
let authReadyResolve: (() => void) | null = null;
const authReadyPromise = new Promise<void>((resolve) => {
  authReadyResolve = resolve;
});

/**
 * Set the auth token getter function.
 * Called from a client component that has access to Clerk's useAuth.
 */
export function setAuthTokenGetter(getter: () => Promise<string | null>) {
  getAuthToken = getter;
  // Signal that auth is ready
  if (authReadyResolve) {
    authReadyResolve();
    authReadyResolve = null;
  }
}

/**
 * Wait for auth to be ready before making API calls.
 * Returns after auth getter is set or after timeout.
 */
async function waitForAuth(timeoutMs = 5000): Promise<void> {
  if (getAuthToken) return;

  // Race between auth being ready and timeout
  await Promise.race([
    authReadyPromise,
    new Promise<void>((resolve) => setTimeout(resolve, timeoutMs)),
  ]);
}

// Backend Asset type (what the API returns in snake_case)
interface BackendAsset {
  id: string;
  org_id: string;
  env_id?: string;
  platform: string;
  account?: string;
  region?: string;
  site?: string;
  instance_id: string;
  name?: string;
  image_ref?: string;
  image_version?: string;
  state: string;
  tags?: Record<string, string>;
  discovered_at: string;
  updated_at: string;
}

// Frontend Asset type (camelCase, enriched with display fields)
export interface Asset {
  id: string;
  orgId?: string;
  envId?: string;
  platform: "aws" | "azure" | "gcp" | "vsphere" | "k8s" | "baremetal";
  account?: string;
  region?: string;
  site?: string;
  instanceId?: string;
  name?: string;
  imageRef?: string;
  imageVersion?: string;
  state?: "running" | "stopped" | "terminated" | "pending" | "unknown";
  tags?: Record<string, string>;
  discoveredAt?: string;
  updatedAt?: string;
  // Computed/display fields for UI convenience
  hostname: string; // name || instanceId
  siteId: string;   // site || ""
  siteName: string; // site || "Unknown"
  // Drift-specific fields (populated by drift endpoints)
  isDrifted?: boolean;
  driftDetectedAt?: string;
  currentImageId?: string;
  currentImageVersion?: string;
  goldenImageId?: string;
  goldenImageVersion?: string;
  environment?: string;
  lastScannedAt?: string;
  createdAt?: string;
  metadata?: Record<string, string>;
}

// Transform backend asset to frontend format
function transformAsset(backend: BackendAsset, isDrifted?: boolean): Asset {
  return {
    id: backend.id,
    orgId: backend.org_id,
    envId: backend.env_id,
    platform: backend.platform as Asset["platform"],
    account: backend.account,
    region: backend.region,
    site: backend.site,
    instanceId: backend.instance_id,
    name: backend.name,
    imageRef: backend.image_ref,
    imageVersion: backend.image_version,
    state: backend.state as Asset["state"],
    tags: backend.tags,
    discoveredAt: backend.discovered_at,
    updatedAt: backend.updated_at,
    // Computed fields
    hostname: backend.name || backend.instance_id,
    siteId: backend.site || "",
    siteName: backend.site || "Unknown",
    isDrifted: isDrifted,
  };
}

export interface Image {
  id: string;
  familyId: string;
  familyName: string;
  version: string;
  description: string;
  status: "production" | "staging" | "testing" | "deprecated" | "pending";
  platforms: Array<"aws" | "azure" | "gcp" | "vsphere" | "k8s">;
  compliance: {
    cis: boolean;
    slsaLevel: number;
    cosignSigned: boolean;
  };
  deployedCount: number;
  createdAt: string;
  createdBy: string;
  promotedAt?: string;
  promotedBy?: string;
  deprecatedAt?: string;
}

export interface ImageFamily {
  id: string;
  name: string;
  description: string;
  owner: string;
  latestVersion: string;
  status: "production" | "staging" | "testing" | "deprecated" | "pending";
  totalDeployed: number;
  versions: Image[];
  createdAt: string;
  updatedAt: string;
}

export interface Site {
  id: string;
  name: string;
  region: string;
  platform: "aws" | "azure" | "gcp" | "vsphere" | "k8s";
  environment: "production" | "staging" | "development" | "dr";
  assetCount: number;
  compliantCount: number;
  driftedCount: number;
  coveragePercentage: number;
  status: "healthy" | "warning" | "critical";
  lastSyncAt: string;
  drPaired?: string;
  metadata: Record<string, string>;
}

export interface DriftSummary {
  totalAssets: number;
  compliantAssets: number;
  driftedAssets: number;
  driftPercentage: number;
  criticalDrift: number;
  averageDriftAge: string;
  byEnvironment: Array<{
    environment: string;
    compliant: number;
    total: number;
    percentage: number;
  }>;
  bySite: Array<{
    siteId: string;
    siteName: string;
    coverage: number;
    status: "success" | "warning" | "critical";
  }>;
  byAge: Array<{
    range: string;
    count: number;
    percentage: number;
  }>;
}

export interface Alert {
  id: string;
  severity: "critical" | "warning" | "info";
  title: string;
  description: string;
  source: string;
  siteId?: string;
  assetId?: string;
  imageId?: string;
  createdAt: string;
  acknowledgedAt?: string;
  resolvedAt?: string;
}

export interface Activity {
  id: string;
  orgId: string;
  type: string;
  action: string;
  detail: string;
  userId?: string;
  siteId?: string;
  assetId?: string;
  imageId?: string;
  timestamp: string;  // Required - matches OpenAPI spec
}

export interface ComplianceFramework {
  id: string;
  name: string;
  description: string;
  score: number;
  passingControls: number;
  totalControls: number;
  status: "passing" | "warning" | "failing";
  level?: number;
  category?: string;
  version?: string;
  regulatoryBody?: string;
}

// =============================================================================
// RBAC Types
// =============================================================================

export type RBACAction = "read" | "write" | "delete" | "execute" | "approve" | "admin";

export interface Role {
  id: string;
  name: string;
  displayName: string;
  description: string;
  orgId?: string;
  isSystemRole: boolean;
  parentRoleId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Permission {
  id: string;
  name: string;
  resourceType: string;
  action: RBACAction;
  description: string;
  isSystem: boolean;
}

export interface UserPermission {
  permissionName: string;
  resourceType: string;
  action: RBACAction;
  source: "role" | "direct" | "team";
}

export interface PermissionCheck {
  allowed: boolean;
  source: string;
  reason: string;
}

export interface Team {
  id: string;
  name: string;
  description: string;
  createdBy: string;
  createdAt: string;
}

export interface TeamMember {
  id: string;
  teamId: string;
  userId: string;
  role: "member" | "admin";
  addedAt: string;
}

// =============================================================================
// Multi-tenancy Types
// =============================================================================

export interface OrganizationQuota {
  orgId: string;
  maxAssets: number;
  maxImages: number;
  maxSites: number;
  maxUsers: number;
  maxTeams: number;
  maxAiTasksPerDay: number;
  maxAiTokensPerMonth: number;
  maxStorageBytes: number;
  apiRateLimitPerMinute: number;
  drEnabled: boolean;
  complianceEnabled: boolean;
  advancedAnalyticsEnabled: boolean;
}

export interface OrganizationUsage {
  orgId: string;
  assetCount: number;
  imageCount: number;
  siteCount: number;
  userCount: number;
  teamCount: number;
  storageUsedBytes: number;
  aiTasksToday: number;
  aiTokensThisMonth: number;
  apiRequestsToday: number;
}

export interface QuotaStatus {
  resourceType: string;
  limit: number;
  used: number;
  remaining: number;
  percentageUsed: number;
  isExceeded: boolean;
}

export interface SubscriptionPlan {
  id: string;
  name: string;
  displayName: string;
  description: string;
  planType: "free" | "starter" | "professional" | "enterprise";
  defaultMaxAssets: number;
  defaultMaxImages: number;
  defaultMaxSites: number;
  defaultMaxUsers: number;
  defaultMaxAiTasksPerDay: number;
  defaultMaxAiTokensPerMonth: number;
  defaultMaxStorageBytes: number;
  monthlyPriceUsd?: number;
  annualPriceUsd?: number;
  drIncluded: boolean;
  complianceIncluded: boolean;
  advancedAnalyticsIncluded: boolean;
  customIntegrationsIncluded: boolean;
  isActive: boolean;
}

export interface Subscription {
  id: string;
  orgId: string;
  planId: string;
  status: "active" | "cancelled" | "suspended" | "trial";
  trialEndsAt?: string;
  currentPeriodStart: string;
  currentPeriodEnd: string;
}

// =============================================================================
// Enhanced Compliance Types
// =============================================================================

export interface ComplianceControl {
  id: string;
  frameworkId: string;
  controlId: string;
  name: string;
  description: string;
  severity: "critical" | "high" | "medium" | "low";
  controlFamily: string;
  automationSupport: "automated" | "hybrid" | "manual";
  priority: string;
}

export interface ComplianceAssessment {
  id: string;
  frameworkId: string;
  name: string;
  description: string;
  assessmentType: "automated" | "manual" | "hybrid";
  status: "pending" | "in_progress" | "completed" | "failed";
  totalControls: number;
  passedControls: number;
  failedControls: number;
  notApplicable: number;
  score: number;
  startedAt?: string;
  completedAt?: string;
  initiatedBy: string;
}

export interface ComplianceEvidence {
  id: string;
  controlId: string;
  evidenceType: "screenshot" | "log" | "config" | "report" | "attestation";
  title: string;
  description: string;
  storageType: string;
  storagePath: string;
  collectedAt: string;
  collectedBy: string;
  isCurrent: boolean;
  reviewStatus: "pending" | "approved" | "rejected";
}

export interface ComplianceExemption {
  id: string;
  controlId: string;
  assetId?: string;
  siteId?: string;
  reason: string;
  riskAcceptance: string;
  compensatingControls: string;
  approvedBy?: string;
  approvedAt?: string;
  expiresAt: string;
  status: "active" | "expired" | "revoked";
}

export interface ComplianceScore {
  orgId: string;
  frameworkId?: string;
  assessmentCount: number;
  averageScore: number;
  totalPassed: number;
  totalFailed: number;
  totalNotApplicable: number;
  passRate: number;
}

export interface FailingControl {
  id: string;
  framework: string;
  title: string;
  severity: "high" | "medium" | "low";
  affectedAssets: number;
  recommendation: string;
}

export interface ImageComplianceStatus {
  familyId: string;
  familyName: string;
  version: string;
  cis: boolean;
  slsaLevel: number;
  cosignSigned: boolean;
  lastScanAt: string;
  issueCount: number;
}

// Image Lineage Types
export interface ImageLineageRelationship {
  id: string;
  imageId: string;
  parentImageId: string;
  relationshipType: "derived_from" | "patched_from" | "rebuilt_from";
  createdAt: string;
  parentImage?: {
    id: string;
    family: string;
    version: string;
    status: string;
  };
  image?: {
    id: string;
    family: string;
    version: string;
    status: string;
  };
}

export interface ImageBuild {
  id: string;
  imageId: string;
  buildNumber: number;
  sourceRepo?: string;
  sourceCommit?: string;
  sourceBranch?: string;
  builderType: string;
  builderVersion?: string;
  buildRunner?: string;
  buildRunnerId?: string;
  buildRunnerUrl?: string;
  buildLogUrl?: string;
  buildDurationSeconds?: number;
  builtBy?: string;
  signedBy?: string;
  status: "pending" | "building" | "success" | "failed";
  errorMessage?: string;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
}

export interface ImageVulnerability {
  id: string;
  imageId: string;
  cveId: string;
  severity: "critical" | "high" | "medium" | "low" | "unknown";
  cvssScore?: number;
  cvssVector?: string;
  packageName?: string;
  packageVersion?: string;
  packageType?: string;
  fixedVersion?: string;
  status: "open" | "fixed" | "wont_fix" | "false_positive";
  statusReason?: string;
  scanner?: string;
  scannedAt?: string;
  fixedInImageId?: string;
  resolvedAt?: string;
  resolvedBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface VulnerabilitySummary {
  imageId: string;
  family: string;
  version: string;
  criticalOpen: number;
  highOpen: number;
  mediumOpen: number;
  lowOpen: number;
  fixedCount: number;
  lastScannedAt?: string;
}

export interface ImageDeployment {
  id: string;
  imageId: string;
  assetId: string;
  deployedAt: string;
  deployedBy?: string;
  deploymentMethod?: string;
  status: "active" | "replaced" | "terminated";
  replacedAt?: string;
  replacedByImageId?: string;
  assetName?: string;
  platform?: string;
  region?: string;
}

export interface ImagePromotion {
  id: string;
  imageId: string;
  fromStatus: string;
  toStatus: string;
  promotedBy: string;
  approvedBy?: string;
  approvalTicket?: string;
  reason?: string;
  validationPassed?: boolean;
  promotedAt: string;
}

export interface ImageComponent {
  id: string;
  imageId: string;
  name: string;
  version: string;
  componentType: "os_package" | "library" | "binary" | "container";
  packageManager?: string;
  license?: string;
  licenseUrl?: string;
  sourceUrl?: string;
  checksum?: string;
  createdAt: string;
}

export interface LineageNode {
  image: {
    id: string;
    family: string;
    version: string;
    status: string;
    osName?: string;
    osVersion?: string;
  };
  depth: number;
  children?: LineageNode[];
  parents?: LineageNode[];
}

export interface ImageLineageTree {
  family: string;
  roots: LineageNode[];
  totalNodes: number;
}

export interface ImageLineageResponse {
  image: Image;
  parents: ImageLineageRelationship[];
  children: ImageLineageRelationship[];
  builds: ImageBuild[];
  vulnerabilitySummary: VulnerabilitySummary;
  activeDeployments: number;
  promotions: ImagePromotion[];
}

// Scanner Import Types
export type ScannerType = "trivy" | "grype" | "snyk" | "clair" | "anchore" | "aqua" | "twistlock" | "qualys";

export interface ScanVulnerability {
  cveId: string;
  severity: "critical" | "high" | "medium" | "low" | "unknown";
  cvssScore?: number;
  cvssVector?: string;
  packageName?: string;
  packageVersion?: string;
  packageType?: string;
  fixedVersion?: string;
  description?: string;
  references?: string[];
}

export interface ImportScanRequest {
  scanner: ScannerType;
  scanVersion?: string;
  scanStartedAt?: string;
  vulnerabilities: ScanVulnerability[];
}

export interface ImportScanResponse {
  imageId: string;
  scanner: string;
  imported: number;
  updated: number;
  fixed: number;
}

// SBOM Import Types
export type SBOMFormat = "spdx" | "cyclonedx" | "syft";

export interface SBOMComponent {
  name: string;
  version: string;
  componentType?: string;
  packageManager?: string;
  license?: string;
  licenseUrl?: string;
  sourceUrl?: string;
  checksum?: string;
  purl?: string;
}

export interface ImportSBOMRequest {
  format: SBOMFormat;
  sbomUrl?: string;
  components: SBOMComponent[];
}

export interface ImportSBOMResponse {
  imageId: string;
  format: string;
  components: number;
}

export interface ComplianceSummary {
  overallScore: number;
  cisCompliance: number;
  slsaLevel: number;
  sigstoreVerified: number;
  lastAuditAt: string;
  frameworks: ComplianceFramework[];
  failingControls: FailingControl[];
  imageCompliance: ImageComplianceStatus[];
}

export interface ResilienceSite {
  id: string;
  name: string;
  region: string;
  platform: "aws" | "azure" | "gcp" | "vsphere" | "k8s";
  type: "primary" | "dr";
  status: "healthy" | "warning" | "critical" | "syncing";
  assetCount: number;
  lastSyncAt: string;
  rpo: string;
  rto: string;
  replicationLag?: string;
}

export interface DRPair {
  id: string;
  name: string;
  primarySite: ResilienceSite;
  drSite: ResilienceSite;
  status: "healthy" | "warning" | "critical" | "syncing";
  lastFailoverTest?: string;
  replicationStatus: "in-sync" | "lagging" | "failed";
}

export interface ResilienceSummary {
  drReadiness: number;
  rpoCompliance: number;
  rtoCompliance: number;
  lastFailoverTest: string;
  totalPairs: number;
  healthyPairs: number;
  drPairs: DRPair[];
  unpairedSites: ResilienceSite[];
}

// Risk Scoring Types
export type RiskLevel = "critical" | "high" | "medium" | "low";

export interface RiskFactor {
  name: string;
  description: string;
  weight: number;
  score: number;
  impact: string;
}

export interface AssetRiskScore {
  assetId: string;
  assetName: string;
  platform: string;
  environment: string;
  site: string;
  riskScore: number;
  riskLevel: RiskLevel;
  factors: RiskFactor[];
  driftAge: number;
  vulnCount: number;
  criticalVulns: number;
  isCompliant: boolean;
  lastUpdated: string;
}

export interface RiskByScope {
  scope: string;
  riskScore: number;
  riskLevel: RiskLevel;
  assetCount: number;
  criticalRisk: number;
  highRisk: number;
}

export interface RiskTrendPoint {
  date: string;
  riskScore: number;
  riskLevel: RiskLevel;
}

export interface RiskSummary {
  orgId: string;
  overallRiskScore: number;
  riskLevel: RiskLevel;
  totalAssets: number;
  criticalRisk: number;
  highRisk: number;
  mediumRisk: number;
  lowRisk: number;
  topRisks: AssetRiskScore[];
  byEnvironment: RiskByScope[];
  byPlatform: RiskByScope[];
  bySite: RiskByScope[];
  trend: RiskTrendPoint[];
  calculatedAt: string;
}

// Predictive Risk Types
export type RiskVelocity = "rapid_increase" | "increasing" | "stable" | "decreasing" | "rapid_decrease";

export interface RiskPrediction {
  assetId?: string;
  scope?: string;
  currentScore: number;
  predictedScore: number;
  predictedLevel: RiskLevel;
  confidence: number;
  predictionHorizon: number;
  velocity: RiskVelocity;
  velocityValue: number;
  factors: string[];
  recommendedAction?: string;
  predictedAt: string;
}

export interface RiskAnomaly {
  id: string;
  assetId?: string;
  scope?: string;
  anomalyType: "spike" | "drop" | "pattern_break";
  severity: RiskLevel;
  description: string;
  expectedScore: number;
  actualScore: number;
  deviation: number;
  detectedAt: string;
  isActive: boolean;
}

export interface RiskRecommendation {
  id: string;
  priority: number;
  category: "patch" | "compliance" | "vulnerability" | "drift";
  title: string;
  description: string;
  impact: string;
  effort: "low" | "medium" | "high";
  affectedAssets: number;
  autoRemediable: boolean;
  actionType: "ai_task" | "manual" | "scheduled";
}

export interface RiskForecast {
  orgId: string;
  currentScore: number;
  predictions: RiskPrediction[];
  velocity: RiskVelocity;
  velocityValue: number;
  anomalies: RiskAnomaly[];
  atRiskAssets: AssetRiskScore[];
  improvingAssets: AssetRiskScore[];
  topRecommendations: RiskRecommendation[];
  generatedAt: string;
}

// Auto-Remediation Types
export interface MaintenanceWindow {
  dayOfWeek: number;
  startHour: number;
  endHour: number;
  timezone: string;
}

export interface AutoRemediationPolicy {
  id: string;
  orgId: string;
  name: string;
  description: string;
  enabled: boolean;
  maxRiskLevel: RiskLevel;
  environments: string[];
  platforms: string[];
  categories: string[];
  requireApproval: boolean;
  notifyOnAction: boolean;
  maxActionsPerDay: number;
  allowedWindows: MaintenanceWindow[];
  createdAt: string;
  updatedAt: string;
}

// =============================================================================
// Certificate Types
// =============================================================================

export type CertificateStatus = "active" | "expiring_soon" | "expired" | "revoked" | "pending_validation";
export type CertificatePlatform = "aws" | "azure" | "gcp" | "k8s" | "vsphere";
export type CertificateSource = "acm" | "key_vault" | "gcp_cert_manager" | "k8s_secret" | "file" | "manual";
export type KeyAlgorithm = "RSA" | "ECDSA" | "Ed25519";
export type RotationStatus = "pending" | "in_progress" | "completed" | "failed" | "rolled_back";
export type RotationType = "renewal" | "replacement" | "revocation";
export type RotationInitiator = "user" | "ai_agent" | "auto_renew" | "system";
export type AlertType = "expiry_warning" | "expiry_critical" | "validation_failed" | "rotation_failed";
export type AlertSeverity = "critical" | "high" | "medium" | "low";
export type AlertStatus = "open" | "acknowledged" | "resolved";
export type UsageType = "load_balancer" | "cloudfront" | "api_gateway" | "ingress" | "service" | "server";

export interface Certificate {
  id: string;
  orgId: string;
  fingerprint: string;
  serialNumber?: string;
  commonName: string;
  subjectAltNames?: string[];
  organization?: string;
  organizationalUnit?: string;
  country?: string;
  issuerCommonName?: string;
  issuerOrganization?: string;
  isSelfSigned: boolean;
  isCA: boolean;
  notBefore: string;
  notAfter: string;
  daysUntilExpiry: number;
  keyAlgorithm: KeyAlgorithm;
  keySize: number;
  signatureAlgorithm?: string;
  source: CertificateSource;
  sourceRef?: string;
  sourceRegion?: string;
  platform: CertificatePlatform;
  status: CertificateStatus;
  autoRenew: boolean;
  renewalThresholdDays?: number;
  lastRotatedAt?: string;
  rotationCount: number;
  tags?: Record<string, string>;
  metadata?: Record<string, unknown>;
  discoveredAt?: string;
  lastScannedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CertificateSummary {
  totalCertificates: number;
  activeCertificates: number;
  expiringSoon: number;
  expired: number;
  expiring7Days: number;
  expiring30Days: number;
  expiring90Days: number;
  autoRenewEnabled: number;
  selfSigned: number;
  platformsCount: number;
}

export interface CertificateUsage {
  id: string;
  certId: string;
  assetId?: string;
  usageType: UsageType;
  usageRef: string;
  usagePort?: number;
  platform: string;
  region?: string;
  serviceName?: string;
  endpoint?: string;
  status: "active" | "inactive" | "unknown";
  lastVerifiedAt?: string;
  tlsVersion?: string;
  metadata?: Record<string, unknown>;
  discoveredAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CertificateRotation {
  id: string;
  orgId: string;
  oldCertId?: string;
  newCertId?: string;
  rotationType: RotationType;
  initiatedBy: RotationInitiator;
  initiatedByUserId?: string;
  aiTaskId?: string;
  aiPlan?: Record<string, unknown>;
  status: RotationStatus;
  startedAt?: string;
  completedAt?: string;
  affectedUsages: number;
  successfulUpdates: number;
  failedUpdates: number;
  rollbackAvailable: boolean;
  rolledBackAt?: string;
  rollbackReason?: string;
  preRotationValidation?: Record<string, unknown>;
  postRotationValidation?: Record<string, unknown>;
  errorMessage?: string;
  errorDetails?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface CertificateAlert {
  id: string;
  orgId: string;
  certId: string;
  alertType: AlertType;
  severity: AlertSeverity;
  title: string;
  message: string;
  daysUntilExpiry?: number;
  thresholdDays?: number;
  status: AlertStatus;
  acknowledgedAt?: string;
  acknowledgedBy?: string;
  resolvedAt?: string;
  autoRotationTriggered: boolean;
  rotationId?: string;
  notificationsSent?: string[];
  createdAt: string;
  updatedAt: string;
}

export interface CertificateListResponse {
  certificates: Certificate[];
  total: number;
  page: number;
  pageSize: number;
}

export interface CertificateUsageResponse {
  usages: CertificateUsage[];
  totalUsages: number;
}

export interface RotationListResponse {
  rotations: CertificateRotation[];
  page: number;
  pageSize: number;
}

export interface AlertListResponse {
  alerts: CertificateAlert[];
  page: number;
  pageSize: number;
}

export interface OverviewMetrics {
  fleetSize: {
    value: number;
    trend: { direction: "up" | "down" | "neutral"; value: string; period: string };
  };
  driftScore: {
    value: number;
    trend: { direction: "up" | "down" | "neutral"; value: string; period: string };
  };
  compliance: {
    value: number;
    trend: { direction: "up" | "down" | "neutral"; value: string; period: string };
  };
  drReadiness: {
    value: number;
    trend: { direction: "up" | "down" | "neutral"; value: string; period: string };
  };
  platformDistribution: Array<{
    platform: "aws" | "azure" | "gcp" | "vsphere" | "k8s";
    count: number;
    percentage: number;
  }>;
  alerts: Array<{
    severity: "critical" | "warning" | "info";
    count: number;
  }>;
  recentActivity: Activity[];
}

// API Error type
export class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// Fetch wrapper with error handling and auth (exported for use by domain-specific API modules)
export async function apiFetch<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`;

  // Wait for auth to be ready before making API calls
  await waitForAuth();

  // Get auth token if available
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };

  if (getAuthToken) {
    const token = await getAuthToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new ApiError(
      response.status,
      response.statusText,
      errorBody || `API request failed: ${response.statusText}`
    );
  }

  // Handle empty responses
  const text = await response.text();
  if (!text) {
    return {} as T;
  }

  return JSON.parse(text) as T;
}

// API Client
export const api = {
  // Overview
  overview: {
    getMetrics: () => apiFetch<OverviewMetrics>("/overview/metrics"),
  },

  // Assets
  assets: {
    list: async (params?: {
      site?: string;
      platform?: string;
      state?: string;
      envId?: string;
      page?: number;
      pageSize?: number;
    }): Promise<{ assets: Asset[]; total: number; page: number; pageSize: number; totalPages: number }> => {
      const searchParams = new URLSearchParams();
      if (params) {
        // Map frontend params to backend params
        if (params.site) searchParams.set("site", params.site);
        if (params.platform) searchParams.set("platform", params.platform);
        if (params.state) searchParams.set("state", params.state);
        if (params.envId) searchParams.set("env_id", params.envId);
        if (params.page) searchParams.set("page", String(params.page));
        if (params.pageSize) searchParams.set("page_size", String(params.pageSize));
      }
      const query = searchParams.toString();

      // Backend response type
      interface BackendAssetListResponse {
        assets: BackendAsset[];
        total: number;
        page: number;
        page_size: number;
        total_pages: number;
      }

      const response = await apiFetch<BackendAssetListResponse>(
        `/assets${query ? `?${query}` : ""}`
      );

      return {
        assets: (response.assets || []).map(a => transformAsset(a)),
        total: response.total,
        page: response.page,
        pageSize: response.page_size,
        totalPages: response.total_pages,
      };
    },

    get: async (id: string): Promise<Asset> => {
      const response = await apiFetch<BackendAsset>(`/assets/${id}`);
      return transformAsset(response);
    },

    getSummary: () => apiFetch<{
      total_assets: number;
      running_assets: number;
      stopped_assets: number;
      by_platform: Record<string, number>;
      by_state: Record<string, number>;
    }>("/assets/summary"),

    // Get drifted assets via top-offenders endpoint (returns already transformed)
    getDrifted: (limit?: number) =>
      apiFetch<Asset[]>(`/drift/top-offenders${limit ? `?limit=${limit}` : ""}`),
  },

  // Images
  images: {
    // List images and group by family for backward compatibility with frontend
    listFamilies: async (): Promise<ImageFamily[]> => {
      // Backend returns snake_case fields, we need to handle the transformation
      interface BackendImage {
        id: string;
        org_id: string;
        family: string;  // Backend uses 'family' not 'familyId'
        version: string;
        os_name?: string;
        os_version?: string;
        signed?: boolean;
        status: string;
        created_at: string;
        updated_at: string;
      }

      // Map backend status to frontend status
      const mapStatus = (backendStatus: string): Image["status"] => {
        const statusMap: Record<string, Image["status"]> = {
          "published": "production",
          "production": "production",
          "staging": "staging",
          "testing": "testing",
          "deprecated": "deprecated",
          "pending": "pending",
        };
        return statusMap[backendStatus] || "pending";
      };

      const response = await apiFetch<{ images: BackendImage[]; total: number }>("/images");
      const backendImages = response.images || [];

      // Group images by family name
      const familyMap = new Map<string, BackendImage[]>();
      for (const img of backendImages) {
        const familyName = img.family || "unknown";
        if (!familyMap.has(familyName)) {
          familyMap.set(familyName, []);
        }
        familyMap.get(familyName)!.push(img);
      }

      // Convert to ImageFamily format
      const families: ImageFamily[] = [];
      for (const [familyName, familyImages] of familyMap) {
        // Find latest image for this family (by version or created_at)
        const sortedImages = [...familyImages].sort(
          (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        );
        const latestImage = sortedImages[0];

        // Transform backend images to frontend Image format
        const transformedVersions: Image[] = familyImages.map(img => ({
          id: img.id,
          familyId: familyName,
          familyName: familyName,
          version: img.version,
          description: `${img.os_name || ''} ${img.os_version || ''}`.trim(),
          status: mapStatus(img.status),
          platforms: ["aws"] as Image["platforms"], // Default, could be derived from coordinates
          compliance: {
            cis: false,
            slsaLevel: 0,
            cosignSigned: img.signed || false,
          },
          deployedCount: 0,
          createdAt: img.created_at,
          createdBy: "system",
        }));

        families.push({
          id: familyName, // Use family name as ID since backend doesn't have separate family entity
          name: familyName,
          description: `${latestImage?.os_name || ''} ${latestImage?.os_version || ''}`.trim() || "Golden image family",
          owner: "system",
          latestVersion: latestImage?.version || "0.0.0",
          status: mapStatus(latestImage?.status || "pending"),
          totalDeployed: 0,
          versions: transformedVersions,
          createdAt: sortedImages[sortedImages.length - 1]?.created_at || new Date().toISOString(),
          updatedAt: latestImage?.created_at || new Date().toISOString(),
        });
      }

      return families;
    },
    getFamily: (id: string) => apiFetch<ImageFamily>(`/images/${id}`),
    listVersions: (familyId: string) =>
      apiFetch<Image[]>(`/images?family=${familyId}`),
    getVersion: (familyId: string, version: string) =>
      apiFetch<Image>(`/images/family/${familyId}/version/${version}`),
    promote: (familyId: string, version: string, targetStatus: string) =>
      apiFetch<Image>(`/images/family/${familyId}/version/${version}/promote`, {
        method: "POST",
        body: JSON.stringify({ status: targetStatus }),
      }),
    deprecate: (familyId: string, version: string) =>
      apiFetch<Image>(`/images/family/${familyId}/version/${version}/promote`, {
        method: "POST",
        body: JSON.stringify({ status: "deprecated" }),
      }),
    // Lineage endpoints
    getLineage: (imageId: string) =>
      apiFetch<ImageLineageResponse>(`/images/${imageId}/lineage`),
    getLineageTree: (family: string) =>
      apiFetch<ImageLineageTree>(`/images/families/${family}/lineage-tree`),
    addParent: (imageId: string, parentImageId: string, relationshipType: string) =>
      apiFetch<ImageLineageRelationship>(`/images/${imageId}/lineage/parents`, {
        method: "POST",
        body: JSON.stringify({ parent_image_id: parentImageId, relationship_type: relationshipType }),
      }),
    getVulnerabilities: (imageId: string) =>
      apiFetch<ImageVulnerability[]>(`/images/${imageId}/vulnerabilities`),
    addVulnerability: (imageId: string, vulnerability: Partial<ImageVulnerability>) =>
      apiFetch<ImageVulnerability>(`/images/${imageId}/vulnerabilities`, {
        method: "POST",
        body: JSON.stringify(vulnerability),
      }),
    getBuilds: (imageId: string) =>
      apiFetch<ImageBuild[]>(`/images/${imageId}/builds`),
    getDeployments: (imageId: string) =>
      apiFetch<ImageDeployment[]>(`/images/${imageId}/deployments`),
    getComponents: (imageId: string) =>
      apiFetch<ImageComponent[]>(`/images/${imageId}/components`),
    // Import endpoints
    importScanResults: (imageId: string, data: ImportScanRequest) =>
      apiFetch<ImportScanResponse>(`/images/${imageId}/vulnerabilities/import`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    importSBOM: (imageId: string, data: ImportSBOMRequest) =>
      apiFetch<ImportSBOMResponse>(`/images/${imageId}/sbom`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
  },

  // Sites
  sites: {
    list: async (): Promise<Site[]> => {
      const response = await apiFetch<{ sites: Site[] }>("/sites");
      return response.sites || [];
    },
    get: (id: string) => apiFetch<Site>(`/sites/${id}`),
    getAssets: (id: string, limit?: number) =>
      apiFetch<Asset[]>(`/sites/${id}/assets${limit ? `?limit=${limit}` : ""}`),
  },

  // Drift
  drift: {
    getSummary: () => apiFetch<DriftSummary>("/drift/summary"),
    getTopOffenders: (limit?: number) =>
      apiFetch<Asset[]>(`/drift/top-offenders${limit ? `?limit=${limit}` : ""}`),
    triggerScan: (siteId?: string) =>
      apiFetch<{ jobId: string }>("/drift/scan", {
        method: "POST",
        body: JSON.stringify({ siteId }),
      }),
  },

  // Alerts
  alerts: {
    list: (params?: { severity?: string; limit?: number }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined) {
            searchParams.set(key, String(value));
          }
        });
      }
      const query = searchParams.toString();
      return apiFetch<Alert[]>(`/alerts${query ? `?${query}` : ""}`);
    },
    acknowledge: (id: string) =>
      apiFetch<Alert>(`/alerts/${id}/acknowledge`, { method: "POST" }),
    resolve: (id: string) =>
      apiFetch<Alert>(`/alerts/${id}/resolve`, { method: "POST" }),
  },

  // Activity
  activity: {
    list: (limit?: number) =>
      apiFetch<Activity[]>(`/activity${limit ? `?limit=${limit}` : ""}`),
  },

  // Compliance
  compliance: {
    getSummary: () => apiFetch<ComplianceSummary>("/compliance/summary"),
    getFrameworks: () => apiFetch<ComplianceFramework[]>("/compliance/frameworks"),
    getFailingControls: (framework?: string) => {
      const query = framework ? `?framework=${framework}` : "";
      return apiFetch<FailingControl[]>(`/compliance/controls/failing${query}`);
    },
    getImageCompliance: () => apiFetch<ImageComplianceStatus[]>("/compliance/images"),
    runAudit: () =>
      apiFetch<{ jobId: string }>("/compliance/audit", { method: "POST" }),
  },

  // Resilience
  resilience: {
    getSummary: () => apiFetch<ResilienceSummary>("/resilience/summary"),
    getDRPairs: () => apiFetch<DRPair[]>("/resilience/dr-pairs"),
    getDRPair: (id: string) => apiFetch<DRPair>(`/resilience/dr-pairs/${id}`),
    triggerFailoverTest: (pairId: string) =>
      apiFetch<{ jobId: string }>(`/resilience/dr-pairs/${pairId}/test`, {
        method: "POST",
      }),
    triggerSync: (pairId: string) =>
      apiFetch<{ jobId: string }>(`/resilience/dr-pairs/${pairId}/sync`, {
        method: "POST",
      }),
  },

  // Risk Scoring
  risk: {
    getSummary: () => apiFetch<RiskSummary>("/risk/summary"),
    getTopRisks: (limit?: number) =>
      apiFetch<AssetRiskScore[]>(`/risk/top${limit ? `?limit=${limit}` : ""}`),
    // Predictive risk endpoints
    getForecast: () => apiFetch<RiskForecast>("/risk/forecast"),
    getRecommendations: () => apiFetch<RiskRecommendation[]>("/risk/recommendations"),
    getAnomalies: () => apiFetch<RiskAnomaly[]>("/risk/anomalies"),
    getAssetPrediction: (assetId: string) =>
      apiFetch<RiskPrediction>(`/risk/assets/${assetId}/prediction`),
  },

  // =============================================================================
  // RBAC API
  // =============================================================================
  rbac: {
    // Roles
    listRoles: () =>
      apiFetch<{ roles: Role[] }>("/rbac/roles"),
    getRole: (roleId: string) =>
      apiFetch<Role>(`/rbac/roles/${roleId}`),

    // Permissions
    listPermissions: () =>
      apiFetch<{ permissions: Permission[] }>("/rbac/permissions"),

    // User roles
    getUserRoles: (userId: string) =>
      apiFetch<{ roles: Role[] }>(`/rbac/users/${userId}/roles`),
    assignRole: (userId: string, roleId: string, expiresAt?: string) =>
      apiFetch<void>(`/rbac/users/${userId}/roles`, {
        method: "POST",
        body: JSON.stringify({ role_id: roleId, expires_at: expiresAt }),
      }),
    revokeRole: (userId: string, roleId: string) =>
      apiFetch<void>(`/rbac/users/${userId}/roles/${roleId}`, {
        method: "DELETE",
      }),

    // User permissions
    getUserPermissions: (userId: string) =>
      apiFetch<{ permissions: UserPermission[] }>(`/rbac/users/${userId}/permissions`),

    // Permission check
    checkPermission: (userId: string, resourceType: string, action: string, resourceId?: string) =>
      apiFetch<PermissionCheck>("/rbac/check", {
        method: "POST",
        body: JSON.stringify({
          user_id: userId,
          resource_type: resourceType,
          action: action,
          resource_id: resourceId,
        }),
      }),

    // Teams
    listTeams: () =>
      apiFetch<{ teams: Team[] }>("/rbac/teams"),
    createTeam: (name: string, description?: string) =>
      apiFetch<Team>("/rbac/teams", {
        method: "POST",
        body: JSON.stringify({ name, description }),
      }),
    getTeamMembers: (teamId: string) =>
      apiFetch<{ members: TeamMember[] }>(`/rbac/teams/${teamId}/members`),
    addTeamMember: (teamId: string, userId: string, role: "member" | "admin" = "member") =>
      apiFetch<void>(`/rbac/teams/${teamId}/members`, {
        method: "POST",
        body: JSON.stringify({ user_id: userId, role }),
      }),
  },

  // =============================================================================
  // Organization / Multi-tenancy API
  // =============================================================================
  organization: {
    // Get current organization
    get: () =>
      apiFetch<{
        id: string;
        name: string;
        slug: string;
        created_at: string;
        updated_at: string;
      }>("/organization"),

    // Check if user has an organization
    check: () =>
      apiFetch<{
        has_organization: boolean;
        organization: {
          id: string;
          name: string;
          slug: string;
          created_at: string;
          updated_at: string;
        } | null;
      }>("/organization/check"),

    // Create organization
    create: (data: { name: string; slug?: string; plan_id?: string }) =>
      apiFetch<{
        organization: {
          id: string;
          name: string;
          slug: string;
          created_at: string;
          updated_at: string;
        };
        quota: OrganizationQuota;
        subscription: Subscription;
      }>("/organizations", {
        method: "POST",
        body: JSON.stringify(data),
      }),

    // Quota
    getQuota: () =>
      apiFetch<OrganizationQuota>("/organization/quota"),

    // Usage
    getUsage: () =>
      apiFetch<OrganizationUsage>("/organization/usage"),

    // Quota status
    getQuotaStatus: () =>
      apiFetch<{ statuses: QuotaStatus[] }>("/organization/quota-status"),

    // Subscription
    getSubscription: () =>
      apiFetch<Subscription>("/organization/subscription"),

    // Plans
    listPlans: () =>
      apiFetch<{ plans: SubscriptionPlan[] }>("/organization/plans"),

    // Seed demo data
    seedDemo: (platform: "aws" | "azure" | "gcp") =>
      apiFetch<{
        sites_created: number;
        assets_created: number;
        images_created: number;
      }>("/organization/seed-demo", {
        method: "POST",
        body: JSON.stringify({ platform }),
      }),
  },

  // =============================================================================
  // Enhanced Compliance API
  // =============================================================================
  complianceV2: {
    // Frameworks
    listFrameworks: () =>
      apiFetch<{ frameworks: ComplianceFramework[] }>("/compliance/frameworks"),

    // Controls
    listControls: (frameworkId: string) =>
      apiFetch<{ controls: ComplianceControl[] }>(`/compliance/frameworks/${frameworkId}/controls`),
    getControlMappings: (controlId: string) =>
      apiFetch<{ mappings: ComplianceControl[] }>(`/compliance/controls/${controlId}/mappings`),

    // Assessments
    listAssessments: (frameworkId?: string, limit?: number) => {
      const params = new URLSearchParams();
      if (frameworkId) params.set("framework_id", frameworkId);
      if (limit) params.set("limit", String(limit));
      const query = params.toString();
      return apiFetch<{ assessments: ComplianceAssessment[] }>(`/compliance/assessments${query ? `?${query}` : ""}`);
    },
    getAssessment: (assessmentId: string) =>
      apiFetch<ComplianceAssessment>(`/compliance/assessments/${assessmentId}`),
    createAssessment: (data: {
      frameworkId: string;
      name: string;
      description?: string;
      assessmentType?: "automated" | "manual" | "hybrid";
      scopeSites?: string[];
      scopeAssets?: string[];
    }) =>
      apiFetch<ComplianceAssessment>("/compliance/assessments", {
        method: "POST",
        body: JSON.stringify({
          framework_id: data.frameworkId,
          name: data.name,
          description: data.description,
          assessment_type: data.assessmentType || "automated",
          scope_sites: data.scopeSites,
          scope_assets: data.scopeAssets,
        }),
      }),

    // Evidence
    listEvidence: (controlId?: string, currentOnly?: boolean) => {
      const params = new URLSearchParams();
      if (controlId) params.set("control_id", controlId);
      if (currentOnly !== undefined) params.set("current_only", String(currentOnly));
      const query = params.toString();
      return apiFetch<{ evidence: ComplianceEvidence[] }>(`/compliance/evidence${query ? `?${query}` : ""}`);
    },
    uploadEvidence: (data: {
      controlId: string;
      evidenceType: "screenshot" | "log" | "config" | "report" | "attestation";
      title: string;
      description?: string;
      storageType: string;
      storagePath?: string;
      validUntil?: string;
    }) =>
      apiFetch<ComplianceEvidence>("/compliance/evidence", {
        method: "POST",
        body: JSON.stringify({
          control_id: data.controlId,
          evidence_type: data.evidenceType,
          title: data.title,
          description: data.description,
          storage_type: data.storageType,
          storage_path: data.storagePath,
          valid_until: data.validUntil,
        }),
      }),

    // Exemptions
    listExemptions: () =>
      apiFetch<{ exemptions: ComplianceExemption[] }>("/compliance/exemptions"),
    createExemption: (data: {
      controlId: string;
      assetId?: string;
      siteId?: string;
      reason: string;
      riskAcceptance?: string;
      compensatingControls?: string;
      expiresAt: string;
    }) =>
      apiFetch<ComplianceExemption>("/compliance/exemptions", {
        method: "POST",
        body: JSON.stringify({
          control_id: data.controlId,
          asset_id: data.assetId,
          site_id: data.siteId,
          reason: data.reason,
          risk_acceptance: data.riskAcceptance,
          compensating_controls: data.compensatingControls,
          expires_at: data.expiresAt,
        }),
      }),

    // Score
    getScore: (frameworkId?: string) => {
      const query = frameworkId ? `?framework_id=${frameworkId}` : "";
      return apiFetch<ComplianceScore>(`/compliance/score${query}`);
    },
  },

  // =============================================================================
  // Certificate Lifecycle Management API
  // =============================================================================
  certificates: {
    // List certificates with optional filters
    list: async (params?: {
      status?: CertificateStatus;
      platform?: CertificatePlatform;
      expiringWithinDays?: number;
      page?: number;
      pageSize?: number;
    }): Promise<CertificateListResponse> => {
      const searchParams = new URLSearchParams();
      if (params?.status) searchParams.set("status", params.status);
      if (params?.platform) searchParams.set("platform", params.platform);
      if (params?.expiringWithinDays) searchParams.set("expiring_within_days", String(params.expiringWithinDays));
      if (params?.page) searchParams.set("page", String(params.page));
      if (params?.pageSize) searchParams.set("page_size", String(params.pageSize));
      const query = searchParams.toString();

      interface BackendCertificateListResponse {
        certificates: Array<{
          id: string;
          org_id: string;
          fingerprint: string;
          serial_number?: string;
          common_name: string;
          subject_alt_names?: string[];
          organization?: string;
          organizational_unit?: string;
          country?: string;
          issuer_common_name?: string;
          issuer_organization?: string;
          is_self_signed: boolean;
          is_ca: boolean;
          not_before: string;
          not_after: string;
          days_until_expiry: number;
          key_algorithm: string;
          key_size: number;
          signature_algorithm?: string;
          source: string;
          source_ref?: string;
          source_region?: string;
          platform: string;
          status: string;
          auto_renew: boolean;
          renewal_threshold_days?: number;
          last_rotated_at?: string;
          rotation_count: number;
          tags?: Record<string, string>;
          metadata?: Record<string, unknown>;
          discovered_at?: string;
          last_scanned_at?: string;
          created_at: string;
          updated_at: string;
        }>;
        total: number;
        page: number;
        page_size: number;
      }

      const response = await apiFetch<BackendCertificateListResponse>(
        `/certificates${query ? `?${query}` : ""}`
      );

      return {
        certificates: (response.certificates || []).map(c => ({
          id: c.id,
          orgId: c.org_id,
          fingerprint: c.fingerprint,
          serialNumber: c.serial_number,
          commonName: c.common_name,
          subjectAltNames: c.subject_alt_names,
          organization: c.organization,
          organizationalUnit: c.organizational_unit,
          country: c.country,
          issuerCommonName: c.issuer_common_name,
          issuerOrganization: c.issuer_organization,
          isSelfSigned: c.is_self_signed,
          isCA: c.is_ca,
          notBefore: c.not_before,
          notAfter: c.not_after,
          daysUntilExpiry: c.days_until_expiry,
          keyAlgorithm: c.key_algorithm as KeyAlgorithm,
          keySize: c.key_size,
          signatureAlgorithm: c.signature_algorithm,
          source: c.source as CertificateSource,
          sourceRef: c.source_ref,
          sourceRegion: c.source_region,
          platform: c.platform as CertificatePlatform,
          status: c.status as CertificateStatus,
          autoRenew: c.auto_renew,
          renewalThresholdDays: c.renewal_threshold_days,
          lastRotatedAt: c.last_rotated_at,
          rotationCount: c.rotation_count,
          tags: c.tags,
          metadata: c.metadata,
          discoveredAt: c.discovered_at,
          lastScannedAt: c.last_scanned_at,
          createdAt: c.created_at,
          updatedAt: c.updated_at,
        })),
        total: response.total,
        page: response.page,
        pageSize: response.page_size,
      };
    },

    // Get certificate summary
    getSummary: async (): Promise<CertificateSummary> => {
      interface BackendSummary {
        total_certificates: number;
        active_certificates: number;
        expiring_soon: number;
        expired: number;
        expiring_7_days: number;
        expiring_30_days: number;
        expiring_90_days: number;
        auto_renew_enabled: number;
        self_signed: number;
        platforms_count: number;
      }
      const response = await apiFetch<BackendSummary>("/certificates/summary");
      return {
        totalCertificates: response.total_certificates,
        activeCertificates: response.active_certificates,
        expiringSoon: response.expiring_soon,
        expired: response.expired,
        expiring7Days: response.expiring_7_days,
        expiring30Days: response.expiring_30_days,
        expiring90Days: response.expiring_90_days,
        autoRenewEnabled: response.auto_renew_enabled,
        selfSigned: response.self_signed,
        platformsCount: response.platforms_count,
      };
    },

    // Get single certificate
    get: (id: string) => apiFetch<Certificate>(`/certificates/${id}`),

    // Get certificate usage (blast radius)
    getUsage: async (id: string): Promise<CertificateUsageResponse> => {
      interface BackendUsageResponse {
        usages: Array<{
          id: string;
          cert_id: string;
          asset_id?: string;
          usage_type: string;
          usage_ref: string;
          usage_port?: number;
          platform: string;
          region?: string;
          service_name?: string;
          endpoint?: string;
          status: string;
          last_verified_at?: string;
          tls_version?: string;
          metadata?: Record<string, unknown>;
          discovered_at?: string;
          created_at: string;
          updated_at: string;
        }>;
        total_usages: number;
      }
      const response = await apiFetch<BackendUsageResponse>(`/certificates/${id}/usage`);
      return {
        usages: (response.usages || []).map(u => ({
          id: u.id,
          certId: u.cert_id,
          assetId: u.asset_id,
          usageType: u.usage_type as UsageType,
          usageRef: u.usage_ref,
          usagePort: u.usage_port,
          platform: u.platform,
          region: u.region,
          serviceName: u.service_name,
          endpoint: u.endpoint,
          status: u.status as "active" | "inactive" | "unknown",
          lastVerifiedAt: u.last_verified_at,
          tlsVersion: u.tls_version,
          metadata: u.metadata,
          discoveredAt: u.discovered_at,
          createdAt: u.created_at,
          updatedAt: u.updated_at,
        })),
        totalUsages: response.total_usages,
      };
    },

    // List rotations
    listRotations: async (params?: {
      status?: RotationStatus;
      page?: number;
      pageSize?: number;
    }): Promise<RotationListResponse> => {
      const searchParams = new URLSearchParams();
      if (params?.status) searchParams.set("status", params.status);
      if (params?.page) searchParams.set("page", String(params.page));
      if (params?.pageSize) searchParams.set("page_size", String(params.pageSize));
      const query = searchParams.toString();

      interface BackendRotation {
        id: string;
        org_id: string;
        old_cert_id?: string;
        new_cert_id?: string;
        rotation_type: string;
        initiated_by: string;
        initiated_by_user_id?: string;
        ai_task_id?: string;
        ai_plan?: Record<string, unknown>;
        status: string;
        started_at?: string;
        completed_at?: string;
        affected_usages: number;
        successful_updates: number;
        failed_updates: number;
        rollback_available: boolean;
        rolled_back_at?: string;
        rollback_reason?: string;
        pre_rotation_validation?: Record<string, unknown>;
        post_rotation_validation?: Record<string, unknown>;
        error_message?: string;
        error_details?: Record<string, unknown>;
        created_at: string;
        updated_at: string;
      }

      interface BackendResponse {
        rotations: BackendRotation[];
        page: number;
        page_size: number;
      }

      const response = await apiFetch<BackendResponse>(
        `/certificates/rotations${query ? `?${query}` : ""}`
      );

      return {
        rotations: (response.rotations || []).map(r => ({
          id: r.id,
          orgId: r.org_id,
          oldCertId: r.old_cert_id,
          newCertId: r.new_cert_id,
          rotationType: r.rotation_type as RotationType,
          initiatedBy: r.initiated_by as RotationInitiator,
          initiatedByUserId: r.initiated_by_user_id,
          aiTaskId: r.ai_task_id,
          aiPlan: r.ai_plan,
          status: r.status as RotationStatus,
          startedAt: r.started_at,
          completedAt: r.completed_at,
          affectedUsages: r.affected_usages,
          successfulUpdates: r.successful_updates,
          failedUpdates: r.failed_updates,
          rollbackAvailable: r.rollback_available,
          rolledBackAt: r.rolled_back_at,
          rollbackReason: r.rollback_reason,
          preRotationValidation: r.pre_rotation_validation,
          postRotationValidation: r.post_rotation_validation,
          errorMessage: r.error_message,
          errorDetails: r.error_details,
          createdAt: r.created_at,
          updatedAt: r.updated_at,
        })),
        page: response.page,
        pageSize: response.page_size,
      };
    },

    // Get single rotation
    getRotation: (id: string) => apiFetch<CertificateRotation>(`/certificates/rotations/${id}`),

    // List alerts
    listAlerts: async (params?: {
      status?: AlertStatus;
      severity?: AlertSeverity;
      page?: number;
      pageSize?: number;
    }): Promise<AlertListResponse> => {
      const searchParams = new URLSearchParams();
      if (params?.status) searchParams.set("status", params.status);
      if (params?.severity) searchParams.set("severity", params.severity);
      if (params?.page) searchParams.set("page", String(params.page));
      if (params?.pageSize) searchParams.set("page_size", String(params.pageSize));
      const query = searchParams.toString();

      interface BackendAlert {
        id: string;
        org_id: string;
        cert_id: string;
        alert_type: string;
        severity: string;
        title: string;
        message: string;
        days_until_expiry?: number;
        threshold_days?: number;
        status: string;
        acknowledged_at?: string;
        acknowledged_by?: string;
        resolved_at?: string;
        auto_rotation_triggered: boolean;
        rotation_id?: string;
        notifications_sent?: string[];
        created_at: string;
        updated_at: string;
      }

      interface BackendResponse {
        alerts: BackendAlert[];
        page: number;
        page_size: number;
      }

      const response = await apiFetch<BackendResponse>(
        `/certificates/alerts${query ? `?${query}` : ""}`
      );

      return {
        alerts: (response.alerts || []).map(a => ({
          id: a.id,
          orgId: a.org_id,
          certId: a.cert_id,
          alertType: a.alert_type as AlertType,
          severity: a.severity as AlertSeverity,
          title: a.title,
          message: a.message,
          daysUntilExpiry: a.days_until_expiry,
          thresholdDays: a.threshold_days,
          status: a.status as AlertStatus,
          acknowledgedAt: a.acknowledged_at,
          acknowledgedBy: a.acknowledged_by,
          resolvedAt: a.resolved_at,
          autoRotationTriggered: a.auto_rotation_triggered,
          rotationId: a.rotation_id,
          notificationsSent: a.notifications_sent,
          createdAt: a.created_at,
          updatedAt: a.updated_at,
        })),
        page: response.page,
        pageSize: response.page_size,
      };
    },

    // Acknowledge alert
    acknowledgeAlert: (id: string) =>
      apiFetch<{ status: string }>(`/certificates/alerts/${id}/acknowledge`, {
        method: "POST",
      }),
  },
};

export default api;

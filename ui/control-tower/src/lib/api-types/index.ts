/**
 * API Types - Generated from OpenAPI Spec
 *
 * This file re-exports types from the generated schema for easier consumption.
 * DO NOT manually edit schema.ts - it is auto-generated.
 *
 * To regenerate: npm run generate:api-types
 */

import type { components, operations } from './schema';

// ==================== Core Types ====================
export type UUID = components['schemas']['UUID'];
export type Timestamp = components['schemas']['Timestamp'];
export type ErrorResponse = components['schemas']['ErrorResponse'];

// ==================== Trend Types ====================
export type TrendDirection = components['schemas']['TrendDirection'];
export type MetricTrend = components['schemas']['MetricTrend'];
export type MetricWithTrend = components['schemas']['MetricWithTrend'];
export type FloatMetricWithTrend = components['schemas']['FloatMetricWithTrend'];

// ==================== Platform Types ====================
export type Platform = components['schemas']['Platform'];
export type PlatformCount = components['schemas']['PlatformCount'];

// ==================== Alert Types ====================
export type AlertSeverity = components['schemas']['AlertSeverity'];
export type AlertStatus = components['schemas']['AlertStatus'];
export type AlertCount = components['schemas']['AlertCount'];
export type Alert = components['schemas']['Alert'];
export type AlertListResponse = components['schemas']['AlertListResponse'];

// ==================== Activity Types ====================
export type ActivityType = components['schemas']['ActivityType'];
export type Activity = components['schemas']['Activity'];

// ==================== Drift Types ====================
export type DriftStatus = components['schemas']['DriftStatus'];
export type DriftBySite = components['schemas']['DriftBySite'];
export type DriftByEnvironment = components['schemas']['DriftByEnvironment'];
export type DriftByAge = components['schemas']['DriftByAge'];
export type DriftSummaryResponse = components['schemas']['DriftSummaryResponse'];

// ==================== Overview Types ====================
export type OverviewMetrics = components['schemas']['OverviewMetrics'];

// ==================== Operation Response Types ====================
export type GetOverviewMetricsResponse = operations['getOverviewMetrics']['responses']['200']['content']['application/json'];
export type GetDriftSummaryResponse = operations['getDriftSummary']['responses']['200']['content']['application/json'];
export type ListAlertsResponse = operations['listAlerts']['responses']['200']['content']['application/json'];

// ==================== Type Guards ====================
export const isValidAlertSeverity = (value: string): value is AlertSeverity => {
  return ['critical', 'high', 'medium', 'warning', 'low', 'info'].includes(value);
};

export const isValidDriftStatus = (value: string): value is DriftStatus => {
  return ['healthy', 'warning', 'critical'].includes(value);
};

export const isValidActivityType = (value: string): value is ActivityType => {
  return ['info', 'warning', 'success', 'critical', 'patch', 'image', 'compliance', 'dr_test', 'ai_task'].includes(value);
};

export const isValidPlatform = (value: string): value is Platform => {
  return ['aws', 'azure', 'gcp', 'vsphere', 'k8s'].includes(value);
};

// ==================== Mapping Functions ====================
/**
 * Maps API AlertSeverity to UI StatusVariant
 * Use this when displaying alerts in UI components that use StatusBadge
 */
export type UIStatusVariant = 'success' | 'warning' | 'critical' | 'neutral' | 'info';

export const mapAlertSeverityToUIStatus = (severity: AlertSeverity): UIStatusVariant => {
  switch (severity) {
    case 'critical':
    case 'high':
      return 'critical';
    case 'medium':
    case 'warning':
      return 'warning';
    case 'low':
      return 'info';
    case 'info':
      return 'neutral';
    default:
      return 'neutral';
  }
};

/**
 * Maps API DriftStatus to UI StatusVariant
 * Use this when displaying drift status in UI components
 */
export const mapDriftStatusToUIStatus = (status: DriftStatus): UIStatusVariant => {
  switch (status) {
    case 'healthy':
      return 'success';
    case 'warning':
      return 'warning';
    case 'critical':
      return 'critical';
    default:
      return 'neutral';
  }
};

/**
 * Determines drift status based on coverage percentage
 */
export const getDriftStatusFromCoverage = (coverage: number): DriftStatus => {
  if (coverage >= 90) return 'healthy';
  if (coverage >= 70) return 'warning';
  return 'critical';
};

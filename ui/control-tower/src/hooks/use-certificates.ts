/**
 * React Query hooks for Certificate Lifecycle Management
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@clerk/nextjs";
import {
  api,
  type Certificate,
  type CertificateSummary,
  type CertificateUsageResponse,
  type CertificateListResponse,
  type RotationListResponse,
  type AlertListResponse,
  type CertificateStatus,
  type CertificatePlatform,
  type RotationStatus,
  type AlertStatus,
  type AlertSeverity,
} from "@/lib/api";

// Query keys
export const certificateKeys = {
  all: ["certificates"] as const,
  lists: () => [...certificateKeys.all, "list"] as const,
  list: (params?: {
    status?: CertificateStatus;
    platform?: CertificatePlatform;
    expiringWithinDays?: number;
    page?: number;
    pageSize?: number;
  }) => [...certificateKeys.lists(), params] as const,
  summary: () => [...certificateKeys.all, "summary"] as const,
  details: () => [...certificateKeys.all, "detail"] as const,
  detail: (id: string) => [...certificateKeys.details(), id] as const,
  usage: (id: string) => [...certificateKeys.all, "usage", id] as const,
  rotations: () => [...certificateKeys.all, "rotations"] as const,
  rotationList: (params?: {
    status?: RotationStatus;
    page?: number;
    pageSize?: number;
  }) => [...certificateKeys.rotations(), params] as const,
  rotation: (id: string) => [...certificateKeys.rotations(), id] as const,
  alerts: () => [...certificateKeys.all, "alerts"] as const,
  alertList: (params?: {
    status?: AlertStatus;
    severity?: AlertSeverity;
    page?: number;
    pageSize?: number;
  }) => [...certificateKeys.alerts(), params] as const,
};

/**
 * Hook to fetch list of certificates
 */
export function useCertificates(params?: {
  status?: CertificateStatus;
  platform?: CertificatePlatform;
  expiringWithinDays?: number;
  page?: number;
  pageSize?: number;
}) {
  const { isLoaded, isSignedIn } = useAuth();

  return useQuery<CertificateListResponse>({
    queryKey: certificateKeys.list(params),
    queryFn: () => api.certificates.list(params),
    staleTime: 1000 * 60 * 5, // 5 minutes
    enabled: isLoaded && isSignedIn,
  });
}

/**
 * Hook to fetch certificate summary
 */
export function useCertificateSummary() {
  const { isLoaded, isSignedIn } = useAuth();

  return useQuery<CertificateSummary>({
    queryKey: certificateKeys.summary(),
    queryFn: () => api.certificates.getSummary(),
    staleTime: 1000 * 60 * 2, // 2 minutes
    enabled: isLoaded && isSignedIn,
  });
}

/**
 * Hook to fetch a specific certificate
 */
export function useCertificate(id: string) {
  const { isLoaded, isSignedIn } = useAuth();

  return useQuery<Certificate>({
    queryKey: certificateKeys.detail(id),
    queryFn: () => api.certificates.get(id),
    enabled: isLoaded && isSignedIn && !!id,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch certificate usage (blast radius)
 */
export function useCertificateUsage(id: string) {
  const { isLoaded, isSignedIn } = useAuth();

  return useQuery<CertificateUsageResponse>({
    queryKey: certificateKeys.usage(id),
    queryFn: () => api.certificates.getUsage(id),
    enabled: isLoaded && isSignedIn && !!id,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch certificate rotations
 */
export function useCertificateRotations(params?: {
  status?: RotationStatus;
  page?: number;
  pageSize?: number;
}) {
  const { isLoaded, isSignedIn } = useAuth();

  return useQuery<RotationListResponse>({
    queryKey: certificateKeys.rotationList(params),
    queryFn: () => api.certificates.listRotations(params),
    staleTime: 1000 * 60 * 2, // 2 minutes
    enabled: isLoaded && isSignedIn,
  });
}

/**
 * Hook to fetch certificate alerts
 */
export function useCertificateAlerts(params?: {
  status?: AlertStatus;
  severity?: AlertSeverity;
  page?: number;
  pageSize?: number;
}) {
  const { isLoaded, isSignedIn } = useAuth();

  return useQuery<AlertListResponse>({
    queryKey: certificateKeys.alertList(params),
    queryFn: () => api.certificates.listAlerts(params),
    staleTime: 1000 * 60 * 1, // 1 minute
    enabled: isLoaded && isSignedIn,
  });
}

/**
 * Hook to acknowledge a certificate alert
 */
export function useAcknowledgeCertificateAlert() {
  const queryClient = useQueryClient();

  return useMutation<{ status: string }, Error, string>({
    mutationFn: (id) => api.certificates.acknowledgeAlert(id),
    onSuccess: () => {
      // Invalidate alerts queries to refetch fresh data
      queryClient.invalidateQueries({ queryKey: certificateKeys.alerts() });
    },
  });
}

/**
 * Hook to get certificates expiring soon (convenience hook)
 */
export function useExpiringCertificates(days: number = 30) {
  return useCertificates({ expiringWithinDays: days });
}

/**
 * Hook to get expired certificates (convenience hook)
 */
export function useExpiredCertificates() {
  return useCertificates({ status: "expired" });
}

/**
 * Hook to get certificates by platform (convenience hook)
 */
export function useCertificatesByPlatform(platform: CertificatePlatform) {
  return useCertificates({ platform });
}

/**
 * Hook to get open alerts (convenience hook)
 */
export function useOpenCertificateAlerts() {
  return useCertificateAlerts({ status: "open" });
}

/**
 * Hook to get critical alerts (convenience hook)
 */
export function useCriticalCertificateAlerts() {
  return useCertificateAlerts({ severity: "critical", status: "open" });
}

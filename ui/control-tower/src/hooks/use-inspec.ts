/**
 * React Query hooks for InSpec data
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  inspecApi,
  InSpecProfile,
  InSpecRun,
  RunResultsResponse,
  ControlMapping,
  ScanSchedule,
  TriggerScanRequest,
  CreateScheduleRequest,
} from "@/lib/api-inspec";

// Query keys
export const inspecKeys = {
  all: ["inspec"] as const,
  profiles: () => [...inspecKeys.all, "profiles"] as const,
  profile: (id: string) => [...inspecKeys.all, "profile", id] as const,
  scans: (params?: { limit?: number; offset?: number }) =>
    [...inspecKeys.all, "scans", params] as const,
  scan: (id: string) => [...inspecKeys.all, "scan", id] as const,
  scanResults: (scanId: string) => [...inspecKeys.all, "scan-results", scanId] as const,
  controlMappings: (profileId: string) =>
    [...inspecKeys.all, "control-mappings", profileId] as const,
  schedules: () => [...inspecKeys.all, "schedules"] as const,
};

/**
 * Hook to fetch all InSpec profiles
 */
export function useProfiles() {
  return useQuery<InSpecProfile[]>({
    queryKey: inspecKeys.profiles(),
    queryFn: () => inspecApi.getProfiles(),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to fetch a single InSpec profile
 */
export function useProfile(id: string) {
  return useQuery<InSpecProfile>({
    queryKey: inspecKeys.profile(id),
    queryFn: () => inspecApi.getProfile(id),
    enabled: !!id,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to trigger an InSpec scan (mutation)
 */
export function useTriggerScan() {
  const queryClient = useQueryClient();

  return useMutation<InSpecRun, Error, TriggerScanRequest>({
    mutationFn: (data) => inspecApi.triggerScan(data),
    onSuccess: () => {
      // Invalidate scans to refetch the list
      queryClient.invalidateQueries({ queryKey: inspecKeys.scans() });
    },
  });
}

/**
 * Hook to fetch InSpec scans/runs
 */
export function useScans(params?: { limit?: number; offset?: number }) {
  return useQuery<{ runs: InSpecRun[]; limit: number; offset: number }>({
    queryKey: inspecKeys.scans(params),
    queryFn: () => inspecApi.getScans(params),
    staleTime: 1000 * 60 * 2, // 2 minutes
  });
}

/**
 * Hook to fetch a single scan
 */
export function useScan(id: string) {
  return useQuery<InSpecRun>({
    queryKey: inspecKeys.scan(id),
    queryFn: () => inspecApi.getScan(id),
    enabled: !!id,
    // Poll every 5 seconds if scan is in progress
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data?.status === "pending" || data?.status === "running") {
        return 5000;
      }
      return false;
    },
  });
}

/**
 * Hook to fetch scan results
 */
export function useScanResults(scanId: string) {
  return useQuery<RunResultsResponse>({
    queryKey: inspecKeys.scanResults(scanId),
    queryFn: () => inspecApi.getScanResults(scanId),
    enabled: !!scanId,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to cancel a scan (mutation)
 */
export function useCancelScan() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: (scanId) => inspecApi.cancelScan(scanId),
    onSuccess: (_, scanId) => {
      // Invalidate the specific scan and the scans list
      queryClient.invalidateQueries({ queryKey: inspecKeys.scan(scanId) });
      queryClient.invalidateQueries({ queryKey: inspecKeys.scans() });
    },
  });
}

/**
 * Hook to fetch control mappings for a profile
 */
export function useControlMappings(profileId: string) {
  return useQuery<ControlMapping[]>({
    queryKey: inspecKeys.controlMappings(profileId),
    queryFn: () => inspecApi.getControlMappings(profileId),
    enabled: !!profileId,
    staleTime: 1000 * 60 * 10, // 10 minutes
  });
}

/**
 * Hook to fetch scan schedules
 */
export function useSchedules() {
  return useQuery<ScanSchedule[]>({
    queryKey: inspecKeys.schedules(),
    queryFn: () => inspecApi.getSchedules(),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to create a scan schedule (mutation)
 */
export function useCreateSchedule() {
  const queryClient = useQueryClient();

  return useMutation<ScanSchedule, Error, CreateScheduleRequest>({
    mutationFn: (data) => inspecApi.createSchedule(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: inspecKeys.schedules() });
    },
  });
}

/**
 * Hook to delete a scan schedule (mutation)
 */
export function useDeleteSchedule() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: (id) => inspecApi.deleteSchedule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: inspecKeys.schedules() });
    },
  });
}

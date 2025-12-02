import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, Alert } from "@/lib/api";

interface UseAlertsParams {
  severity?: string;
  limit?: number;
}

export const alertKeys = {
  all: ["alerts"] as const,
  lists: () => [...alertKeys.all, "list"] as const,
  list: (filters: UseAlertsParams) =>
    [...alertKeys.lists(), filters] as const,
};

export function useAlerts(params?: UseAlertsParams) {
  return useQuery<Alert[]>({
    queryKey: alertKeys.list(params ?? {}),
    queryFn: () => api.alerts.list(params),
    staleTime: 30 * 1000,
    refetchInterval: 30 * 1000, // More frequent for alerts
  });
}

export function useAcknowledgeAlert() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.alerts.acknowledge(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.all });
    },
  });
}

export function useResolveAlert() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.alerts.resolve(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.all });
    },
  });
}

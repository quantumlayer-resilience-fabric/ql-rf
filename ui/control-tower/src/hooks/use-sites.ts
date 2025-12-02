import { useQuery } from "@tanstack/react-query";
import { api, Site, Asset } from "@/lib/api";

export const siteKeys = {
  all: ["sites"] as const,
  lists: () => [...siteKeys.all, "list"] as const,
  details: () => [...siteKeys.all, "detail"] as const,
  detail: (id: string) => [...siteKeys.details(), id] as const,
  assets: (id: string, limit?: number) =>
    [...siteKeys.all, "assets", id, limit] as const,
};

export function useSites() {
  return useQuery<Site[]>({
    queryKey: siteKeys.lists(),
    queryFn: () => api.sites.list(),
    staleTime: 60 * 1000,
  });
}

export function useSite(id: string) {
  return useQuery<Site>({
    queryKey: siteKeys.detail(id),
    queryFn: () => api.sites.get(id),
    enabled: !!id,
    staleTime: 60 * 1000,
  });
}

export function useSiteAssets(id: string, limit?: number) {
  return useQuery<Asset[]>({
    queryKey: siteKeys.assets(id, limit),
    queryFn: () => api.sites.getAssets(id, limit),
    enabled: !!id,
    staleTime: 30 * 1000,
  });
}

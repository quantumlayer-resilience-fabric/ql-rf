import { useQuery } from "@tanstack/react-query";
import { api, Asset } from "@/lib/api";

interface UseAssetsParams {
  site?: string;
  platform?: string;
  state?: string;
  envId?: string;
  page?: number;
  pageSize?: number;
}

interface AssetListResponse {
  assets: Asset[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export const assetKeys = {
  all: ["assets"] as const,
  lists: () => [...assetKeys.all, "list"] as const,
  list: (filters: UseAssetsParams) =>
    [...assetKeys.lists(), filters] as const,
  details: () => [...assetKeys.all, "detail"] as const,
  detail: (id: string) => [...assetKeys.details(), id] as const,
  drifted: (limit?: number) => [...assetKeys.all, "drifted", limit] as const,
  summary: () => [...assetKeys.all, "summary"] as const,
};

export function useAssets(params?: UseAssetsParams) {
  return useQuery<AssetListResponse>({
    queryKey: assetKeys.list(params ?? {}),
    queryFn: () => api.assets.list(params),
    staleTime: 30 * 1000,
    retry: 2,
  });
}

export function useAsset(id: string) {
  return useQuery<Asset>({
    queryKey: assetKeys.detail(id),
    queryFn: () => api.assets.get(id),
    enabled: !!id,
    staleTime: 60 * 1000,
    retry: 2,
  });
}

export function useDriftedAssets(limit?: number) {
  return useQuery<Asset[]>({
    queryKey: assetKeys.drifted(limit),
    queryFn: () => api.assets.getDrifted(limit),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
    retry: 2,
  });
}

export function useAssetSummary() {
  return useQuery({
    queryKey: assetKeys.summary(),
    queryFn: () => api.assets.getSummary(),
    staleTime: 60 * 1000,
    retry: 2,
  });
}

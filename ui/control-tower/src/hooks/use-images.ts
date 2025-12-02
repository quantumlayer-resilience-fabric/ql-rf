import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, ImageFamily, Image } from "@/lib/api";

export const imageKeys = {
  all: ["images"] as const,
  families: () => [...imageKeys.all, "families"] as const,
  family: (id: string) => [...imageKeys.families(), id] as const,
  versions: (familyId: string) =>
    [...imageKeys.all, "versions", familyId] as const,
  version: (familyId: string, version: string) =>
    [...imageKeys.versions(familyId), version] as const,
};

export function useImageFamilies() {
  return useQuery<ImageFamily[]>({
    queryKey: imageKeys.families(),
    queryFn: () => api.images.listFamilies(),
    staleTime: 60 * 1000,
  });
}

export function useImageFamily(id: string) {
  return useQuery<ImageFamily>({
    queryKey: imageKeys.family(id),
    queryFn: () => api.images.getFamily(id),
    enabled: !!id,
    staleTime: 60 * 1000,
  });
}

export function useImageVersions(familyId: string) {
  return useQuery<Image[]>({
    queryKey: imageKeys.versions(familyId),
    queryFn: () => api.images.listVersions(familyId),
    enabled: !!familyId,
    staleTime: 60 * 1000,
  });
}

export function useImageVersion(familyId: string, version: string) {
  return useQuery<Image>({
    queryKey: imageKeys.version(familyId, version),
    queryFn: () => api.images.getVersion(familyId, version),
    enabled: !!familyId && !!version,
    staleTime: 60 * 1000,
  });
}

export function usePromoteImage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      familyId,
      version,
      targetStatus,
    }: {
      familyId: string;
      version: string;
      targetStatus: string;
    }) => api.images.promote(familyId, version, targetStatus),
    onSuccess: (_, variables) => {
      // Invalidate related queries
      queryClient.invalidateQueries({ queryKey: imageKeys.families() });
      queryClient.invalidateQueries({
        queryKey: imageKeys.family(variables.familyId),
      });
      queryClient.invalidateQueries({
        queryKey: imageKeys.versions(variables.familyId),
      });
    },
  });
}

export function useDeprecateImage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      familyId,
      version,
    }: {
      familyId: string;
      version: string;
    }) => api.images.deprecate(familyId, version),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: imageKeys.families() });
      queryClient.invalidateQueries({
        queryKey: imageKeys.family(variables.familyId),
      });
      queryClient.invalidateQueries({
        queryKey: imageKeys.versions(variables.familyId),
      });
    },
  });
}

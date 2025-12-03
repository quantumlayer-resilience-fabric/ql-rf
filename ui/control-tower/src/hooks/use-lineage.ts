import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  api,
  ImageLineageResponse,
  ImageLineageTree,
  ImageVulnerability,
  ImageBuild,
  ImageDeployment,
  ImageComponent,
  ImageLineageRelationship,
} from "@/lib/api";

export const lineageKeys = {
  all: ["lineage"] as const,
  lineage: (imageId: string) => [...lineageKeys.all, "image", imageId] as const,
  tree: (family: string) => [...lineageKeys.all, "tree", family] as const,
  vulnerabilities: (imageId: string) =>
    [...lineageKeys.all, "vulnerabilities", imageId] as const,
  builds: (imageId: string) => [...lineageKeys.all, "builds", imageId] as const,
  deployments: (imageId: string) =>
    [...lineageKeys.all, "deployments", imageId] as const,
  components: (imageId: string) =>
    [...lineageKeys.all, "components", imageId] as const,
};

export function useImageLineage(imageId: string) {
  return useQuery<ImageLineageResponse>({
    queryKey: lineageKeys.lineage(imageId),
    queryFn: () => api.images.getLineage(imageId),
    enabled: !!imageId,
    staleTime: 60 * 1000,
  });
}

export function useImageLineageTree(family: string) {
  return useQuery<ImageLineageTree>({
    queryKey: lineageKeys.tree(family),
    queryFn: () => api.images.getLineageTree(family),
    enabled: !!family,
    staleTime: 60 * 1000,
  });
}

export function useImageVulnerabilities(imageId: string) {
  return useQuery<ImageVulnerability[]>({
    queryKey: lineageKeys.vulnerabilities(imageId),
    queryFn: () => api.images.getVulnerabilities(imageId),
    enabled: !!imageId,
    staleTime: 60 * 1000,
  });
}

export function useImageBuilds(imageId: string) {
  return useQuery<ImageBuild[]>({
    queryKey: lineageKeys.builds(imageId),
    queryFn: () => api.images.getBuilds(imageId),
    enabled: !!imageId,
    staleTime: 60 * 1000,
  });
}

export function useImageDeployments(imageId: string) {
  return useQuery<ImageDeployment[]>({
    queryKey: lineageKeys.deployments(imageId),
    queryFn: () => api.images.getDeployments(imageId),
    enabled: !!imageId,
    staleTime: 30 * 1000, // More frequent updates for deployments
  });
}

export function useImageComponents(imageId: string) {
  return useQuery<ImageComponent[]>({
    queryKey: lineageKeys.components(imageId),
    queryFn: () => api.images.getComponents(imageId),
    enabled: !!imageId,
    staleTime: 5 * 60 * 1000, // SBOM data changes less frequently
  });
}

export function useAddParentImage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      imageId,
      parentImageId,
      relationshipType,
    }: {
      imageId: string;
      parentImageId: string;
      relationshipType: string;
    }) => api.images.addParent(imageId, parentImageId, relationshipType),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: lineageKeys.lineage(variables.imageId),
      });
      queryClient.invalidateQueries({
        queryKey: lineageKeys.all,
      });
    },
  });
}

export function useAddVulnerability() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      imageId,
      vulnerability,
    }: {
      imageId: string;
      vulnerability: Partial<ImageVulnerability>;
    }) => api.images.addVulnerability(imageId, vulnerability),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: lineageKeys.vulnerabilities(variables.imageId),
      });
      queryClient.invalidateQueries({
        queryKey: lineageKeys.lineage(variables.imageId),
      });
    },
  });
}

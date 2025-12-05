"use client";

import { useMemo } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Package, Image as ImageIcon, Server, AlertTriangle, ChevronRight, GitBranch } from "lucide-react";
import { cn } from "@/lib/utils";

interface AffectedPackage {
  name: string;
  version: string;
  type: string;
  fixed_version?: string;
}

interface AffectedImage {
  id: string;
  name: string;
  version: string;
  lineage_depth: number;
  is_direct: boolean;
  children_count?: number;
}

interface AffectedAsset {
  id: string;
  name: string;
  platform: string;
  region: string;
  environment: string;
  is_production: boolean;
}

interface BlastRadiusDiagramProps {
  packages: AffectedPackage[];
  images: AffectedImage[];
  assets: AffectedAsset[];
  className?: string;
}

// Platform colors
const platformColors: Record<string, string> = {
  aws: "bg-[#FF9900]/20 text-[#FF9900] border-[#FF9900]/30",
  azure: "bg-[#0078D4]/20 text-[#0078D4] border-[#0078D4]/30",
  gcp: "bg-[#4285F4]/20 text-[#4285F4] border-[#4285F4]/30",
  kubernetes: "bg-[#326CE5]/20 text-[#326CE5] border-[#326CE5]/30",
  vsphere: "bg-[#6DB33F]/20 text-[#6DB33F] border-[#6DB33F]/30",
};

// Environment styles
const envStyles: Record<string, string> = {
  production: "bg-status-red/10 text-status-red border-status-red/30",
  staging: "bg-status-amber/10 text-status-amber border-status-amber/30",
  development: "bg-blue-500/10 text-blue-500 border-blue-500/30",
  test: "bg-gray-500/10 text-gray-500 border-gray-500/30",
};

function PackageNode({ pkg }: { pkg: AffectedPackage }) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex items-center gap-2 rounded-lg border border-purple-500/30 bg-purple-500/10 px-3 py-2 text-sm transition-all hover:bg-purple-500/20 hover:scale-105 cursor-pointer">
            <Package className="h-4 w-4 text-purple-500" />
            <span className="font-medium text-purple-700 dark:text-purple-300">{pkg.name}</span>
            <Badge variant="outline" className="text-xs">
              {pkg.version}
            </Badge>
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">
          <div className="space-y-1 text-xs">
            <p><strong>Package:</strong> {pkg.name}</p>
            <p><strong>Version:</strong> {pkg.version}</p>
            <p><strong>Type:</strong> {pkg.type}</p>
            {pkg.fixed_version && (
              <p className="text-green-500"><strong>Fix:</strong> {pkg.fixed_version}</p>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function ImageNode({ image, showLineage = false }: { image: AffectedImage; showLineage?: boolean }) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className={cn(
            "flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-all hover:scale-105 cursor-pointer",
            image.is_direct
              ? "border-blue-500/30 bg-blue-500/10 hover:bg-blue-500/20"
              : "border-cyan-500/30 bg-cyan-500/10 hover:bg-cyan-500/20"
          )}>
            <ImageIcon className={cn(
              "h-4 w-4",
              image.is_direct ? "text-blue-500" : "text-cyan-500"
            )} />
            <span className={cn(
              "font-medium",
              image.is_direct ? "text-blue-700 dark:text-blue-300" : "text-cyan-700 dark:text-cyan-300"
            )}>
              {image.name}
            </span>
            <Badge variant="outline" className="text-xs">
              {image.version}
            </Badge>
            {showLineage && image.lineage_depth > 0 && (
              <Badge variant="secondary" className="text-xs gap-1">
                <GitBranch className="h-3 w-3" />
                L{image.lineage_depth}
              </Badge>
            )}
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">
          <div className="space-y-1 text-xs">
            <p><strong>Image:</strong> {image.name}:{image.version}</p>
            <p><strong>Type:</strong> {image.is_direct ? "Direct" : "Inherited"}</p>
            <p><strong>Lineage Depth:</strong> {image.lineage_depth}</p>
            {image.children_count && image.children_count > 0 && (
              <p><strong>Child Images:</strong> {image.children_count}</p>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function AssetNode({ asset }: { asset: AffectedAsset }) {
  const platformClass = platformColors[asset.platform.toLowerCase()] || "bg-gray-500/10 text-gray-500 border-gray-500/30";
  const envClass = envStyles[asset.environment.toLowerCase()] || envStyles.development;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className={cn(
            "flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-all hover:scale-105 cursor-pointer",
            asset.is_production
              ? "border-status-red/30 bg-status-red/10 hover:bg-status-red/20"
              : "border-green-500/30 bg-green-500/10 hover:bg-green-500/20"
          )}>
            <Server className={cn(
              "h-4 w-4",
              asset.is_production ? "text-status-red" : "text-green-500"
            )} />
            <span className={cn(
              "font-medium",
              asset.is_production ? "text-status-red" : "text-green-700 dark:text-green-300"
            )}>
              {asset.name}
            </span>
            {asset.is_production && (
              <Badge variant="destructive" className="text-xs">
                PROD
              </Badge>
            )}
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">
          <div className="space-y-1 text-xs">
            <p><strong>Asset:</strong> {asset.name}</p>
            <div className="flex items-center gap-1">
              <strong>Platform:</strong>
              <span className={cn("px-1.5 py-0.5 rounded border text-[10px]", platformClass)}>
                {asset.platform}
              </span>
            </div>
            <p><strong>Region:</strong> {asset.region}</p>
            <div className="flex items-center gap-1">
              <strong>Environment:</strong>
              <span className={cn("px-1.5 py-0.5 rounded border text-[10px]", envClass)}>
                {asset.environment}
              </span>
            </div>
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function ConnectionLine({ className }: { className?: string }) {
  return (
    <div className={cn("flex items-center justify-center", className)}>
      <div className="h-0.5 w-8 bg-gradient-to-r from-muted-foreground/20 to-muted-foreground/40" />
      <ChevronRight className="h-4 w-4 text-muted-foreground/40 -mx-1" />
      <div className="h-0.5 w-8 bg-gradient-to-r from-muted-foreground/40 to-muted-foreground/20" />
    </div>
  );
}

function VerticalLine() {
  return (
    <div className="flex flex-col items-center">
      <div className="w-0.5 h-4 bg-gradient-to-b from-muted-foreground/40 to-muted-foreground/20" />
    </div>
  );
}

export function BlastRadiusDiagram({
  packages,
  images,
  assets,
  className,
}: BlastRadiusDiagramProps) {
  // Group images by lineage depth
  const imagesByDepth = useMemo(() => {
    const grouped: Record<number, AffectedImage[]> = {};
    images.forEach((img) => {
      const depth = img.lineage_depth;
      if (!grouped[depth]) grouped[depth] = [];
      grouped[depth].push(img);
    });
    return grouped;
  }, [images]);

  // Group assets by environment
  const assetsByEnv = useMemo(() => {
    const prod = assets.filter((a) => a.is_production);
    const nonProd = assets.filter((a) => !a.is_production);
    return { prod, nonProd };
  }, [assets]);

  const productionCount = assetsByEnv.prod.length;
  const nonProdCount = assetsByEnv.nonProd.length;

  return (
    <Card className={cn("overflow-hidden", className)}>
      <CardHeader className="pb-4">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-status-amber" />
              Blast Radius Visualization
            </CardTitle>
            <CardDescription>
              Impact propagation from vulnerable packages to production assets
            </CardDescription>
          </div>
          <div className="flex items-center gap-4 text-sm">
            <div className="flex items-center gap-2">
              <div className="h-3 w-3 rounded-full bg-purple-500" />
              <span className="text-muted-foreground">{packages.length} Packages</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="h-3 w-3 rounded-full bg-blue-500" />
              <span className="text-muted-foreground">{images.length} Images</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="h-3 w-3 rounded-full bg-status-red" />
              <span className="text-muted-foreground">{productionCount} Production</span>
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {/* Flow Diagram */}
        <div className="flex items-start justify-between gap-4 overflow-x-auto pb-4">
          {/* Packages Column */}
          <div className="flex flex-col items-center min-w-[180px]">
            <Badge variant="secondary" className="mb-4 gap-1">
              <Package className="h-3 w-3" />
              Vulnerable Packages
            </Badge>
            <div className="flex flex-col gap-2">
              {packages.slice(0, 5).map((pkg, idx) => (
                <PackageNode key={`${pkg.name}-${idx}`} pkg={pkg} />
              ))}
              {packages.length > 5 && (
                <Badge variant="outline" className="self-center">
                  +{packages.length - 5} more
                </Badge>
              )}
            </div>
          </div>

          {/* Connection */}
          <ConnectionLine className="mt-16" />

          {/* Images Column */}
          <div className="flex flex-col items-center min-w-[200px]">
            <Badge variant="secondary" className="mb-4 gap-1">
              <ImageIcon className="h-3 w-3" />
              Affected Images
            </Badge>
            <div className="flex flex-col gap-2">
              {Object.entries(imagesByDepth).slice(0, 3).map(([depth, imgs]) => (
                <div key={depth} className="space-y-2">
                  {Number(depth) === 0 && imgs.length > 0 && (
                    <div className="text-xs text-muted-foreground text-center mb-1">Direct</div>
                  )}
                  {Number(depth) > 0 && imgs.length > 0 && (
                    <>
                      <VerticalLine />
                      <div className="text-xs text-muted-foreground text-center mb-1">
                        Inherited (L{depth})
                      </div>
                    </>
                  )}
                  {imgs.slice(0, 3).map((img) => (
                    <ImageNode key={img.id} image={img} showLineage />
                  ))}
                  {imgs.length > 3 && (
                    <Badge variant="outline" className="self-center">
                      +{imgs.length - 3} more
                    </Badge>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Connection */}
          <ConnectionLine className="mt-16" />

          {/* Assets Column */}
          <div className="flex flex-col items-center min-w-[200px]">
            <Badge variant="secondary" className="mb-4 gap-1">
              <Server className="h-3 w-3" />
              Affected Assets
            </Badge>
            <div className="flex flex-col gap-4">
              {/* Production Assets */}
              {assetsByEnv.prod.length > 0 && (
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-xs text-status-red">
                    <AlertTriangle className="h-3 w-3" />
                    Production ({productionCount})
                  </div>
                  {assetsByEnv.prod.slice(0, 3).map((asset) => (
                    <AssetNode key={asset.id} asset={asset} />
                  ))}
                  {productionCount > 3 && (
                    <Badge variant="destructive" className="self-center">
                      +{productionCount - 3} more production
                    </Badge>
                  )}
                </div>
              )}

              {/* Non-Production Assets */}
              {assetsByEnv.nonProd.length > 0 && (
                <div className="space-y-2">
                  <div className="text-xs text-muted-foreground text-center">
                    Non-Production ({nonProdCount})
                  </div>
                  {assetsByEnv.nonProd.slice(0, 2).map((asset) => (
                    <AssetNode key={asset.id} asset={asset} />
                  ))}
                  {nonProdCount > 2 && (
                    <Badge variant="outline" className="self-center">
                      +{nonProdCount - 2} more
                    </Badge>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Summary Stats */}
        <div className="mt-6 pt-4 border-t border-border">
          <div className="grid grid-cols-4 gap-4 text-center">
            <div>
              <div className="text-2xl font-bold text-purple-500">{packages.length}</div>
              <div className="text-xs text-muted-foreground">Vulnerable Packages</div>
            </div>
            <div>
              <div className="text-2xl font-bold text-blue-500">{images.length}</div>
              <div className="text-xs text-muted-foreground">Affected Images</div>
            </div>
            <div>
              <div className="text-2xl font-bold text-green-500">{assets.length}</div>
              <div className="text-xs text-muted-foreground">Total Assets</div>
            </div>
            <div>
              <div className="text-2xl font-bold text-status-red">{productionCount}</div>
              <div className="text-xs text-muted-foreground">Production at Risk</div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export default BlastRadiusDiagram;

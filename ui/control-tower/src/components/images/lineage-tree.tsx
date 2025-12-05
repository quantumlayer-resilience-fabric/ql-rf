"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/status/status-badge";
import { LineageNode, ImageLineageTree } from "@/lib/api";
import {
  ChevronDown,
  ChevronRight,
  GitBranch,
  Box,
  ArrowRight,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface LineageTreeProps {
  tree: ImageLineageTree;
  onSelectImage?: (imageId: string) => void;
  selectedImageId?: string;
}

type ImageStatus = "production" | "staging" | "deprecated" | "pending";

const statusConfig: Record<ImageStatus, { label: string; variant: "success" | "warning" | "critical" | "info" }> = {
  production: { label: "Production", variant: "success" },
  staging: { label: "Staging", variant: "warning" },
  deprecated: { label: "Deprecated", variant: "critical" },
  pending: { label: "Pending", variant: "info" },
};


function TreeNode({
  node,
  depth,
  onSelect,
  selectedId,
  isLast,
}: {
  node: LineageNode;
  depth: number;
  onSelect?: (imageId: string) => void;
  selectedId?: string;
  isLast: boolean;
}) {
  const [isExpanded, setIsExpanded] = useState(depth < 2);
  const hasChildren = node.children && node.children.length > 0;
  const isSelected = selectedId === node.image.id;
  const config = statusConfig[node.image.status as ImageStatus] || statusConfig.pending;

  return (
    <div className="relative">
      {/* Tree line connectors */}
      {depth > 0 && (
        <div className="absolute left-[-24px] top-0 h-6 w-6 border-l-2 border-b-2 border-border rounded-bl-lg" />
      )}
      {depth > 0 && !isLast && (
        <div className="absolute left-[-24px] top-6 bottom-0 w-0.5 bg-border" />
      )}

      <div className="flex items-start gap-2 mb-2">
        {/* Expand/collapse button */}
        {hasChildren ? (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0"
            onClick={() => setIsExpanded(!isExpanded)}
          >
            {isExpanded ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
          </Button>
        ) : (
          <div className="w-6" />
        )}

        {/* Node content */}
        <div
          className={cn(
            "flex-1 flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors",
            isSelected
              ? "bg-brand-accent/10 border-brand-accent"
              : "bg-background hover:bg-muted/50 border-border"
          )}
          onClick={() => onSelect?.(node.image.id)}
        >
          <div className="flex h-8 w-8 items-center justify-center rounded-md bg-muted">
            <Box className="h-4 w-4 text-muted-foreground" />
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium truncate">{node.image.family}</span>
              <code className="text-xs rounded bg-muted px-1.5 py-0.5">
                v{node.image.version}
              </code>
            </div>
            {node.image.osName && (
              <div className="text-xs text-muted-foreground">
                {node.image.osName} {node.image.osVersion}
              </div>
            )}
          </div>

          <StatusBadge status={config.variant} size="sm">
            {config.label}
          </StatusBadge>
        </div>
      </div>

      {/* Children */}
      {hasChildren && isExpanded && (
        <div className="ml-8 pl-4 border-l-2 border-border">
          {node.children!.map((child, index) => (
            <TreeNode
              key={child.image.id}
              node={child}
              depth={depth + 1}
              onSelect={onSelect}
              selectedId={selectedId}
              isLast={index === node.children!.length - 1}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export function LineageTreeView({ tree, onSelectImage, selectedImageId }: LineageTreeProps) {
  if (!tree.roots || tree.roots.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12 text-center">
          <GitBranch className="h-12 w-12 text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium">No Lineage Data</h3>
          <p className="text-sm text-muted-foreground mt-1">
            No parent-child relationships have been defined for this image family.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <GitBranch className="h-5 w-5" />
            Lineage Tree: {tree.family}
          </CardTitle>
          <Badge variant="secondary">{tree.totalNodes} versions</Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {tree.roots.map((root, index) => (
            <TreeNode
              key={root.image.id}
              node={root}
              depth={0}
              onSelect={onSelectImage}
              selectedId={selectedImageId}
              isLast={index === tree.roots.length - 1}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

// Compact horizontal lineage view
interface LineagePathProps {
  parents: Array<{ id: string; family: string; version: string; status: string }>;
  current: { id: string; family: string; version: string; status: string };
  childImages: Array<{ id: string; family: string; version: string; status: string }>;
  onSelectImage?: (imageId: string) => void;
}

export function LineagePath({ parents, current, childImages, onSelectImage }: LineagePathProps) {
  const renderNode = (
    image: { id: string; family: string; version: string; status: string },
    isCurrent: boolean = false
  ) => {
    const config = statusConfig[image.status as ImageStatus] || statusConfig.pending;
    return (
      <div
        key={image.id}
        className={cn(
          "flex items-center gap-2 p-2 rounded-lg border cursor-pointer transition-colors",
          isCurrent
            ? "bg-brand-accent/10 border-brand-accent"
            : "bg-muted/50 hover:bg-muted border-border"
        )}
        onClick={() => onSelectImage?.(image.id)}
      >
        <Box className="h-4 w-4 text-muted-foreground" />
        <div className="min-w-0">
          <div className="text-sm font-medium truncate">{image.family}</div>
          <code className="text-xs text-muted-foreground">v{image.version}</code>
        </div>
        <StatusBadge status={config.variant} size="sm">
          {config.label}
        </StatusBadge>
      </div>
    );
  };

  return (
    <div className="flex items-center gap-2 flex-wrap">
      {/* Parents */}
      {parents.length > 0 && (
        <>
          <div className="flex items-center gap-2">
            {parents.map((parent) => renderNode(parent))}
          </div>
          <ArrowRight className="h-4 w-4 text-muted-foreground" />
        </>
      )}

      {/* Current */}
      {renderNode(current, true)}

      {/* Children */}
      {childImages.length > 0 && (
        <>
          <ArrowRight className="h-4 w-4 text-muted-foreground" />
          <div className="flex items-center gap-2">
            {childImages.slice(0, 3).map((child) => renderNode(child))}
            {childImages.length > 3 && (
              <Badge variant="secondary">+{childImages.length - 3} more</Badge>
            )}
          </div>
        </>
      )}
    </div>
  );
}

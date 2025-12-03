"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LineageNode, ImageLineageTree } from "@/lib/api";
import {
  ZoomIn,
  ZoomOut,
  Maximize2,
  RefreshCw,
  GitBranch,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface GraphNode {
  id: string;
  family: string;
  version: string;
  status: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  depth: number;
  isRoot: boolean;
}

interface GraphLink {
  source: string;
  target: string;
  type: string;
}

interface LineageGraphProps {
  tree: ImageLineageTree;
  onSelectImage?: (imageId: string) => void;
  selectedImageId?: string;
  className?: string;
}

const statusColors: Record<string, string> = {
  production: "#10b981",
  staging: "#f59e0b",
  deprecated: "#ef4444",
  pending: "#6366f1",
};

// Convert tree to flat nodes and links
function flattenTree(tree: ImageLineageTree): { nodes: GraphNode[]; links: GraphLink[] } {
  const nodes: GraphNode[] = [];
  const links: GraphLink[] = [];
  const nodeMap = new Map<string, boolean>();

  function traverse(node: LineageNode, depth: number, isRoot: boolean) {
    if (nodeMap.has(node.image.id)) return;
    nodeMap.set(node.image.id, true);

    nodes.push({
      id: node.image.id,
      family: node.image.family,
      version: node.image.version,
      status: node.image.status,
      x: 0,
      y: 0,
      vx: 0,
      vy: 0,
      depth,
      isRoot,
    });

    if (node.children) {
      for (const child of node.children) {
        links.push({
          source: node.image.id,
          target: child.image.id,
          type: "derived_from",
        });
        traverse(child, depth + 1, false);
      }
    }
  }

  for (const root of tree.roots) {
    traverse(root, 0, true);
  }

  return { nodes, links };
}

export function LineageGraph({
  tree,
  onSelectImage,
  selectedImageId,
  className,
}: LineageGraphProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [nodes, setNodes] = useState<GraphNode[]>([]);
  const [links, setLinks] = useState<GraphLink[]>([]);
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragNode, setDragNode] = useState<GraphNode | null>(null);
  const [hoveredNode, setHoveredNode] = useState<GraphNode | null>(null);
  const animationRef = useRef<number | null>(null);

  // Initialize graph layout
  useEffect(() => {
    const { nodes: flatNodes, links: flatLinks } = flattenTree(tree);

    // Position nodes in a hierarchical layout
    const levelGroups: Map<number, GraphNode[]> = new Map();
    for (const node of flatNodes) {
      const group = levelGroups.get(node.depth) || [];
      group.push(node);
      levelGroups.set(node.depth, group);
    }

    const levelWidth = 180;
    const nodeSpacing = 80;

    levelGroups.forEach((group, depth) => {
      const startY = -(group.length - 1) * nodeSpacing / 2;
      group.forEach((node, index) => {
        node.x = depth * levelWidth + 100;
        node.y = startY + index * nodeSpacing + 250;
      });
    });

    setNodes(flatNodes);
    setLinks(flatLinks);
  }, [tree]);

  // Draw function
  const draw = useCallback(() => {
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext("2d");
    if (!canvas || !ctx) return;

    const dpr = window.devicePixelRatio || 1;
    const rect = canvas.getBoundingClientRect();

    canvas.width = rect.width * dpr;
    canvas.height = rect.height * dpr;
    ctx.scale(dpr, dpr);

    // Clear canvas
    ctx.fillStyle = "hsl(var(--background))";
    ctx.fillRect(0, 0, rect.width, rect.height);

    // Apply transformations
    ctx.save();
    ctx.translate(pan.x + rect.width / 2, pan.y + rect.height / 2);
    ctx.scale(zoom, zoom);
    ctx.translate(-rect.width / 2, -rect.height / 2);

    // Build node map for quick lookup
    const nodeMap = new Map(nodes.map(n => [n.id, n]));

    // Draw links
    ctx.strokeStyle = "hsl(var(--border))";
    ctx.lineWidth = 2;

    for (const link of links) {
      const source = nodeMap.get(link.source);
      const target = nodeMap.get(link.target);
      if (!source || !target) continue;

      ctx.beginPath();

      // Curved line
      const midX = (source.x + target.x) / 2;
      ctx.moveTo(source.x + 40, source.y);
      ctx.bezierCurveTo(
        midX, source.y,
        midX, target.y,
        target.x - 40, target.y
      );
      ctx.stroke();

      // Draw arrow
      const arrowSize = 8;
      const angle = Math.atan2(target.y - source.y, target.x - source.x);
      const arrowX = target.x - 40;
      const arrowY = target.y;

      ctx.beginPath();
      ctx.moveTo(arrowX, arrowY);
      ctx.lineTo(
        arrowX - arrowSize * Math.cos(angle - Math.PI / 6),
        arrowY - arrowSize * Math.sin(angle - Math.PI / 6)
      );
      ctx.lineTo(
        arrowX - arrowSize * Math.cos(angle + Math.PI / 6),
        arrowY - arrowSize * Math.sin(angle + Math.PI / 6)
      );
      ctx.closePath();
      ctx.fill();
    }

    // Draw nodes
    for (const node of nodes) {
      const isSelected = node.id === selectedImageId;
      const isHovered = node.id === hoveredNode?.id;

      // Node background
      ctx.fillStyle = isSelected
        ? "hsl(var(--brand-accent) / 0.2)"
        : isHovered
          ? "hsl(var(--muted))"
          : "hsl(var(--card))";
      ctx.strokeStyle = isSelected
        ? "hsl(var(--brand-accent))"
        : "hsl(var(--border))";
      ctx.lineWidth = isSelected ? 2 : 1;

      // Rounded rectangle
      const width = 120;
      const height = 60;
      const radius = 8;
      const x = node.x - width / 2;
      const y = node.y - height / 2;

      ctx.beginPath();
      ctx.roundRect(x, y, width, height, radius);
      ctx.fill();
      ctx.stroke();

      // Status indicator
      const statusColor = statusColors[node.status] || statusColors.pending;
      ctx.fillStyle = statusColor;
      ctx.beginPath();
      ctx.arc(x + width - 10, y + 10, 5, 0, Math.PI * 2);
      ctx.fill();

      // Node text
      ctx.fillStyle = "hsl(var(--foreground))";
      ctx.font = "bold 11px system-ui";
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";

      // Truncate family name if too long
      let familyText = node.family;
      if (ctx.measureText(familyText).width > width - 20) {
        while (ctx.measureText(familyText + "...").width > width - 20 && familyText.length > 3) {
          familyText = familyText.slice(0, -1);
        }
        familyText += "...";
      }
      ctx.fillText(familyText, node.x, node.y - 8);

      // Version text
      ctx.font = "10px system-ui";
      ctx.fillStyle = "hsl(var(--muted-foreground))";
      ctx.fillText(`v${node.version}`, node.x, node.y + 10);
    }

    ctx.restore();
  }, [nodes, links, zoom, pan, selectedImageId, hoveredNode]);

  // Animation loop
  useEffect(() => {
    const animate = () => {
      draw();
      animationRef.current = requestAnimationFrame(animate);
    };
    animate();
    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [draw]);

  // Mouse event handlers
  const getMousePos = (e: React.MouseEvent<HTMLCanvasElement>): { x: number; y: number } => {
    const canvas = canvasRef.current;
    if (!canvas) return { x: 0, y: 0 };
    const rect = canvas.getBoundingClientRect();
    const x = (e.clientX - rect.left - pan.x - rect.width / 2) / zoom + rect.width / 2;
    const y = (e.clientY - rect.top - pan.y - rect.height / 2) / zoom + rect.height / 2;
    return { x, y };
  };

  const findNodeAtPosition = (x: number, y: number): GraphNode | null => {
    for (const node of nodes) {
      const dx = x - node.x;
      const dy = y - node.y;
      if (Math.abs(dx) < 60 && Math.abs(dy) < 30) {
        return node;
      }
    }
    return null;
  };

  const handleMouseDown = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const pos = getMousePos(e);
    const node = findNodeAtPosition(pos.x, pos.y);
    if (node) {
      setDragNode(node);
    } else {
      setIsDragging(true);
    }
  };

  const handleMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const pos = getMousePos(e);

    if (dragNode) {
      setNodes(nodes.map(n =>
        n.id === dragNode.id
          ? { ...n, x: pos.x, y: pos.y }
          : n
      ));
    } else if (isDragging) {
      setPan({ x: pan.x + e.movementX, y: pan.y + e.movementY });
    } else {
      const node = findNodeAtPosition(pos.x, pos.y);
      setHoveredNode(node);
      const canvas = canvasRef.current;
      if (canvas) {
        canvas.style.cursor = node ? "pointer" : "grab";
      }
    }
  };

  const handleMouseUp = () => {
    if (dragNode && !isDragging) {
      onSelectImage?.(dragNode.id);
    }
    setDragNode(null);
    setIsDragging(false);
  };

  const handleWheel = (e: React.WheelEvent<HTMLCanvasElement>) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setZoom(Math.max(0.3, Math.min(3, zoom * delta)));
  };

  const resetView = () => {
    setZoom(1);
    setPan({ x: 0, y: 0 });
  };

  if (!tree.roots || tree.roots.length === 0) {
    return (
      <Card className={className}>
        <CardContent className="flex flex-col items-center justify-center py-12 text-center">
          <GitBranch className="h-12 w-12 text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium">No Lineage Data</h3>
          <p className="text-sm text-muted-foreground mt-1">
            No parent-child relationships have been defined.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={className}>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-base">
            <GitBranch className="h-4 w-4" />
            Lineage Graph
          </CardTitle>
          <div className="flex items-center gap-2">
            <Badge variant="secondary">{nodes.length} versions</Badge>
            <div className="flex items-center gap-1 border rounded-md">
              <Button
                variant="ghost"
                size="sm"
                className="h-8 w-8 p-0"
                onClick={() => setZoom(Math.min(3, zoom * 1.2))}
              >
                <ZoomIn className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 w-8 p-0"
                onClick={() => setZoom(Math.max(0.3, zoom * 0.8))}
              >
                <ZoomOut className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 w-8 p-0"
                onClick={resetView}
              >
                <Maximize2 className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div ref={containerRef} className="relative w-full h-[500px]">
          <canvas
            ref={canvasRef}
            className="w-full h-full"
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
            onMouseUp={handleMouseUp}
            onMouseLeave={handleMouseUp}
            onWheel={handleWheel}
          />

          {/* Legend */}
          <div className="absolute bottom-4 left-4 flex items-center gap-4 bg-card/80 backdrop-blur-sm rounded-lg p-2 text-xs">
            <div className="flex items-center gap-1">
              <div className="w-3 h-3 rounded-full bg-[#10b981]" />
              <span>Production</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-3 h-3 rounded-full bg-[#f59e0b]" />
              <span>Staging</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-3 h-3 rounded-full bg-[#ef4444]" />
              <span>Deprecated</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-3 h-3 rounded-full bg-[#6366f1]" />
              <span>Pending</span>
            </div>
          </div>

          {/* Zoom indicator */}
          <div className="absolute bottom-4 right-4 bg-card/80 backdrop-blur-sm rounded-lg px-2 py-1 text-xs">
            {Math.round(zoom * 100)}%
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

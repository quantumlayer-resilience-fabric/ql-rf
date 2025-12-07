"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { MetricCard } from "@/components/data/metric-card";
import { StatusBadge } from "@/components/status/status-badge";
import { PlatformIcon } from "@/components/status/platform-icon";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import { api } from "@/lib/api";
import {
  Plus,
  RefreshCw,
  Cable,
  Server,
  CheckCircle2,
  XCircle,
  MoreHorizontal,
  Play,
  Pause,
  Trash2,
  TestTube,
  Loader2,
  Clock,
  AlertTriangle,
} from "lucide-react";
import { toast } from "sonner";

// Types
interface Connector {
  id: string;
  name: string;
  platform: "aws" | "azure" | "gcp" | "vsphere" | "k8s";
  enabled: boolean;
  config?: Record<string, unknown>;
  last_sync_at?: string;
  last_sync_status?: string;
  last_sync_error?: string;
  created_at: string;
  updated_at: string;
}

// Platform configuration fields
const platformFields: Record<string, Array<{
  name: string;
  label: string;
  type: "text" | "password" | "textarea";
  required: boolean;
  placeholder?: string;
}>> = {
  aws: [
    { name: "region", label: "Default Region", type: "text", required: true, placeholder: "us-east-1" },
    { name: "assume_role_arn", label: "Assume Role ARN (optional)", type: "text", required: false, placeholder: "arn:aws:iam::123456789012:role/RoleName" },
  ],
  azure: [
    { name: "subscription_id", label: "Subscription ID", type: "text", required: true, placeholder: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" },
    { name: "tenant_id", label: "Tenant ID", type: "text", required: true, placeholder: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" },
    { name: "client_id", label: "Client ID", type: "text", required: true, placeholder: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" },
    { name: "client_secret", label: "Client Secret", type: "password", required: true, placeholder: "Your client secret" },
  ],
  gcp: [
    { name: "project_id", label: "Project ID", type: "text", required: true, placeholder: "my-gcp-project" },
    { name: "credentials_file", label: "Service Account JSON", type: "textarea", required: false, placeholder: "Paste service account JSON here" },
  ],
  vsphere: [
    { name: "host", label: "vCenter Host", type: "text", required: true, placeholder: "vcenter.example.com" },
    { name: "username", label: "Username", type: "text", required: true, placeholder: "administrator@vsphere.local" },
    { name: "password", label: "Password", type: "password", required: true, placeholder: "Your password" },
  ],
  k8s: [
    { name: "cluster_name", label: "Cluster Name", type: "text", required: false, placeholder: "production-cluster" },
    { name: "context", label: "Kubeconfig Context (optional)", type: "text", required: false, placeholder: "my-context" },
  ],
};

const platformLabels: Record<string, string> = {
  aws: "Amazon Web Services",
  azure: "Microsoft Azure",
  gcp: "Google Cloud Platform",
  vsphere: "VMware vSphere",
  k8s: "Kubernetes",
};

export default function ConnectorsPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [newConnector, setNewConnector] = useState({
    name: "",
    platform: "aws" as Connector["platform"],
    config: {} as Record<string, string>,
    syncSchedule: "1h",
  });
  const [testingId, setTestingId] = useState<string | null>(null);
  const [syncingId, setSyncingId] = useState<string | null>(null);

  // Fetch connectors
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["connectors"],
    queryFn: async () => {
      const response = await api.connectors.list();
      return response.connectors;
    },
  });

  // Create connector mutation
  const createMutation = useMutation({
    mutationFn: async (params: { name: string; platform: Connector["platform"]; config: Record<string, unknown>; syncSchedule?: string }) => {
      return api.connectors.create(params);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["connectors"] });
      setIsCreateDialogOpen(false);
      setNewConnector({ name: "", platform: "aws", config: {}, syncSchedule: "1h" });
      toast.success("Connector created successfully");
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to create connector");
    },
  });

  // Delete connector mutation
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      return api.connectors.delete(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["connectors"] });
      toast.success("Connector deleted");
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to delete connector");
    },
  });

  // Test connection mutation
  const testMutation = useMutation({
    mutationFn: async (id: string) => {
      setTestingId(id);
      return api.connectors.test(id);
    },
    onSuccess: (result) => {
      if (result.success) {
        toast.success("Connection test passed");
      } else {
        toast.error(result.message || "Connection test failed");
      }
      setTestingId(null);
    },
    onError: (err: Error) => {
      toast.error(err.message || "Connection test failed");
      setTestingId(null);
    },
  });

  // Sync connector mutation
  const syncMutation = useMutation({
    mutationFn: async (id: string) => {
      setSyncingId(id);
      return api.connectors.sync(id);
    },
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ["connectors"] });
      toast.success(`Sync complete: ${result.assets_found || 0} assets discovered`);
      setSyncingId(null);
    },
    onError: (err: Error) => {
      toast.error(err.message || "Sync failed");
      setSyncingId(null);
    },
  });

  // Enable/Disable mutations
  const enableMutation = useMutation({
    mutationFn: async (id: string) => api.connectors.enable(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["connectors"] });
      toast.success("Connector enabled");
    },
  });

  const disableMutation = useMutation({
    mutationFn: async (id: string) => api.connectors.disable(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["connectors"] });
      toast.success("Connector disabled");
    },
  });

  const handleCreateConnector = useCallback(() => {
    const fields = platformFields[newConnector.platform] || [];
    const missingRequired = fields
      .filter((f) => f.required && !newConnector.config[f.name])
      .map((f) => f.label);

    if (!newConnector.name) {
      toast.error("Please enter a connector name");
      return;
    }

    if (missingRequired.length > 0) {
      toast.error(`Missing required fields: ${missingRequired.join(", ")}`);
      return;
    }

    createMutation.mutate({
      name: newConnector.name,
      platform: newConnector.platform,
      config: newConnector.config,
      syncSchedule: newConnector.syncSchedule,
    });
  }, [newConnector, createMutation]);

  const formatLastSync = (dateString?: string) => {
    if (!dateString) return "Never";
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));

    if (diffMins < 1) return "just now";
    if (diffMins < 60) return `${diffMins} min ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Connectors
            </h1>
            <p className="text-muted-foreground">
              Manage cloud platform connections for asset discovery.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={4} showChart={false} showTable />
      </div>
    );
  }

  if (error) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Connectors
          </h1>
          <p className="text-muted-foreground">
            Manage cloud platform connections for asset discovery.
          </p>
        </div>
        <ErrorState
          error={error}
          retry={refetch}
          title="Failed to load connectors"
          description="We couldn't fetch the connectors. Please try again."
        />
      </div>
    );
  }

  const connectors = data || [];

  // Calculate metrics
  const metrics = {
    total: connectors.length,
    active: connectors.filter((c) => c.enabled).length,
    synced: connectors.filter((c) => c.last_sync_status === "completed").length,
    failed: connectors.filter((c) => c.last_sync_status === "failed").length,
  };

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Connectors
          </h1>
          <p className="text-muted-foreground">
            Manage cloud platform connections for asset discovery.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="mr-2 h-4 w-4" />
                Add Connector
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[500px]">
              <DialogHeader>
                <DialogTitle>Add New Connector</DialogTitle>
                <DialogDescription>
                  Connect a cloud platform to discover and manage assets.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                <div className="space-y-2">
                  <Label htmlFor="name">Connector Name</Label>
                  <Input
                    id="name"
                    value={newConnector.name}
                    onChange={(e) =>
                      setNewConnector({ ...newConnector, name: e.target.value })
                    }
                    placeholder="Production AWS"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="platform">Platform</Label>
                  <Select
                    value={newConnector.platform}
                    onValueChange={(value: Connector["platform"]) =>
                      setNewConnector({
                        ...newConnector,
                        platform: value,
                        config: {},
                      })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select platform" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="aws">
                        <div className="flex items-center gap-2">
                          <PlatformIcon platform="aws" size="sm" />
                          Amazon Web Services
                        </div>
                      </SelectItem>
                      <SelectItem value="azure">
                        <div className="flex items-center gap-2">
                          <PlatformIcon platform="azure" size="sm" />
                          Microsoft Azure
                        </div>
                      </SelectItem>
                      <SelectItem value="gcp">
                        <div className="flex items-center gap-2">
                          <PlatformIcon platform="gcp" size="sm" />
                          Google Cloud Platform
                        </div>
                      </SelectItem>
                      <SelectItem value="vsphere">
                        <div className="flex items-center gap-2">
                          <PlatformIcon platform="vsphere" size="sm" />
                          VMware vSphere
                        </div>
                      </SelectItem>
                      <SelectItem value="k8s">
                        <div className="flex items-center gap-2">
                          <PlatformIcon platform="k8s" size="sm" />
                          Kubernetes
                        </div>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {/* Sync Schedule */}
                <div className="space-y-2">
                  <Label htmlFor="syncSchedule">Sync Schedule</Label>
                  <Select
                    value={newConnector.syncSchedule}
                    onValueChange={(value) =>
                      setNewConnector({ ...newConnector, syncSchedule: value })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select sync interval" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="15m">Every 15 minutes</SelectItem>
                      <SelectItem value="30m">Every 30 minutes</SelectItem>
                      <SelectItem value="1h">Every hour</SelectItem>
                      <SelectItem value="2h">Every 2 hours</SelectItem>
                      <SelectItem value="6h">Every 6 hours</SelectItem>
                      <SelectItem value="12h">Every 12 hours</SelectItem>
                      <SelectItem value="24h">Daily</SelectItem>
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">
                    How often to automatically discover assets from this connector
                  </p>
                </div>

                {/* Dynamic platform fields */}
                {platformFields[newConnector.platform]?.map((field) => (
                  <div key={field.name} className="space-y-2">
                    <Label htmlFor={field.name}>
                      {field.label}
                      {field.required && <span className="text-destructive ml-1">*</span>}
                    </Label>
                    {field.type === "textarea" ? (
                      <textarea
                        id={field.name}
                        className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                        value={newConnector.config[field.name] || ""}
                        onChange={(e) =>
                          setNewConnector({
                            ...newConnector,
                            config: {
                              ...newConnector.config,
                              [field.name]: e.target.value,
                            },
                          })
                        }
                        placeholder={field.placeholder}
                      />
                    ) : (
                      <Input
                        id={field.name}
                        type={field.type}
                        value={newConnector.config[field.name] || ""}
                        onChange={(e) =>
                          setNewConnector({
                            ...newConnector,
                            config: {
                              ...newConnector.config,
                              [field.name]: e.target.value,
                            },
                          })
                        }
                        placeholder={field.placeholder}
                      />
                    )}
                  </div>
                ))}
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setIsCreateDialogOpen(false)}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleCreateConnector}
                  disabled={createMutation.isPending}
                >
                  {createMutation.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Create Connector
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Connectors"
          value={metrics.total}
          subtitle="configured"
          status="neutral"
          icon={<Cable className="h-5 w-5" />}
        />
        <MetricCard
          title="Active"
          value={metrics.active}
          subtitle="enabled"
          status="success"
          icon={<CheckCircle2 className="h-5 w-5" />}
        />
        <MetricCard
          title="Last Sync OK"
          value={metrics.synced}
          subtitle="completed"
          status="success"
          icon={<Server className="h-5 w-5" />}
        />
        <MetricCard
          title="Sync Failed"
          value={metrics.failed}
          subtitle="needs attention"
          status={metrics.failed > 0 ? "critical" : "success"}
          icon={<AlertTriangle className="h-5 w-5" />}
        />
      </div>

      {/* Connectors Table */}
      {connectors.length === 0 ? (
        <Card>
          <CardContent className="p-8">
            <EmptyState
              variant="default"
              title="No connectors configured"
              description="Add your first connector to start discovering infrastructure assets."
              action={{
                label: "Add Connector",
                onClick: () => setIsCreateDialogOpen(true),
              }}
            />
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Cloud Platform Connectors</CardTitle>
            <CardDescription>
              Manage your connected cloud platforms and trigger asset discovery.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Platform</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Last Sync</TableHead>
                  <TableHead>Sync Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {connectors.map((connector) => (
                  <TableRow key={connector.id}>
                    <TableCell className="font-medium">
                      {connector.name}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <PlatformIcon platform={connector.platform} size="sm" />
                        <span className="text-sm text-muted-foreground">
                          {platformLabels[connector.platform]}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusBadge
                        status={connector.enabled ? "success" : "neutral"}
                        size="sm"
                      >
                        {connector.enabled ? "Active" : "Disabled"}
                      </StatusBadge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1 text-sm text-muted-foreground">
                        <Clock className="h-3 w-3" />
                        {formatLastSync(connector.last_sync_at)}
                      </div>
                    </TableCell>
                    <TableCell>
                      {syncingId === connector.id ? (
                        <div className="flex items-center gap-1 text-sm text-muted-foreground">
                          <Loader2 className="h-3 w-3 animate-spin" />
                          Syncing...
                        </div>
                      ) : connector.last_sync_status === "completed" ? (
                        <div className="flex items-center gap-1 text-sm text-status-green">
                          <CheckCircle2 className="h-3 w-3" />
                          Success
                        </div>
                      ) : connector.last_sync_status === "failed" ? (
                        <div className="flex items-center gap-1 text-sm text-status-red" title={connector.last_sync_error || "Unknown error"}>
                          <XCircle className="h-3 w-3" />
                          Failed
                        </div>
                      ) : connector.last_sync_status === "syncing" ? (
                        <div className="flex items-center gap-1 text-sm text-muted-foreground">
                          <Loader2 className="h-3 w-3 animate-spin" />
                          Syncing...
                        </div>
                      ) : (
                        <span className="text-sm text-muted-foreground">-</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => syncMutation.mutate(connector.id)}
                          disabled={!connector.enabled || syncingId === connector.id}
                        >
                          {syncingId === connector.id ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                          ) : (
                            <RefreshCw className="h-4 w-4" />
                          )}
                        </Button>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem
                              onClick={() => testMutation.mutate(connector.id)}
                              disabled={testingId === connector.id}
                            >
                              {testingId === connector.id ? (
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                              ) : (
                                <TestTube className="mr-2 h-4 w-4" />
                              )}
                              Test Connection
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => syncMutation.mutate(connector.id)}
                              disabled={!connector.enabled || syncingId === connector.id}
                            >
                              <RefreshCw className="mr-2 h-4 w-4" />
                              Sync Now
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            {connector.enabled ? (
                              <DropdownMenuItem
                                onClick={() => disableMutation.mutate(connector.id)}
                              >
                                <Pause className="mr-2 h-4 w-4" />
                                Disable
                              </DropdownMenuItem>
                            ) : (
                              <DropdownMenuItem
                                onClick={() => enableMutation.mutate(connector.id)}
                              >
                                <Play className="mr-2 h-4 w-4" />
                                Enable
                              </DropdownMenuItem>
                            )}
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              className="text-destructive focus:text-destructive"
                              onClick={() => {
                                if (confirm("Are you sure you want to delete this connector?")) {
                                  deleteMutation.mutate(connector.id);
                                }
                              }}
                            >
                              <Trash2 className="mr-2 h-4 w-4" />
                              Delete
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

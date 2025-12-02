"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { StatusBadge } from "@/components/status/status-badge";
import { PlatformIcon } from "@/components/status/platform-icon";
import {
  Cloud,
  Bell,
  Users,
  Key,
  ScrollText,
  Plus,
  Trash2,
  RefreshCw,
  Check,
  Copy,
  Eye,
  EyeOff,
  Mail,
  Slack,
  Webhook,
  Shield,
  Clock,
  MoreHorizontal,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

// Mock data
const connectors = [
  {
    id: "aws-prod",
    name: "AWS Production",
    platform: "aws" as const,
    status: "connected",
    lastSync: "2 min ago",
    assetsDiscovered: 4231,
    region: "us-east-1, eu-west-1, ap-south-1",
  },
  {
    id: "azure-main",
    name: "Azure Main",
    platform: "azure" as const,
    status: "connected",
    lastSync: "5 min ago",
    assetsDiscovered: 2419,
    region: "eastus, westeurope",
  },
  {
    id: "gcp-central",
    name: "GCP Central",
    platform: "gcp" as const,
    status: "syncing",
    lastSync: "syncing...",
    assetsDiscovered: 654,
    region: "us-central1",
  },
  {
    id: "vsphere-dc",
    name: "On-Prem vSphere",
    platform: "vsphere" as const,
    status: "error",
    lastSync: "1 hour ago",
    assetsDiscovered: 753,
    region: "London DC, Singapore DC",
  },
];

const teamMembers = [
  { id: "1", name: "Sarah Chen", email: "sarah@company.com", role: "Admin", status: "active" },
  { id: "2", name: "Mike Johnson", email: "mike@company.com", role: "Editor", status: "active" },
  { id: "3", name: "Emily Davis", email: "emily@company.com", role: "Viewer", status: "active" },
  { id: "4", name: "Alex Kim", email: "alex@company.com", role: "Editor", status: "pending" },
];

const apiKeys = [
  { id: "key-1", name: "Production API", prefix: "qlrf_prod_", created: "2024-01-10", lastUsed: "2 hours ago" },
  { id: "key-2", name: "CI/CD Pipeline", prefix: "qlrf_ci_", created: "2024-01-05", lastUsed: "5 min ago" },
  { id: "key-3", name: "Monitoring Integration", prefix: "qlrf_mon_", created: "2023-12-20", lastUsed: "1 day ago" },
];

const auditLogs = [
  { id: "1", action: "Connector synced", user: "System", target: "AWS Production", time: "2 min ago", type: "info" },
  { id: "2", action: "Image promoted", user: "Sarah Chen", target: "ql-base-linux:1.6.4", time: "1 hour ago", type: "success" },
  { id: "3", action: "Team member invited", user: "Sarah Chen", target: "alex@company.com", time: "3 hours ago", type: "info" },
  { id: "4", action: "API key created", user: "Mike Johnson", target: "CI/CD Pipeline", time: "1 day ago", type: "info" },
  { id: "5", action: "Connector error", user: "System", target: "On-Prem vSphere", time: "1 hour ago", type: "warning" },
  { id: "6", action: "Drift scan triggered", user: "Emily Davis", target: "All sites", time: "2 days ago", type: "info" },
];

export default function SettingsPage() {
  const [showApiKey, setShowApiKey] = useState<string | null>(null);

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground">
          Settings
        </h1>
        <p className="text-muted-foreground">
          Manage your connectors, team, and platform configuration.
        </p>
      </div>

      {/* Settings Tabs */}
      <Tabs defaultValue="connectors" className="space-y-6">
        <TabsList className="grid w-full grid-cols-5 lg:w-auto lg:inline-grid">
          <TabsTrigger value="connectors" className="gap-2">
            <Cloud className="h-4 w-4" />
            <span className="hidden sm:inline">Connectors</span>
          </TabsTrigger>
          <TabsTrigger value="notifications" className="gap-2">
            <Bell className="h-4 w-4" />
            <span className="hidden sm:inline">Notifications</span>
          </TabsTrigger>
          <TabsTrigger value="team" className="gap-2">
            <Users className="h-4 w-4" />
            <span className="hidden sm:inline">Team</span>
          </TabsTrigger>
          <TabsTrigger value="api" className="gap-2">
            <Key className="h-4 w-4" />
            <span className="hidden sm:inline">API</span>
          </TabsTrigger>
          <TabsTrigger value="audit" className="gap-2">
            <ScrollText className="h-4 w-4" />
            <span className="hidden sm:inline">Audit Log</span>
          </TabsTrigger>
        </TabsList>

        {/* Connectors Tab */}
        <TabsContent value="connectors" className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold">Cloud Connectors</h2>
              <p className="text-sm text-muted-foreground">
                Connect your cloud providers to discover and monitor assets.
              </p>
            </div>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Add Connector
            </Button>
          </div>

          <div className="grid gap-4">
            {connectors.map((connector) => (
              <Card key={connector.id}>
                <CardContent className="flex items-center justify-between p-4">
                  <div className="flex items-center gap-4">
                    <PlatformIcon platform={connector.platform} size="lg" />
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold">{connector.name}</h3>
                        <StatusBadge
                          status={
                            connector.status === "connected"
                              ? "success"
                              : connector.status === "syncing"
                              ? "info"
                              : "critical"
                          }
                          size="sm"
                          pulse={connector.status === "syncing"}
                        >
                          {connector.status}
                        </StatusBadge>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {connector.region}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-6">
                    <div className="text-right">
                      <div className="text-sm font-medium">
                        {connector.assetsDiscovered.toLocaleString()} assets
                      </div>
                      <div className="text-xs text-muted-foreground">
                        Last sync: {connector.lastSync}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button variant="outline" size="sm">
                        <RefreshCw className="h-4 w-4" />
                      </Button>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem>Edit</DropdownMenuItem>
                          <DropdownMenuItem>View Logs</DropdownMenuItem>
                          <DropdownMenuItem className="text-status-red">
                            <Trash2 className="mr-2 h-4 w-4" />
                            Remove
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* Notifications Tab */}
        <TabsContent value="notifications" className="space-y-4">
          <div>
            <h2 className="text-lg font-semibold">Notification Preferences</h2>
            <p className="text-sm text-muted-foreground">
              Configure how you receive alerts and updates.
            </p>
          </div>

          <div className="grid gap-4 md:grid-cols-3">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Mail className="h-5 w-5" />
                  Email
                </CardTitle>
                <CardDescription>
                  Receive notifications via email
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm">Critical Alerts</span>
                  <Badge variant="secondary">Enabled</Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm">Drift Reports</span>
                  <Badge variant="secondary">Daily</Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm">Weekly Summary</span>
                  <Badge variant="secondary">Enabled</Badge>
                </div>
                <Button variant="outline" className="w-full" size="sm">
                  Configure
                </Button>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Slack className="h-5 w-5" />
                  Slack
                </CardTitle>
                <CardDescription>
                  Send alerts to Slack channels
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm">Status</span>
                  <Badge variant="outline" className="text-status-green border-status-green/30">
                    Connected
                  </Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm">Channel</span>
                  <span className="text-sm text-muted-foreground">#infra-alerts</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm">Alert Types</span>
                  <span className="text-sm text-muted-foreground">Critical only</span>
                </div>
                <Button variant="outline" className="w-full" size="sm">
                  Configure
                </Button>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Webhook className="h-5 w-5" />
                  Webhook
                </CardTitle>
                <CardDescription>
                  Send events to custom endpoints
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm">Status</span>
                  <Badge variant="outline" className="text-muted-foreground">
                    Not configured
                  </Badge>
                </div>
                <div className="pt-8">
                  <Button variant="outline" className="w-full" size="sm">
                    <Plus className="mr-2 h-4 w-4" />
                    Add Webhook
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Team Tab */}
        <TabsContent value="team" className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold">Team Members</h2>
              <p className="text-sm text-muted-foreground">
                Manage who has access to your organization.
              </p>
            </div>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Invite Member
            </Button>
          </div>

          <Card>
            <CardContent className="p-0">
              <div className="rounded-lg border">
                <table className="w-full">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-4 py-3 text-left text-sm font-medium">Member</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">Role</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">Status</th>
                      <th className="px-4 py-3 text-right text-sm font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {teamMembers.map((member, i) => (
                      <tr key={member.id} className={i !== teamMembers.length - 1 ? "border-b" : ""}>
                        <td className="px-4 py-3">
                          <div>
                            <div className="font-medium">{member.name}</div>
                            <div className="text-sm text-muted-foreground">{member.email}</div>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <Select defaultValue={member.role.toLowerCase()}>
                            <SelectTrigger className="w-[120px]">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="admin">Admin</SelectItem>
                              <SelectItem value="editor">Editor</SelectItem>
                              <SelectItem value="viewer">Viewer</SelectItem>
                            </SelectContent>
                          </Select>
                        </td>
                        <td className="px-4 py-3">
                          <StatusBadge
                            status={member.status === "active" ? "success" : "warning"}
                            size="sm"
                          >
                            {member.status}
                          </StatusBadge>
                        </td>
                        <td className="px-4 py-3 text-right">
                          <Button variant="ghost" size="sm" className="text-status-red">
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* API Tab */}
        <TabsContent value="api" className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold">API Keys</h2>
              <p className="text-sm text-muted-foreground">
                Manage API keys for programmatic access.
              </p>
            </div>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Generate Key
            </Button>
          </div>

          <Card>
            <CardContent className="p-0">
              <div className="rounded-lg border">
                <table className="w-full">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-4 py-3 text-left text-sm font-medium">Name</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">Key</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">Created</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">Last Used</th>
                      <th className="px-4 py-3 text-right text-sm font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {apiKeys.map((key, i) => (
                      <tr key={key.id} className={i !== apiKeys.length - 1 ? "border-b" : ""}>
                        <td className="px-4 py-3 font-medium">{key.name}</td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <code className="rounded bg-muted px-2 py-1 text-sm">
                              {showApiKey === key.id
                                ? `${key.prefix}xxxxxxxxxxxxxxxxxxxx`
                                : `${key.prefix}••••••••••••`}
                            </code>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => setShowApiKey(showApiKey === key.id ? null : key.id)}
                            >
                              {showApiKey === key.id ? (
                                <EyeOff className="h-4 w-4" />
                              ) : (
                                <Eye className="h-4 w-4" />
                              )}
                            </Button>
                            <Button variant="ghost" size="sm">
                              <Copy className="h-4 w-4" />
                            </Button>
                          </div>
                        </td>
                        <td className="px-4 py-3 text-sm text-muted-foreground">{key.created}</td>
                        <td className="px-4 py-3 text-sm text-muted-foreground">{key.lastUsed}</td>
                        <td className="px-4 py-3 text-right">
                          <Button variant="ghost" size="sm" className="text-status-red">
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">API Documentation</CardTitle>
              <CardDescription>
                Learn how to integrate with the QL-RF API.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-4">
                <Button variant="outline">View API Docs</Button>
                <Button variant="outline">Download OpenAPI Spec</Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Audit Log Tab */}
        <TabsContent value="audit" className="space-y-4">
          <div>
            <h2 className="text-lg font-semibold">Audit Log</h2>
            <p className="text-sm text-muted-foreground">
              Track all actions and changes in your organization.
            </p>
          </div>

          <Card>
            <CardContent className="p-0">
              <div className="rounded-lg border">
                <table className="w-full">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-4 py-3 text-left text-sm font-medium">Action</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">User</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">Target</th>
                      <th className="px-4 py-3 text-left text-sm font-medium">Time</th>
                    </tr>
                  </thead>
                  <tbody>
                    {auditLogs.map((log, i) => (
                      <tr key={log.id} className={i !== auditLogs.length - 1 ? "border-b" : ""}>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <div
                              className={`h-2 w-2 rounded-full ${
                                log.type === "success"
                                  ? "bg-status-green"
                                  : log.type === "warning"
                                  ? "bg-status-amber"
                                  : "bg-brand-accent"
                              }`}
                            />
                            <span className="font-medium">{log.action}</span>
                          </div>
                        </td>
                        <td className="px-4 py-3 text-sm">{log.user}</td>
                        <td className="px-4 py-3 text-sm text-muted-foreground">{log.target}</td>
                        <td className="px-4 py-3 text-sm text-muted-foreground">{log.time}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>

          <div className="flex justify-center">
            <Button variant="outline">Load More</Button>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}

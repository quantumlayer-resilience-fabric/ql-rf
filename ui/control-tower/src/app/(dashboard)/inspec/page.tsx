"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { MetricCard } from "@/components/data/metric-card";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import {
  useProfiles,
  useScans,
  useSchedules,
  useTriggerScan,
} from "@/hooks/use-inspec";
import { ProfileCard } from "@/components/inspec/profile-card";
import { ScanStatusBadge } from "@/components/inspec/scan-status-badge";
import {
  Shield,
  CheckCircle,
  Play,
  Calendar,
  TrendingUp,
  FileText,
  Clock,
  AlertCircle,
} from "lucide-react";
import { InSpecProfile } from "@/lib/api-inspec";

export default function InSpecPage() {
  const router = useRouter();
  const [selectedProfile, setSelectedProfile] = useState<InSpecProfile | null>(null);

  const { data: profiles, isLoading: profilesLoading, error: profilesError, refetch: refetchProfiles } = useProfiles();
  const { data: scansData, isLoading: scansLoading, error: scansError, refetch: refetchScans } = useScans({ limit: 10 });
  const { data: schedules } = useSchedules();
  const triggerScan = useTriggerScan();

  const scans = scansData?.runs || [];

  const handleRunProfile = (profile: InSpecProfile) => {
    setSelectedProfile(profile);
    // In a real app, you'd show a dialog to select the target asset
    // For now, we'll just set the selected profile
    console.log("Run profile:", profile);
  };

  // Calculate metrics
  const totalProfiles = profiles?.length || 0;
  const recentScans = scans.filter((scan) => {
    const scanDate = new Date(scan.createdAt);
    const thirtyDaysAgo = new Date();
    thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
    return scanDate >= thirtyDaysAgo;
  });
  const lastScan = scans[0];
  const lastScanScore = lastScan ? (lastScan.passRate || 0) : 0;
  const totalControlsPassed = scans.reduce((sum, scan) => sum + (scan.passedTests || 0), 0);

  if (profilesLoading || scansLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              InSpec Compliance Scanning
            </h1>
            <p className="text-muted-foreground">
              Automated compliance assessment with InSpec profiles.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={4} showChart={false} showTable={true} tableRows={5} />
      </div>
    );
  }

  if (profilesError || scansError) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            InSpec Compliance Scanning
          </h1>
          <p className="text-muted-foreground">
            Automated compliance assessment with InSpec profiles.
          </p>
        </div>
        <ErrorState
          error={profilesError || scansError}
          retry={() => {
            refetchProfiles();
            refetchScans();
          }}
          title="Failed to load InSpec data"
          description="We couldn't fetch the InSpec data. Please try again."
        />
      </div>
    );
  }

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            InSpec Compliance Scanning
          </h1>
          <p className="text-muted-foreground">
            Automated compliance assessment with InSpec profiles.
          </p>
        </div>
        <Button size="sm">
          <Play className="mr-2 h-4 w-4" />
          Run Scan
        </Button>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Profiles Available"
          value={totalProfiles}
          subtitle="profiles"
          status="neutral"
          icon={<Shield className="h-5 w-5" />}
        />
        <MetricCard
          title="Last Scan Score"
          value={`${lastScanScore.toFixed(1)}%`}
          subtitle="pass rate"
          status={lastScanScore >= 95 ? "success" : lastScanScore >= 80 ? "warning" : "critical"}
          icon={<CheckCircle className="h-5 w-5" />}
        />
        <MetricCard
          title="Controls Passed"
          value={totalControlsPassed}
          subtitle="total"
          status="success"
          icon={<FileText className="h-5 w-5" />}
        />
        <MetricCard
          title="Scans This Month"
          value={recentScans.length}
          subtitle="runs"
          status="neutral"
          icon={<TrendingUp className="h-5 w-5" />}
        />
      </div>

      {/* Tabs */}
      <Tabs defaultValue="profiles" className="space-y-4">
        <TabsList>
          <TabsTrigger value="profiles">Profiles</TabsTrigger>
          <TabsTrigger value="scans">Recent Scans</TabsTrigger>
          <TabsTrigger value="schedules">Schedules</TabsTrigger>
        </TabsList>

        {/* Profiles Tab */}
        <TabsContent value="profiles" className="space-y-4">
          {profiles && profiles.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {profiles.map((profile) => (
                <div key={profile.id} onClick={() => router.push(`/inspec/profiles/${profile.id}`)}>
                  <ProfileCard
                    profile={profile}
                    onRun={handleRunProfile}
                    isRunning={triggerScan.isPending && selectedProfile?.id === profile.id}
                  />
                </div>
              ))}
            </div>
          ) : (
            <Card>
              <CardContent className="p-8">
                <EmptyState
                  variant="data"
                  title="No profiles available"
                  description="Configure InSpec profiles to start scanning your infrastructure."
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Recent Scans Tab */}
        <TabsContent value="scans" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Recent Scans</CardTitle>
            </CardHeader>
            <CardContent>
              {scans.length > 0 ? (
                <div className="rounded-lg border">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b bg-muted/50">
                        <th className="px-4 py-3 text-left text-sm font-medium">Date</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Profile</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Status</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Score</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Duration</th>
                        <th className="px-4 py-3 text-left text-sm font-medium">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {scans.map((scan, i) => (
                        <tr
                          key={scan.id}
                          className={i !== scans.length - 1 ? "border-b" : ""}
                        >
                          <td className="px-4 py-3 text-sm">
                            <div className="flex items-center gap-1">
                              <Clock className="h-3 w-3 text-muted-foreground" />
                              {new Date(scan.createdAt).toLocaleDateString()}
                            </div>
                          </td>
                          <td className="px-4 py-3">
                            <div>
                              <div className="text-sm font-medium">{scan.profileName || "Unknown"}</div>
                              {scan.framework && (
                                <div className="text-xs text-muted-foreground">{scan.framework}</div>
                              )}
                            </div>
                          </td>
                          <td className="px-4 py-3">
                            <ScanStatusBadge status={scan.status} size="sm" />
                          </td>
                          <td className="px-4 py-3">
                            {scan.status === "completed" ? (
                              <div className="flex items-center gap-2">
                                <span className="text-sm font-medium">
                                  {scan.passRate?.toFixed(1)}%
                                </span>
                                <span className="text-xs text-muted-foreground">
                                  ({scan.passedTests}/{scan.totalTests})
                                </span>
                              </div>
                            ) : (
                              <span className="text-sm text-muted-foreground">-</span>
                            )}
                          </td>
                          <td className="px-4 py-3 text-sm text-muted-foreground">
                            {scan.duration ? `${scan.duration}s` : "-"}
                          </td>
                          <td className="px-4 py-3">
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => router.push(`/inspec/scans/${scan.id}`)}
                            >
                              View Results
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <EmptyState
                  variant="data"
                  title="No scans yet"
                  description="Run an InSpec profile to see scan results here."
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Schedules Tab */}
        <TabsContent value="schedules" className="space-y-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-base">Scheduled Scans</CardTitle>
              <Button size="sm">
                <Calendar className="mr-2 h-4 w-4" />
                New Schedule
              </Button>
            </CardHeader>
            <CardContent>
              {schedules && schedules.length > 0 ? (
                <div className="space-y-3">
                  {schedules.map((schedule) => (
                    <div
                      key={schedule.id}
                      className="flex items-center justify-between rounded-lg border p-4"
                    >
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <h4 className="font-medium">{schedule.profileName || "Unknown Profile"}</h4>
                          <Badge variant={schedule.enabled ? "default" : "secondary"}>
                            {schedule.enabled ? "Enabled" : "Disabled"}
                          </Badge>
                        </div>
                        <div className="mt-1 flex items-center gap-4 text-sm text-muted-foreground">
                          <span className="flex items-center gap-1">
                            <Calendar className="h-3 w-3" />
                            {schedule.cronExpression}
                          </span>
                          {schedule.nextRunAt && (
                            <span className="flex items-center gap-1">
                              <Clock className="h-3 w-3" />
                              Next: {new Date(schedule.nextRunAt).toLocaleString()}
                            </span>
                          )}
                        </div>
                      </div>
                      <Button size="sm" variant="ghost">
                        Configure
                      </Button>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState
                  variant="data"
                  title="No schedules configured"
                  description="Create a schedule to run InSpec profiles automatically."
                  action={{
                    label: "Create Schedule",
                    onClick: () => console.log("Create schedule"),
                    icon: <Calendar className="mr-2 h-4 w-4" />,
                  }}
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Info Card */}
      {lastScan && (
        <Card>
          <CardContent className="flex items-center justify-between p-4">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-muted p-2">
                <AlertCircle className="h-5 w-5 text-muted-foreground" />
              </div>
              <div>
                <p className="font-medium">Last Scan</p>
                <p className="text-sm text-muted-foreground">
                  {lastScan.profileName || "Unknown Profile"} •{" "}
                  {new Date(lastScan.createdAt).toLocaleString()}
                </p>
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => router.push(`/inspec/scans/${lastScan.id}`)}
            >
              View Details
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

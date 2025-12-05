"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { PageSkeleton, ErrorState } from "@/components/feedback";
import { useProfile, useControlMappings } from "@/hooks/use-inspec";
import {
  ArrowLeft,
  Play,
  Shield,
  FileText,
  ExternalLink,
  CheckCircle,
  XCircle,
  Link as LinkIcon,
  Package,
} from "lucide-react";

export default function ProfileDetailPage({ params }: { params: { id: string } }) {
  const router = useRouter();
  const profileId = params.id;

  const { data: profile, isLoading, error, refetch } = useProfile(profileId);
  const { data: mappings, isLoading: mappingsLoading } = useControlMappings(profileId);

  const [isRunning, setIsRunning] = useState(false);

  const handleRunProfile = () => {
    setIsRunning(true);
    // TODO: Show dialog to select target asset
    console.log("Run profile:", profile);
    setTimeout(() => setIsRunning(false), 2000);
  };

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Profile Details
            </h1>
            <p className="text-muted-foreground">Loading profile...</p>
          </div>
        </div>
        <PageSkeleton metricCards={0} showChart={false} showTable={false} />
      </div>
    );
  }

  if (error || !profile) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Profile Details
            </h1>
          </div>
        </div>
        <ErrorState
          error={error}
          retry={refetch}
          title="Failed to load profile"
          description="We couldn't fetch the profile details. Please try again."
        />
      </div>
    );
  }

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              {profile.title}
            </h1>
            <p className="text-muted-foreground">
              {profile.name} • v{profile.version}
            </p>
          </div>
        </div>
        <Button size="sm" onClick={handleRunProfile} disabled={isRunning}>
          <Play className="mr-2 h-4 w-4" />
          Run This Profile
        </Button>
      </div>

      {/* Profile Info Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-brand-accent/10 p-3">
              <Shield className="h-6 w-6 text-brand-accent" />
            </div>
            <div className="flex-1">
              <CardTitle className="text-lg">{profile.title}</CardTitle>
              <p className="text-sm text-muted-foreground mt-1">{profile.summary}</p>
            </div>
            <Badge variant="secondary">v{profile.version}</Badge>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-3">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Framework</span>
                <span className="font-medium">{profile.framework || "N/A"}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Maintainer</span>
                <span className="font-medium">{profile.maintainer || "N/A"}</span>
              </div>
              {profile.controlCount !== undefined && (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Controls</span>
                  <span className="font-medium flex items-center gap-1">
                    <FileText className="h-3 w-3" />
                    {profile.controlCount}
                  </span>
                </div>
              )}
            </div>
            <div className="space-y-3">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Platforms</span>
                <div className="flex gap-1 flex-wrap justify-end">
                  {profile.platforms.map((platform) => (
                    <Badge key={platform} variant="outline" className="text-xs">
                      <Package className="h-3 w-3 mr-1" />
                      {platform}
                    </Badge>
                  ))}
                </div>
              </div>
              {profile.profileUrl && (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Source</span>
                  <a
                    href={profile.profileUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1 text-brand-accent hover:underline"
                  >
                    <LinkIcon className="h-3 w-3" />
                    View Source
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </div>
              )}
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Created</span>
                <span className="font-medium">
                  {new Date(profile.createdAt).toLocaleDateString()}
                </span>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Control Mappings */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <LinkIcon className="h-4 w-4" />
            Control Mappings
            {mappings && (
              <Badge variant="secondary" className="ml-2">
                {mappings.length}
              </Badge>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {mappingsLoading ? (
            <div className="space-y-2">
              {[...Array(3)].map((_, i) => (
                <div key={i} className="h-16 bg-muted animate-pulse rounded" />
              ))}
            </div>
          ) : mappings && mappings.length > 0 ? (
            <div className="space-y-2">
              {mappings.map((mapping) => (
                <div
                  key={mapping.id}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div className="flex items-center gap-3">
                    <div className="rounded bg-brand-accent/10 p-2">
                      <LinkIcon className="h-4 w-4 text-brand-accent" />
                    </div>
                    <div>
                      <code className="text-sm font-mono">{mapping.inspecControlId}</code>
                      {mapping.notes && (
                        <p className="text-xs text-muted-foreground mt-1">{mapping.notes}</p>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">
                      {(mapping.mappingConfidence * 100).toFixed(0)}% confidence
                    </Badge>
                    {mapping.mappingConfidence >= 0.9 ? (
                      <CheckCircle className="h-4 w-4 text-status-green" />
                    ) : mapping.mappingConfidence >= 0.7 ? (
                      <CheckCircle className="h-4 w-4 text-status-amber" />
                    ) : (
                      <XCircle className="h-4 w-4 text-muted-foreground" />
                    )}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <LinkIcon className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p className="text-sm">No control mappings configured</p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Description */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">About This Profile</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground leading-relaxed">
            {profile.summary || "No description available for this profile."}
          </p>
          {profile.maintainer && (
            <p className="text-sm text-muted-foreground mt-4">
              Maintained by <span className="font-medium">{profile.maintainer}</span>
            </p>
          )}
        </CardContent>
      </Card>

      {/* Actions */}
      <Card>
        <CardContent className="flex items-center justify-between p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-muted p-2">
              <Shield className="h-5 w-5 text-muted-foreground" />
            </div>
            <div>
              <p className="font-medium">Ready to scan?</p>
              <p className="text-sm text-muted-foreground">
                Run this profile against your infrastructure assets
              </p>
            </div>
          </div>
          <Button onClick={handleRunProfile} disabled={isRunning}>
            <Play className="mr-2 h-4 w-4" />
            Run Profile
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

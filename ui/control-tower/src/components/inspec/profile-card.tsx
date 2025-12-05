"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Shield, Play, FileCheck, Loader2 } from "lucide-react";
import { InSpecProfile } from "@/lib/api-inspec";

interface ProfileCardProps {
  profile: InSpecProfile;
  onRun?: (profile: InSpecProfile) => void;
  isRunning?: boolean;
}

export function ProfileCard({ profile, onRun, isRunning }: ProfileCardProps) {
  return (
    <Card className="cursor-pointer hover:border-brand-accent transition-all">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <CardTitle className="text-base flex items-center gap-2">
              <Shield className="h-4 w-4 text-brand-accent" />
              {profile.title}
            </CardTitle>
            <p className="text-xs text-muted-foreground mt-1">
              {profile.summary}
            </p>
          </div>
          <Badge variant="secondary" className="text-xs">
            v{profile.version}
          </Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Framework</span>
            <span className="font-medium">{profile.framework || "N/A"}</span>
          </div>
          {profile.controlCount !== undefined && (
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Controls</span>
              <span className="font-medium flex items-center gap-1">
                <FileCheck className="h-3 w-3" />
                {profile.controlCount}
              </span>
            </div>
          )}
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Platforms</span>
            <div className="flex gap-1">
              {profile.platforms.slice(0, 3).map((platform) => (
                <Badge key={platform} variant="outline" className="text-xs">
                  {platform}
                </Badge>
              ))}
              {profile.platforms.length > 3 && (
                <Badge variant="outline" className="text-xs">
                  +{profile.platforms.length - 3}
                </Badge>
              )}
            </div>
          </div>
          {onRun && (
            <Button
              className="w-full mt-2"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onRun(profile);
              }}
              disabled={isRunning}
            >
              {isRunning ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Running...
                </>
              ) : (
                <>
                  <Play className="mr-2 h-4 w-4" />
                  Run Profile
                </>
              )}
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

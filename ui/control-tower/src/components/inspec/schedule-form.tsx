"use client";

import { useState } from "react";
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
import { Calendar, Loader2 } from "lucide-react";
import { CreateScheduleRequest } from "@/lib/api-inspec";

interface ScheduleFormProps {
  profiles: Array<{ id: string; title: string }>;
  assets?: Array<{ id: string; name: string }>;
  onSubmit: (data: CreateScheduleRequest) => void;
  onCancel: () => void;
  isLoading?: boolean;
}

const cronPresets = [
  { label: "Every day at midnight", value: "0 0 * * *" },
  { label: "Every week on Sunday", value: "0 0 * * 0" },
  { label: "Every month on 1st", value: "0 0 1 * *" },
  { label: "Every 6 hours", value: "0 */6 * * *" },
  { label: "Every hour", value: "0 * * * *" },
  { label: "Custom", value: "custom" },
];

export function ScheduleForm({
  profiles,
  assets,
  onSubmit,
  onCancel,
  isLoading,
}: ScheduleFormProps) {
  const [profileId, setProfileId] = useState("");
  const [assetId, setAssetId] = useState("");
  const [cronPreset, setCronPreset] = useState("0 0 * * *");
  const [customCron, setCustomCron] = useState("");
  const [enabled, setEnabled] = useState(true);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const cronExpression = cronPreset === "custom" ? customCron : cronPreset;

    onSubmit({
      profileId,
      assetId: assetId || undefined,
      cronExpression,
      enabled,
    });
  };

  const isValid = profileId && (cronPreset !== "custom" || customCron);

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="profile">Profile</Label>
        <Select value={profileId} onValueChange={setProfileId}>
          <SelectTrigger id="profile">
            <SelectValue placeholder="Select a profile" />
          </SelectTrigger>
          <SelectContent>
            {profiles.map((profile) => (
              <SelectItem key={profile.id} value={profile.id}>
                {profile.title}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {assets && assets.length > 0 && (
        <div className="space-y-2">
          <Label htmlFor="asset">Asset (Optional)</Label>
          <Select value={assetId} onValueChange={setAssetId}>
            <SelectTrigger id="asset">
              <SelectValue placeholder="All assets" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">All assets</SelectItem>
              {assets.map((asset) => (
                <SelectItem key={asset.id} value={asset.id}>
                  {asset.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      <div className="space-y-2">
        <Label htmlFor="schedule">Schedule</Label>
        <Select value={cronPreset} onValueChange={setCronPreset}>
          <SelectTrigger id="schedule">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {cronPresets.map((preset) => (
              <SelectItem key={preset.value} value={preset.value}>
                {preset.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {cronPreset === "custom" && (
        <div className="space-y-2">
          <Label htmlFor="cron">Cron Expression</Label>
          <Input
            id="cron"
            placeholder="0 0 * * *"
            value={customCron}
            onChange={(e) => setCustomCron(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            Format: minute hour day month weekday
          </p>
        </div>
      )}

      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          id="enabled"
          checked={enabled}
          onChange={(e) => setEnabled(e.target.checked)}
          className="h-4 w-4 rounded border-border"
        />
        <Label htmlFor="enabled" className="cursor-pointer">
          Enable schedule
        </Label>
      </div>

      <div className="flex justify-end gap-2 pt-4">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={!isValid || isLoading}>
          {isLoading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Creating...
            </>
          ) : (
            <>
              <Calendar className="mr-2 h-4 w-4" />
              Create Schedule
            </>
          )}
        </Button>
      </div>
    </form>
  );
}

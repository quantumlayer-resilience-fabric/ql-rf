"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useModifyTask, TaskWithPlan } from "@/hooks/use-ai";
import {
  Edit,
  Plus,
  Trash2,
  GripVertical,
  Loader2,
  AlertTriangle,
  Server,
  Clock,
} from "lucide-react";

interface Phase {
  name: string;
  assets?: number | string[];
  wait_time?: string;
  rollback_if?: string;
}

interface PlanModificationDialogProps {
  task: TaskWithPlan;
  trigger?: React.ReactNode;
  disabled?: boolean;
}

export function PlanModificationDialog({
  task,
  trigger,
  disabled = false,
}: PlanModificationDialogProps) {
  const [open, setOpen] = useState(false);
  const modifyTask = useModifyTask();

  // Extract current plan data
  const planPayload = task.plan?.payload;
  const currentPhases = (planPayload?.phases as Phase[]) || [];
  const currentRiskLevel = task.risk_level || "low";
  const currentEnvironment =
    (task.task_spec as Record<string, unknown>)?.environment as string ||
    "production";

  // Form state
  const [environment, setEnvironment] = useState(currentEnvironment);
  const [riskLevel, setRiskLevel] = useState(currentRiskLevel);
  const [notes, setNotes] = useState("");
  const [phases, setPhases] = useState<Phase[]>(currentPhases);

  // Reset form when dialog opens
  const handleOpenChange = (newOpen: boolean) => {
    if (newOpen) {
      setEnvironment(currentEnvironment);
      setRiskLevel(currentRiskLevel);
      setNotes("");
      setPhases(currentPhases);
    }
    setOpen(newOpen);
  };

  // Phase management
  const addPhase = () => {
    setPhases([
      ...phases,
      {
        name: `Phase ${phases.length + 1}`,
        assets: 10,
        wait_time: "5m",
        rollback_if: "",
      },
    ]);
  };

  const removePhase = (index: number) => {
    setPhases(phases.filter((_, i) => i !== index));
  };

  const updatePhase = (index: number, field: keyof Phase, value: string | number) => {
    setPhases(
      phases.map((phase, i) =>
        i === index ? { ...phase, [field]: value } : phase
      )
    );
  };

  const handleSubmit = async () => {
    try {
      await modifyTask.mutateAsync({
        taskId: task.id,
        modifications: {
          environment,
          risk_level: riskLevel,
          phases,
          notes: notes || undefined,
        },
      });
      setOpen(false);
    } catch (error) {
      console.error("Failed to modify task:", error);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="outline" className="w-full" disabled={disabled}>
            <Edit className="mr-2 h-4 w-4" />
            Modify Plan
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-w-2xl max-h-[90vh]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Edit className="h-5 w-5" />
            Modify Execution Plan
          </DialogTitle>
          <DialogDescription>
            Adjust the plan parameters before approval. Changes will be logged
            for audit purposes.
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="max-h-[60vh] pr-4">
          <div className="space-y-6 py-4">
            {/* Environment and Risk Level */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="environment">Environment</Label>
                <Select value={environment} onValueChange={setEnvironment}>
                  <SelectTrigger id="environment">
                    <SelectValue placeholder="Select environment" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="development">Development</SelectItem>
                    <SelectItem value="staging">Staging</SelectItem>
                    <SelectItem value="production">Production</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="risk-level">Risk Level</Label>
                <Select value={riskLevel} onValueChange={setRiskLevel}>
                  <SelectTrigger id="risk-level">
                    <SelectValue placeholder="Select risk level" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="low">
                      <div className="flex items-center gap-2">
                        <div className="h-2 w-2 rounded-full bg-status-green" />
                        Low
                      </div>
                    </SelectItem>
                    <SelectItem value="medium">
                      <div className="flex items-center gap-2">
                        <div className="h-2 w-2 rounded-full bg-status-amber" />
                        Medium
                      </div>
                    </SelectItem>
                    <SelectItem value="high">
                      <div className="flex items-center gap-2">
                        <div className="h-2 w-2 rounded-full bg-status-red" />
                        High
                      </div>
                    </SelectItem>
                    <SelectItem value="critical">
                      <div className="flex items-center gap-2">
                        <div className="h-2 w-2 rounded-full bg-destructive" />
                        Critical
                      </div>
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <Separator />

            {/* Execution Phases */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <Label className="text-base">Execution Phases</Label>
                <Button variant="outline" size="sm" onClick={addPhase}>
                  <Plus className="mr-2 h-3 w-3" />
                  Add Phase
                </Button>
              </div>

              {phases.length === 0 ? (
                <div className="rounded-lg border border-dashed p-8 text-center text-muted-foreground">
                  <Server className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p>No phases defined</p>
                  <p className="text-sm">Add phases to define the rollout strategy</p>
                </div>
              ) : (
                <div className="space-y-3">
                  {phases.map((phase, index) => (
                    <div
                      key={index}
                      className="rounded-lg border bg-muted/30 p-4 space-y-3"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <GripVertical className="h-4 w-4 text-muted-foreground cursor-move" />
                          <Badge variant="outline">Phase {index + 1}</Badge>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => removePhase(index)}
                          className="text-destructive hover:text-destructive"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>

                      <div className="grid grid-cols-2 gap-3">
                        <div className="space-y-1">
                          <Label className="text-xs">Phase Name</Label>
                          <Input
                            value={phase.name}
                            onChange={(e) =>
                              updatePhase(index, "name", e.target.value)
                            }
                            placeholder="e.g., Canary Deployment"
                          />
                        </div>
                        <div className="space-y-1">
                          <Label className="text-xs">
                            <Server className="h-3 w-3 inline mr-1" />
                            Assets (count or %)
                          </Label>
                          <Input
                            value={
                              typeof phase.assets === "number"
                                ? phase.assets
                                : Array.isArray(phase.assets)
                                ? phase.assets.length
                                : ""
                            }
                            onChange={(e) =>
                              updatePhase(
                                index,
                                "assets",
                                parseInt(e.target.value) || 0
                              )
                            }
                            type="number"
                            min={0}
                            placeholder="10"
                          />
                        </div>
                      </div>

                      <div className="grid grid-cols-2 gap-3">
                        <div className="space-y-1">
                          <Label className="text-xs">
                            <Clock className="h-3 w-3 inline mr-1" />
                            Wait Time (after phase)
                          </Label>
                          <Input
                            value={phase.wait_time || ""}
                            onChange={(e) =>
                              updatePhase(index, "wait_time", e.target.value)
                            }
                            placeholder="e.g., 5m, 1h, 30s"
                          />
                        </div>
                        <div className="space-y-1">
                          <Label className="text-xs">
                            <AlertTriangle className="h-3 w-3 inline mr-1" />
                            Rollback Condition
                          </Label>
                          <Input
                            value={phase.rollback_if || ""}
                            onChange={(e) =>
                              updatePhase(index, "rollback_if", e.target.value)
                            }
                            placeholder="e.g., error_rate > 5%"
                          />
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <Separator />

            {/* Modification Notes */}
            <div className="space-y-2">
              <Label htmlFor="notes">Modification Notes (Optional)</Label>
              <Textarea
                id="notes"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Explain why you're modifying the plan..."
                rows={3}
              />
              <p className="text-xs text-muted-foreground">
                These notes will be included in the audit trail.
              </p>
            </div>
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={modifyTask.isPending}>
            {modifyTask.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <Edit className="mr-2 h-4 w-4" />
                Save Modifications
              </>
            )}
          </Button>
        </DialogFooter>

        {modifyTask.isError && (
          <div className="mt-2 p-3 rounded-lg bg-destructive/10 text-destructive text-sm">
            <AlertTriangle className="h-4 w-4 inline mr-2" />
            {modifyTask.error.message}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

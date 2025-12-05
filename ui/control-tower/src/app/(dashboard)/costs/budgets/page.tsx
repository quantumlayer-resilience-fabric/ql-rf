"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import { BudgetProgressCard } from "@/components/finops/budget-progress-card";
import { useBudgets, useCreateBudget } from "@/hooks/use-finops";
import { CreateBudgetRequest } from "@/lib/api-finops";
import { Plus, Loader2 } from "lucide-react";

export default function BudgetsPage() {
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [showInactive, setShowInactive] = useState(false);

  const { data: budgetsData, isLoading, error, refetch } = useBudgets(!showInactive);
  const createBudget = useCreateBudget();

  // Form state - explicit type to ensure scope includes all valid values
  const [formData, setFormData] = useState({
    name: "",
    description: "",
    amount: 0,
    currency: "USD",
    period: "monthly" as const,
    scope: "organization" as "organization" | "cloud" | "service" | "site",
    scopeValue: "" as string | undefined,
    alertThreshold: 80,
    startDate: new Date().toISOString().split("T")[0],
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      await createBudget.mutateAsync(formData);
      setIsCreateDialogOpen(false);
      // Reset form
      setFormData({
        name: "",
        description: "",
        amount: 0,
        currency: "USD",
        period: "monthly" as const,
        scope: "organization" as "organization" | "cloud" | "service" | "site",
        scopeValue: "" as string | undefined,
        alertThreshold: 80,
        startDate: new Date().toISOString().split("T")[0],
      });
    } catch (error) {
      console.error("Failed to create budget:", error);
    }
  };

  if (isLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Budgets
            </h1>
            <p className="text-muted-foreground">
              Manage cost budgets and spending alerts.
            </p>
          </div>
        </div>
        <PageSkeleton metricCards={0} showChart={false} showTable={true} tableRows={5} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="page-transition space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Budgets
          </h1>
          <p className="text-muted-foreground">
            Manage cost budgets and spending alerts.
          </p>
        </div>
        <ErrorState
          error={error}
          retry={refetch}
          title="Failed to load budgets"
          description="We couldn't fetch the budgets. Please try again."
        />
      </div>
    );
  }

  const budgets = budgetsData?.budgets || [];

  return (
    <div className="page-transition space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Budgets
          </h1>
          <p className="text-muted-foreground">
            Manage cost budgets and spending alerts.
          </p>
        </div>
        <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="mr-2 h-4 w-4" />
              Create Budget
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[500px]">
            <form onSubmit={handleSubmit}>
              <DialogHeader>
                <DialogTitle>Create Budget</DialogTitle>
                <DialogDescription>
                  Set up a new budget to track spending and receive alerts.
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="space-y-2">
                  <Label htmlFor="name">Name</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) =>
                      setFormData({ ...formData, name: e.target.value })
                    }
                    placeholder="e.g., Monthly Cloud Budget"
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="description">Description (Optional)</Label>
                  <Textarea
                    id="description"
                    value={formData.description}
                    onChange={(e) =>
                      setFormData({ ...formData, description: e.target.value })
                    }
                    placeholder="Budget description"
                    rows={2}
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="amount">Amount</Label>
                    <Input
                      id="amount"
                      type="number"
                      min="0"
                      step="0.01"
                      value={formData.amount || ""}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          amount: parseFloat(e.target.value) || 0,
                        })
                      }
                      placeholder="50000"
                      required
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="currency">Currency</Label>
                    <Select
                      value={formData.currency}
                      onValueChange={(value) =>
                        setFormData({ ...formData, currency: value })
                      }
                    >
                      <SelectTrigger id="currency">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="USD">USD</SelectItem>
                        <SelectItem value="EUR">EUR</SelectItem>
                        <SelectItem value="GBP">GBP</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="period">Period</Label>
                    <Select
                      value={formData.period}
                      onValueChange={(value: any) =>
                        setFormData({ ...formData, period: value })
                      }
                    >
                      <SelectTrigger id="period">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="daily">Daily</SelectItem>
                        <SelectItem value="weekly">Weekly</SelectItem>
                        <SelectItem value="monthly">Monthly</SelectItem>
                        <SelectItem value="quarterly">Quarterly</SelectItem>
                        <SelectItem value="yearly">Yearly</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="alertThreshold">Alert Threshold (%)</Label>
                    <Input
                      id="alertThreshold"
                      type="number"
                      min="0"
                      max="100"
                      value={formData.alertThreshold}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          alertThreshold: parseFloat(e.target.value) || 80,
                        })
                      }
                      required
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="scope">Scope</Label>
                  <Select
                    value={formData.scope}
                    onValueChange={(value: any) =>
                      setFormData({ ...formData, scope: value })
                    }
                  >
                    <SelectTrigger id="scope">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="organization">Organization</SelectItem>
                      <SelectItem value="cloud">Cloud</SelectItem>
                      <SelectItem value="service">Service</SelectItem>
                      <SelectItem value="site">Site</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {formData.scope !== "organization" && (
                  <div className="space-y-2">
                    <Label htmlFor="scopeValue">Scope Value</Label>
                    <Input
                      id="scopeValue"
                      value={formData.scopeValue || ""}
                      onChange={(e) =>
                        setFormData({ ...formData, scopeValue: e.target.value })
                      }
                      placeholder={
                        formData.scope === "cloud"
                          ? "e.g., aws"
                          : formData.scope === "service"
                          ? "e.g., ec2"
                          : "e.g., us-east-1"
                      }
                      required
                    />
                  </div>
                )}

                <div className="space-y-2">
                  <Label htmlFor="startDate">Start Date</Label>
                  <Input
                    id="startDate"
                    type="date"
                    value={formData.startDate}
                    onChange={(e) =>
                      setFormData({ ...formData, startDate: e.target.value })
                    }
                    required
                  />
                </div>
              </div>
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setIsCreateDialogOpen(false)}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={createBudget.isPending}>
                  {createBudget.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    "Create Budget"
                  )}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {/* Budget Cards */}
      {budgets.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {budgets.map((budget) => (
            <BudgetProgressCard key={budget.id} budget={budget} />
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="p-8">
            <EmptyState
              variant="data"
              title="No budgets configured"
              description="Create your first budget to track spending and receive alerts."
              action={{
                label: "Create Budget",
                onClick: () => setIsCreateDialogOpen(true),
                icon: <Plus className="mr-2 h-4 w-4" />,
              }}
            />
          </CardContent>
        </Card>
      )}
    </div>
  );
}

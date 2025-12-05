"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Scale, AlertTriangle, CheckCircle, FileText } from "lucide-react";
import type { LicenseSummary } from "@/lib/api-sbom";

interface LicenseDistributionChartProps {
  licenseSummary: LicenseSummary;
}

export function LicenseDistributionChart({
  licenseSummary,
}: LicenseDistributionChartProps) {
  const getCategoryIcon = (category: string) => {
    switch (category) {
      case "permissive":
        return <CheckCircle className="h-4 w-4 text-status-green" />;
      case "copyleft":
        return <AlertTriangle className="h-4 w-4 text-status-amber" />;
      case "proprietary":
        return <AlertTriangle className="h-4 w-4 text-status-red" />;
      default:
        return <FileText className="h-4 w-4 text-muted-foreground" />;
    }
  };

  const getCategoryColor = (category: string) => {
    switch (category) {
      case "permissive":
        return "bg-status-green/10 border-status-green/30 text-status-green";
      case "copyleft":
        return "bg-status-amber/10 border-status-amber/30 text-status-amber";
      case "proprietary":
        return "bg-status-red/10 border-status-red/30 text-status-red";
      default:
        return "bg-muted border-border text-muted-foreground";
    }
  };

  const getRiskLevelBadge = (riskScore: number) => {
    if (riskScore >= 80) {
      return (
        <Badge className="bg-status-red/10 text-status-red border-status-red/30">
          High Risk
        </Badge>
      );
    } else if (riskScore >= 50) {
      return (
        <Badge className="bg-status-amber/10 text-status-amber border-status-amber/30">
          Medium Risk
        </Badge>
      );
    } else {
      return (
        <Badge className="bg-status-green/10 text-status-green border-status-green/30">
          Low Risk
        </Badge>
      );
    }
  };

  // Sort licenses by count (descending)
  const sortedLicenses = [...licenseSummary.licenses].sort(
    (a, b) => b.count - a.count
  );

  return (
    <div className="space-y-4">
      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-brand-accent/10 p-2">
                <Scale className="h-5 w-5 text-brand-accent" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Total Licenses</p>
                <p className="text-2xl font-bold">{licenseSummary.licenses.length}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-status-amber/10 p-2">
                <AlertTriangle className="h-5 w-5 text-status-amber" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Unlicensed Packages</p>
                <p className="text-2xl font-bold">{licenseSummary.unlicensedPackages}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-muted p-2">
                <CheckCircle className="h-5 w-5" />
              </div>
              <div className="flex items-center justify-between w-full">
                <div>
                  <p className="text-sm text-muted-foreground">Risk Score</p>
                  <p className="text-2xl font-bold">{licenseSummary.riskScore}</p>
                </div>
                {getRiskLevelBadge(licenseSummary.riskScore)}
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* License List */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">License Distribution</CardTitle>
        </CardHeader>
        <CardContent>
          {sortedLicenses.length > 0 ? (
            <div className="space-y-3">
              {sortedLicenses.map((license) => {
                const percentage = (license.count / licenseSummary.totalPackages) * 100;
                return (
                  <div key={license.name} className="space-y-2">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        {getCategoryIcon(license.category)}
                        <span className="font-medium">{license.name}</span>
                        <Badge
                          variant="outline"
                          className={`text-xs ${getCategoryColor(license.category)}`}
                        >
                          {license.category}
                        </Badge>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm text-muted-foreground">
                          {license.count} packages
                        </span>
                        <span className="text-sm font-medium">
                          {percentage.toFixed(1)}%
                        </span>
                      </div>
                    </div>
                    <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                      <div
                        className={`h-full transition-all ${
                          license.category === "permissive"
                            ? "bg-status-green"
                            : license.category === "copyleft"
                            ? "bg-status-amber"
                            : license.category === "proprietary"
                            ? "bg-status-red"
                            : "bg-muted-foreground"
                        }`}
                        style={{ width: `${percentage}%` }}
                      />
                    </div>
                    {license.packages.length > 0 && (
                      <details className="text-xs">
                        <summary className="cursor-pointer text-muted-foreground hover:text-foreground">
                          View packages ({license.packages.length})
                        </summary>
                        <div className="mt-2 flex flex-wrap gap-1">
                          {license.packages.slice(0, 10).map((pkg) => (
                            <Badge key={pkg} variant="secondary" className="text-xs">
                              {pkg}
                            </Badge>
                          ))}
                          {license.packages.length > 10 && (
                            <Badge variant="outline" className="text-xs">
                              +{license.packages.length - 10} more
                            </Badge>
                          )}
                        </div>
                      </details>
                    )}
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="py-8 text-center text-muted-foreground">
              No license data available
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

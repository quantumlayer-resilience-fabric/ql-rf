"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertTriangle,
  Shield,
  TrendingUp,
  TrendingDown,
  Activity,
  Server,
  Cloud,
  MapPin,
  Target,
  Lightbulb,
  ArrowRight,
  CheckCircle2,
  Clock,
  Bot,
  Sparkles,
  Bell,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { useRiskSummary, useRiskForecast, useRiskRecommendations } from "@/hooks/use-risk";
import {
  RiskLevel,
  AssetRiskScore,
  RiskByScope,
  RiskVelocity,
  RiskPrediction,
  RiskRecommendation,
  RiskAnomaly,
  RiskForecast,
} from "@/lib/api";

// Risk level badge colors
const riskLevelColors: Record<RiskLevel, string> = {
  critical: "bg-red-500 hover:bg-red-600",
  high: "bg-orange-500 hover:bg-orange-600",
  medium: "bg-yellow-500 hover:bg-yellow-600",
  low: "bg-green-500 hover:bg-green-600",
};

const riskLevelTextColors: Record<RiskLevel, string> = {
  critical: "text-red-600",
  high: "text-orange-600",
  medium: "text-yellow-600",
  low: "text-green-600",
};

function RiskLevelBadge({ level }: { level: RiskLevel }) {
  return (
    <Badge className={`${riskLevelColors[level]} text-white`}>
      {level.charAt(0).toUpperCase() + level.slice(1)}
    </Badge>
  );
}

function RiskScoreGauge({ score, level }: { score: number; level: RiskLevel }) {
  return (
    <div className="flex flex-col items-center">
      <div className="relative w-32 h-32">
        <svg className="w-full h-full transform -rotate-90" viewBox="0 0 100 100">
          <circle
            cx="50"
            cy="50"
            r="45"
            stroke="currentColor"
            strokeWidth="10"
            fill="none"
            className="text-muted"
          />
          <circle
            cx="50"
            cy="50"
            r="45"
            stroke="currentColor"
            strokeWidth="10"
            fill="none"
            strokeDasharray={`${score * 2.83} 283`}
            className={riskLevelTextColors[level]}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className={`text-3xl font-bold ${riskLevelTextColors[level]}`}>
            {Math.round(score)}
          </span>
          <span className="text-xs text-muted-foreground">Risk Score</span>
        </div>
      </div>
    </div>
  );
}

function TopRisksTable({ risks }: { risks: AssetRiskScore[] }) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Asset</TableHead>
          <TableHead>Environment</TableHead>
          <TableHead>Platform</TableHead>
          <TableHead>Risk Score</TableHead>
          <TableHead>Level</TableHead>
          <TableHead>Key Factors</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {risks.map((risk) => (
          <TableRow key={risk.assetId}>
            <TableCell className="font-medium">{risk.assetName}</TableCell>
            <TableCell>
              <Badge variant="outline">{risk.environment}</Badge>
            </TableCell>
            <TableCell>{risk.platform}</TableCell>
            <TableCell>
              <span className={`font-bold ${riskLevelTextColors[risk.riskLevel]}`}>
                {Math.round(risk.riskScore)}
              </span>
            </TableCell>
            <TableCell>
              <RiskLevelBadge level={risk.riskLevel} />
            </TableCell>
            <TableCell>
              <div className="flex flex-wrap gap-1">
                {risk.criticalVulns > 0 && (
                  <Badge variant="destructive" className="text-xs">
                    {risk.criticalVulns} Critical CVE
                  </Badge>
                )}
                {risk.driftAge > 0 && (
                  <Badge variant="secondary" className="text-xs">
                    {risk.driftAge}d drift
                  </Badge>
                )}
                {!risk.isCompliant && (
                  <Badge variant="outline" className="text-xs">
                    Non-compliant
                  </Badge>
                )}
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function RiskByScopeCards({ data, icon: Icon }: { data: RiskByScope[]; icon: React.ElementType }) {
  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {data.map((scope) => (
        <Card key={scope.scope}>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{scope.scope}</CardTitle>
            <Icon className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between mb-2">
              <span className={`text-2xl font-bold ${riskLevelTextColors[scope.riskLevel]}`}>
                {Math.round(scope.riskScore)}
              </span>
              <RiskLevelBadge level={scope.riskLevel} />
            </div>
            <Progress
              value={scope.riskScore}
              className="h-2"
            />
            <div className="flex justify-between mt-2 text-xs text-muted-foreground">
              <span>{scope.assetCount} assets</span>
              <span>
                {scope.criticalRisk > 0 && (
                  <span className="text-red-600">{scope.criticalRisk} critical</span>
                )}
                {scope.criticalRisk > 0 && scope.highRisk > 0 && " / "}
                {scope.highRisk > 0 && (
                  <span className="text-orange-600">{scope.highRisk} high</span>
                )}
              </span>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function RiskTrendChart({ trend }: { trend: { date: string; riskScore: number }[] }) {
  const maxScore = Math.max(...trend.map((t) => t.riskScore), 100);
  const minScore = Math.min(...trend.map((t) => t.riskScore), 0);

  return (
    <div className="h-48 flex items-end gap-1">
      {trend.slice(-30).map((point, index) => {
        const height = ((point.riskScore - minScore) / (maxScore - minScore)) * 100;
        const date = new Date(point.date);
        return (
          <div
            key={index}
            className="flex-1 flex flex-col items-center group"
          >
            <div className="relative w-full">
              <div
                className={`w-full rounded-t transition-all ${
                  point.riskScore >= 80
                    ? "bg-red-500"
                    : point.riskScore >= 60
                    ? "bg-orange-500"
                    : point.riskScore >= 40
                    ? "bg-yellow-500"
                    : "bg-green-500"
                }`}
                style={{ height: `${height}%`, minHeight: "4px" }}
              />
              <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 bg-popover text-popover-foreground text-xs rounded px-2 py-1 opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-10 shadow-lg">
                {date.toLocaleDateString()}: {Math.round(point.riskScore)}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

// Velocity indicator colors and labels
const velocityConfig: Record<RiskVelocity, { color: string; bgColor: string; icon: typeof TrendingUp; label: string }> = {
  rapid_increase: { color: "text-red-600", bgColor: "bg-red-100", icon: TrendingUp, label: "Rapidly Increasing" },
  increasing: { color: "text-orange-600", bgColor: "bg-orange-100", icon: TrendingUp, label: "Increasing" },
  stable: { color: "text-blue-600", bgColor: "bg-blue-100", icon: Activity, label: "Stable" },
  decreasing: { color: "text-green-600", bgColor: "bg-green-100", icon: TrendingDown, label: "Decreasing" },
  rapid_decrease: { color: "text-emerald-600", bgColor: "bg-emerald-100", icon: TrendingDown, label: "Rapidly Decreasing" },
};

function VelocityIndicator({ velocity, pointsPerDay }: { velocity: RiskVelocity; pointsPerDay: number }) {
  const config = velocityConfig[velocity];
  const Icon = config.icon;

  return (
    <div className={`flex items-center gap-2 px-3 py-2 rounded-lg ${config.bgColor}`}>
      <Icon className={`h-5 w-5 ${config.color}`} />
      <div>
        <div className={`font-semibold ${config.color}`}>{config.label}</div>
        <div className="text-xs text-muted-foreground">
          {pointsPerDay > 0 ? "+" : ""}{pointsPerDay.toFixed(1)} pts/day
        </div>
      </div>
    </div>
  );
}

function PredictionCard({ prediction, label }: { prediction: RiskPrediction; label: string }) {
  const level = prediction.predictedLevel;

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-purple-500" />
          {label}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between mb-2">
          <span className={`text-2xl font-bold ${riskLevelTextColors[level]}`}>
            {Math.round(prediction.predictedScore)}
          </span>
          <RiskLevelBadge level={level} />
        </div>
        <div className="text-xs text-muted-foreground">
          Confidence: {Math.round(prediction.confidence * 100)}%
        </div>
        <Progress value={prediction.confidence * 100} className="h-1 mt-1" />
        {prediction.factors && prediction.factors.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1">
            {prediction.factors.slice(0, 2).map((factor, i) => (
              <Badge key={i} variant="outline" className="text-xs">
                {factor}
              </Badge>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function RecommendationsList({ recommendations }: { recommendations: RiskRecommendation[] }) {
  // Map numeric priority to color classes
  const getPriorityColor = (priority: number): string => {
    if (priority <= 1) return "border-l-red-500"; // Critical
    if (priority <= 2) return "border-l-orange-500"; // High
    if (priority <= 3) return "border-l-yellow-500"; // Medium
    return "border-l-green-500"; // Low
  };

  // Map effort to display text
  const getEffortLabel = (effort: "low" | "medium" | "high"): string => {
    const labels = { low: "Low effort", medium: "Medium effort", high: "High effort" };
    return labels[effort] || effort;
  };

  return (
    <div className="space-y-3">
      {recommendations.map((rec) => (
        <Card key={rec.id} className={`border-l-4 ${getPriorityColor(rec.priority)}`}>
          <CardContent className="py-4">
            <div className="flex items-start justify-between gap-4">
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-1">
                  <Lightbulb className="h-4 w-4 text-amber-500" />
                  <span className="font-semibold">{rec.title}</span>
                  {rec.autoRemediable && (
                    <Badge variant="secondary" className="text-xs">
                      <Bot className="h-3 w-3 mr-1" />
                      Auto-fix
                    </Badge>
                  )}
                </div>
                <p className="text-sm text-muted-foreground mb-2">{rec.description}</p>
                <div className="flex items-center gap-4 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <Target className="h-3 w-3" />
                    {rec.affectedAssets} assets affected
                  </span>
                  <span className="flex items-center gap-1">
                    <TrendingDown className="h-3 w-3 text-green-500" />
                    {rec.impact}
                  </span>
                  <span className="flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    {getEffortLabel(rec.effort)}
                  </span>
                </div>
              </div>
              <Button size="sm" variant="outline" className="shrink-0">
                View Details
                <ArrowRight className="h-4 w-4 ml-1" />
              </Button>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function AnomalyAlerts({ anomalies }: { anomalies: RiskAnomaly[] }) {
  if (anomalies.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        <CheckCircle2 className="h-12 w-12 mx-auto mb-2 text-green-500" />
        <p>No anomalies detected</p>
        <p className="text-xs">Risk patterns are within normal ranges</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {anomalies.map((anomaly) => (
        <Card key={anomaly.id} className="border-l-4 border-l-purple-500 bg-purple-50/50 dark:bg-purple-950/20">
          <CardContent className="py-4">
            <div className="flex items-start gap-3">
              <Bell className="h-5 w-5 text-purple-500 mt-0.5" />
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-semibold">{anomaly.scope || anomaly.assetId || "System"}</span>
                  <Badge variant="outline" className="text-xs">
                    {anomaly.anomalyType.replace("_", " ")}
                  </Badge>
                  {anomaly.isActive && (
                    <Badge variant="destructive" className="text-xs">Active</Badge>
                  )}
                </div>
                <p className="text-sm text-muted-foreground mb-2">{anomaly.description}</p>
                <div className="flex items-center gap-4 text-xs">
                  <span className="text-muted-foreground">
                    Expected: {Math.round(anomaly.expectedScore)}
                  </span>
                  <span className={anomaly.actualScore > anomaly.expectedScore ? "text-red-600" : "text-green-600"}>
                    Actual: {Math.round(anomaly.actualScore)}
                  </span>
                  <span className="text-purple-600">
                    {anomaly.deviation.toFixed(1)} std dev
                  </span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function ForecastSection({ forecast }: { forecast: RiskForecast }) {
  return (
    <div className="space-y-6">
      {/* Velocity & Current State */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="col-span-1 lg:col-span-2">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Risk Velocity</CardTitle>
            <CardDescription>Rate of change over the past 7 days</CardDescription>
          </CardHeader>
          <CardContent>
            <VelocityIndicator
              velocity={forecast.velocity}
              pointsPerDay={forecast.velocityValue}
            />
          </CardContent>
        </Card>

        {/* Predictions */}
        {forecast.predictions.map((pred) => (
          <PredictionCard
            key={pred.predictionHorizon}
            prediction={pred}
            label={`${pred.predictionHorizon}-Day Forecast`}
          />
        ))}
      </div>

      {/* Anomalies */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bell className="h-5 w-5 text-purple-500" />
            Detected Anomalies
          </CardTitle>
          <CardDescription>
            Unusual risk patterns that deviate significantly from expected values
          </CardDescription>
        </CardHeader>
        <CardContent>
          <AnomalyAlerts anomalies={forecast.anomalies} />
        </CardContent>
      </Card>
    </div>
  );
}

export default function RiskPage() {
  const { data: riskSummary, isLoading, error } = useRiskSummary();
  const { data: forecast, isLoading: forecastLoading } = useRiskForecast();
  const { data: recommendations, isLoading: recsLoading } = useRiskRecommendations();

  if (isLoading) {
    return (
      <div className="space-y-6 p-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold tracking-tight">Risk Analysis</h1>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Card key={i}>
              <CardHeader className="pb-2">
                <Skeleton className="h-4 w-24" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-8 w-16" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  if (error || !riskSummary) {
    return (
      <div className="flex items-center justify-center h-96">
        <Card className="w-96">
          <CardContent className="pt-6">
            <div className="flex flex-col items-center text-center">
              <AlertTriangle className="h-12 w-12 text-destructive mb-4" />
              <h3 className="font-semibold">Failed to load risk data</h3>
              <p className="text-sm text-muted-foreground mt-2">
                Please try refreshing the page
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const trendDirection =
    riskSummary.trend.length >= 2
      ? riskSummary.trend[riskSummary.trend.length - 1].riskScore >
        riskSummary.trend[riskSummary.trend.length - 2].riskScore
        ? "up"
        : "down"
      : "neutral";

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Risk Analysis</h1>
          <p className="text-muted-foreground">
            AI-powered risk scoring across your infrastructure
          </p>
        </div>
        <div className="text-sm text-muted-foreground">
          Last calculated: {new Date(riskSummary.calculatedAt).toLocaleString()}
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
        {/* Overall Risk Score */}
        <Card className="col-span-1">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Overall Risk</CardTitle>
          </CardHeader>
          <CardContent>
            <RiskScoreGauge
              score={riskSummary.overallRiskScore}
              level={riskSummary.riskLevel}
            />
          </CardContent>
        </Card>

        {/* Risk Distribution */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Critical Risk</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">
              {riskSummary.criticalRisk}
            </div>
            <p className="text-xs text-muted-foreground">
              assets at critical risk
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">High Risk</CardTitle>
            <Shield className="h-4 w-4 text-orange-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-orange-600">
              {riskSummary.highRisk}
            </div>
            <p className="text-xs text-muted-foreground">
              assets at high risk
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Medium Risk</CardTitle>
            <Activity className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-yellow-600">
              {riskSummary.mediumRisk}
            </div>
            <p className="text-xs text-muted-foreground">
              assets at medium risk
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Low Risk</CardTitle>
            <Shield className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {riskSummary.lowRisk}
            </div>
            <p className="text-xs text-muted-foreground">
              assets at low risk
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Risk Trend */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Risk Trend</CardTitle>
              <CardDescription>30-day risk score history</CardDescription>
            </div>
            <div className="flex items-center gap-2">
              {trendDirection === "up" ? (
                <TrendingUp className="h-4 w-4 text-red-500" />
              ) : (
                <TrendingDown className="h-4 w-4 text-green-500" />
              )}
              <span
                className={
                  trendDirection === "up" ? "text-red-500" : "text-green-500"
                }
              >
                {trendDirection === "up" ? "Increasing" : "Decreasing"} risk
              </span>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <RiskTrendChart trend={riskSummary.trend} />
        </CardContent>
      </Card>

      {/* Tabbed Content */}
      <Tabs defaultValue="top-risks" className="space-y-4">
        <TabsList>
          <TabsTrigger value="top-risks">Top Risks</TabsTrigger>
          <TabsTrigger value="predictions" className="flex items-center gap-1">
            <Sparkles className="h-3 w-3" />
            Predictions
          </TabsTrigger>
          <TabsTrigger value="recommendations" className="flex items-center gap-1">
            <Lightbulb className="h-3 w-3" />
            Recommendations
          </TabsTrigger>
          <TabsTrigger value="by-environment">By Environment</TabsTrigger>
          <TabsTrigger value="by-platform">By Platform</TabsTrigger>
          <TabsTrigger value="by-site">By Site</TabsTrigger>
        </TabsList>

        <TabsContent value="top-risks">
          <Card>
            <CardHeader>
              <CardTitle>Highest Risk Assets</CardTitle>
              <CardDescription>
                Assets requiring immediate attention based on risk factors
              </CardDescription>
            </CardHeader>
            <CardContent>
              {riskSummary.topRisks.length > 0 ? (
                <TopRisksTable risks={riskSummary.topRisks} />
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  No high-risk assets found
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="predictions">
          {forecastLoading ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              {[...Array(4)].map((_, i) => (
                <Card key={i}>
                  <CardHeader className="pb-2">
                    <Skeleton className="h-4 w-24" />
                  </CardHeader>
                  <CardContent>
                    <Skeleton className="h-8 w-16 mb-2" />
                    <Skeleton className="h-2 w-full" />
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : forecast ? (
            <ForecastSection forecast={forecast} />
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <AlertTriangle className="h-12 w-12 mx-auto mb-2" />
              <p>Unable to load predictions</p>
            </div>
          )}
        </TabsContent>

        <TabsContent value="recommendations">
          {recsLoading ? (
            <div className="space-y-3">
              {[...Array(3)].map((_, i) => (
                <Card key={i}>
                  <CardContent className="py-4">
                    <Skeleton className="h-4 w-48 mb-2" />
                    <Skeleton className="h-3 w-full mb-2" />
                    <Skeleton className="h-3 w-32" />
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : recommendations && recommendations.length > 0 ? (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Lightbulb className="h-5 w-5 text-amber-500" />
                  Risk Mitigation Recommendations
                </CardTitle>
                <CardDescription>
                  AI-generated recommendations to reduce organizational risk, sorted by impact
                </CardDescription>
              </CardHeader>
              <CardContent>
                <RecommendationsList recommendations={recommendations} />
              </CardContent>
            </Card>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <CheckCircle2 className="h-12 w-12 mx-auto mb-2 text-green-500" />
              <p>No recommendations at this time</p>
              <p className="text-xs">Your infrastructure risk is well-managed</p>
            </div>
          )}
        </TabsContent>

        <TabsContent value="by-environment">
          <RiskByScopeCards data={riskSummary.byEnvironment} icon={Server} />
        </TabsContent>

        <TabsContent value="by-platform">
          <RiskByScopeCards data={riskSummary.byPlatform} icon={Cloud} />
        </TabsContent>

        <TabsContent value="by-site">
          <RiskByScopeCards data={riskSummary.bySite} icon={MapPin} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

"use client";

import { useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { PageSkeleton, ErrorState, EmptyState } from "@/components/feedback";
import { useScan, useScanResults } from "@/hooks/use-inspec";
import { useSendAIMessage, useAIContext } from "@/hooks/use-ai";
import { ScanStatusBadge } from "@/components/inspec/scan-status-badge";
import { ScanSummaryCard } from "@/components/inspec/scan-summary-card";
import { ControlResultRow } from "@/components/inspec/control-result-row";
import {
  ArrowLeft,
  Download,
  Clock,
  Calendar,
  Shield,
  Loader2,
  Zap,
  FileText,
} from "lucide-react";
import { InSpecResult } from "@/lib/api-inspec";
import { useState } from "react";

export default function ScanResultsPage({ params }: { params: { id: string } }) {
  const router = useRouter();
  const scanId = params.id;
  const [isRemediating, setIsRemediating] = useState(false);

  const { data: scan, isLoading: scanLoading, error: scanError, refetch: refetchScan } = useScan(scanId);
  const { data: resultsData, isLoading: resultsLoading, error: resultsError, refetch: refetchResults } = useScanResults(scanId);

  const aiContext = useAIContext();
  const sendAIMessage = useSendAIMessage();

  const results = resultsData?.results || [];
  const passedResults = results.filter((r) => r.status === "passed");
  const failedResults = results.filter((r) => r.status === "failed");
  const skippedResults = results.filter((r) => r.status === "skipped");

  const handleRemediateControl = async (result: InSpecResult) => {
    setIsRemediating(true);
    try {
      const intent = `Fix InSpec control ${result.controlId} (${result.controlTitle}) that failed during scan. ${result.message || ""}`;
      await sendAIMessage.mutateAsync({
        message: intent,
        context: aiContext,
      });
      router.push("/ai");
    } catch (error) {
      console.error("Failed to create AI task:", error);
    } finally {
      setIsRemediating(false);
    }
  };

  const handleRemediateAll = async () => {
    if (failedResults.length === 0) return;

    setIsRemediating(true);
    try {
      const intent = `Fix ${failedResults.length} failing InSpec controls from scan ${scanId}. Overall pass rate is ${scan?.passRate?.toFixed(1)}%.`;
      await sendAIMessage.mutateAsync({
        message: intent,
        context: aiContext,
      });
      router.push("/ai");
    } catch (error) {
      console.error("Failed to create AI task:", error);
    } finally {
      setIsRemediating(false);
    }
  };

  const handleExportResults = () => {
    // TODO: Implement export functionality
    console.log("Export results");
  };

  if (scanLoading || resultsLoading) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Scan Results
            </h1>
            <p className="text-muted-foreground">Loading scan details...</p>
          </div>
        </div>
        <PageSkeleton metricCards={3} showChart={false} showTable={true} tableRows={10} />
      </div>
    );
  }

  if (scanError || resultsError || !scan) {
    return (
      <div className="page-transition space-y-6">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Scan Results
            </h1>
          </div>
        </div>
        <ErrorState
          error={scanError || resultsError}
          retry={() => {
            refetchScan();
            refetchResults();
          }}
          title="Failed to load scan results"
          description="We couldn't fetch the scan results. Please try again."
        />
      </div>
    );
  }

  const scanDuration = scan.duration || 0;
  const scanScore = scan.passRate || 0;

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
              Scan Results
            </h1>
            <p className="text-muted-foreground">
              {scan.profileName || "Unknown Profile"} • {scan.framework || "N/A"}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={handleExportResults}>
            <Download className="mr-2 h-4 w-4" />
            Export
          </Button>
          {failedResults.length > 0 && (
            <Button size="sm" onClick={handleRemediateAll} disabled={isRemediating}>
              {isRemediating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating Task...
                </>
              ) : (
                <>
                  <Zap className="mr-2 h-4 w-4" />
                  Fix All with AI
                </>
              )}
            </Button>
          )}
        </div>
      </div>

      {/* Scan Summary Header */}
      <Card>
        <CardContent className="p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-6">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-brand-accent/10 p-3">
                  <Shield className="h-6 w-6 text-brand-accent" />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-semibold">{scan.profileName || "Unknown Profile"}</h3>
                    <ScanStatusBadge status={scan.status} />
                  </div>
                  <div className="flex items-center gap-4 mt-1 text-sm text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Calendar className="h-3 w-3" />
                      {new Date(scan.createdAt).toLocaleString()}
                    </span>
                    {scanDuration > 0 && (
                      <span className="flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {scanDuration}s
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </div>
            {scan.status === "completed" && (
              <div className="text-right">
                <div className="text-3xl font-bold">
                  {scanScore.toFixed(1)}%
                </div>
                <div className="text-sm text-muted-foreground">Overall Score</div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Summary Cards */}
      {scan.status === "completed" && (
        <div className="grid gap-4 md:grid-cols-3">
          <ScanSummaryCard type="passed" count={passedResults.length} total={results.length} />
          <ScanSummaryCard type="failed" count={failedResults.length} total={results.length} />
          <ScanSummaryCard type="skipped" count={skippedResults.length} total={results.length} />
        </div>
      )}

      {/* AI Remediation Card */}
      {failedResults.length > 0 && (
        <Card className="border-l-4 border-l-status-amber bg-gradient-to-r from-status-amber/5 to-transparent">
          <CardContent className="flex items-start gap-4 p-6">
            <div className="rounded-lg p-2 bg-status-amber/10">
              <Zap className="h-5 w-5 text-status-amber" />
            </div>
            <div className="flex-1">
              <h3 className="font-semibold">
                {failedResults.length} Control{failedResults.length > 1 ? "s" : ""} Failed
              </h3>
              <p className="mt-1 text-sm text-muted-foreground">
                AI can analyze failures and generate remediation scripts to fix compliance gaps.
              </p>
            </div>
            <Button size="sm" onClick={handleRemediateAll} disabled={isRemediating}>
              {isRemediating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Zap className="mr-2 h-4 w-4" />
                  Remediate All
                </>
              )}
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Results Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-base flex items-center gap-2">
              <FileText className="h-4 w-4" />
              Control Results ({results.length})
            </CardTitle>
            <div className="flex gap-2">
              <Badge variant="outline" className="text-status-green">
                {passedResults.length} Passed
              </Badge>
              <Badge variant="outline" className="text-status-red">
                {failedResults.length} Failed
              </Badge>
              <Badge variant="outline" className="text-muted-foreground">
                {skippedResults.length} Skipped
              </Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {results.length > 0 ? (
            <div className="rounded-lg border">
              {results.map((result) => (
                <ControlResultRow
                  key={result.id}
                  result={result}
                  onRemediate={result.status === "failed" ? handleRemediateControl : undefined}
                />
              ))}
            </div>
          ) : (
            <EmptyState
              variant="data"
              title="No results available"
              description="This scan doesn't have any results yet."
            />
          )}
        </CardContent>
      </Card>

      {/* Scan Error */}
      {scan.status === "failed" && scan.errorMessage && (
        <Card className="border-status-red">
          <CardHeader>
            <CardTitle className="text-base text-status-red">Scan Error</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="text-sm bg-muted p-4 rounded-lg overflow-auto">
              {scan.errorMessage}
            </pre>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

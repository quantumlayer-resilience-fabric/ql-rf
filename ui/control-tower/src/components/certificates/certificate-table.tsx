"use client";

import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { type Certificate, type CertificateStatus, type CertificatePlatform } from "@/lib/api";
import {
  Key,
  Cloud,
  Server,
  CheckCircle,
  XCircle,
} from "lucide-react";

// Platform icon mapping
export const platformIcons: Record<CertificatePlatform, React.ReactNode> = {
  aws: <Cloud className="h-4 w-4 text-orange-400" />,
  azure: <Cloud className="h-4 w-4 text-blue-400" />,
  gcp: <Cloud className="h-4 w-4 text-red-400" />,
  k8s: <Server className="h-4 w-4 text-blue-500" />,
  vsphere: <Server className="h-4 w-4 text-green-500" />,
};

// Status badge configuration
export const statusConfig: Record<
  CertificateStatus,
  { variant: "default" | "secondary" | "destructive" | "outline"; label: string }
> = {
  active: { variant: "default", label: "Active" },
  expiring_soon: { variant: "secondary", label: "Expiring Soon" },
  expired: { variant: "destructive", label: "Expired" },
  revoked: { variant: "destructive", label: "Revoked" },
  pending_validation: { variant: "outline", label: "Pending" },
};

interface CertificateRowProps {
  certificate: Certificate;
  onRowClick?: (certificate: Certificate) => void;
}

export function CertificateRow({ certificate, onRowClick }: CertificateRowProps) {
  const statusCfg = statusConfig[certificate.status];
  const daysUntilExpiry = certificate.daysUntilExpiry;

  return (
    <TableRow
      className={onRowClick ? "cursor-pointer hover:bg-muted/50" : undefined}
      onClick={() => onRowClick?.(certificate)}
    >
      <TableCell>
        <div className="flex items-center gap-2">
          <Key className="h-4 w-4 text-muted-foreground" />
          <div>
            <div className="font-medium">{certificate.commonName}</div>
            {certificate.subjectAltNames && certificate.subjectAltNames.length > 0 && (
              <div className="text-xs text-muted-foreground">
                +{certificate.subjectAltNames.length} SANs
              </div>
            )}
          </div>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          {platformIcons[certificate.platform]}
          <span className="capitalize">{certificate.platform}</span>
        </div>
      </TableCell>
      <TableCell>
        <span className="text-sm text-muted-foreground capitalize">
          {certificate.source.replace(/_/g, " ")}
        </span>
      </TableCell>
      <TableCell>
        <ExpiryDisplay daysUntilExpiry={daysUntilExpiry} />
      </TableCell>
      <TableCell>
        <Badge variant={statusCfg.variant}>{statusCfg.label}</Badge>
      </TableCell>
      <TableCell>
        {certificate.autoRenew ? (
          <CheckCircle className="h-4 w-4 text-status-green" />
        ) : (
          <XCircle className="h-4 w-4 text-muted-foreground" />
        )}
      </TableCell>
    </TableRow>
  );
}

interface ExpiryDisplayProps {
  daysUntilExpiry: number;
  showLabel?: boolean;
}

export function ExpiryDisplay({ daysUntilExpiry, showLabel = false }: ExpiryDisplayProps) {
  const colorClass = daysUntilExpiry < 0
    ? "text-status-red"
    : daysUntilExpiry < 7
      ? "text-status-amber"
      : "";

  if (daysUntilExpiry < 0) {
    return (
      <div className={colorClass}>
        <span className="font-medium">
          Expired {Math.abs(daysUntilExpiry)} days ago
        </span>
      </div>
    );
  }

  return (
    <div className={colorClass}>
      <span>{daysUntilExpiry} days</span>
      {showLabel && <span className="text-muted-foreground"> until expiry</span>}
    </div>
  );
}

interface CertificateTableProps {
  certificates: Certificate[];
  onRowClick?: (certificate: Certificate) => void;
  showSource?: boolean;
}

export function CertificateTable({
  certificates,
  onRowClick,
  showSource = true,
}: CertificateTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Common Name</TableHead>
          <TableHead>Platform</TableHead>
          {showSource && <TableHead>Source</TableHead>}
          <TableHead>Expires</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Auto-Renew</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {certificates.map((cert) => (
          <CertificateRow
            key={cert.id}
            certificate={cert}
            onRowClick={onRowClick}
          />
        ))}
      </TableBody>
    </Table>
  );
}

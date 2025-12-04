"use client";

import { jsPDF } from "jspdf";
import autoTable from "jspdf-autotable";

interface ComplianceReportData {
  overallScore: number;
  cisCompliance: number;
  slsaLevel: number;
  sigstoreVerified: number;
  lastAuditAt: string;
  frameworks: Array<{
    id: string;
    name: string;
    description: string;
    score: number;
    status: string;
    passingControls: number;
    totalControls: number;
    level?: number;
  }>;
  failingControls: Array<{
    id: string;
    framework: string;
    title: string;
    severity: string;
    recommendation: string;
    affectedAssets: number;
  }>;
  imageCompliance: Array<{
    familyId: string;
    familyName: string;
    version: string;
    cis: boolean;
    slsaLevel: number;
    cosignSigned: boolean;
    lastScanAt: string;
    issueCount: number;
  }>;
}

export function exportComplianceReport(data: ComplianceReportData): void {
  const doc = new jsPDF();
  const pageWidth = doc.internal.pageSize.getWidth();
  const margin = 20;
  let yPosition = 20;

  // Title
  doc.setFontSize(24);
  doc.setFont("helvetica", "bold");
  doc.text("Compliance Report", margin, yPosition);
  yPosition += 10;

  // Subtitle with date
  doc.setFontSize(10);
  doc.setFont("helvetica", "normal");
  doc.setTextColor(128, 128, 128);
  doc.text(`Generated: ${new Date().toLocaleString()}`, margin, yPosition);
  doc.text(`Last Audit: ${new Date(data.lastAuditAt).toLocaleString()}`, pageWidth - margin - 60, yPosition);
  yPosition += 15;

  // Reset text color
  doc.setTextColor(0, 0, 0);

  // Executive Summary Section
  doc.setFontSize(14);
  doc.setFont("helvetica", "bold");
  doc.text("Executive Summary", margin, yPosition);
  yPosition += 10;

  // Metrics Grid
  const metrics = [
    ["Overall Score", `${data.overallScore.toFixed(1)}%`],
    ["CIS Compliance", `${data.cisCompliance.toFixed(1)}%`],
    ["SLSA Level", `Level ${data.slsaLevel}`],
    ["Sigstore Verified", `${data.sigstoreVerified.toFixed(1)}%`],
  ];

  doc.setFontSize(10);
  doc.setFont("helvetica", "normal");
  const colWidth = (pageWidth - 2 * margin) / 4;
  metrics.forEach((metric, index) => {
    const x = margin + (index * colWidth);
    doc.setFont("helvetica", "normal");
    doc.setTextColor(128, 128, 128);
    doc.text(metric[0], x, yPosition);
    doc.setFont("helvetica", "bold");
    doc.setTextColor(0, 0, 0);
    doc.text(metric[1], x, yPosition + 5);
  });
  yPosition += 20;

  // Frameworks Section
  if (data.frameworks.length > 0) {
    doc.setFontSize(14);
    doc.setFont("helvetica", "bold");
    doc.text("Framework Compliance", margin, yPosition);
    yPosition += 5;

    const frameworkTableData = data.frameworks.map(fw => [
      fw.name + (fw.level ? ` (L${fw.level})` : ""),
      `${fw.score.toFixed(1)}%`,
      `${fw.passingControls}/${fw.totalControls}`,
      fw.status.toUpperCase(),
    ]);

    autoTable(doc, {
      startY: yPosition,
      head: [["Framework", "Score", "Controls", "Status"]],
      body: frameworkTableData,
      margin: { left: margin, right: margin },
      headStyles: {
        fillColor: [30, 41, 59],
        textColor: [255, 255, 255],
        fontStyle: "bold",
      },
      styles: {
        fontSize: 9,
        cellPadding: 3,
      },
      alternateRowStyles: {
        fillColor: [248, 250, 252],
      },
      columnStyles: {
        0: { cellWidth: 70 },
        1: { cellWidth: 30, halign: "center" },
        2: { cellWidth: 40, halign: "center" },
        3: { cellWidth: 30, halign: "center" },
      },
    });

    yPosition = (doc as jsPDF & { lastAutoTable: { finalY: number } }).lastAutoTable.finalY + 15;
  }

  // Failing Controls Section
  if (data.failingControls.length > 0) {
    // Check if we need a new page
    if (yPosition > 200) {
      doc.addPage();
      yPosition = 20;
    }

    doc.setFontSize(14);
    doc.setFont("helvetica", "bold");
    doc.text("Failing Controls", margin, yPosition);
    yPosition += 5;

    const controlTableData = data.failingControls.map(ctrl => [
      ctrl.id,
      ctrl.framework,
      ctrl.title.substring(0, 40) + (ctrl.title.length > 40 ? "..." : ""),
      ctrl.severity.toUpperCase(),
      ctrl.affectedAssets.toString(),
    ]);

    autoTable(doc, {
      startY: yPosition,
      head: [["Control ID", "Framework", "Title", "Severity", "Assets"]],
      body: controlTableData,
      margin: { left: margin, right: margin },
      headStyles: {
        fillColor: [30, 41, 59],
        textColor: [255, 255, 255],
        fontStyle: "bold",
      },
      styles: {
        fontSize: 8,
        cellPadding: 2,
      },
      alternateRowStyles: {
        fillColor: [248, 250, 252],
      },
      columnStyles: {
        0: { cellWidth: 30 },
        1: { cellWidth: 25 },
        2: { cellWidth: 70 },
        3: { cellWidth: 25, halign: "center" },
        4: { cellWidth: 20, halign: "center" },
      },
      didParseCell: function(data) {
        // Color severity cells
        if (data.column.index === 3 && data.section === "body") {
          const severity = data.cell.raw as string;
          if (severity === "HIGH") {
            data.cell.styles.textColor = [220, 38, 38];
            data.cell.styles.fontStyle = "bold";
          } else if (severity === "MEDIUM") {
            data.cell.styles.textColor = [217, 119, 6];
          }
        }
      },
    });

    yPosition = (doc as jsPDF & { lastAutoTable: { finalY: number } }).lastAutoTable.finalY + 15;
  }

  // Image Compliance Section
  if (data.imageCompliance.length > 0) {
    // Check if we need a new page
    if (yPosition > 200) {
      doc.addPage();
      yPosition = 20;
    }

    doc.setFontSize(14);
    doc.setFont("helvetica", "bold");
    doc.text("Image Compliance", margin, yPosition);
    yPosition += 5;

    const imageTableData = data.imageCompliance.map(img => [
      img.familyName,
      `v${img.version}`,
      img.cis ? "PASS" : "FAIL",
      `Level ${img.slsaLevel}`,
      img.cosignSigned ? "Yes" : "No",
      img.issueCount === 0 ? "Clean" : `${img.issueCount} issues`,
    ]);

    autoTable(doc, {
      startY: yPosition,
      head: [["Image", "Version", "CIS", "SLSA", "Signed", "Status"]],
      body: imageTableData,
      margin: { left: margin, right: margin },
      headStyles: {
        fillColor: [30, 41, 59],
        textColor: [255, 255, 255],
        fontStyle: "bold",
      },
      styles: {
        fontSize: 8,
        cellPadding: 2,
      },
      alternateRowStyles: {
        fillColor: [248, 250, 252],
      },
      columnStyles: {
        0: { cellWidth: 45 },
        1: { cellWidth: 25, halign: "center" },
        2: { cellWidth: 20, halign: "center" },
        3: { cellWidth: 25, halign: "center" },
        4: { cellWidth: 20, halign: "center" },
        5: { cellWidth: 30, halign: "center" },
      },
      didParseCell: function(data) {
        // Color CIS cells
        if (data.column.index === 2 && data.section === "body") {
          const value = data.cell.raw as string;
          if (value === "PASS") {
            data.cell.styles.textColor = [34, 197, 94];
          } else {
            data.cell.styles.textColor = [220, 38, 38];
          }
        }
        // Color status cells
        if (data.column.index === 5 && data.section === "body") {
          const value = data.cell.raw as string;
          if (value === "Clean") {
            data.cell.styles.textColor = [34, 197, 94];
          } else {
            data.cell.styles.textColor = [217, 119, 6];
          }
        }
      },
    });
  }

  // Footer on each page
  const pageCount = doc.getNumberOfPages();
  for (let i = 1; i <= pageCount; i++) {
    doc.setPage(i);
    doc.setFontSize(8);
    doc.setTextColor(128, 128, 128);
    doc.text(
      `Page ${i} of ${pageCount} | QL-RF Compliance Report`,
      pageWidth / 2,
      doc.internal.pageSize.getHeight() - 10,
      { align: "center" }
    );
  }

  // Download
  const fileName = `compliance-report-${new Date().toISOString().split("T")[0]}.pdf`;
  doc.save(fileName);
}

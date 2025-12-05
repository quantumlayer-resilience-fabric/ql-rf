import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Overview",
  description: "Real-time visibility into your infrastructure health, drift score, compliance status, and DR readiness across all cloud platforms.",
};

export default function OverviewLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}

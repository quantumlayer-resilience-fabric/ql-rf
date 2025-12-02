"use client";

import { DashboardSidebar, DashboardHeader } from "@/components/layout";
import { AuthProvider } from "@/providers/auth-provider";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AuthProvider>
      <div className="min-h-screen bg-background">
        <DashboardSidebar />
        <div className="transition-all duration-300 md:pl-64">
          <DashboardHeader />
          <main className="p-4 pt-16 md:p-6 md:pt-6">{children}</main>
        </div>
      </div>
    </AuthProvider>
  );
}

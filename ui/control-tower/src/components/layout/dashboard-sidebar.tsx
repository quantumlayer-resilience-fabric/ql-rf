"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { LogoIcon } from "@/components/brand";
import { Sheet, SheetContent } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  LayoutDashboard,
  Image,
  TrendingDown,
  MapPin,
  Shield,
  RefreshCw,
  Sparkles,
  Settings,
  ChevronLeft,
  ChevronRight,
  Menu,
  AlertTriangle,
  Package,
  DollarSign,
  ClipboardCheck,
  KeyRound,
  ShieldAlert,
  Rocket,
  Cable,
} from "lucide-react";
import { useState } from "react";

interface NavItem {
  label: string;
  href: string;
  icon: typeof LayoutDashboard;
  badge?: string;
  badgeVariant?: "default" | "warning" | "critical";
}

const mainNavItems: NavItem[] = [
  { label: "Overview", href: "/overview", icon: LayoutDashboard },
  { label: "Risk", href: "/risk", icon: AlertTriangle },
  { label: "Vulnerabilities", href: "/vulnerabilities", icon: ShieldAlert },
  { label: "Patch Campaigns", href: "/patch-campaigns", icon: Rocket },
  { label: "Images", href: "/images", icon: Image },
  { label: "SBOM", href: "/sbom", icon: Package },
  { label: "Certificates", href: "/certificates", icon: KeyRound },
  { label: "Drift", href: "/drift", icon: TrendingDown, badge: "3", badgeVariant: "warning" },
  { label: "Compliance", href: "/compliance", icon: Shield },
  { label: "InSpec", href: "/inspec", icon: ClipboardCheck },
  { label: "Resilience", href: "/resilience", icon: RefreshCw },
  { label: "Costs", href: "/costs", icon: DollarSign },
  { label: "Sites", href: "/sites", icon: MapPin },
  { label: "AI Copilot", href: "/ai", icon: Sparkles },
];

const bottomNavItems: NavItem[] = [
  { label: "Connectors", href: "/connectors", icon: Cable },
  { label: "Settings", href: "/settings", icon: Settings },
];

const badgeStyles = {
  default: "bg-brand-accent text-white",
  warning: "bg-status-amber text-white animate-pulse-status",
  critical: "bg-status-red text-white animate-glow-critical",
};

function NavItemComponent({
  item,
  isActive,
  collapsed,
  onNavClick,
}: {
  item: NavItem;
  isActive: boolean;
  collapsed: boolean;
  onNavClick?: () => void;
}) {
  const Icon = item.icon;

  const linkContent = (
    <Link
      href={item.href}
      onClick={onNavClick}
      className={cn(
        "group relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200",
        isActive
          ? "bg-sidebar-accent text-sidebar-accent-foreground shadow-sm"
          : "text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground",
        collapsed && "justify-center px-2"
      )}
    >
      {/* Active indicator */}
      {isActive && (
        <span className="absolute left-0 top-1/2 h-6 w-1 -translate-y-1/2 rounded-r-full bg-brand-accent" />
      )}

      <Icon
        className={cn(
          "h-5 w-5 shrink-0 transition-transform duration-200",
          !isActive && "group-hover:scale-110"
        )}
      />

      {!collapsed && (
        <>
          <span className="flex-1">{item.label}</span>
          {item.badge && (
            <span
              className={cn(
                "flex h-5 min-w-5 items-center justify-center rounded-full px-1.5 text-xs font-medium",
                badgeStyles[item.badgeVariant || "default"]
              )}
            >
              {item.badge}
            </span>
          )}
        </>
      )}

      {/* Collapsed badge */}
      {collapsed && item.badge && (
        <span
          className={cn(
            "absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[10px] font-medium",
            badgeStyles[item.badgeVariant || "default"]
          )}
        >
          {item.badge}
        </span>
      )}
    </Link>
  );

  // Wrap in tooltip when collapsed
  if (collapsed) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>{linkContent}</TooltipTrigger>
        <TooltipContent side="right" sideOffset={8}>
          <span className="font-medium">{item.label}</span>
          {item.badge && (
            <span className="ml-2 text-xs text-muted-foreground">
              ({item.badge})
            </span>
          )}
        </TooltipContent>
      </Tooltip>
    );
  }

  return linkContent;
}

function SidebarContent({
  collapsed,
  setCollapsed,
  onNavClick,
}: {
  collapsed: boolean;
  setCollapsed: (v: boolean) => void;
  onNavClick?: () => void;
}) {
  const pathname = usePathname();

  return (
    <TooltipProvider delayDuration={0}>
      {/* Logo */}
      <div
        className={cn(
          "flex items-center border-b border-sidebar-border transition-all duration-300",
          collapsed ? "justify-center px-2 h-16" : "justify-between px-4 h-20"
        )}
      >
        <Link
          href="/overview"
          className="flex items-center gap-3 transition-opacity hover:opacity-80"
          onClick={onNavClick}
        >
          <LogoIcon size={collapsed ? "sm" : "md"} />
          {!collapsed && (
            <div className="flex flex-col">
              <span
                className="font-bold gradient-quantum-text tracking-tight text-lg"
                style={{ fontFamily: "var(--font-display)" }}
              >
                QuantumLayer
              </span>
              <span className="text-[10px] font-medium text-muted-foreground tracking-widest uppercase">
                Resilience Fabric
              </span>
            </div>
          )}
        </Link>
      </div>

      {/* Main Navigation */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        <ul className="space-y-1">
          {mainNavItems.map((item) => {
            const isActive =
              pathname === item.href || pathname.startsWith(item.href + "/");

            return (
              <li key={item.href}>
                <NavItemComponent
                  item={item}
                  isActive={isActive}
                  collapsed={collapsed}
                  onNavClick={onNavClick}
                />
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Bottom Navigation */}
      <div className="border-t border-sidebar-border px-3 py-4">
        <ul className="space-y-1">
          {bottomNavItems.map((item) => {
            const isActive = pathname === item.href;

            return (
              <li key={item.href}>
                <NavItemComponent
                  item={item}
                  isActive={isActive}
                  collapsed={collapsed}
                  onNavClick={onNavClick}
                />
              </li>
            );
          })}
        </ul>

        {/* Collapse Toggle - Hidden on mobile */}
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              onClick={() => setCollapsed(!collapsed)}
              aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
              aria-expanded={!collapsed}
              className={cn(
                "mt-4 hidden w-full items-center justify-center rounded-lg py-2 text-sidebar-foreground/50 transition-all duration-200 hover:bg-sidebar-accent hover:text-sidebar-foreground md:flex",
                collapsed ? "hover:scale-110" : ""
              )}
            >
              {collapsed ? (
                <ChevronRight className="h-5 w-5" aria-hidden="true" />
              ) : (
                <ChevronLeft className="h-5 w-5" aria-hidden="true" />
              )}
              <span className="sr-only">{collapsed ? "Expand sidebar" : "Collapse sidebar"}</span>
            </button>
          </TooltipTrigger>
          {collapsed && (
            <TooltipContent side="right" sideOffset={8}>
              Expand sidebar
            </TooltipContent>
          )}
        </Tooltip>
      </div>
    </TooltipProvider>
  );
}

export function DashboardSidebar() {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <>
      {/* Mobile Menu Button */}
      <Button
        variant="ghost"
        size="icon"
        className="fixed left-4 top-4 z-50 md:hidden shadow-md bg-background/80 backdrop-blur-sm"
        onClick={() => setMobileOpen(true)}
        aria-label="Open navigation menu"
        aria-expanded={mobileOpen}
        aria-controls="mobile-sidebar"
      >
        <Menu className="h-6 w-6" aria-hidden="true" />
        <span className="sr-only">Open navigation menu</span>
      </Button>

      {/* Mobile Sheet */}
      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetContent
          side="left"
          className="w-64 p-0 bg-sidebar border-sidebar-border"
        >
          <div className="flex h-full flex-col">
            <SidebarContent
              collapsed={false}
              setCollapsed={setCollapsed}
              onNavClick={() => setMobileOpen(false)}
            />
          </div>
        </SheetContent>
      </Sheet>

      {/* Desktop Sidebar */}
      <aside
        className={cn(
          "fixed left-0 top-0 z-40 hidden h-screen flex-col border-r border-sidebar-border bg-sidebar transition-all duration-300 ease-[var(--ease-out-expo)] md:flex",
          collapsed ? "w-16" : "w-64"
        )}
      >
        <SidebarContent collapsed={collapsed} setCollapsed={setCollapsed} />
      </aside>
    </>
  );
}

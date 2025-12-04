"use client";

import { useState } from "react";
import Link from "next/link";
import dynamic from "next/dynamic";
import { cn } from "@/lib/utils";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Search, Bell, Sun, Moon, HelpCircle, Sparkles } from "lucide-react";

// Dev bypass - set NEXT_PUBLIC_DEV_AUTH_BYPASS=true to skip Clerk entirely
const devAuthBypass = process.env.NEXT_PUBLIC_DEV_AUTH_BYPASS === "true";

// Check if Clerk is configured (and dev bypass is not enabled)
const hasClerkKey =
  !devAuthBypass &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY &&
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.startsWith("pk_") &&
  !process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY.includes("xxxxx");

// Only load UserButton if Clerk is configured
const UserButton = hasClerkKey
  ? dynamic(() => import("@clerk/nextjs").then((mod) => mod.UserButton), {
      loading: () => (
        <Avatar className="h-8 w-8">
          <AvatarFallback>U</AvatarFallback>
        </Avatar>
      ),
    })
  : null;

// Fallback avatar for dev mode
function DevModeAvatar() {
  return (
    <Avatar className="h-8 w-8 cursor-pointer ring-2 ring-transparent transition-all hover:ring-brand-accent/30">
      <AvatarFallback className="bg-gradient-to-br from-brand-accent to-primary text-white font-medium">
        D
      </AvatarFallback>
    </Avatar>
  );
}

const notificationTypeStyles = {
  warning: "bg-status-amber",
  critical: "bg-status-red",
  success: "bg-status-green",
  info: "bg-brand-accent",
};

interface DashboardHeaderProps {
  className?: string;
}

export function DashboardHeader({ className }: DashboardHeaderProps) {
  const [theme, setTheme] = useState<"light" | "dark">("light");

  const toggleTheme = () => {
    const newTheme = theme === "light" ? "dark" : "light";
    setTheme(newTheme);
    document.documentElement.classList.toggle("dark", newTheme === "dark");
  };

  const notifications = [
    {
      title: "Drift detected in ap-south-1",
      description: "Coverage dropped to 62.4%",
      time: "5m ago",
      type: "warning" as const,
    },
    {
      title: "New image version available",
      description: "ql-base-linux v1.6.5 is ready for promotion",
      time: "1h ago",
      type: "info" as const,
    },
    {
      title: "DR drill completed",
      description: "All RTO/RPO targets met for dc-london",
      time: "2h ago",
      type: "success" as const,
    },
  ];

  const unreadCount = notifications.length;

  return (
    <TooltipProvider>
      <header
        className={cn(
          "sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-background/95 px-6 backdrop-blur-md supports-[backdrop-filter]:bg-background/80",
          className
        )}
      >
        {/* Search */}
        <div className="flex items-center gap-4">
          <div className="relative w-64 lg:w-96 group">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition-colors group-focus-within:text-brand-accent" />
            <Input
              type="search"
              placeholder="Search assets, images, sites..."
              className="h-9 pl-9 pr-12 transition-shadow focus:shadow-[0_0_0_2px_rgba(37,99,235,0.1)] focus:border-brand-accent/50"
            />
            <kbd className="absolute right-3 top-1/2 -translate-y-1/2 rounded border border-border bg-muted px-1.5 text-[10px] font-medium text-muted-foreground">
              ⌘K
            </kbd>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1.5">
          {/* AI Quick Action */}
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-9 w-9 text-muted-foreground hover:text-brand-accent"
                asChild
              >
                <Link href="/ai">
                  <Sparkles className="h-4 w-4" />
                </Link>
              </Button>
            </TooltipTrigger>
            <TooltipContent>AI Copilot</TooltipContent>
          </Tooltip>

          {/* Theme Toggle */}
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                onClick={toggleTheme}
                className="h-9 w-9 text-muted-foreground hover:text-foreground"
              >
                {theme === "light" ? (
                  <Moon className="h-4 w-4" />
                ) : (
                  <Sun className="h-4 w-4" />
                )}
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              {theme === "light" ? "Dark mode" : "Light mode"}
            </TooltipContent>
          </Tooltip>

          {/* Help */}
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-9 w-9 text-muted-foreground hover:text-foreground"
                asChild
              >
                <Link href="/docs">
                  <HelpCircle className="h-4 w-4" />
                </Link>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Help & Documentation</TooltipContent>
          </Tooltip>

          {/* Notifications */}
          <DropdownMenu>
            <Tooltip>
              <TooltipTrigger asChild>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="relative h-9 w-9 text-muted-foreground hover:text-foreground"
                  >
                    <Bell className="h-4 w-4" />
                    {unreadCount > 0 && (
                      <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-status-amber px-1 text-[10px] font-medium text-white animate-in zoom-in-50 duration-200">
                        {unreadCount}
                      </span>
                    )}
                  </Button>
                </DropdownMenuTrigger>
              </TooltipTrigger>
              <TooltipContent>Notifications</TooltipContent>
            </Tooltip>
            <DropdownMenuContent align="end" className="w-80">
              <DropdownMenuLabel className="flex items-center justify-between">
                <span>Notifications</span>
                {unreadCount > 0 && (
                  <span className="text-xs font-normal text-muted-foreground">
                    {unreadCount} unread
                  </span>
                )}
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <div className="max-h-72 overflow-y-auto">
                {notifications.map((notification, i) => (
                  <DropdownMenuItem
                    key={i}
                    className="flex flex-col items-start gap-1.5 py-3 cursor-pointer"
                  >
                    <div className="flex w-full items-center gap-2">
                      <span
                        className={cn(
                          "h-2 w-2 rounded-full shrink-0",
                          notificationTypeStyles[notification.type]
                        )}
                      />
                      <span className="font-medium flex-1 truncate">
                        {notification.title}
                      </span>
                      <span className="text-[10px] text-muted-foreground shrink-0">
                        {notification.time}
                      </span>
                    </div>
                    <span className="text-xs text-muted-foreground pl-4">
                      {notification.description}
                    </span>
                  </DropdownMenuItem>
                ))}
              </div>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="justify-center text-sm font-medium text-brand-accent hover:text-brand-accent">
                View all notifications
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          {/* Divider */}
          <div className="mx-2 h-6 w-px bg-border" />

          {/* User Menu */}
          {UserButton ? (
            <UserButton
              afterSignOutUrl="/"
              appearance={{
                elements: {
                  avatarBox: "h-8 w-8 ring-2 ring-transparent hover:ring-brand-accent/30 transition-all",
                },
              }}
            />
          ) : (
            <DevModeAvatar />
          )}
        </div>
      </header>
    </TooltipProvider>
  );
}

import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const cardVariants = cva(
  "bg-card text-card-foreground flex flex-col gap-6 rounded-xl border py-6",
  {
    variants: {
      variant: {
        default: "shadow-[var(--shadow-card)]",
        elevated: "shadow-[var(--shadow-md)] hover:shadow-[var(--shadow-lg)] transition-shadow duration-200",
        interactive: "shadow-[var(--shadow-card)] card-interactive",
        glass: "bg-card/80 backdrop-blur-sm shadow-[var(--shadow-sm)]",
        ghost: "border-transparent shadow-none bg-transparent",
        brand: "border-brand-accent/20 bg-gradient-to-br from-brand-accent/5 to-transparent shadow-[var(--shadow-card)]",
        ai: "border-[var(--ai-start)]/20 bg-gradient-to-br from-[var(--ai-start)]/5 to-[var(--ai-end)]/5 shadow-[var(--shadow-card)]",
        success: "border-status-green/20 bg-status-green-bg shadow-[var(--shadow-card)]",
        warning: "border-status-amber/20 bg-status-amber-bg shadow-[var(--shadow-card)]",
        critical: "border-status-red/20 bg-status-red-bg shadow-[var(--shadow-card)]",
      },
      hover: {
        none: "",
        lift: "card-hover",
        glow: "hover:shadow-[var(--shadow-glow-brand)] transition-shadow duration-300",
        scale: "hover:scale-[1.02] transition-transform duration-200",
      },
    },
    defaultVariants: {
      variant: "default",
      hover: "none",
    },
  }
)

export interface CardProps
  extends React.ComponentProps<"div">,
    VariantProps<typeof cardVariants> {
  accentColor?: "brand" | "ai" | "success" | "warning" | "critical"
}

function Card({ className, variant, hover, accentColor, ...props }: CardProps) {
  const accentStyles = accentColor
    ? {
        brand: "border-t-4 border-t-brand-accent",
        ai: "border-t-4 border-t-[var(--ai-start)]",
        success: "border-t-4 border-t-status-green",
        warning: "border-t-4 border-t-status-amber",
        critical: "border-t-4 border-t-status-red",
      }[accentColor]
    : ""

  return (
    <div
      data-slot="card"
      className={cn(cardVariants({ variant, hover }), accentStyles, className)}
      {...props}
    />
  )
}

function CardHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-header"
      className={cn(
        "@container/card-header grid auto-rows-min grid-rows-[auto_auto] items-start gap-2 px-6 has-data-[slot=card-action]:grid-cols-[1fr_auto] [.border-b]:pb-6",
        className
      )}
      {...props}
    />
  )
}

function CardTitle({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-title"
      className={cn("leading-none font-semibold tracking-tight", className)}
      style={{ fontFamily: 'var(--font-display)' }}
      {...props}
    />
  )
}

function CardDescription({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-description"
      className={cn("text-muted-foreground text-sm", className)}
      {...props}
    />
  )
}

function CardAction({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-action"
      className={cn(
        "col-start-2 row-span-2 row-start-1 self-start justify-self-end",
        className
      )}
      {...props}
    />
  )
}

function CardContent({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-content"
      className={cn("px-6", className)}
      {...props}
    />
  )
}

function CardFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-footer"
      className={cn("flex items-center px-6 [.border-t]:pt-6", className)}
      {...props}
    />
  )
}

export {
  Card,
  CardHeader,
  CardFooter,
  CardTitle,
  CardAction,
  CardDescription,
  CardContent,
  cardVariants,
}

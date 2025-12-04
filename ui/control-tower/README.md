# QL-RF Control Tower

The Control Tower is the web-based dashboard for QL-RF (Resilience Framework) - an AI-powered infrastructure resilience and compliance platform.

## Overview

Control Tower provides:
- **Real-time Infrastructure Monitoring**: Live view of assets across AWS, Azure, GCP, and vSphere
- **Drift Detection & Analysis**: Visual drift indicators with remediation guidance
- **Golden Image Management**: Catalog, versioning, and compliance tracking
- **Compliance Dashboard**: Control status, evidence generation, audit trails
- **DR Readiness**: Disaster recovery site status and RTO/RPO metrics
- **AI Copilot**: Natural language interface for infrastructure operations

## Tech Stack

- **Framework**: Next.js 16 (App Router)
- **Language**: TypeScript
- **Styling**: Tailwind CSS + shadcn/ui
- **State Management**: React Query + Zustand
- **Real-time**: Socket.IO
- **Charts**: Recharts
- **Icons**: Lucide React

## Getting Started

### Prerequisites

- Node.js 18+
- npm or pnpm
- Backend API running (see `/services/api`)

### Installation

```bash
# Install dependencies
npm install

# Create environment file
cp .env.example .env.local

# Start development server
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) to view the dashboard.

### Environment Variables

```bash
# .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080
NEXT_PUBLIC_AI_ENABLED=true
```

## Project Structure

```
src/
├── app/                    # Next.js App Router
│   ├── (dashboard)/        # Protected dashboard routes
│   │   ├── overview/       # Overview dashboard
│   │   ├── drift/          # Drift analysis
│   │   ├── images/         # Golden images
│   │   ├── compliance/     # Compliance dashboard
│   │   ├── resilience/     # DR readiness
│   │   ├── sites/          # Multi-site management
│   │   ├── alerts/         # Alert management
│   │   ├── settings/       # User/org settings
│   │   └── ai/             # AI Copilot
│   ├── (marketing)/        # Public marketing pages
│   └── api/                # API routes (BFF)
├── components/
│   ├── ui/                 # shadcn/ui components
│   ├── dashboard/          # Dashboard-specific components
│   ├── charts/             # Chart components
│   └── ai/                 # AI Copilot components
├── lib/                    # Utilities and helpers
├── hooks/                  # Custom React hooks
└── types/                  # TypeScript type definitions
```

## Key Pages

| Route | Description |
|-------|-------------|
| `/overview` | Fleet-wide health summary |
| `/drift` | Drift analysis with filters |
| `/images` | Golden image catalog |
| `/compliance` | Compliance controls & evidence |
| `/resilience` | DR readiness dashboard |
| `/sites` | Multi-site topology |
| `/alerts` | Alert management |
| `/ai` | AI Copilot chat interface |

## Design System

The Control Tower follows the "Command Center" design philosophy documented in `/docs/FRONTEND_DESIGN_SYSTEM.md`:

- **Dark theme** with high contrast for NOC environments
- **Status colors**: Green (healthy), Amber (warning), Red (critical)
- **Information density**: Maximized for ops workflows
- **Real-time updates**: Socket.IO for live status

### Color Tokens

```css
--color-healthy: #22c55e;    /* Green */
--color-warning: #f59e0b;    /* Amber */
--color-critical: #ef4444;   /* Red */
--color-neutral: #6b7280;    /* Gray */
```

## AI Copilot Integration

The AI Copilot (`/ai` route) provides natural language control:

```
User: "Show me drifted servers in production"
AI: [Displays filtered drift view]

User: "Fix drift on prod web servers"
AI: [Generates remediation plan → HITL approval → Execute]
```

See [ADR-007: LLM-First Orchestration](../../docs/adr/ADR-007-llm-first-orchestration.md) for architecture details.

## Development

### Commands

```bash
# Development
npm run dev         # Start dev server (port 3000)
npm run build       # Production build
npm run start       # Start production server
npm run lint        # Run ESLint
npm run type-check  # TypeScript check

# Testing
npm run test        # Run tests
npm run test:e2e    # E2E tests (Playwright)
```

### Adding Components

We use shadcn/ui for base components:

```bash
# Add a new component
npx shadcn@latest add button
npx shadcn@latest add card
npx shadcn@latest add dialog
```

### Code Style

- Use TypeScript strictly (no `any`)
- Prefer server components where possible
- Co-locate components with their pages
- Use React Query for server state
- Use Zustand for client state

## API Integration

The Control Tower communicates with the backend API:

```typescript
// lib/api.ts
const api = {
  assets: {
    list: () => fetch('/api/v1/assets'),
    drift: (id) => fetch(`/api/v1/assets/${id}/drift`),
  },
  images: {
    list: () => fetch('/api/v1/images'),
    promote: (id) => fetch(`/api/v1/images/${id}/promote`, { method: 'POST' }),
  },
  ai: {
    execute: (intent) => fetch('/api/v1/ai/execute', {
      method: 'POST',
      body: JSON.stringify({ user_intent: intent }),
    }),
    approve: (taskId) => fetch(`/api/v1/ai/plans/${taskId}/approve`, {
      method: 'POST',
    }),
  },
};
```

## Real-time Updates

Socket.IO provides live updates:

```typescript
// hooks/useRealtime.ts
const socket = io(process.env.NEXT_PUBLIC_WS_URL);

socket.on('asset:drift_detected', (data) => {
  // Update UI with new drift info
});

socket.on('ai:task_completed', (data) => {
  // Notify user of AI task completion
});
```

## Related Documentation

- [Frontend Design System](../../docs/FRONTEND_DESIGN_SYSTEM.md)
- [Product Requirements](../../docs/PRD.md)
- [ADR-007: LLM-First Orchestration](../../docs/adr/ADR-007-llm-first-orchestration.md)
- [AI Schemas](../../schemas/ai/README.md)

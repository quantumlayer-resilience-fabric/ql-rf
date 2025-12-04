# QuantumLayer Resilience Fabric
## Frontend Design System & Architecture

---

## 1. Design Philosophy

### 1.1 Aesthetic Direction: "Command Center"

The Control Tower embraces a **Mission Control** aesthetic—dense, information-rich, but never overwhelming. Think aerospace command centers, financial trading floors, and network operations centers.

**Core Principles:**
- **Data Density with Clarity**: Show more, explain less—trained operators need information, not hand-holding
- **Ambient Awareness**: Background signals (color, motion) communicate status before reading
- **Drill-Down Architecture**: Overview → Region → Site → Asset (progressive disclosure)
- **Dark-First**: Reduces eye strain for 24/7 operations; colors pop for alerts

### 1.2 Design Tokens

```css
/* ========================================
   QUANTUMLAYER RESILIENCE FABRIC
   Design Tokens v1.0
   ======================================== */

:root {
  /* === COLORS: DARK THEME === */
  
  /* Backgrounds */
  --rf-bg-void: #0a0a0f;          /* Deepest background */
  --rf-bg-surface: #12121a;        /* Card backgrounds */
  --rf-bg-elevated: #1a1a24;       /* Elevated elements */
  --rf-bg-hover: #22222e;          /* Hover states */
  
  /* Text */
  --rf-text-primary: #f0f0f5;      /* Primary text */
  --rf-text-secondary: #8888a0;    /* Secondary/muted */
  --rf-text-tertiary: #5555670;    /* Disabled/hints */
  
  /* Status Colors - RAG */
  --rf-status-green: #00d4aa;      /* Compliant/Healthy */
  --rf-status-green-bg: #00d4aa15; /* Green background */
  --rf-status-amber: #ffaa00;      /* Warning/Drift */
  --rf-status-amber-bg: #ffaa0015;
  --rf-status-red: #ff4466;        /* Critical/Failed */
  --rf-status-red-bg: #ff446615;
  
  /* Accent */
  --rf-accent-primary: #6366f1;    /* Primary actions */
  --rf-accent-secondary: #818cf8;  /* Secondary */
  --rf-accent-glow: #6366f140;     /* Glow effects */
  
  /* Platform Colors */
  --rf-aws: #ff9900;
  --rf-azure: #0078d4;
  --rf-gcp: #4285f4;
  --rf-vsphere: #6d9e37;
  --rf-k8s: #326ce5;
  --rf-baremetal: #8b8b8b;
  
  /* Borders */
  --rf-border-subtle: #ffffff08;
  --rf-border-default: #ffffff12;
  --rf-border-strong: #ffffff20;
  
  /* === TYPOGRAPHY === */
  
  /* Font Families */
  --rf-font-display: 'JetBrains Mono', 'SF Mono', monospace;
  --rf-font-body: 'IBM Plex Sans', -apple-system, sans-serif;
  --rf-font-data: 'JetBrains Mono', monospace;
  
  /* Font Sizes */
  --rf-text-xs: 0.6875rem;   /* 11px - micro labels */
  --rf-text-sm: 0.75rem;     /* 12px - secondary */
  --rf-text-base: 0.875rem;  /* 14px - body */
  --rf-text-lg: 1rem;        /* 16px - emphasis */
  --rf-text-xl: 1.25rem;     /* 20px - headings */
  --rf-text-2xl: 1.5rem;     /* 24px - page titles */
  --rf-text-3xl: 2rem;       /* 32px - hero numbers */
  --rf-text-4xl: 3rem;       /* 48px - big metrics */
  
  /* Font Weights */
  --rf-weight-normal: 400;
  --rf-weight-medium: 500;
  --rf-weight-semibold: 600;
  --rf-weight-bold: 700;
  
  /* === SPACING === */
  --rf-space-1: 0.25rem;     /* 4px */
  --rf-space-2: 0.5rem;      /* 8px */
  --rf-space-3: 0.75rem;     /* 12px */
  --rf-space-4: 1rem;        /* 16px */
  --rf-space-5: 1.5rem;      /* 24px */
  --rf-space-6: 2rem;        /* 32px */
  --rf-space-8: 3rem;        /* 48px */
  
  /* === EFFECTS === */
  --rf-radius-sm: 4px;
  --rf-radius-md: 8px;
  --rf-radius-lg: 12px;
  --rf-radius-xl: 16px;
  
  --rf-shadow-sm: 0 1px 2px rgba(0,0,0,0.4);
  --rf-shadow-md: 0 4px 12px rgba(0,0,0,0.5);
  --rf-shadow-lg: 0 8px 32px rgba(0,0,0,0.6);
  --rf-shadow-glow: 0 0 20px var(--rf-accent-glow);
  
  /* === ANIMATION === */
  --rf-ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --rf-ease-in-out: cubic-bezier(0.65, 0, 0.35, 1);
  --rf-duration-fast: 150ms;
  --rf-duration-normal: 250ms;
  --rf-duration-slow: 400ms;
}
```

---

## 2. Information Architecture

### 2.1 Navigation Structure

```
┌─────────────────────────────────────────────────────────────────┐
│  CONTROL TOWER                                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  📊 Overview          ← Executive dashboard, KPIs, alerts        │
│  │                                                               │
│  ├── 🖼️ Images        ← Golden image registry & versions        │
│  │   ├── Registry                                                │
│  │   ├── Versions                                                │
│  │   └── Compliance                                              │
│  │                                                               │
│  ├── 📉 Drift         ← Patch drift analysis                    │
│  │   ├── By Environment                                          │
│  │   ├── By Platform                                             │
│  │   └── Trends                                                  │
│  │                                                               │
│  ├── 🏢 Sites         ← Data center & cloud regions             │
│  │   ├── Topology Map                                            │
│  │   ├── Site Details                                            │
│  │   └── Heatmaps                                                │
│  │                                                               │
│  ├── 🛡️ Compliance    ← Audit & evidence                        │
│  │   ├── Posture                                                 │
│  │   ├── Evidence Packs                                          │
│  │   └── Exceptions                                              │
│  │                                                               │
│  ├── 🔄 Resilience    ← BCP/DR status                           │
│  │   ├── DR Readiness                                            │
│  │   ├── Drills                                                  │
│  │   └── Failover Status                                         │
│  │                                                               │
│  ├── 🤖 AI Copilot    ← Natural language interface              │
│  │                                                               │
│  └── ⚙️ Settings      ← Configuration                           │
│      ├── Connectors                                              │
│      ├── Policies                                                │
│      ├── Notifications                                           │
│      └── RBAC                                                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 User Flows

#### Flow 1: Executive Daily Check
```
Login → Overview Dashboard → Check RAG Status → Drill into Amber/Red → Review Trends → Export Report
```

#### Flow 2: Ops Investigation
```
Alert Notification → Drift Details → Filter by Platform → View Affected Assets → Check Image Version → Initiate Rollout
```

#### Flow 3: Compliance Audit
```
Compliance Tab → Select Framework (CIS/ISO) → Generate Evidence Pack → Download Bundle
```

#### Flow 4: DR Drill
```
Resilience Tab → Select Workload → Configure Drill → Execute → Monitor RTO/RPO → Review Results
```

---

## 3. Page Layouts

### 3.1 Overview Dashboard

```
┌──────────────────────────────────────────────────────────────────────────┐
│ [Logo] Control Tower              [Search] [Notifications] [User Menu]   │
├────────┬─────────────────────────────────────────────────────────────────┤
│        │                                                                  │
│  NAV   │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌────────────┐ │
│        │  │ FLEET       │ │ DRIFT       │ │ COMPLIANCE  │ │ DR READY   │ │
│ Overview│  │   12,847    │ │    94.2%    │ │    97.8%    │ │   98.1%    │ │
│ Images │  │   assets    │ │   current   │ │   passing   │ │  readiness │ │
│ Drift  │  │ ↑ +234      │ │ ↓ -2.1%     │ │ → stable    │ │ ↑ +0.3%    │ │
│ Sites  │  └─────────────┘ └─────────────┘ └─────────────┘ └────────────┘ │
│ Compli │                                                                  │
│ Resili │  ┌────────────────────────────────┐ ┌───────────────────────────┐│
│ AI     │  │ PLATFORM DISTRIBUTION          │ │ ACTIVE ALERTS             ││
│        │  │ ██████████████ AWS    4,231    │ │ 🔴 3 Critical             ││
│ ────── │  │ ████████████   Azure  3,892    │ │ 🟡 12 Warning             ││
│ Settings│  │ ██████████     GCP    2,156    │ │ 🟢 847 Info               ││
│        │  │ ████████       vSphere 1,834   │ │                           ││
│        │  │ ██████         K8s     734     │ │ [View All →]              ││
│        │  └────────────────────────────────┘ └───────────────────────────┘│
│        │                                                                  │
│        │  ┌────────────────────────────────────────────────────────────┐ │
│        │  │ DRIFT HEATMAP BY SITE                                       │ │
│        │  │                                                             │ │
│        │  │   ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐        │ │
│        │  │   │🟢│ │🟢│ │🟡│ │🟢│ │🔴│ │🟢│ │🟢│ │🟡│        │ │
│        │  │   └───┘ └───┘ └───┘ └───┘ └───┘ └───┘ └───┘ └───┘        │ │
│        │  │   eu-w1 eu-w2 us-e1 us-w2 ap-s1 ap-n1 dc-ln dc-sg        │ │
│        │  │                                                             │ │
│        │  └────────────────────────────────────────────────────────────┘ │
│        │                                                                  │
│        │  ┌──────────────────────────┐ ┌──────────────────────────────┐  │
│        │  │ COVERAGE TREND (30 DAYS) │ │ RECENT ACTIVITY              │  │
│        │  │ ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█│ │ • Image promoted: ql-base... │  │
│        │  │                          │ │ • Drift detected: ap-south-1 │  │
│        │  │ 94.2% ↑ +3.1% vs 30d ago │ │ • DR drill completed: dc-lon │  │
│        │  └──────────────────────────┘ └──────────────────────────────┘  │
│        │                                                                  │
└────────┴─────────────────────────────────────────────────────────────────┘
```

### 3.2 Drift Analysis Page

```
┌──────────────────────────────────────────────────────────────────────────┐
│ Drift Analysis                                    [Filter ▼] [Export]    │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Environment: [All ▼]  Platform: [All ▼]  Site: [All ▼]  Age: [All ▼]   │
│                                                                          │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ DRIFT BY ENVIRONMENT                                                │ │
│  │                                                                      │ │
│  │  Production    ████████████████████████████░░░░░░░  87.3%  🟡       │ │
│  │  Staging       █████████████████████████████████░░  96.1%  🟢       │ │
│  │  Development   ████████████████████████████████░░░  92.8%  🟢       │ │
│  │  DR-Secondary  ██████████████████░░░░░░░░░░░░░░░░░  62.4%  🔴       │ │
│  │                                                                      │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ TOP OFFENDERS                                          [View All →] │ │
│  ├─────────────────────┬───────────┬─────────┬───────────┬────────────┤ │
│  │ Asset               │ Platform  │ Current │ Expected  │ Drift Age  │ │
│  ├─────────────────────┼───────────┼─────────┼───────────┼────────────┤ │
│  │ i-0abc123def456     │ AWS       │ 1.6.1   │ 1.6.4     │ 32 days 🔴 │ │
│  │ vm-prod-api-023     │ vSphere   │ 1.6.2   │ 1.6.4     │ 18 days 🟡 │ │
│  │ vmss-web-001        │ Azure     │ 1.6.2   │ 1.6.4     │ 18 days 🟡 │ │
│  │ mig-backend-eu      │ GCP       │ 1.6.3   │ 1.6.4     │ 7 days 🟢  │ │
│  └─────────────────────┴───────────┴─────────┴───────────┴────────────┘ │
│                                                                          │
│  ┌─────────────────────────────────┐ ┌──────────────────────────────────┐│
│  │ DRIFT AGE DISTRIBUTION         │ │ AI INSIGHT                        ││
│  │                                 │ │                                   ││
│  │ 0-7d   ████████████████ 4,231  │ │ "DR-Secondary site in Singapore  ││
│  │ 7-14d  ████████████     2,156  │ │  has significant drift (62.4%).  ││
│  │ 14-30d ████████         1,234  │ │  This poses a risk to failover   ││
│  │ 30d+   ████              567   │ │  readiness. Recommend immediate  ││
│  │                                 │ │  patch rollout before next DR    ││
│  │                                 │ │  drill on Dec 15."               ││
│  └─────────────────────────────────┘ └──────────────────────────────────┘│
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 3.3 Site Topology View

```
┌──────────────────────────────────────────────────────────────────────────┐
│ Sites & Topology                              [Map View] [List View]     │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                                                                      │ │
│  │                           🌍 GLOBAL TOPOLOGY                         │ │
│  │                                                                      │ │
│  │         ┌─────┐                              ┌─────┐                 │ │
│  │         │ 🟢  │ eu-west-1                    │ 🟢  │ us-east-1       │ │
│  │         │ 98% │ AWS                          │ 96% │ AWS             │ │
│  │         └──┬──┘                              └──┬──┘                 │ │
│  │            │                                    │                    │ │
│  │         ┌──┴──┐                              ┌──┴──┐                 │ │
│  │         │ 🟢  │ dc-london                    │ 🟡  │ dc-newyork      │ │
│  │         │ 94% │ vSphere                      │ 88% │ vSphere         │ │
│  │         └─────┘                              └─────┘                 │ │
│  │                                                                      │ │
│  │                           ┌─────┐                                    │ │
│  │                           │ 🔴  │ ap-south-1                         │ │
│  │                           │ 62% │ AWS + DC                           │ │
│  │                           └─────┘                                    │ │
│  │                                                                      │ │
│  │  ─── Primary Traffic    ╌╌╌ DR Failover Path                        │ │
│  │                                                                      │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────────┐│
│  │ SITE DETAILS                                                          ││
│  ├───────────────┬──────────┬────────┬─────────┬──────────┬─────────────┤│
│  │ Site          │ Platform │ Assets │ Drift % │ DR Ready │ Last Sync   ││
│  ├───────────────┼──────────┼────────┼─────────┼──────────┼─────────────┤│
│  │ eu-west-1     │ AWS      │ 2,341  │ 98.2%   │ ✓ Yes    │ 2 min ago   ││
│  │ dc-london     │ vSphere  │ 1,234  │ 94.1%   │ ✓ Yes    │ 5 min ago   ││
│  │ us-east-1     │ AWS      │ 1,890  │ 96.4%   │ ✓ Yes    │ 2 min ago   ││
│  │ dc-newyork    │ vSphere  │ 987    │ 88.3%   │ ⚠ Warn   │ 5 min ago   ││
│  │ ap-south-1    │ Multi    │ 1,456  │ 62.4%   │ ✗ No     │ 3 min ago   ││
│  └───────────────┴──────────┴────────┴─────────┴──────────┴─────────────┘│
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Component Library

### 4.1 Core Components

#### Status Badge
```jsx
// Variants: success, warning, critical, neutral
<StatusBadge status="success">98.2%</StatusBadge>
<StatusBadge status="warning" pulse>Drifting</StatusBadge>
<StatusBadge status="critical" pulse>Failed</StatusBadge>
```

#### Metric Card
```jsx
<MetricCard
  title="Fleet Coverage"
  value="12,847"
  subtitle="assets"
  trend={{ direction: "up", value: "+234", period: "24h" }}
  status="success"
/>
```

#### Platform Icon
```jsx
<PlatformIcon platform="aws" size="md" />
<PlatformIcon platform="azure" size="md" />
<PlatformIcon platform="gcp" size="md" />
<PlatformIcon platform="vsphere" size="md" />
```

#### Progress Bar
```jsx
<ProgressBar 
  value={94.2} 
  status="success"  // auto-colors based on thresholds
  showLabel
  size="md"
/>
```

#### Heatmap Cell
```jsx
<HeatmapCell 
  value={98.2}
  label="eu-west-1"
  onClick={() => drillDown('eu-west-1')}
/>
```

#### Data Table
```jsx
<DataTable
  columns={columns}
  data={assets}
  sortable
  filterable
  selectable
  pagination={{ pageSize: 25 }}
  rowStatus={(row) => row.driftAge > 30 ? 'critical' : 'default'}
/>
```

#### Sparkline
```jsx
<Sparkline
  data={coverageTrend}
  color="success"
  height={40}
  showArea
/>
```

#### AI Insight Card
```jsx
<AIInsightCard
  severity="warning"
  title="DR Site Drift Detected"
  content="Singapore DR site has 62.4% coverage..."
  actions={[
    { label: "View Details", onClick: () => {} },
    { label: "Acknowledge", onClick: () => {} }
  ]}
/>
```

### 4.2 Composite Components

#### Site Card
```jsx
<SiteCard
  name="eu-west-1"
  platform="aws"
  assets={2341}
  coverage={98.2}
  drReady={true}
  lastSync="2 min ago"
  onClick={() => navigate('/sites/eu-west-1')}
/>
```

#### Image Version Row
```jsx
<ImageVersionRow
  family="ql-base-linux"
  version="1.6.4"
  platforms={['aws', 'azure', 'gcp', 'vsphere']}
  compliance={{ cis: 'pass', slsa: 3, signed: true }}
  fleetCoverage={94.2}
  actions={['promote', 'view', 'deprecate']}
/>
```

#### Alert Row
```jsx
<AlertRow
  severity="critical"
  title="Patch drift exceeded SLA"
  source="ap-south-1"
  timestamp="5 min ago"
  acknowledged={false}
/>
```

---

## 5. Responsive Breakpoints

```css
/* Mobile First */
--rf-bp-sm: 640px;   /* Tablets */
--rf-bp-md: 768px;   /* Small laptops */
--rf-bp-lg: 1024px;  /* Laptops */
--rf-bp-xl: 1280px;  /* Desktops */
--rf-bp-2xl: 1536px; /* Large monitors */
--rf-bp-3xl: 1920px; /* Full HD */
--rf-bp-4xl: 2560px; /* 2K monitors */
```

### Responsive Behavior

| Breakpoint | Sidebar | Grid Columns | Data Density |
|------------|---------|--------------|--------------|
| < 768px    | Hidden (hamburger) | 1 | Compact |
| 768-1024px | Collapsed (icons) | 2 | Normal |
| 1024-1280px | Expanded | 3 | Normal |
| 1280-1920px | Expanded | 4 | Comfortable |
| > 1920px   | Expanded | 6 | Spacious |

---

## 6. Motion & Animation

### 6.1 Page Transitions
```css
.page-enter {
  opacity: 0;
  transform: translateY(8px);
}
.page-enter-active {
  opacity: 1;
  transform: translateY(0);
  transition: all 300ms var(--rf-ease-out);
}
```

### 6.2 Status Pulse
```css
@keyframes pulse-critical {
  0%, 100% { box-shadow: 0 0 0 0 var(--rf-status-red); }
  50% { box-shadow: 0 0 0 8px transparent; }
}

.status-critical-pulse {
  animation: pulse-critical 2s infinite;
}
```

### 6.3 Data Loading
```css
@keyframes shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}

.skeleton {
  background: linear-gradient(
    90deg,
    var(--rf-bg-surface) 0%,
    var(--rf-bg-elevated) 50%,
    var(--rf-bg-surface) 100%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}
```

### 6.4 Real-time Updates
```css
@keyframes flash-update {
  0% { background-color: var(--rf-accent-glow); }
  100% { background-color: transparent; }
}

.data-updated {
  animation: flash-update 1s ease-out;
}
```

---

## 7. Accessibility

### 7.1 Color Contrast
- All text meets WCAG AA (4.5:1 for normal, 3:1 for large)
- Status colors include icons/patterns for color-blind users
- Never rely on color alone for meaning

### 7.2 Keyboard Navigation
- Full keyboard navigation with visible focus states
- Skip links for main content
- ARIA labels for all interactive elements
- Escape closes modals/dropdowns

### 7.3 Screen Reader Support
- Semantic HTML structure
- ARIA live regions for real-time updates
- Descriptive alt text for charts/graphs
- Status announcements for alerts

---

## 8. Performance Guidelines

### 8.1 Data Loading
- Skeleton states for all async content
- Progressive loading for large datasets
- Virtual scrolling for tables > 100 rows
- Debounced search/filter inputs

### 8.2 Real-time Updates
- WebSocket for live data (Socket.IO)
- Optimistic UI updates
- Background sync every 30s
- Visual indicators for stale data

### 8.3 Bundle Optimization
- Code splitting by route
- Lazy load heavy components (charts, maps)
- Preload critical routes
- Service worker for caching

---

## 9. Technology Stack

| Layer | Technology |
|-------|------------|
| Framework | Next.js 16 (App Router) |
| Styling | Tailwind CSS + CSS Variables |
| Components | shadcn/ui (customized) |
| State | TanStack Query (React Query) |
| Charts | Recharts + custom SVG |
| Maps | Mapbox GL / Custom SVG |
| Real-time | Socket.IO |
| Forms | React Hook Form + Zod |
| Tables | TanStack Table |
| Animation | Framer Motion |
| Icons | Lucide React |

---

## 10. File Structure

```
ui/control-tower/
├── app/
│   ├── (auth)/
│   │   ├── login/
│   │   └── layout.tsx
│   ├── (dashboard)/
│   │   ├── overview/
│   │   ├── images/
│   │   ├── drift/
│   │   ├── sites/
│   │   ├── compliance/
│   │   ├── resilience/
│   │   ├── ai/
│   │   ├── settings/
│   │   └── layout.tsx
│   ├── layout.tsx
│   └── globals.css
├── components/
│   ├── ui/               # Base shadcn components
│   ├── data/             # Data display components
│   │   ├── metric-card.tsx
│   │   ├── data-table.tsx
│   │   ├── sparkline.tsx
│   │   └── progress-bar.tsx
│   ├── charts/           # Chart components
│   │   ├── area-chart.tsx
│   │   ├── bar-chart.tsx
│   │   └── heatmap.tsx
│   ├── status/           # Status indicators
│   │   ├── status-badge.tsx
│   │   ├── platform-icon.tsx
│   │   └── trend-indicator.tsx
│   ├── layout/           # Layout components
│   │   ├── sidebar.tsx
│   │   ├── header.tsx
│   │   └── page-header.tsx
│   └── ai/               # AI-specific components
│       ├── ai-chat.tsx
│       └── ai-insight-card.tsx
├── hooks/
│   ├── use-drift.ts
│   ├── use-assets.ts
│   ├── use-images.ts
│   └── use-realtime.ts
├── lib/
│   ├── api.ts
│   ├── socket.ts
│   └── utils.ts
├── styles/
│   └── tokens.css
└── types/
    └── index.ts
```

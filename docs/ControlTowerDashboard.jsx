import React, { useState, useEffect } from 'react';
import { 
  Activity, Shield, Server, AlertTriangle, CheckCircle, 
  XCircle, TrendingUp, TrendingDown, ChevronRight, Bell,
  Search, Settings, Cpu, Cloud, Database, Globe, Layers,
  BarChart3, GitBranch, RefreshCw, Zap, Eye, Box
} from 'lucide-react';

// Platform icons with brand colors
const PlatformIcon = ({ platform, size = 16 }) => {
  const colors = {
    aws: '#FF9900',
    azure: '#0078D4',
    gcp: '#4285F4',
    vsphere: '#6D9E37',
    k8s: '#326CE5',
    baremetal: '#8B8B8B'
  };
  
  const icons = {
    aws: Cloud,
    azure: Cloud,
    gcp: Cloud,
    vsphere: Server,
    k8s: Box,
    baremetal: Cpu
  };
  
  const Icon = icons[platform] || Cloud;
  return <Icon size={size} style={{ color: colors[platform] }} />;
};

// Status Badge Component
const StatusBadge = ({ status, children, pulse = false, size = 'md' }) => {
  const colors = {
    success: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30',
    warning: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
    critical: 'bg-red-500/20 text-red-400 border-red-500/30',
    neutral: 'bg-slate-500/20 text-slate-400 border-slate-500/30'
  };
  
  const sizes = {
    sm: 'text-xs px-2 py-0.5',
    md: 'text-sm px-2.5 py-1',
    lg: 'text-base px-3 py-1.5'
  };
  
  return (
    <span className={`
      inline-flex items-center gap-1.5 rounded-full border font-medium
      ${colors[status]} ${sizes[size]}
      ${pulse ? 'animate-pulse' : ''}
    `}>
      {status === 'success' && <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />}
      {status === 'warning' && <span className="w-1.5 h-1.5 rounded-full bg-amber-400" />}
      {status === 'critical' && <span className="w-1.5 h-1.5 rounded-full bg-red-400" />}
      {children}
    </span>
  );
};

// Metric Card Component
const MetricCard = ({ title, value, subtitle, trend, status = 'neutral', icon: Icon }) => {
  const statusColors = {
    success: 'border-emerald-500/20',
    warning: 'border-amber-500/20',
    critical: 'border-red-500/20',
    neutral: 'border-slate-700/50'
  };
  
  return (
    <div className={`
      bg-slate-900/80 rounded-xl p-5 border ${statusColors[status]}
      hover:bg-slate-800/80 transition-all duration-200
      backdrop-blur-sm
    `}>
      <div className="flex items-start justify-between mb-3">
        <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">
          {title}
        </span>
        {Icon && (
          <div className="p-2 rounded-lg bg-slate-800/80">
            <Icon size={16} className="text-slate-400" />
          </div>
        )}
      </div>
      <div className="flex items-baseline gap-2">
        <span className="text-3xl font-bold text-white font-mono tracking-tight">
          {value}
        </span>
        <span className="text-sm text-slate-500">{subtitle}</span>
      </div>
      {trend && (
        <div className={`
          flex items-center gap-1 mt-2 text-xs font-medium
          ${trend.direction === 'up' ? 'text-emerald-400' : 'text-red-400'}
        `}>
          {trend.direction === 'up' ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
          {trend.value}
          <span className="text-slate-500 font-normal">vs {trend.period}</span>
        </div>
      )}
    </div>
  );
};

// Heatmap Cell
const HeatmapCell = ({ site, coverage, onClick }) => {
  const getStatus = (cov) => {
    if (cov >= 95) return 'success';
    if (cov >= 85) return 'warning';
    return 'critical';
  };
  
  const status = getStatus(coverage);
  const bgColors = {
    success: 'bg-emerald-500/20 hover:bg-emerald-500/30 border-emerald-500/30',
    warning: 'bg-amber-500/20 hover:bg-amber-500/30 border-amber-500/30',
    critical: 'bg-red-500/20 hover:bg-red-500/30 border-red-500/30'
  };
  
  return (
    <button
      onClick={onClick}
      className={`
        flex flex-col items-center justify-center p-3 rounded-lg border
        ${bgColors[status]} transition-all duration-200
        hover:scale-105 cursor-pointer
      `}
    >
      <span className="text-lg font-bold text-white font-mono">{coverage}%</span>
      <span className="text-xs text-slate-400 mt-1">{site}</span>
    </button>
  );
};

// Progress Bar
const ProgressBar = ({ value, label, status = 'success', showValue = true }) => {
  const colors = {
    success: 'bg-emerald-500',
    warning: 'bg-amber-500',
    critical: 'bg-red-500'
  };
  
  return (
    <div className="space-y-1.5">
      <div className="flex justify-between text-sm">
        <span className="text-slate-300">{label}</span>
        {showValue && <span className="text-slate-400 font-mono">{value}%</span>}
      </div>
      <div className="h-2 bg-slate-800 rounded-full overflow-hidden">
        <div 
          className={`h-full ${colors[status]} rounded-full transition-all duration-500`}
          style={{ width: `${value}%` }}
        />
      </div>
    </div>
  );
};

// Alert Row
const AlertRow = ({ severity, title, source, time }) => {
  const icons = {
    critical: XCircle,
    warning: AlertTriangle,
    info: CheckCircle
  };
  const colors = {
    critical: 'text-red-400',
    warning: 'text-amber-400',
    info: 'text-emerald-400'
  };
  const Icon = icons[severity];
  
  return (
    <div className="flex items-center gap-3 p-3 rounded-lg bg-slate-800/50 hover:bg-slate-800 transition-colors cursor-pointer">
      <Icon size={18} className={colors[severity]} />
      <div className="flex-1 min-w-0">
        <p className="text-sm text-slate-200 truncate">{title}</p>
        <p className="text-xs text-slate-500">{source}</p>
      </div>
      <span className="text-xs text-slate-500 whitespace-nowrap">{time}</span>
    </div>
  );
};

// Sparkline Component
const Sparkline = ({ data, color = '#10B981', height = 40 }) => {
  const max = Math.max(...data);
  const min = Math.min(...data);
  const range = max - min || 1;
  
  const points = data.map((val, i) => {
    const x = (i / (data.length - 1)) * 100;
    const y = height - ((val - min) / range) * height;
    return `${x},${y}`;
  }).join(' ');
  
  return (
    <svg viewBox={`0 0 100 ${height}`} className="w-full" preserveAspectRatio="none">
      <defs>
        <linearGradient id={`grad-${color}`} x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor={color} stopOpacity="0.3" />
          <stop offset="100%" stopColor={color} stopOpacity="0" />
        </linearGradient>
      </defs>
      <polygon 
        points={`0,${height} ${points} 100,${height}`}
        fill={`url(#grad-${color})`}
      />
      <polyline
        points={points}
        fill="none"
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
};

// Activity Item
const ActivityItem = ({ icon: Icon, title, time, status }) => (
  <div className="flex items-start gap-3 py-2">
    <div className={`
      p-1.5 rounded-lg mt-0.5
      ${status === 'success' ? 'bg-emerald-500/20 text-emerald-400' : 
        status === 'warning' ? 'bg-amber-500/20 text-amber-400' : 
        'bg-slate-700/50 text-slate-400'}
    `}>
      <Icon size={14} />
    </div>
    <div className="flex-1 min-w-0">
      <p className="text-sm text-slate-300 truncate">{title}</p>
      <p className="text-xs text-slate-500">{time}</p>
    </div>
  </div>
);

// Sidebar Navigation
const Sidebar = ({ activeItem, onNavigate }) => {
  const navItems = [
    { id: 'overview', icon: BarChart3, label: 'Overview' },
    { id: 'images', icon: Layers, label: 'Images' },
    { id: 'drift', icon: GitBranch, label: 'Drift' },
    { id: 'sites', icon: Globe, label: 'Sites' },
    { id: 'compliance', icon: Shield, label: 'Compliance' },
    { id: 'resilience', icon: RefreshCw, label: 'Resilience' },
    { id: 'ai', icon: Zap, label: 'AI Copilot' },
  ];
  
  return (
    <div className="w-64 bg-slate-950 border-r border-slate-800 flex flex-col">
      {/* Logo */}
      <div className="p-5 border-b border-slate-800">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center">
            <Activity size={20} className="text-white" />
          </div>
          <div>
            <h1 className="text-sm font-bold text-white">QuantumLayer</h1>
            <p className="text-xs text-slate-500">Resilience Fabric</p>
          </div>
        </div>
      </div>
      
      {/* Navigation */}
      <nav className="flex-1 p-3 space-y-1">
        {navItems.map(item => (
          <button
            key={item.id}
            onClick={() => onNavigate(item.id)}
            className={`
              w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm
              transition-all duration-150
              ${activeItem === item.id 
                ? 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/30' 
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'}
            `}
          >
            <item.icon size={18} />
            {item.label}
          </button>
        ))}
      </nav>
      
      {/* Settings */}
      <div className="p-3 border-t border-slate-800">
        <button className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-slate-400 hover:text-slate-200 hover:bg-slate-800/50 transition-colors">
          <Settings size={18} />
          Settings
        </button>
      </div>
    </div>
  );
};

// Main Dashboard Component
export default function ControlTowerDashboard() {
  const [activeNav, setActiveNav] = useState('overview');
  const [currentTime, setCurrentTime] = useState(new Date());
  
  // Update time every minute
  useEffect(() => {
    const timer = setInterval(() => setCurrentTime(new Date()), 60000);
    return () => clearInterval(timer);
  }, []);
  
  // Mock data
  const metrics = {
    fleet: { value: '12,847', trend: { direction: 'up', value: '+234', period: '24h' } },
    drift: { value: '94.2%', trend: { direction: 'up', value: '+2.1%', period: '7d' } },
    compliance: { value: '97.8%', trend: { direction: 'up', value: '+0.5%', period: '30d' } },
    drReady: { value: '98.1%', trend: { direction: 'up', value: '+0.3%', period: '7d' } }
  };
  
  const platforms = [
    { name: 'AWS', count: 4231, platform: 'aws' },
    { name: 'Azure', count: 3892, platform: 'azure' },
    { name: 'GCP', count: 2156, platform: 'gcp' },
    { name: 'vSphere', count: 1834, platform: 'vsphere' },
    { name: 'Kubernetes', count: 734, platform: 'k8s' }
  ];
  
  const sites = [
    { name: 'eu-west-1', coverage: 98 },
    { name: 'eu-west-2', coverage: 96 },
    { name: 'us-east-1', coverage: 88 },
    { name: 'us-west-2', coverage: 97 },
    { name: 'ap-south-1', coverage: 62 },
    { name: 'ap-northeast-1', coverage: 95 },
    { name: 'dc-london', coverage: 94 },
    { name: 'dc-singapore', coverage: 71 }
  ];
  
  const alerts = [
    { severity: 'critical', title: 'Patch drift exceeded SLA', source: 'ap-south-1', time: '5m ago' },
    { severity: 'critical', title: 'DR readiness below threshold', source: 'dc-singapore', time: '12m ago' },
    { severity: 'critical', title: 'CVE-2025-1234 detected', source: 'ql-base-linux', time: '23m ago' },
    { severity: 'warning', title: 'Certificate expiring soon', source: 'us-east-1', time: '1h ago' },
    { severity: 'warning', title: 'Image rebuild required', source: 'ql-base-linux@1.6.3', time: '2h ago' }
  ];
  
  const trendData = [88, 89, 87, 90, 91, 89, 92, 93, 91, 94, 93, 94, 95, 94, 94];
  
  const activities = [
    { icon: CheckCircle, title: 'Image promoted: ql-base-linux@1.6.4 → prod', time: '10m ago', status: 'success' },
    { icon: AlertTriangle, title: 'Drift detected: ap-south-1 (62.4%)', time: '25m ago', status: 'warning' },
    { icon: RefreshCw, title: 'DR drill completed: dc-london', time: '1h ago', status: 'success' },
    { icon: Layers, title: 'New image registered: ql-base-win@2.1.0', time: '2h ago', status: 'success' },
    { icon: Shield, title: 'Compliance scan completed: CIS L1', time: '3h ago', status: 'success' }
  ];
  
  return (
    <div className="min-h-screen bg-slate-950 flex">
      {/* Sidebar */}
      <Sidebar activeItem={activeNav} onNavigate={setActiveNav} />
      
      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <header className="h-16 bg-slate-900/80 border-b border-slate-800 flex items-center justify-between px-6 backdrop-blur-sm">
          <div className="flex items-center gap-4">
            <h2 className="text-lg font-semibold text-white">Control Tower</h2>
            <StatusBadge status="success" size="sm">
              <span className="flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                Live
              </span>
            </StatusBadge>
          </div>
          
          <div className="flex items-center gap-4">
            {/* Search */}
            <div className="relative">
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
              <input
                type="text"
                placeholder="Search assets, images..."
                className="w-64 pl-9 pr-4 py-2 bg-slate-800/50 border border-slate-700 rounded-lg text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-indigo-500/50"
              />
            </div>
            
            {/* Notifications */}
            <button className="relative p-2 rounded-lg hover:bg-slate-800 transition-colors">
              <Bell size={20} className="text-slate-400" />
              <span className="absolute top-1 right-1 w-2 h-2 rounded-full bg-red-500" />
            </button>
            
            {/* User */}
            <div className="flex items-center gap-3 pl-4 border-l border-slate-700">
              <div className="text-right">
                <p className="text-sm font-medium text-slate-200">Satish G.</p>
                <p className="text-xs text-slate-500">Admin</p>
              </div>
              <div className="w-9 h-9 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-medium text-sm">
                SG
              </div>
            </div>
          </div>
        </header>
        
        {/* Dashboard Content */}
        <main className="flex-1 p-6 overflow-auto">
          {/* Timestamp */}
          <div className="flex items-center justify-between mb-6">
            <div>
              <h1 className="text-2xl font-bold text-white">Overview</h1>
              <p className="text-sm text-slate-500 mt-1">
                Last updated: {currentTime.toLocaleTimeString()} • Auto-refresh: 30s
              </p>
            </div>
            <button className="flex items-center gap-2 px-4 py-2 bg-slate-800 hover:bg-slate-700 rounded-lg text-sm text-slate-200 transition-colors">
              <RefreshCw size={16} />
              Refresh
            </button>
          </div>
          
          {/* Metrics Grid */}
          <div className="grid grid-cols-4 gap-4 mb-6">
            <MetricCard
              title="Fleet Assets"
              value={metrics.fleet.value}
              subtitle="total"
              trend={metrics.fleet.trend}
              status="neutral"
              icon={Server}
            />
            <MetricCard
              title="Drift Coverage"
              value={metrics.drift.value}
              subtitle="compliant"
              trend={metrics.drift.trend}
              status="success"
              icon={GitBranch}
            />
            <MetricCard
              title="Compliance"
              value={metrics.compliance.value}
              subtitle="passing"
              trend={metrics.compliance.trend}
              status="success"
              icon={Shield}
            />
            <MetricCard
              title="DR Readiness"
              value={metrics.drReady.value}
              subtitle="ready"
              trend={metrics.drReady.trend}
              status="success"
              icon={RefreshCw}
            />
          </div>
          
          {/* Two Column Layout */}
          <div className="grid grid-cols-3 gap-6 mb-6">
            {/* Platform Distribution */}
            <div className="bg-slate-900/80 rounded-xl p-5 border border-slate-800">
              <h3 className="text-sm font-semibold text-slate-200 mb-4">Platform Distribution</h3>
              <div className="space-y-3">
                {platforms.map(p => (
                  <div key={p.name} className="flex items-center gap-3">
                    <PlatformIcon platform={p.platform} size={18} />
                    <div className="flex-1">
                      <div className="flex justify-between text-sm mb-1">
                        <span className="text-slate-300">{p.name}</span>
                        <span className="text-slate-500 font-mono">{p.count.toLocaleString()}</span>
                      </div>
                      <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
                        <div 
                          className="h-full bg-indigo-500 rounded-full"
                          style={{ width: `${(p.count / platforms[0].count) * 100}%` }}
                        />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
            
            {/* Alerts */}
            <div className="bg-slate-900/80 rounded-xl p-5 border border-slate-800">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-slate-200">Active Alerts</h3>
                <div className="flex items-center gap-2">
                  <StatusBadge status="critical" size="sm">3</StatusBadge>
                  <StatusBadge status="warning" size="sm">12</StatusBadge>
                </div>
              </div>
              <div className="space-y-2">
                {alerts.slice(0, 4).map((alert, i) => (
                  <AlertRow key={i} {...alert} />
                ))}
              </div>
              <button className="w-full mt-3 py-2 text-sm text-indigo-400 hover:text-indigo-300 flex items-center justify-center gap-1">
                View all alerts <ChevronRight size={16} />
              </button>
            </div>
            
            {/* Recent Activity */}
            <div className="bg-slate-900/80 rounded-xl p-5 border border-slate-800">
              <h3 className="text-sm font-semibold text-slate-200 mb-4">Recent Activity</h3>
              <div className="space-y-1">
                {activities.map((activity, i) => (
                  <ActivityItem key={i} {...activity} />
                ))}
              </div>
            </div>
          </div>
          
          {/* Heatmap */}
          <div className="bg-slate-900/80 rounded-xl p-5 border border-slate-800 mb-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-slate-200">Drift Coverage by Site</h3>
              <div className="flex items-center gap-4 text-xs">
                <div className="flex items-center gap-1.5">
                  <span className="w-3 h-3 rounded bg-emerald-500/40 border border-emerald-500/50" />
                  <span className="text-slate-400">≥95%</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="w-3 h-3 rounded bg-amber-500/40 border border-amber-500/50" />
                  <span className="text-slate-400">85-94%</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="w-3 h-3 rounded bg-red-500/40 border border-red-500/50" />
                  <span className="text-slate-400">&lt;85%</span>
                </div>
              </div>
            </div>
            <div className="grid grid-cols-8 gap-3">
              {sites.map(site => (
                <HeatmapCell 
                  key={site.name}
                  site={site.name}
                  coverage={site.coverage}
                  onClick={() => console.log('Drill into', site.name)}
                />
              ))}
            </div>
          </div>
          
          {/* Coverage Trend & AI Insight */}
          <div className="grid grid-cols-2 gap-6">
            {/* Trend */}
            <div className="bg-slate-900/80 rounded-xl p-5 border border-slate-800">
              <h3 className="text-sm font-semibold text-slate-200 mb-4">Coverage Trend (30 Days)</h3>
              <div className="h-24 mb-3">
                <Sparkline data={trendData} color="#10B981" height={96} />
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-2xl font-bold text-white font-mono">94.2%</span>
                <div className="flex items-center gap-1 text-emerald-400">
                  <TrendingUp size={16} />
                  <span>+6.2% vs 30d ago</span>
                </div>
              </div>
            </div>
            
            {/* AI Insight */}
            <div className="bg-gradient-to-br from-indigo-500/10 to-purple-500/10 rounded-xl p-5 border border-indigo-500/20">
              <div className="flex items-start gap-3">
                <div className="p-2 rounded-lg bg-indigo-500/20">
                  <Zap size={20} className="text-indigo-400" />
                </div>
                <div>
                  <h3 className="text-sm font-semibold text-indigo-300 mb-2">AI Insight</h3>
                  <p className="text-sm text-slate-300 leading-relaxed">
                    <strong className="text-amber-400">DR site drift detected.</strong> The Singapore DC 
                    (dc-singapore) and ap-south-1 have significant patch drift (62-71%). This poses a 
                    risk to failover readiness. Recommend prioritizing patch rollout before the 
                    scheduled DR drill on <strong className="text-white">December 15th</strong>.
                  </p>
                  <div className="flex gap-2 mt-4">
                    <button className="px-3 py-1.5 bg-indigo-500/20 hover:bg-indigo-500/30 border border-indigo-500/30 rounded-lg text-sm text-indigo-300 transition-colors">
                      View Details
                    </button>
                    <button className="px-3 py-1.5 hover:bg-slate-800/50 rounded-lg text-sm text-slate-400 transition-colors">
                      Acknowledge
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}

import React, { useState } from 'react';
import { 
  GitBranch, Filter, Download, ChevronDown, ChevronRight,
  AlertTriangle, CheckCircle, XCircle, Search, RefreshCw,
  Server, Cloud, Box, Cpu, TrendingUp, TrendingDown,
  Calendar, Clock, Zap, Eye, ArrowUpRight, BarChart3
} from 'lucide-react';

// Reusable Components
const StatusBadge = ({ status, children, size = 'md' }) => {
  const colors = {
    success: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30',
    warning: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
    critical: 'bg-red-500/20 text-red-400 border-red-500/30',
    neutral: 'bg-slate-500/20 text-slate-400 border-slate-500/30'
  };
  const sizes = {
    sm: 'text-xs px-2 py-0.5',
    md: 'text-sm px-2.5 py-1',
  };
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full border font-medium ${colors[status]} ${sizes[size]}`}>
      {children}
    </span>
  );
};

const PlatformIcon = ({ platform, size = 16 }) => {
  const colors = {
    aws: '#FF9900', azure: '#0078D4', gcp: '#4285F4',
    vsphere: '#6D9E37', k8s: '#326CE5', baremetal: '#8B8B8B'
  };
  const icons = {
    aws: Cloud, azure: Cloud, gcp: Cloud,
    vsphere: Server, k8s: Box, baremetal: Cpu
  };
  const Icon = icons[platform] || Cloud;
  return <Icon size={size} style={{ color: colors[platform] }} />;
};

// Filter Dropdown
const FilterDropdown = ({ label, options, value, onChange }) => {
  const [isOpen, setIsOpen] = useState(false);
  
  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 hover:border-slate-600 transition-colors"
      >
        <span className="text-slate-500">{label}:</span>
        <span>{value}</span>
        <ChevronDown size={16} className={`text-slate-500 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>
      {isOpen && (
        <div className="absolute top-full left-0 mt-1 w-48 bg-slate-800 border border-slate-700 rounded-lg shadow-xl z-10 py-1">
          {options.map(opt => (
            <button
              key={opt}
              onClick={() => { onChange(opt); setIsOpen(false); }}
              className={`w-full text-left px-3 py-2 text-sm hover:bg-slate-700 transition-colors ${
                value === opt ? 'text-indigo-400' : 'text-slate-300'
              }`}
            >
              {opt}
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

// Environment Progress Bar
const EnvProgressBar = ({ env, coverage, assets, status }) => {
  const statusColors = {
    success: { bar: 'bg-emerald-500', dot: 'bg-emerald-400' },
    warning: { bar: 'bg-amber-500', dot: 'bg-amber-400' },
    critical: { bar: 'bg-red-500', dot: 'bg-red-400' }
  };
  
  return (
    <div className="group hover:bg-slate-800/50 rounded-lg p-3 -mx-3 transition-colors cursor-pointer">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-3">
          <span className={`w-2 h-2 rounded-full ${statusColors[status].dot}`} />
          <span className="text-sm font-medium text-slate-200">{env}</span>
          <span className="text-xs text-slate-500">{assets.toLocaleString()} assets</span>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm font-mono text-slate-300">{coverage}%</span>
          <ChevronRight size={16} className="text-slate-600 group-hover:text-slate-400 transition-colors" />
        </div>
      </div>
      <div className="h-2 bg-slate-800 rounded-full overflow-hidden">
        <div 
          className={`h-full ${statusColors[status].bar} rounded-full transition-all duration-500`}
          style={{ width: `${coverage}%` }}
        />
      </div>
    </div>
  );
};

// Asset Row
const AssetRow = ({ asset, isSelected, onSelect }) => {
  const getStatus = (age) => {
    if (age > 30) return 'critical';
    if (age > 14) return 'warning';
    return 'success';
  };
  
  const status = getStatus(asset.driftAge);
  const statusColors = {
    success: 'text-emerald-400',
    warning: 'text-amber-400',
    critical: 'text-red-400'
  };
  
  return (
    <tr className={`
      border-b border-slate-800 hover:bg-slate-800/50 transition-colors cursor-pointer
      ${isSelected ? 'bg-indigo-500/10' : ''}
    `}>
      <td className="py-3 px-4">
        <input
          type="checkbox"
          checked={isSelected}
          onChange={onSelect}
          className="w-4 h-4 rounded border-slate-600 bg-slate-800 text-indigo-500 focus:ring-indigo-500/20"
        />
      </td>
      <td className="py-3 px-4">
        <div className="flex items-center gap-2">
          <PlatformIcon platform={asset.platform} size={16} />
          <span className="text-sm font-mono text-slate-200">{asset.id}</span>
        </div>
      </td>
      <td className="py-3 px-4">
        <span className="text-sm text-slate-400">{asset.site}</span>
      </td>
      <td className="py-3 px-4">
        <span className="text-sm font-mono text-slate-300">{asset.current}</span>
      </td>
      <td className="py-3 px-4">
        <span className="text-sm font-mono text-emerald-400">{asset.expected}</span>
      </td>
      <td className="py-3 px-4">
        <StatusBadge status={status} size="sm">
          {asset.driftAge} days
        </StatusBadge>
      </td>
      <td className="py-3 px-4">
        <button className="p-1.5 hover:bg-slate-700 rounded-lg transition-colors">
          <Eye size={16} className="text-slate-500" />
        </button>
      </td>
    </tr>
  );
};

// Distribution Bar Chart
const DistributionChart = ({ data }) => {
  const max = Math.max(...data.map(d => d.count));
  
  return (
    <div className="space-y-3">
      {data.map((item, i) => (
        <div key={i} className="flex items-center gap-3">
          <span className="w-16 text-xs text-slate-500 text-right">{item.label}</span>
          <div className="flex-1 h-6 bg-slate-800 rounded overflow-hidden">
            <div 
              className={`h-full ${item.color} rounded transition-all duration-500 flex items-center justify-end pr-2`}
              style={{ width: `${(item.count / max) * 100}%` }}
            >
              <span className="text-xs font-mono text-white/80">{item.count.toLocaleString()}</span>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
};

// Main Component
export default function DriftAnalysisPage() {
  const [envFilter, setEnvFilter] = useState('All');
  const [platformFilter, setPlatformFilter] = useState('All');
  const [siteFilter, setSiteFilter] = useState('All');
  const [ageFilter, setAgeFilter] = useState('All');
  const [selectedAssets, setSelectedAssets] = useState(new Set());
  const [searchQuery, setSearchQuery] = useState('');
  
  // Mock data
  const environments = [
    { name: 'Production', coverage: 87.3, assets: 5234, status: 'warning' },
    { name: 'Staging', coverage: 96.1, assets: 2341, status: 'success' },
    { name: 'Development', coverage: 92.8, assets: 3456, status: 'success' },
    { name: 'DR-Secondary', coverage: 62.4, assets: 1816, status: 'critical' }
  ];
  
  const assets = [
    { id: 'i-0abc123def456', platform: 'aws', site: 'ap-south-1', current: '1.6.1', expected: '1.6.4', driftAge: 32 },
    { id: 'vm-prod-api-023', platform: 'vsphere', site: 'dc-singapore', current: '1.6.2', expected: '1.6.4', driftAge: 18 },
    { id: 'vmss-web-001', platform: 'azure', site: 'westeurope', current: '1.6.2', expected: '1.6.4', driftAge: 18 },
    { id: 'mig-backend-eu', platform: 'gcp', site: 'europe-west1', current: '1.6.3', expected: '1.6.4', driftAge: 7 },
    { id: 'i-0xyz789abc012', platform: 'aws', site: 'ap-south-1', current: '1.6.1', expected: '1.6.4', driftAge: 35 },
    { id: 'vm-dr-web-001', platform: 'vsphere', site: 'dc-singapore', current: '1.6.1', expected: '1.6.4', driftAge: 41 },
    { id: 'pod-api-k8s-034', platform: 'k8s', site: 'us-east-1', current: '1.6.3', expected: '1.6.4', driftAge: 5 },
    { id: 'bm-db-master-01', platform: 'baremetal', site: 'dc-london', current: '1.6.3', expected: '1.6.4', driftAge: 7 }
  ];
  
  const driftDistribution = [
    { label: '0-7d', count: 4231, color: 'bg-emerald-500' },
    { label: '7-14d', count: 2156, color: 'bg-emerald-500/70' },
    { label: '14-30d', count: 1234, color: 'bg-amber-500' },
    { label: '30d+', count: 567, color: 'bg-red-500' }
  ];
  
  const toggleAsset = (id) => {
    const newSelected = new Set(selectedAssets);
    if (newSelected.has(id)) {
      newSelected.delete(id);
    } else {
      newSelected.add(id);
    }
    setSelectedAssets(newSelected);
  };
  
  const selectAll = () => {
    if (selectedAssets.size === assets.length) {
      setSelectedAssets(new Set());
    } else {
      setSelectedAssets(new Set(assets.map(a => a.id)));
    }
  };
  
  return (
    <div className="min-h-screen bg-slate-950 text-white">
      {/* Header */}
      <header className="border-b border-slate-800 bg-slate-900/80 backdrop-blur-sm sticky top-0 z-10">
        <div className="px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-indigo-500/20">
                <GitBranch size={20} className="text-indigo-400" />
              </div>
              <div>
                <h1 className="text-xl font-bold text-white">Drift Analysis</h1>
                <p className="text-sm text-slate-500">Patch drift across your infrastructure</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <button className="flex items-center gap-2 px-4 py-2 bg-slate-800 hover:bg-slate-700 border border-slate-700 rounded-lg text-sm text-slate-300 transition-colors">
                <Download size={16} />
                Export
              </button>
              <button className="flex items-center gap-2 px-4 py-2 bg-indigo-500 hover:bg-indigo-600 rounded-lg text-sm text-white transition-colors">
                <RefreshCw size={16} />
                Refresh
              </button>
            </div>
          </div>
        </div>
        
        {/* Filters */}
        <div className="px-6 py-3 border-t border-slate-800 bg-slate-900/50">
          <div className="flex items-center gap-3">
            <Filter size={16} className="text-slate-500" />
            <FilterDropdown 
              label="Environment" 
              options={['All', 'Production', 'Staging', 'Development', 'DR-Secondary']}
              value={envFilter}
              onChange={setEnvFilter}
            />
            <FilterDropdown 
              label="Platform" 
              options={['All', 'AWS', 'Azure', 'GCP', 'vSphere', 'K8s']}
              value={platformFilter}
              onChange={setPlatformFilter}
            />
            <FilterDropdown 
              label="Site" 
              options={['All', 'eu-west-1', 'us-east-1', 'ap-south-1', 'dc-london', 'dc-singapore']}
              value={siteFilter}
              onChange={setSiteFilter}
            />
            <FilterDropdown 
              label="Drift Age" 
              options={['All', '0-7 days', '7-14 days', '14-30 days', '30+ days']}
              value={ageFilter}
              onChange={setAgeFilter}
            />
            
            {(envFilter !== 'All' || platformFilter !== 'All' || siteFilter !== 'All' || ageFilter !== 'All') && (
              <button 
                onClick={() => { setEnvFilter('All'); setPlatformFilter('All'); setSiteFilter('All'); setAgeFilter('All'); }}
                className="text-sm text-indigo-400 hover:text-indigo-300 transition-colors"
              >
                Clear filters
              </button>
            )}
          </div>
        </div>
      </header>
      
      {/* Main Content */}
      <main className="p-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-4 gap-4 mb-6">
          <div className="bg-slate-900/80 rounded-xl p-4 border border-slate-800">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-slate-500 uppercase tracking-wider">Total Assets</span>
              <Server size={16} className="text-slate-600" />
            </div>
            <div className="text-2xl font-bold text-white font-mono">12,847</div>
            <div className="text-xs text-emerald-400 flex items-center gap-1 mt-1">
              <TrendingUp size={12} /> +234 (24h)
            </div>
          </div>
          <div className="bg-slate-900/80 rounded-xl p-4 border border-emerald-500/20">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-slate-500 uppercase tracking-wider">Compliant</span>
              <CheckCircle size={16} className="text-emerald-500" />
            </div>
            <div className="text-2xl font-bold text-emerald-400 font-mono">11,159</div>
            <div className="text-xs text-slate-400 mt-1">86.9% of fleet</div>
          </div>
          <div className="bg-slate-900/80 rounded-xl p-4 border border-amber-500/20">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-slate-500 uppercase tracking-wider">Drifting</span>
              <AlertTriangle size={16} className="text-amber-500" />
            </div>
            <div className="text-2xl font-bold text-amber-400 font-mono">1,121</div>
            <div className="text-xs text-slate-400 mt-1">8.7% of fleet</div>
          </div>
          <div className="bg-slate-900/80 rounded-xl p-4 border border-red-500/20">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-slate-500 uppercase tracking-wider">Critical (&gt;30d)</span>
              <XCircle size={16} className="text-red-500" />
            </div>
            <div className="text-2xl font-bold text-red-400 font-mono">567</div>
            <div className="text-xs text-slate-400 mt-1">4.4% of fleet</div>
          </div>
        </div>
        
        {/* Two Column Layout */}
        <div className="grid grid-cols-3 gap-6 mb-6">
          {/* Environment Breakdown */}
          <div className="bg-slate-900/80 rounded-xl p-5 border border-slate-800">
            <h3 className="text-sm font-semibold text-slate-200 mb-4">Drift by Environment</h3>
            <div className="space-y-2">
              {environments.map(env => (
                <EnvProgressBar 
                  key={env.name}
                  env={env.name}
                  coverage={env.coverage}
                  assets={env.assets}
                  status={env.status}
                />
              ))}
            </div>
          </div>
          
          {/* Age Distribution */}
          <div className="bg-slate-900/80 rounded-xl p-5 border border-slate-800">
            <h3 className="text-sm font-semibold text-slate-200 mb-4">Drift Age Distribution</h3>
            <DistributionChart data={driftDistribution} />
            <div className="mt-4 pt-4 border-t border-slate-800">
              <div className="flex items-center justify-between text-sm">
                <span className="text-slate-400">Avg drift age</span>
                <span className="font-mono text-white">8.3 days</span>
              </div>
              <div className="flex items-center justify-between text-sm mt-2">
                <span className="text-slate-400">Median drift age</span>
                <span className="font-mono text-white">5 days</span>
              </div>
            </div>
          </div>
          
          {/* AI Insight */}
          <div className="bg-gradient-to-br from-amber-500/10 to-red-500/10 rounded-xl p-5 border border-amber-500/20">
            <div className="flex items-start gap-3">
              <div className="p-2 rounded-lg bg-amber-500/20">
                <Zap size={18} className="text-amber-400" />
              </div>
              <div>
                <h3 className="text-sm font-semibold text-amber-300 mb-2">AI Insight</h3>
                <p className="text-sm text-slate-300 leading-relaxed">
                  <strong className="text-red-400">DR site at risk.</strong> The DR-Secondary 
                  environment has only 62.4% coverage, with 567 assets drifting 30+ days. 
                  This violates your 85% DR parity SLA.
                </p>
                <div className="mt-4 space-y-2">
                  <div className="flex items-center gap-2 text-xs">
                    <Calendar size={12} className="text-slate-500" />
                    <span className="text-slate-400">Next DR drill: Dec 15, 2025</span>
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <Clock size={12} className="text-slate-500" />
                    <span className="text-slate-400">Est. remediation: 4-6 hours</span>
                  </div>
                </div>
                <button className="mt-4 flex items-center gap-2 px-3 py-1.5 bg-amber-500/20 hover:bg-amber-500/30 border border-amber-500/30 rounded-lg text-sm text-amber-300 transition-colors">
                  Generate remediation plan
                  <ArrowUpRight size={14} />
                </button>
              </div>
            </div>
          </div>
        </div>
        
        {/* Assets Table */}
        <div className="bg-slate-900/80 rounded-xl border border-slate-800">
          <div className="flex items-center justify-between p-4 border-b border-slate-800">
            <div className="flex items-center gap-4">
              <h3 className="text-sm font-semibold text-slate-200">Top Offenders</h3>
              <StatusBadge status="critical" size="sm">{assets.filter(a => a.driftAge > 30).length} critical</StatusBadge>
            </div>
            <div className="flex items-center gap-3">
              <div className="relative">
                <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                <input
                  type="text"
                  placeholder="Search assets..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-64 pl-9 pr-4 py-2 bg-slate-800/50 border border-slate-700 rounded-lg text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-indigo-500/50"
                />
              </div>
              {selectedAssets.size > 0 && (
                <button className="flex items-center gap-2 px-3 py-2 bg-indigo-500/20 border border-indigo-500/30 rounded-lg text-sm text-indigo-300 hover:bg-indigo-500/30 transition-colors">
                  Bulk remediate ({selectedAssets.size})
                </button>
              )}
            </div>
          </div>
          
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-xs text-slate-500 uppercase tracking-wider border-b border-slate-800">
                  <th className="py-3 px-4 text-left">
                    <input
                      type="checkbox"
                      checked={selectedAssets.size === assets.length}
                      onChange={selectAll}
                      className="w-4 h-4 rounded border-slate-600 bg-slate-800 text-indigo-500 focus:ring-indigo-500/20"
                    />
                  </th>
                  <th className="py-3 px-4 text-left">Asset</th>
                  <th className="py-3 px-4 text-left">Site</th>
                  <th className="py-3 px-4 text-left">Current</th>
                  <th className="py-3 px-4 text-left">Expected</th>
                  <th className="py-3 px-4 text-left">Drift Age</th>
                  <th className="py-3 px-4 text-left">Actions</th>
                </tr>
              </thead>
              <tbody>
                {assets.map(asset => (
                  <AssetRow
                    key={asset.id}
                    asset={asset}
                    isSelected={selectedAssets.has(asset.id)}
                    onSelect={() => toggleAsset(asset.id)}
                  />
                ))}
              </tbody>
            </table>
          </div>
          
          {/* Pagination */}
          <div className="flex items-center justify-between p-4 border-t border-slate-800">
            <span className="text-sm text-slate-500">Showing 1-8 of 567 assets</span>
            <div className="flex items-center gap-2">
              <button className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 rounded-lg text-sm text-slate-400 transition-colors">
                Previous
              </button>
              <button className="px-3 py-1.5 bg-indigo-500/20 border border-indigo-500/30 rounded-lg text-sm text-indigo-300">
                1
              </button>
              <button className="px-3 py-1.5 hover:bg-slate-800 rounded-lg text-sm text-slate-400 transition-colors">
                2
              </button>
              <button className="px-3 py-1.5 hover:bg-slate-800 rounded-lg text-sm text-slate-400 transition-colors">
                3
              </button>
              <span className="text-slate-600">...</span>
              <button className="px-3 py-1.5 hover:bg-slate-800 rounded-lg text-sm text-slate-400 transition-colors">
                71
              </button>
              <button className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 rounded-lg text-sm text-slate-400 transition-colors">
                Next
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

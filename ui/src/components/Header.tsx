import { Search, Download, Wifi, WifiOff, Pause, Play, Filter, RefreshCw, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { formatBytes } from '../utils'

interface FilterOptions {
  searchText: string
  showOnlyHTTP: boolean
  showDNS: boolean
  showAllPorts: boolean
}

interface HeaderProps {
  connected: boolean
  flowCount: number
  pcapSize: number
  filter: string
  onFilterChange: (filter: string) => void
  filterOptions: FilterOptions
  onFilterOptionsChange: (options: FilterOptions) => void
  onDownloadPCAP: () => void
  isPaused: boolean
  onTogglePause: () => void
}

export function Header({
  connected,
  flowCount,
  pcapSize,
  filter,
  onFilterChange,
  filterOptions,
  onFilterOptionsChange,
  onDownloadPCAP,
  isPaused,
  onTogglePause,
}: HeaderProps) {
  const [bpfFilter, setBpfFilter] = useState('')
  const [currentBPFFilter, setCurrentBPFFilter] = useState('')
  const [applyingFilter, setApplyingFilter] = useState(false)
  const [resettingPCAP, setResettingPCAP] = useState(false)

  const handleApplyBPFFilter = async () => {
    if (!bpfFilter.trim()) return

    setApplyingFilter(true)
    try {
      const res = await fetch('/api/bpf-filter', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ filter: bpfFilter.trim() }),
      })

      const data = await res.json()

      if (res.ok) {
        setCurrentBPFFilter(bpfFilter.trim())
        console.log('BPF filter applied:', data.message)
      } else {
        console.error('Failed to apply BPF filter:', data.error)
        alert(`Invalid BPF filter: ${data.error}\n\nExample valid filters:\n- tcp port 80\n- udp port 53\n- tcp port 8080 or tcp port 443`)
      }
    } catch (err) {
      console.error('Error applying BPF filter:', err)
      alert('Error applying BPF filter')
    } finally {
      setApplyingFilter(false)
    }
  }

  const handleClearBPFFilter = async () => {
    setApplyingFilter(true)
    try {
      const res = await fetch('/api/bpf-filter', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ filter: '' }),
      })

      if (res.ok) {
        setBpfFilter('')
        setCurrentBPFFilter('')
        console.log('BPF filter cleared')
      }
    } catch (err) {
      console.error('Error clearing BPF filter:', err)
    } finally {
      setApplyingFilter(false)
    }
  }

  const handleResetPCAP = async () => {
    if (!confirm('Are you sure you want to reset the PCAP buffer? This will delete all captured data.')) {
      return
    }

    setResettingPCAP(true)
    try {
      const res = await fetch('/api/pcap/reset', {
        method: 'POST',
      })

      if (res.ok) {
        const data = await res.json()
        console.log('PCAP reset:', data.message)
        alert('PCAP buffer reset successfully. Capture size will reset on page refresh.')
      } else {
        console.error('Failed to reset PCAP')
        alert('Failed to reset PCAP buffer')
      }
    } catch (err) {
      console.error('Error resetting PCAP:', err)
      alert('Error resetting PCAP buffer')
    } finally {
      setResettingPCAP(false)
    }
  }

  return (
    <header className="bg-slate-800 border-b border-slate-700 px-4 py-3">
      <div className="flex items-center justify-between">
        {/* Logo and Title */}
        <div className="flex items-center gap-3">
          <div className="text-2xl">🦈</div>
          <div>
            <h1 className="text-xl font-bold text-white">PodScope</h1>
            <p className="text-xs text-slate-400">Kubernetes Traffic Analyzer</p>
          </div>
        </div>

        {/* Search/Filter */}
        <div className="flex-1 max-w-xl mx-8">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Filter by IP, Pod, Service, URL, SNI..."
              value={filter}
              onChange={(e) => onFilterChange(e.target.value)}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg pl-10 pr-4 py-2 text-sm text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-podscope-500 focus:border-transparent"
            />
          </div>
        </div>

        {/* Status and Actions */}
        <div className="flex items-center gap-6">
          {/* Connection Status */}
          <div className="flex items-center gap-2">
            {connected ? (
              <>
                <Wifi className="w-4 h-4 text-green-400" />
                <span className="text-sm text-green-400">Live</span>
              </>
            ) : (
              <>
                <WifiOff className="w-4 h-4 text-red-400" />
                <span className="text-sm text-red-400">Disconnected</span>
              </>
            )}
          </div>

          {/* Stats */}
          <div className="flex items-center gap-4 text-sm text-slate-400">
            <span>{flowCount} flows</span>
            <span>{formatBytes(pcapSize)} captured</span>
          </div>

          {/* Pause/Resume Button */}
          <button
            onClick={onTogglePause}
            className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
              isPaused
                ? 'bg-yellow-600 hover:bg-yellow-700 text-white'
                : 'bg-slate-700 hover:bg-slate-600 text-slate-300'
            }`}
            title={isPaused ? 'Resume streaming' : 'Pause streaming'}
          >
            {isPaused ? (
              <>
                <Play className="w-4 h-4" />
                Resume
              </>
            ) : (
              <>
                <Pause className="w-4 h-4" />
                Pause
              </>
            )}
          </button>

          {/* Download Button */}
          <button
            onClick={onDownloadPCAP}
            className="flex items-center gap-2 bg-podscope-600 hover:bg-podscope-700 px-4 py-2 rounded-lg text-sm font-medium transition-colors"
          >
            <Download className="w-4 h-4" />
            Download PCAP
          </button>

          {/* Reset PCAP Button */}
          <button
            onClick={handleResetPCAP}
            disabled={resettingPCAP}
            className="flex items-center gap-2 bg-red-600 hover:bg-red-700 disabled:bg-slate-600 disabled:cursor-not-allowed px-4 py-2 rounded-lg text-sm font-medium text-white transition-colors"
            title="Reset PCAP buffer (deletes all captured data)"
          >
            <Trash2 className="w-4 h-4" />
            {resettingPCAP ? 'Resetting...' : 'Reset PCAP'}
          </button>
        </div>
      </div>

      {/* Filter Toggles Row */}
      <div className="mt-3 flex items-center gap-2">
        <Filter className="w-4 h-4 text-slate-400" />
        <span className="text-xs text-slate-400 mr-2">Filter:</span>

        <button
          onClick={() => onFilterOptionsChange({ ...filterOptions, showOnlyHTTP: !filterOptions.showOnlyHTTP, showAllPorts: false })}
          className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
            filterOptions.showOnlyHTTP
              ? 'bg-podscope-600 text-white'
              : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
          }`}
        >
          {filterOptions.showOnlyHTTP ? '✓ ' : ''}HTTP/HTTPS Only
        </button>

        <button
          onClick={() => onFilterOptionsChange({ ...filterOptions, showDNS: !filterOptions.showDNS })}
          className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
            filterOptions.showDNS
              ? 'bg-podscope-600 text-white'
              : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
          }`}
        >
          {filterOptions.showDNS ? '✓ ' : ''}Show DNS
        </button>

        <button
          onClick={() => onFilterOptionsChange({ ...filterOptions, showAllPorts: !filterOptions.showAllPorts, showOnlyHTTP: false })}
          className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
            filterOptions.showAllPorts
              ? 'bg-podscope-600 text-white'
              : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
          }`}
        >
          {filterOptions.showAllPorts ? '✓ ' : ''}Show All Ports
        </button>

        <span className="text-xs text-slate-500 ml-2">
          (Ports: {filterOptions.showAllPorts ? 'All' : filterOptions.showOnlyHTTP ? '80, 443, 8080, 8443, etc.' : 'Custom'})
        </span>
      </div>

      {/* BPF Filter Row */}
      <div className="mt-2 flex items-center gap-2">
        <RefreshCw className="w-4 h-4 text-slate-400" />
        <span className="text-xs text-slate-400 mr-2">BPF Filter:</span>

        <input
          type="text"
          placeholder="e.g., tcp port 80 or udp port 53"
          value={bpfFilter}
          onChange={(e) => setBpfFilter(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              handleApplyBPFFilter()
            }
          }}
          className="flex-1 max-w-md bg-slate-700 border border-slate-600 rounded px-3 py-1 text-xs text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-podscope-500"
        />

        <button
          onClick={handleApplyBPFFilter}
          disabled={applyingFilter || !bpfFilter.trim()}
          className="px-3 py-1 rounded-md text-xs font-medium bg-green-600 hover:bg-green-700 disabled:bg-slate-600 disabled:cursor-not-allowed text-white transition-colors"
          title="Apply BPF filter to all agents (updates within 5 seconds)"
        >
          {applyingFilter ? 'Applying...' : 'Apply Filter'}
        </button>

        {currentBPFFilter && (
          <button
            onClick={handleClearBPFFilter}
            disabled={applyingFilter}
            className="px-3 py-1 rounded-md text-xs font-medium bg-red-600 hover:bg-red-700 disabled:bg-slate-600 disabled:cursor-not-allowed text-white transition-colors"
            title="Clear BPF filter (capture all traffic)"
          >
            Clear
          </button>
        )}

        {currentBPFFilter && (
          <span className="text-xs text-green-400">
            Active: {currentBPFFilter}
          </span>
        )}
      </div>
    </header>
  )
}

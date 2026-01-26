import { Search, Download, Pause, Play, Filter, ChevronDown, Sparkles, HardDrive, Activity, Waves, Trash2, X, Check, AlertTriangle } from 'lucide-react'
import { useState, useRef, useEffect } from 'react'
import { formatBytes } from '../utils'
import { bpfPresets, type BPFPreset } from '../lib/bpfPresets'

interface FilterOptions {
  searchText: string
  showOnlyHTTP: boolean
  showAllPorts: boolean
}

interface HeaderProps {
  connected: boolean
  flowCount: number
  filteredCount: number
  pcapSize: number
  pcapFull: boolean
  filter: string
  onFilterChange: (filter: string) => void
  filterOptions: FilterOptions
  onFilterOptionsChange: (options: FilterOptions) => void
  onDownloadPCAP: () => void
  onClearPCAP: () => void
  isPaused: boolean
  onTogglePause: () => void
}

export function Header({
  connected,
  flowCount,
  filteredCount,
  pcapSize,
  pcapFull,
  filter,
  onFilterChange,
  filterOptions,
  onFilterOptionsChange,
  onDownloadPCAP,
  onClearPCAP,
  isPaused,
  onTogglePause,
}: HeaderProps) {
  const [bpfFilter, setBpfFilter] = useState('')
  const [currentBPFFilter, setCurrentBPFFilter] = useState('')
  const [applyingFilter, setApplyingFilter] = useState(false)
  const [showPresets, setShowPresets] = useState(false)
  const [generatingFilter, setGeneratingFilter] = useState(false)
  const [showFilters, setShowFilters] = useState(false)
  const presetsRef = useRef<HTMLDivElement>(null)

  const anthropicEnabled = Boolean(import.meta.env.VITE_ANTHROPIC_API_KEY)
  const azureEnabled = Boolean(
    import.meta.env.VITE_AZURE_OPENAI_ENDPOINT &&
    import.meta.env.VITE_AZURE_OPENAI_API_KEY &&
    import.meta.env.VITE_AZURE_OPENAI_DEPLOYMENT
  )
  const aiEnabled = anthropicEnabled || azureEnabled
  const aiProvider = anthropicEnabled ? 'anthropic' : 'azure'

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (presetsRef.current && !presetsRef.current.contains(event.target as Node)) {
        setShowPresets(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Fetch current BPF filter from server on mount (persists across refreshes)
  useEffect(() => {
    const fetchCurrentFilter = async () => {
      try {
        const res = await fetch('/api/bpf-filter')
        if (res.ok) {
          const data = await res.json()
          if (data.filter) {
            setCurrentBPFFilter(data.filter)
            setBpfFilter(data.filter)
          }
        }
      } catch (err) {
        console.error('Failed to fetch current BPF filter:', err)
      }
    }
    fetchCurrentFilter()
  }, [])

  const handleSelectPreset = (preset: BPFPreset) => {
    setBpfFilter(preset.filter)
    setShowPresets(false)
  }

  const handleGenerateWithAI = async () => {
    if (!bpfFilter.trim()) return
    setGeneratingFilter(true)

    const systemPrompt = `You are a BPF (Berkeley Packet Filter) expert. Convert the user's natural language description into a valid BPF filter expression.
Rules:
- Output ONLY the BPF filter string, nothing else
- Use standard tcpdump/libpcap BPF syntax`

    try {
      let generatedBPF = ''

      if (aiProvider === 'anthropic') {
        const apiKey = import.meta.env.VITE_ANTHROPIC_API_KEY

        const response = await fetch('/api/ai/anthropic', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            apiKey,
            system: systemPrompt,
            message: bpfFilter,
          }),
        })

        if (!response.ok) throw new Error(`API error: ${response.status}`)
        const data = await response.json()
        generatedBPF = data.content?.trim() || ''
      } else {
        const endpoint = import.meta.env.VITE_AZURE_OPENAI_ENDPOINT
        const apiKey = import.meta.env.VITE_AZURE_OPENAI_API_KEY
        const deployment = import.meta.env.VITE_AZURE_OPENAI_DEPLOYMENT

        const response = await fetch(
          `${endpoint}/openai/deployments/${deployment}/chat/completions?api-version=2024-02-15-preview`,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'api-key': apiKey,
            },
            body: JSON.stringify({
              messages: [
                { role: 'system', content: systemPrompt },
                { role: 'user', content: bpfFilter }
              ],
              max_tokens: 100,
              temperature: 0,
            }),
          }
        )

        if (!response.ok) throw new Error(`API error: ${response.status}`)
        const data = await response.json()
        generatedBPF = data.choices[0]?.message?.content?.trim() || ''
      }

      // Replace the input text with the generated BPF filter
      setBpfFilter(generatedBPF)
    } catch (err) {
      console.error('Error generating BPF filter with AI:', err)
      alert('Failed to generate BPF filter.')
    } finally {
      setGeneratingFilter(false)
    }
  }

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
      } else {
        alert(`Invalid BPF filter: ${data.error}`)
      }
    } catch (err) {
      console.error('Error applying BPF filter:', err)
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
      }
    } catch (err) {
      console.error('Error clearing BPF filter:', err)
    } finally {
      setApplyingFilter(false)
    }
  }

  return (
    <header className="relative z-50 flex-shrink-0">
      {/* Main header bar */}
      <div className="px-6 py-4 flex items-center justify-between gap-6 border-b border-glow-400/10 bg-void-900/80 backdrop-blur-xl">
        {/* Logo */}
        <div className="flex items-center gap-4">
          <div className="relative">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-glow-400 to-glow-600 flex items-center justify-center shadow-glow">
              <Waves className="w-5 h-5 text-void-900" />
            </div>
            {connected && (
              <div className="absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full bg-glow-400 border-2 border-void-900 animate-pulse-glow" />
            )}
          </div>
          <div>
            <h1 className="text-lg font-semibold tracking-tight text-white">
              Pod<span className="text-glow-400">Scope</span>
            </h1>
            <p className="text-[11px] text-gray-500 font-medium tracking-wide uppercase">
              Traffic Observatory
            </p>
          </div>
        </div>

        {/* Search */}
        <div className="flex-1 max-w-2xl">
          <div className="relative group">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500 group-focus-within:text-glow-400 transition-colors" />
            <input
              type="text"
              placeholder="Search flows by IP, pod, service, URL, SNI..."
              value={filter}
              onChange={(e) => onFilterChange(e.target.value)}
              className="w-full bg-void-800/60 border border-void-600 rounded-xl pl-11 pr-4 py-3 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-glow-400/50 focus:shadow-glow-sm transition-all duration-200"
            />
          </div>
        </div>

        {/* Stats & Actions */}
        <div className="flex items-center gap-3">
          {/* Connection Status */}
          <div className={`flex items-center gap-2 px-3 py-2 rounded-lg ${connected ? 'bg-glow-400/10 border border-glow-400/20' : 'bg-status-error/10 border border-status-error/20'}`}>
            <div className={`w-2 h-2 rounded-full ${connected ? 'bg-glow-400 animate-pulse-glow' : 'bg-status-error'}`} />
            <span className={`text-xs font-medium ${connected ? 'text-glow-400' : 'text-status-error'}`}>
              {connected ? 'Live' : 'Offline'}
            </span>
          </div>

          {/* Stats pills */}
          <div className={`flex items-center gap-2 px-3 py-2 rounded-lg ${pcapFull ? 'bg-status-warning/10 border border-status-warning/30' : 'bg-void-800/60 border border-void-700'}`}>
            <Activity className="w-3.5 h-3.5 text-gray-400" />
            <span className="text-xs text-gray-400 font-mono">
              {filteredCount === flowCount ? flowCount : `${filteredCount}/${flowCount}`}
            </span>
            <div className="w-px h-3 bg-void-600" />
            {pcapFull ? (
              <AlertTriangle className="w-3.5 h-3.5 text-status-warning" />
            ) : (
              <HardDrive className="w-3.5 h-3.5 text-gray-400" />
            )}
            <span className={`text-xs font-mono ${pcapFull ? 'text-status-warning' : 'text-gray-400'}`}>
              {pcapFull ? 'Buffer Full' : formatBytes(pcapSize)}
            </span>
          </div>

          {/* Pause */}
          <button
            onClick={onTogglePause}
            className={`btn-ghost ${isPaused ? 'text-ember-400 bg-ember-400/10' : ''}`}
            title={isPaused ? 'Resume capture' : 'Pause capture'}
          >
            {isPaused ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
          </button>

          {/* Filter toggle */}
          <button
            onClick={() => setShowFilters(!showFilters)}
            className={`btn-ghost ${showFilters ? 'text-glow-400 bg-glow-400/10' : ''}`}
            title="Toggle filters"
          >
            <Filter className="w-4 h-4" />
          </button>

          {/* Clear PCAP */}
          <button
            onClick={onClearPCAP}
            className={`btn-ghost ${pcapFull ? 'text-status-warning bg-status-warning/20 animate-pulse' : 'text-status-warning hover:text-status-error hover:bg-status-error/10'}`}
            title={pcapFull ? 'Buffer full - click to reset and continue capturing' : 'Clear PCAP data'}
          >
            <Trash2 className="w-4 h-4" />
          </button>

          {/* Download */}
          <button onClick={onDownloadPCAP} className="btn-primary">
            <Download className="w-4 h-4" />
            <span className="hidden sm:inline">Download</span>
          </button>
        </div>
      </div>

      {/* Filter bar - collapsible */}
      {showFilters && (
        <div className="px-6 py-3 border-b border-glow-400/5 bg-void-900/60 backdrop-blur-xl animate-slide-up">
          {/* Protocol filters */}
          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2">
              <span className="text-xs text-gray-500 uppercase tracking-wider font-medium">Protocol</span>
              <div className="flex items-center gap-1">
                <FilterChip
                  active={filterOptions.showOnlyHTTP}
                  onClick={() => onFilterOptionsChange({ ...filterOptions, showOnlyHTTP: !filterOptions.showOnlyHTTP, showAllPorts: false })}
                >
                  HTTP/S
                </FilterChip>
                <FilterChip
                  active={filterOptions.showAllPorts}
                  onClick={() => onFilterOptionsChange({ ...filterOptions, showAllPorts: !filterOptions.showAllPorts, showOnlyHTTP: false })}
                >
                  All Traffic
                </FilterChip>
              </div>
            </div>

            <div className="h-4 w-px bg-void-700" />

            {/* BPF Filter */}
            <div className="flex items-center gap-2 flex-1">
              <span className="text-xs text-gray-500 uppercase tracking-wider font-medium">BPF</span>

              <div className="relative" ref={presetsRef}>
                <button
                  onClick={() => setShowPresets(!showPresets)}
                  className="btn-ghost text-xs py-1.5"
                >
                  Presets
                  <ChevronDown className="w-3 h-3" />
                </button>
                {showPresets && (
                  <div className="absolute top-full left-0 mt-2 w-72 bg-void-900/95 backdrop-blur-xl border border-void-600 rounded-xl p-1 z-[200] animate-fade-in shadow-xl">
                    {bpfPresets.map((preset, index) => (
                      <button
                        key={index}
                        onClick={() => handleSelectPreset(preset)}
                        className="w-full text-left px-3 py-2.5 rounded-lg hover:bg-glow-400/10 transition-colors"
                      >
                        <div className="text-xs font-medium text-white">{preset.label}</div>
                        <div className="text-[10px] text-gray-500 font-mono truncate">{preset.filter || '(none)'}</div>
                      </button>
                    ))}
                  </div>
                )}
              </div>

              <div className="relative flex-1 max-w-sm">
                <input
                  type="text"
                  placeholder={aiEnabled ? "BPF filter or natural language..." : "tcp port 80 or udp port 53"}
                  value={bpfFilter}
                  onChange={(e) => setBpfFilter(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleApplyBPFFilter()}
                  className={`w-full bg-void-800/60 border border-void-600 rounded-lg px-3 py-1.5 text-xs text-white font-mono placeholder-gray-600 focus:outline-none focus:border-glow-400/50 transition-all ${aiEnabled ? 'pr-9' : ''}`}
                />
                {aiEnabled && (
                  <button
                    onClick={handleGenerateWithAI}
                    disabled={generatingFilter || !bpfFilter.trim()}
                    className="absolute right-1.5 top-1/2 -translate-y-1/2 p-1 rounded text-purple-400 hover:text-purple-300 hover:bg-purple-400/10 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                    title="Generate BPF filter with AI"
                  >
                    <Sparkles className={`w-3.5 h-3.5 ${generatingFilter ? 'animate-pulse' : ''}`} />
                  </button>
                )}
              </div>

              <button
                onClick={handleApplyBPFFilter}
                disabled={applyingFilter || !bpfFilter.trim()}
                className="px-3 py-1.5 rounded-lg text-xs font-medium bg-glow-500/20 text-glow-400 border border-glow-500/30 hover:bg-glow-500/30 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
              >
                {applyingFilter ? 'Applying...' : 'Apply'}
              </button>

              {currentBPFFilter && (
                <div className="flex items-center gap-1.5 px-2 py-1 rounded-lg bg-glow-400/10 border border-glow-400/30 max-w-[280px]">
                  <Check className="w-3 h-3 text-glow-400 flex-shrink-0" />
                  <code className="text-xs text-glow-400 font-mono truncate" title={currentBPFFilter}>
                    {currentBPFFilter}
                  </code>
                  <button
                    onClick={handleClearBPFFilter}
                    className="p-0.5 rounded hover:bg-glow-400/20 text-glow-400/60 hover:text-glow-400 transition-colors flex-shrink-0"
                    title="Clear filter"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </div>
              )}
            </div>
          </div>

        </div>
      )}
    </header>
  )
}

function FilterChip({ children, active, onClick }: { children: React.ReactNode; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all duration-200 ${
        active
          ? 'bg-glow-400/20 text-glow-400 border border-glow-400/30 shadow-glow-sm'
          : 'bg-void-800/60 text-gray-400 border border-void-600 hover:border-glow-400/20 hover:text-gray-300'
      }`}
    >
      {children}
    </button>
  )
}

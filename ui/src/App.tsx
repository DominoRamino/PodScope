import { useState, useEffect, useCallback, useRef, useMemo } from 'react'
import { FlowList } from './components/FlowList'
import { FlowDetail } from './components/FlowDetail'
import { Header } from './components/Header'
import { Terminal } from './components/Terminal'
import { Flow } from './types'
import { mockFlows, generateMockFlow } from './lib/mockData'

// Enable demo mode when not connected to a real hub (set to true for UI development)
const DEMO_MODE = import.meta.env.DEV && !window.location.port?.includes('8')

interface TerminalTarget {
  namespace: string
  podName: string
  container?: string
}

interface FilterOptions {
  searchText: string
  showOnlyHTTP: boolean
  showAllPorts: boolean
}

function App() {
  const [flows, setFlows] = useState<Flow[]>([])
  const [selectedFlow, setSelectedFlow] = useState<Flow | null>(null)
  const [connected, setConnected] = useState(false)
  const [filter, setFilter] = useState('')
  const [filterOptions, setFilterOptions] = useState<FilterOptions>({
    searchText: '',
    showOnlyHTTP: true,
    showAllPorts: false,
  })
  const [stats, setStats] = useState({ flows: 0, wsClients: 0, pcapSize: 0, pcapFull: false, paused: false })
  const [terminalTarget, setTerminalTarget] = useState<TerminalTarget | null>(null)
  const [terminalMaximized, setTerminalMaximized] = useState(false)
  const [downloading, setDownloading] = useState(false)

  const isPausedRef = useRef(false)
  useEffect(() => {
    isPausedRef.current = stats.paused
  }, [stats.paused])

  // WebSocket connection for live updates (or demo mode)
  useEffect(() => {
    if (DEMO_MODE) {
      // Demo mode: load mock data and simulate live updates
      setConnected(true)
      setFlows(mockFlows)
      setStats({ flows: mockFlows.length, wsClients: 1, pcapSize: 245678, pcapFull: false, paused: false })

      // Simulate new flows arriving
      const interval = setInterval(() => {
        if (!isPausedRef.current) {
          const newFlow = generateMockFlow()
          setFlows(prev => [newFlow, ...prev].slice(0, 1000))
          setStats(prev => ({ ...prev, flows: prev.flows + 1, pcapSize: prev.pcapSize + Math.floor(Math.random() * 5000) }))
        }
      }, 3000)

      return () => clearInterval(interval)
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/flows/ws`

    let ws: WebSocket | null = null
    let reconnectTimer: ReturnType<typeof setTimeout>

    const connect = () => {
      ws = new WebSocket(wsUrl)

      ws.onopen = () => {
        console.log('WebSocket connected')
        setConnected(true)
      }

      ws.onmessage = (event) => {
        if (isPausedRef.current) return

        try {
          const message = JSON.parse(event.data)

          if (message.type === 'catchup' || message.type === 'batch') {
            const newFlows = message.flows as Flow[]
            console.log(`Received ${message.type}:`, newFlows.length, 'flows')

            setFlows(prev => {
              const flowMap = new Map(prev.map(f => [f.id, f]))
              for (const flow of newFlows) {
                flowMap.set(flow.id, flow)
              }
              const allFlows = Array.from(flowMap.values())
              allFlows.sort((a, b) =>
                new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
              )
              return allFlows.slice(0, 1000)
            })
          } else {
            const flow = message as Flow
            console.log('Received flow:', flow.id, flow.protocol, flow.srcPort, '->', flow.dstPort)
            setFlows(prev => {
              const existingIndex = prev.findIndex(f => f.id === flow.id)
              if (existingIndex >= 0) {
                const updated = [...prev]
                updated[existingIndex] = flow
                return updated
              }
              return [flow, ...prev].slice(0, 1000)
            })
          }
        } catch (err) {
          console.error('Failed to parse message:', err)
        }
      }

      ws.onclose = () => {
        console.log('WebSocket disconnected')
        setConnected(false)
        reconnectTimer = setTimeout(connect, 2000)
      }

      ws.onerror = (err) => {
        console.error('WebSocket error:', err)
        ws?.close()
      }
    }

    connect()

    return () => {
      clearTimeout(reconnectTimer)
      ws?.close()
    }
  }, [])

  // Fetch stats periodically (skip in demo mode)
  useEffect(() => {
    if (DEMO_MODE) return

    const fetchStats = async () => {
      try {
        const res = await fetch('/api/stats')
        if (res.ok) {
          const data = await res.json()
          setStats(data)
        }
      } catch (err) {
        console.error('Failed to fetch stats:', err)
      }
    }

    fetchStats()
    const interval = setInterval(fetchStats, 5000)
    return () => clearInterval(interval)
  }, [])

  const filteredFlows = useMemo(() => {
    return flows.filter(flow => {
      if (filterOptions.showAllPorts) {
        // Show everything
      } else if (filterOptions.showOnlyHTTP) {
        const isHTTPProtocol = flow.protocol === 'HTTP' ||
                               flow.protocol === 'HTTPS' ||
                               flow.protocol === 'TLS'
        const hasHTTPData = flow.http != null
        if (!isHTTPProtocol && !hasHTTPData) return false
      }

      if (!filter && !filterOptions.searchText) return true
      const searchLower = (filter || filterOptions.searchText).toLowerCase()
      return (
        flow.srcIp?.toLowerCase().includes(searchLower) ||
        flow.dstIp?.toLowerCase().includes(searchLower) ||
        flow.srcPod?.toLowerCase().includes(searchLower) ||
        flow.dstPod?.toLowerCase().includes(searchLower) ||
        flow.dstService?.toLowerCase().includes(searchLower) ||
        flow.protocol?.toLowerCase().includes(searchLower) ||
        flow.http?.url?.toLowerCase().includes(searchLower) ||
        flow.http?.host?.toLowerCase().includes(searchLower) ||
        flow.tls?.sni?.toLowerCase().includes(searchLower)
      )
    })
  }, [flows, filter, filterOptions])

  const parsePodName = (podName: string): { namespace: string; name: string } | null => {
    if (!podName) return null
    const parts = podName.split('/')
    if (parts.length === 2) {
      return { namespace: parts[0], name: parts[1] }
    }
    return { namespace: 'default', name: podName }
  }

  const handleOpenTerminal = useCallback((podName: string) => {
    const parsed = parsePodName(podName)
    if (parsed) {
      setTerminalTarget({ namespace: parsed.namespace, podName: parsed.name })
    }
  }, [])

  const handleTogglePause = useCallback(async () => {
    try {
      const res = await fetch('/api/pause', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ paused: !stats.paused }),
      })
      if (res.ok) {
        const data = await res.json()
        setStats(prev => ({ ...prev, paused: data.paused }))
      }
    } catch (err) {
      console.error('Failed to toggle pause:', err)
    }
  }, [stats.paused])

  const handleDownloadPCAP = useCallback((streamId?: string) => {
    let url = streamId ? `/api/pcap/${streamId}` : '/api/pcap'
    const params = new URLSearchParams()

    if (filterOptions.showOnlyHTTP) params.set('onlyHTTP', 'true')
    if (filterOptions.showAllPorts) params.set('allPorts', 'true')
    if (filter) params.set('search', filter)

    if (params.toString()) {
      url += '?' + params.toString()
    }

    // Show brief loading state for visual feedback
    setDownloading(true)
    setTimeout(() => setDownloading(false), 2000)

    // Use direct navigation so the browser handles the download natively.
    // The server sets Content-Disposition: attachment, so the browser will
    // show this in its download manager with a progress bar.
    const a = document.createElement('a')
    a.href = url
    a.download = ''
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }, [filterOptions, filter])

  const handleClearPCAP = useCallback(async () => {
    try {
      const res = await fetch('/api/pcap/reset', { method: 'POST' })
      if (res.ok) {
        setStats(prev => ({ ...prev, pcapSize: 0, pcapFull: false }))
      }
    } catch (err) {
      console.error('Failed to clear PCAP:', err)
    }
  }, [])

  return (
    <div className="h-screen flex flex-col bg-void-950 text-white overflow-hidden relative">
      {/* Subtle noise overlay */}
      <div className="noise-overlay" />

      {/* Background glow */}
      <div className="glow-bg absolute inset-0 pointer-events-none" />

      {/* Grid pattern */}
      <div className="grid-bg absolute inset-0 pointer-events-none opacity-50" />

      {/* Main content */}
      <div className="relative z-10 flex flex-col h-full">
        <Header
          connected={connected}
          flowCount={flows.length}
          filteredCount={filteredFlows.length}
          pcapSize={stats.pcapSize}
          pcapFull={stats.pcapFull}
          filter={filter}
          onFilterChange={setFilter}
          filterOptions={filterOptions}
          onFilterOptionsChange={setFilterOptions}
          onDownloadPCAP={() => handleDownloadPCAP()}
          onClearPCAP={handleClearPCAP}
          isPaused={stats.paused}
          onTogglePause={handleTogglePause}
          isDownloading={downloading}
        />

        <div className="flex-1 flex min-h-0 overflow-hidden">
          {/* Flow List - Main Panel */}
          <div className={`transition-all duration-300 ease-out ${selectedFlow ? 'w-[55%]' : 'w-full'}`}>
            <FlowList
              flows={filteredFlows}
              selectedId={selectedFlow?.id}
              onSelect={setSelectedFlow}
            />
          </div>

          {/* Flow Detail - Slide-in Panel */}
          {selectedFlow && (
            <div className="w-[45%] border-l border-glow-400/10 animate-slide-in">
              <FlowDetail
                flow={selectedFlow}
                onClose={() => setSelectedFlow(null)}
                onDownloadPCAP={() => handleDownloadPCAP(selectedFlow.id)}
                onOpenTerminal={handleOpenTerminal}
                isDownloading={downloading}
              />
            </div>
          )}
        </div>

        {/* Terminal Panel */}
        {terminalTarget && (
          <div className={`${terminalMaximized ? 'fixed inset-0 z-50' : 'h-80 flex-shrink-0'} border-t border-glow-400/10`}>
            <Terminal
              namespace={terminalTarget.namespace}
              podName={terminalTarget.podName}
              container={terminalTarget.container}
              onClose={() => {
                setTerminalTarget(null)
                setTerminalMaximized(false)
              }}
              isMaximized={terminalMaximized}
              onToggleMaximize={() => setTerminalMaximized(m => !m)}
            />
          </div>
        )}
      </div>
    </div>
  )
}

export default App

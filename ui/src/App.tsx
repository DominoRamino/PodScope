import { useState, useEffect, useCallback, useRef, useMemo } from 'react'
import { FlowList } from './components/FlowList'
import { FlowDetail } from './components/FlowDetail'
import { Header } from './components/Header'
import { Terminal } from './components/Terminal'
import { Flow } from './types'

interface TerminalTarget {
  namespace: string
  podName: string
  container?: string
}

interface FilterOptions {
  searchText: string
  showOnlyHTTP: boolean
  showDNS: boolean
  showAllPorts: boolean
}

function App() {
  const [flows, setFlows] = useState<Flow[]>([])
  const [selectedFlow, setSelectedFlow] = useState<Flow | null>(null)
  const [connected, setConnected] = useState(false)
  const [filter, setFilter] = useState('')
  const [filterOptions, setFilterOptions] = useState<FilterOptions>({
    searchText: '',
    showOnlyHTTP: true, // Default: show only HTTP/HTTPS traffic
    showDNS: false,
    showAllPorts: false,
  })
  const [stats, setStats] = useState({ flows: 0, wsClients: 0, pcapSize: 0, paused: false })
  const [terminalTarget, setTerminalTarget] = useState<TerminalTarget | null>(null)
  const [terminalMaximized, setTerminalMaximized] = useState(false)

  // Use ref for pause state to avoid WebSocket reconnection on state change
  const isPausedRef = useRef(false)
  useEffect(() => {
    isPausedRef.current = stats.paused
  }, [stats.paused])

  // WebSocket connection for live updates
  useEffect(() => {
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
        // Skip processing when paused (WebSocket stays connected)
        if (isPausedRef.current) return

        try {
          const message = JSON.parse(event.data)

          // Handle batch and catchup messages from Hub
          if (message.type === 'catchup' || message.type === 'batch') {
            const newFlows = message.flows as Flow[]
            console.log(`Received ${message.type}:`, newFlows.length, 'flows')

            setFlows(prev => {
              // Create a map for O(1) lookups
              const flowMap = new Map(prev.map(f => [f.id, f]))

              // Update or add each flow
              for (const flow of newFlows) {
                flowMap.set(flow.id, flow)
              }

              // Convert back to array and sort by timestamp (newest first)
              const allFlows = Array.from(flowMap.values())
              allFlows.sort((a, b) =>
                new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
              )
              return allFlows.slice(0, 1000) // Keep max 1000 flows
            })
          } else {
            // Legacy single flow message (backward compatibility)
            const flow = message as Flow
            console.log('Received flow:', flow.id, flow.protocol, flow.srcPort, '->', flow.dstPort)
            setFlows(prev => {
              // Check if flow already exists (update) or new
              const existingIndex = prev.findIndex(f => f.id === flow.id)
              if (existingIndex >= 0) {
                const updated = [...prev]
                updated[existingIndex] = flow
                return updated
              }
              return [flow, ...prev].slice(0, 1000) // Keep max 1000 flows
            })
          }
        } catch (err) {
          console.error('Failed to parse message:', err)
        }
      }

      ws.onclose = () => {
        console.log('WebSocket disconnected')
        setConnected(false)
        // Reconnect after 2 seconds
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

  // Fetch stats periodically
  useEffect(() => {
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

  const DNS_PORT = 53

  // Memoized filtered flows - only recalculates when dependencies change
  const filteredFlows = useMemo(() => {
    return flows.filter(flow => {
      // Protocol/Port filtering
      if (filterOptions.showAllPorts) {
        // Show everything (no port filtering)
      } else if (filterOptions.showOnlyHTTP) {
        // Show only flows with detected HTTP/HTTPS/TLS protocol or parsed HTTP data
        const isHTTPProtocol = flow.protocol === 'HTTP' ||
                               flow.protocol === 'HTTPS' ||
                               flow.protocol === 'TLS'
        const hasHTTPData = flow.http != null

        if (!isHTTPProtocol && !hasHTTPData) return false
      }

      // DNS filtering (skip if showAllPorts is enabled)
      if (!filterOptions.showAllPorts && !filterOptions.showDNS) {
        const isDNS = flow.srcPort === DNS_PORT || flow.dstPort === DNS_PORT
        if (isDNS) return false
      }

      // Text search filter
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

  // Parse pod name to extract namespace and pod
  const parsePodName = (podName: string): { namespace: string; name: string } | null => {
    // Pod names are typically in format "namespace/podname" or just "podname"
    if (!podName) return null
    const parts = podName.split('/')
    if (parts.length === 2) {
      return { namespace: parts[0], name: parts[1] }
    }
    // Default to 'default' namespace if not specified
    return { namespace: 'default', name: podName }
  }

  const handleOpenTerminal = useCallback((podName: string) => {
    const parsed = parsePodName(podName)
    if (parsed) {
      setTerminalTarget({ namespace: parsed.namespace, podName: parsed.name })
    }
  }, [])

  // Toggle pause state - calls the hub API to pause/resume PCAP capture
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

  const handleDownloadPCAP = useCallback(async (streamId?: string) => {
    try {
      // Build URL with filter parameters
      let url = streamId ? `/api/pcap/${streamId}` : '/api/pcap'
      const params = new URLSearchParams()

      // Add filter parameters
      if (filterOptions.showOnlyHTTP) params.set('onlyHTTP', 'true')
      if (filterOptions.showDNS) params.set('includeDNS', 'true')
      if (filterOptions.showAllPorts) params.set('allPorts', 'true')
      if (filter) params.set('search', filter)

      if (params.toString()) {
        url += '?' + params.toString()
      }

      const res = await fetch(url)
      if (!res.ok) throw new Error('Download failed')

      const blob = await res.blob()
      const downloadUrl = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = downloadUrl
      const filterSuffix = filterOptions.showOnlyHTTP ? '-http' : filterOptions.showAllPorts ? '-all' : ''
      a.download = streamId ? `stream-${streamId}${filterSuffix}.pcap` : `podscope-session${filterSuffix}.pcap`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(downloadUrl)
    } catch (err) {
      console.error('Failed to download PCAP:', err)
    }
  }, [filterOptions, filter])

  return (
    <div className="h-screen flex flex-col bg-slate-900 text-white overflow-hidden">
      <Header
        connected={connected}
        flowCount={flows.length}
        pcapSize={stats.pcapSize}
        filter={filter}
        onFilterChange={setFilter}
        filterOptions={filterOptions}
        onFilterOptionsChange={setFilterOptions}
        onDownloadPCAP={() => handleDownloadPCAP()}
        isPaused={stats.paused}
        onTogglePause={handleTogglePause}
      />

      <div className="flex-1 flex min-h-0 overflow-hidden">
        {/* Flow List - Left Panel */}
        <div className={`${selectedFlow ? 'w-1/2' : 'w-full'} border-r border-slate-700 overflow-hidden`}>
          <FlowList
            flows={filteredFlows}
            selectedId={selectedFlow?.id}
            onSelect={setSelectedFlow}
          />
        </div>

        {/* Flow Detail - Right Panel */}
        {selectedFlow && (
          <div className="w-1/2 overflow-hidden">
            <FlowDetail
              flow={selectedFlow}
              onClose={() => setSelectedFlow(null)}
              onDownloadPCAP={() => handleDownloadPCAP(selectedFlow.id)}
              onOpenTerminal={handleOpenTerminal}
            />
          </div>
        )}
      </div>

      {/* Terminal Panel */}
      {terminalTarget && (
        <div className={`${terminalMaximized ? 'fixed inset-0 z-50' : 'h-80 flex-shrink-0'}`}>
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
  )
}

export default App

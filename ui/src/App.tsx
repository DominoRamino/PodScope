import { useState, useEffect, useCallback, useRef } from 'react'
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

function App() {
  const [flows, setFlows] = useState<Flow[]>([])
  const [selectedFlow, setSelectedFlow] = useState<Flow | null>(null)
  const [connected, setConnected] = useState(false)
  const [filter, setFilter] = useState('')
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
          const flow = JSON.parse(event.data) as Flow
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
        } catch (err) {
          console.error('Failed to parse flow:', err)
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

  // Filter flows
  const filteredFlows = flows.filter(flow => {
    if (!filter) return true
    const searchLower = filter.toLowerCase()
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
      const url = streamId ? `/api/pcap/${streamId}` : '/api/pcap'
      const res = await fetch(url)
      if (!res.ok) throw new Error('Download failed')

      const blob = await res.blob()
      const downloadUrl = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = downloadUrl
      a.download = streamId ? `stream-${streamId}.pcap` : 'podscope-session.pcap'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(downloadUrl)
    } catch (err) {
      console.error('Failed to download PCAP:', err)
    }
  }, [])

  return (
    <div className="h-screen flex flex-col bg-slate-900 text-white">
      <Header
        connected={connected}
        flowCount={flows.length}
        pcapSize={stats.pcapSize}
        filter={filter}
        onFilterChange={setFilter}
        onDownloadPCAP={() => handleDownloadPCAP()}
        isPaused={stats.paused}
        onTogglePause={handleTogglePause}
      />

      <div className={`flex-1 flex overflow-hidden ${terminalTarget && !terminalMaximized ? 'h-[calc(100%-20rem)]' : ''}`}>
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
        <div className={`${terminalMaximized ? 'fixed inset-0 z-50' : 'h-80'}`}>
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

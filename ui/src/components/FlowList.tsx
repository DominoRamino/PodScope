import { Flow } from '../types'
import { ArrowRight, Lock, Radar, Globe, Server } from 'lucide-react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { useRef, memo } from 'react'
import { formatBytes, formatTime } from '../utils'

interface FlowListProps {
  flows: Flow[]
  selectedId?: string
  onSelect: (flow: Flow) => void
}

export function FlowList({ flows, selectedId, onSelect }: FlowListProps) {
  const parentRef = useRef<HTMLDivElement>(null)

  const rowVirtualizer = useVirtualizer({
    count: flows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 64,
    overscan: 10,
  })

  return (
    <div className="h-full flex flex-col">
      {/* Table Header */}
      <div className="bg-void-900/80 backdrop-blur-xl border-b border-glow-400/5 px-6 py-3 grid grid-cols-12 gap-4 text-[11px] font-semibold text-gray-500 uppercase tracking-wider flex-shrink-0">
        <div className="col-span-2">Timestamp</div>
        <div className="col-span-3">Source</div>
        <div className="col-span-3">Destination</div>
        <div className="col-span-1 text-center">Protocol</div>
        <div className="col-span-1 text-center">Status</div>
        <div className="col-span-1 text-right">Latency</div>
        <div className="col-span-1 text-right">Size</div>
      </div>

      {/* Virtualized Flow Rows */}
      <div ref={parentRef} className="flex-1 overflow-y-auto">
        {flows.length === 0 ? (
          <EmptyState />
        ) : (
          <div
            style={{
              height: `${rowVirtualizer.getTotalSize()}px`,
              width: '100%',
              position: 'relative',
            }}
          >
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const flow = flows[virtualRow.index]
              return (
                <div
                  key={flow.id}
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    width: '100%',
                    height: `${virtualRow.size}px`,
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                >
                  <FlowRowMemo
                    flow={flow}
                    selected={flow.id === selectedId}
                    onClick={() => onSelect(flow)}
                    index={virtualRow.index}
                  />
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

function EmptyState() {
  return (
    <div className="flex items-center justify-center h-full">
      <div className="text-center max-w-md px-8">
        <div className="w-20 h-20 mx-auto mb-6 rounded-2xl bg-void-800/60 border border-glow-400/10 flex items-center justify-center">
          <Radar className="w-10 h-10 text-glow-400/40" />
        </div>
        <h3 className="text-lg font-medium text-white mb-2">Awaiting Traffic</h3>
        <p className="text-sm text-gray-500 leading-relaxed">
          Network flows will appear here in real-time as they're captured from your Kubernetes pods.
        </p>
        <div className="mt-6 flex items-center justify-center gap-2 text-xs text-gray-600">
          <div className="w-1.5 h-1.5 rounded-full bg-glow-400/50 animate-pulse-glow" />
          <span>Listening for connections...</span>
        </div>
      </div>
    </div>
  )
}

interface FlowRowProps {
  flow: Flow
  selected: boolean
  onClick: () => void
  index: number
}

const FlowRowMemo = memo(function FlowRow({ flow, selected, onClick, index }: FlowRowProps) {
  const getDisplayStatus = (): string => {
    if (flow.http?.statusCode) {
      return `${flow.http.statusCode}`
    }
    return flow.status
  }

  const getLatency = (): string => {
    if (flow.ttfbMs) return `${flow.ttfbMs.toFixed(0)}ms`
    if (flow.tcpHandshakeMs) return `${flow.tcpHandshakeMs.toFixed(0)}ms`
    return '-'
  }

  const getDestination = (): string => {
    if (flow.http?.host) return flow.http.host
    if (flow.tls?.sni) return flow.tls.sni
    if (flow.dstService) return flow.dstService
    if (flow.dstPod) return flow.dstPod
    return `${flow.dstIp}:${flow.dstPort}`
  }

  const getSource = (): string => {
    if (flow.srcPod) return flow.srcPod
    return `${flow.srcIp}:${flow.srcPort}`
  }

  const isEncrypted = flow.protocol === 'HTTPS' || flow.protocol === 'TLS'
  const totalBytes = flow.bytesSent + flow.bytesReceived

  const getProtocolStyle = () => {
    switch (flow.protocol) {
      case 'HTTP':
        return 'protocol-http'
      case 'HTTPS':
      case 'TLS':
        return 'protocol-https'
      default:
        return 'protocol-tcp'
    }
  }

  const getStatusStyle = () => {
    if (flow.http?.statusCode) {
      const code = flow.http.statusCode
      if (code >= 200 && code < 300) return 'text-status-success'
      if (code >= 300 && code < 400) return 'text-status-info'
      if (code >= 400 && code < 500) return 'text-status-warning'
      return 'text-status-error'
    }
    switch (flow.status) {
      case 'CLOSED': return 'text-status-success'
      case 'RESET': return 'text-status-error'
      case 'TIMEOUT': return 'text-status-warning'
      default: return 'text-status-info'
    }
  }

  return (
    <div
      onClick={onClick}
      className={`
        row-glow px-6 py-3 grid grid-cols-12 gap-4 text-sm cursor-pointer border-b border-void-800/50
        transition-all duration-150 h-full items-center
        ${selected
          ? 'bg-glow-400/5 border-l-2 border-l-glow-400'
          : 'hover:bg-void-800/30 border-l-2 border-l-transparent'
        }
      `}
      style={{ animationDelay: `${index * 0.02}s` }}
    >
      {/* Timestamp */}
      <div className="col-span-2 font-mono text-xs text-gray-400">
        {formatTime(flow.timestamp)}
      </div>

      {/* Source */}
      <div className="col-span-3 min-w-0">
        <div className="flex items-center gap-2">
          <div className="w-5 h-5 rounded-md bg-void-700/80 flex items-center justify-center flex-shrink-0">
            <Server className="w-3 h-3 text-gray-500" />
          </div>
          <div className="min-w-0">
            <div className="text-sm text-gray-200 truncate">{getSource()}</div>
            {flow.srcNamespace && (
              <div className="text-[10px] text-gray-600 truncate">{flow.srcNamespace}</div>
            )}
          </div>
        </div>
      </div>

      {/* Destination */}
      <div className="col-span-3 min-w-0">
        <div className="flex items-center gap-2">
          <ArrowRight className="w-3 h-3 text-glow-400/40 flex-shrink-0" />
          {isEncrypted ? (
            <div className="w-5 h-5 rounded-md bg-amber-500/10 flex items-center justify-center flex-shrink-0">
              <Lock className="w-3 h-3 text-amber-400" />
            </div>
          ) : (
            <div className="w-5 h-5 rounded-md bg-void-700/80 flex items-center justify-center flex-shrink-0">
              <Globe className="w-3 h-3 text-gray-500" />
            </div>
          )}
          <div className="min-w-0">
            <div className="text-sm text-gray-200 truncate">{getDestination()}</div>
            {flow.http?.url && flow.http.url !== '/' && (
              <div className="text-[10px] text-gray-600 truncate">
                <span className="text-glow-400/60">{flow.http.method}</span> {flow.http.url}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Protocol */}
      <div className="col-span-1 flex justify-center">
        <span className={`status-badge ${getProtocolStyle()}`}>
          {flow.protocol}
        </span>
      </div>

      {/* Status */}
      <div className={`col-span-1 text-center font-mono text-xs font-medium ${getStatusStyle()}`}>
        {getDisplayStatus()}
      </div>

      {/* Latency */}
      <div className="col-span-1 text-right font-mono text-xs text-gray-400">
        {getLatency()}
      </div>

      {/* Size */}
      <div className="col-span-1 text-right font-mono text-xs text-gray-400">
        {formatBytes(totalBytes)}
      </div>
    </div>
  )
}, (prev, next) => {
  return prev.flow.id === next.flow.id &&
         prev.flow.status === next.flow.status &&
         prev.flow.bytesSent === next.flow.bytesSent &&
         prev.flow.bytesReceived === next.flow.bytesReceived &&
         prev.selected === next.selected
})

import { Flow, Protocol } from '../types'
import { ArrowRight, Lock, Server } from 'lucide-react'

interface FlowListProps {
  flows: Flow[]
  selectedId?: string
  onSelect: (flow: Flow) => void
}

export function FlowList({ flows, selectedId, onSelect }: FlowListProps) {
  return (
    <div className="h-full flex flex-col">
      {/* Table Header */}
      <div className="bg-slate-800 border-b border-slate-700 px-4 py-2 grid grid-cols-12 gap-2 text-xs font-medium text-slate-400 uppercase tracking-wider">
        <div className="col-span-2">Time</div>
        <div className="col-span-3">Source</div>
        <div className="col-span-3">Destination</div>
        <div className="col-span-1">Protocol</div>
        <div className="col-span-1">Status</div>
        <div className="col-span-1">Latency</div>
        <div className="col-span-1">Size</div>
      </div>

      {/* Flow Rows */}
      <div className="flex-1 overflow-y-auto">
        {flows.length === 0 ? (
          <div className="flex items-center justify-center h-full text-slate-500">
            <div className="text-center">
              <Server className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>No flows captured yet</p>
              <p className="text-sm mt-1">Traffic will appear here in real-time</p>
            </div>
          </div>
        ) : (
          flows.map((flow) => (
            <FlowRow
              key={flow.id}
              flow={flow}
              selected={flow.id === selectedId}
              onClick={() => onSelect(flow)}
            />
          ))
        )}
      </div>
    </div>
  )
}

interface FlowRowProps {
  flow: Flow
  selected: boolean
  onClick: () => void
}

function FlowRow({ flow, selected, onClick }: FlowRowProps) {
  const formatTime = (timestamp: string): string => {
    const date = new Date(timestamp)
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }) + '.' + String(date.getMilliseconds()).padStart(3, '0')
  }

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  }

  const getProtocolColor = (protocol: Protocol): string => {
    switch (protocol) {
      case 'HTTP':
        return 'text-green-400 bg-green-400/10'
      case 'HTTPS':
      case 'TLS':
        return 'text-yellow-400 bg-yellow-400/10'
      default:
        return 'text-blue-400 bg-blue-400/10'
    }
  }

  const getStatusColor = (status: string, httpCode?: number): string => {
    if (httpCode) {
      if (httpCode >= 200 && httpCode < 300) return 'text-green-400'
      if (httpCode >= 300 && httpCode < 400) return 'text-blue-400'
      if (httpCode >= 400 && httpCode < 500) return 'text-yellow-400'
      if (httpCode >= 500) return 'text-red-400'
    }
    switch (status) {
      case 'CLOSED':
        return 'text-green-400'
      case 'RESET':
        return 'text-red-400'
      case 'TIMEOUT':
        return 'text-yellow-400'
      default:
        return 'text-blue-400'
    }
  }

  const getDisplayStatus = (): string => {
    if (flow.http?.statusCode) {
      return `${flow.http.statusCode}`
    }
    return flow.status
  }

  const getLatency = (): string => {
    if (flow.ttfbMs) return `${flow.ttfbMs.toFixed(1)}ms`
    if (flow.tcpHandshakeMs) return `${flow.tcpHandshakeMs.toFixed(1)}ms`
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

  return (
    <div
      onClick={onClick}
      className={`
        px-4 py-2 grid grid-cols-12 gap-2 text-sm cursor-pointer border-b border-slate-700/50
        hover:bg-slate-800/50 transition-colors
        ${selected ? 'bg-podscope-900/30 border-l-2 border-l-podscope-500' : ''}
      `}
    >
      {/* Time */}
      <div className="col-span-2 text-slate-400 font-mono text-xs">
        {formatTime(flow.timestamp)}
      </div>

      {/* Source */}
      <div className="col-span-3 truncate text-slate-300">
        <div className="flex items-center gap-1">
          <span className="truncate">{getSource()}</span>
        </div>
        {flow.srcNamespace && (
          <div className="text-xs text-slate-500 truncate">{flow.srcNamespace}</div>
        )}
      </div>

      {/* Destination */}
      <div className="col-span-3 truncate">
        <div className="flex items-center gap-1">
          <ArrowRight className="w-3 h-3 text-slate-500 flex-shrink-0" />
          {isEncrypted && <Lock className="w-3 h-3 text-yellow-500 flex-shrink-0" />}
          <span className="truncate text-slate-300">{getDestination()}</span>
        </div>
        {flow.http?.url && flow.http.url !== '/' && (
          <div className="text-xs text-slate-500 truncate pl-4">
            {flow.http.method} {flow.http.url}
          </div>
        )}
      </div>

      {/* Protocol */}
      <div className="col-span-1">
        <span className={`px-2 py-0.5 rounded text-xs font-medium ${getProtocolColor(flow.protocol)}`}>
          {flow.protocol}
        </span>
      </div>

      {/* Status */}
      <div className={`col-span-1 font-medium ${getStatusColor(flow.status, flow.http?.statusCode)}`}>
        {getDisplayStatus()}
      </div>

      {/* Latency */}
      <div className="col-span-1 text-slate-400">
        {getLatency()}
      </div>

      {/* Size */}
      <div className="col-span-1 text-slate-400">
        {formatBytes(totalBytes)}
      </div>
    </div>
  )
}

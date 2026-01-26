import { useState } from 'react'
import { Flow } from '../types'
import { X, Download, ArrowRight, Lock, Terminal, Clock, Send, Inbox, Shield, Globe, Server, ChevronDown, Activity } from 'lucide-react'
import { formatBytes } from '../utils'

interface FlowDetailProps {
  flow: Flow
  onClose: () => void
  onDownloadPCAP: () => void
  onOpenTerminal?: (podName: string) => void
}

export function FlowDetail({ flow, onClose, onDownloadPCAP, onOpenTerminal }: FlowDetailProps) {
  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp)
    return date.toLocaleString()
  }

  return (
    <div className="h-full flex flex-col bg-void-900/60 backdrop-blur-xl">
      {/* Header */}
      <div className="px-6 py-4 flex items-center justify-between border-b border-glow-400/10">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-glow-400/10 flex items-center justify-center">
            <Globe className="w-4 h-4 text-glow-400" />
          </div>
          <div>
            <h2 className="font-semibold text-white">Flow Details</h2>
            <span className="text-[10px] text-gray-500 font-mono">{flow.id}</span>
          </div>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={onDownloadPCAP}
            className="btn-ghost"
            title="Download PCAP"
          >
            <Download className="w-4 h-4" />
          </button>
          <button onClick={onClose} className="btn-ghost">
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {/* Quick Stats */}
        <div className="grid grid-cols-4 gap-3">
          <StatCard
            label="Protocol"
            value={flow.protocol}
            accent={flow.protocol === 'HTTP' ? 'emerald' : flow.protocol === 'HTTPS' || flow.protocol === 'TLS' ? 'amber' : 'blue'}
          />
          <StatCard label="Status" value={flow.status} accent={flow.status === 'CLOSED' ? 'emerald' : flow.status === 'RESET' ? 'red' : 'blue'} />
          <StatCard label="Duration" value={`${flow.duration}ms`} />
          <StatCard label="Total" value={formatBytes(flow.bytesSent + flow.bytesReceived)} />
        </div>

        {/* Connection Flow */}
        <Section title="Connection" icon={<ArrowRight className="w-4 h-4" />}>
          <div className="flex items-stretch gap-4">
            <EndpointCard
              type="source"
              ip={flow.srcIp}
              port={flow.srcPort}
              pod={flow.srcPod}
              namespace={flow.srcNamespace}
              onOpenTerminal={onOpenTerminal}
            />
            <div className="flex flex-col items-center justify-center py-4">
              <div className="w-8 h-8 rounded-full bg-glow-400/10 flex items-center justify-center">
                <ArrowRight className="w-4 h-4 text-glow-400" />
              </div>
              <div className="flex-1 w-px bg-gradient-to-b from-glow-400/30 via-glow-400/10 to-transparent mt-2" />
            </div>
            <EndpointCard
              type="destination"
              ip={flow.dstIp}
              port={flow.dstPort}
              pod={flow.dstPod}
              namespace={flow.dstNamespace}
              service={flow.dstService}
              onOpenTerminal={onOpenTerminal}
            />
          </div>
        </Section>

        {/* Timing */}
        <Section title="Timing" icon={<Clock className="w-4 h-4" />}>
          <div className="text-xs text-gray-500 mb-3">{formatTimestamp(flow.timestamp)}</div>
          <TimingBar flow={flow} />
        </Section>

        {/* Data Transfer */}
        <Section title="Data Transfer" icon={<Send className="w-4 h-4" />}>
          <div className="grid grid-cols-2 gap-3">
            <div className="glass-card p-4">
              <div className="flex items-center gap-2 mb-2">
                <Send className="w-3.5 h-3.5 text-status-success" />
                <span className="text-xs text-gray-500 uppercase tracking-wider">Sent</span>
              </div>
              <div className="text-xl font-semibold text-status-success">{formatBytes(flow.bytesSent)}</div>
              <div className="text-xs text-gray-600 mt-1">{flow.packetsSent} packets</div>
            </div>
            <div className="glass-card p-4">
              <div className="flex items-center gap-2 mb-2">
                <Inbox className="w-3.5 h-3.5 text-status-info" />
                <span className="text-xs text-gray-500 uppercase tracking-wider">Received</span>
              </div>
              <div className="text-xl font-semibold text-status-info">{formatBytes(flow.bytesReceived)}</div>
              <div className="text-xs text-gray-600 mt-1">{flow.packetsReceived} packets</div>
            </div>
          </div>
        </Section>

        {/* Advanced Metrics */}
        <CollapsibleSection title="Advanced Metrics" icon={<Activity className="w-4 h-4" />}>
          <div className="grid grid-cols-3 gap-4">
            <MetricItem
              label="Time to First Byte"
              value={flow.ttfbMs ? `${flow.ttfbMs.toFixed(1)}ms` : 'N/A'}
              indicator={getTTFBIndicator(flow.ttfbMs)}
            />
            <MetricItem
              label="Throughput"
              value={calculateThroughput(flow.bytesSent, flow.bytesReceived, flow.duration)}
            />
            <ProtocolVersionBadge version={getProtocolVersionFromALPN(flow.tls?.alpn)} />
          </div>
        </CollapsibleSection>

        {/* TLS Info */}
        {flow.tls && (
          <Section title="TLS / Encryption" icon={<Shield className="w-4 h-4" />}>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <InfoItem label="Version" value={flow.tls.version} />
                {flow.tls.sni && <InfoItem label="SNI" value={flow.tls.sni} />}
              </div>
              {flow.tls.cipherSuite && <InfoItem label="Cipher Suite" value={flow.tls.cipherSuite} />}
              {flow.tls.alpn && flow.tls.alpn.length > 0 && (
                <InfoItem label="ALPN" value={flow.tls.alpn.join(', ')} />
              )}
              <div className="flex items-center gap-3 p-3 rounded-lg bg-amber-500/5 border border-amber-500/20">
                <Lock className="w-4 h-4 text-amber-400" />
                <span className="text-xs text-amber-300">Encrypted payload - only metadata visible</span>
              </div>
            </div>
          </Section>
        )}

        {/* HTTP Info */}
        {flow.http && (
          <>
            <Section title="HTTP Request" icon={<Send className="w-4 h-4" />}>
              <div className="space-y-4">
                <div className="glass-card p-3 flex items-center gap-3">
                  <span className="px-2.5 py-1 rounded-md text-xs font-semibold bg-glow-400/20 text-glow-400 border border-glow-400/30">
                    {flow.http.method}
                  </span>
                  <span className="text-sm text-gray-200 font-mono truncate">{flow.http.url}</span>
                </div>
                {flow.http.host && <InfoItem label="Host" value={flow.http.host} />}
                {flow.http.requestHeaders && Object.keys(flow.http.requestHeaders).length > 0 && (
                  <HeadersTable headers={flow.http.requestHeaders} title="Headers" />
                )}
                {flow.http.requestBody && (
                  <CodeBlock title="Request Body" content={flow.http.requestBody} />
                )}
              </div>
            </Section>

            <Section title="HTTP Response" icon={<Inbox className="w-4 h-4" />}>
              <div className="space-y-4">
                <div className="glass-card p-3">
                  <StatusBadge code={flow.http.statusCode} text={flow.http.statusText} />
                </div>
                {flow.http.contentType && <InfoItem label="Content-Type" value={flow.http.contentType} />}
                {flow.http.contentLength !== undefined && (
                  <InfoItem label="Content-Length" value={formatBytes(flow.http.contentLength)} />
                )}
                {flow.http.responseHeaders && Object.keys(flow.http.responseHeaders).length > 0 && (
                  <HeadersTable headers={flow.http.responseHeaders} title="Headers" />
                )}
                {flow.http.responseBody && (
                  <CodeBlock title="Response Body (truncated)" content={flow.http.responseBody} />
                )}
              </div>
            </Section>
          </>
        )}
      </div>
    </div>
  )
}

function Section({ title, icon, children }: { title: string; icon?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="animate-fade-in">
      <div className="flex items-center gap-2 mb-4">
        {icon && <span className="text-glow-400/60">{icon}</span>}
        <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wider">{title}</h3>
      </div>
      {children}
    </div>
  )
}

function CollapsibleSection({ title, icon, children, defaultOpen = true }: { title: string; icon?: React.ReactNode; children: React.ReactNode; defaultOpen?: boolean }) {
  const [isOpen, setIsOpen] = useState(defaultOpen)

  return (
    <div className="animate-fade-in glass-card overflow-hidden">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center justify-between p-4 hover:bg-void-700/30 transition-colors"
      >
        <div className="flex items-center gap-2">
          {icon && <span className="text-glow-400/60">{icon}</span>}
          <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wider">{title}</h3>
        </div>
        <ChevronDown className={`w-4 h-4 text-gray-500 transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`} />
      </button>
      {isOpen && (
        <div className="p-4 pt-0 border-t border-void-700/30">
          {children}
        </div>
      )}
    </div>
  )
}

function StatCard({ label, value, accent }: { label: string; value: string | number; accent?: string }) {
  const getAccentClass = () => {
    switch (accent) {
      case 'emerald': return 'text-emerald-400'
      case 'amber': return 'text-amber-400'
      case 'red': return 'text-status-error'
      case 'blue': return 'text-status-info'
      default: return 'text-white'
    }
  }

  return (
    <div className="glass-card p-3 text-center">
      <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-1">{label}</div>
      <div className={`text-sm font-semibold ${getAccentClass()}`}>{value}</div>
    </div>
  )
}

function EndpointCard({
  type,
  ip,
  port,
  pod,
  namespace,
  service,
  onOpenTerminal
}: {
  type: 'source' | 'destination'
  ip: string
  port: number
  pod?: string
  namespace?: string
  service?: string
  onOpenTerminal?: (podName: string) => void
}) {
  return (
    <div className="flex-1 glass-card p-4">
      <div className="flex items-center gap-2 mb-3">
        <div className={`w-6 h-6 rounded-md flex items-center justify-center ${type === 'source' ? 'bg-glow-400/10' : 'bg-status-info/10'}`}>
          <Server className={`w-3.5 h-3.5 ${type === 'source' ? 'text-glow-400' : 'text-status-info'}`} />
        </div>
        <span className="text-[10px] text-gray-500 uppercase tracking-wider">
          {type === 'source' ? 'Source' : 'Destination'}
        </span>
      </div>

      <div className="font-mono text-sm text-white mb-2">{ip}:{port}</div>

      {pod && (
        <div className="flex items-center gap-2 mt-3">
          <span className="text-xs text-gray-400 truncate">{namespace}/{pod}</span>
          {onOpenTerminal && (
            <button
              onClick={() => onOpenTerminal(`${namespace}/${pod}`)}
              className="p-1.5 rounded-md hover:bg-glow-400/10 transition-colors"
              title="Open terminal"
            >
              <Terminal className="w-3.5 h-3.5 text-gray-500 hover:text-glow-400" />
            </button>
          )}
        </div>
      )}

      {service && (
        <div className="text-xs text-glow-400 mt-2 truncate">{service}</div>
      )}
    </div>
  )
}

function InfoItem({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex justify-between items-center py-2 border-b border-void-700/50">
      <span className="text-xs text-gray-500">{label}</span>
      <span className="text-sm text-white font-mono truncate max-w-[60%]">{value}</span>
    </div>
  )
}

function HeadersTable({ headers, title }: { headers: Record<string, string>; title: string }) {
  return (
    <div>
      <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-2">{title}</div>
      <div className="glass-card overflow-hidden">
        {Object.entries(headers).map(([key, value]) => (
          <div key={key} className="flex border-b border-void-700/30 last:border-0">
            <div className="w-1/3 px-3 py-2 text-[11px] text-gray-400 bg-void-800/30 font-mono truncate">
              {key}
            </div>
            <div className="flex-1 px-3 py-2 text-[11px] text-gray-200 font-mono break-all">
              {value}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function CodeBlock({ title, content }: { title: string; content: string }) {
  return (
    <div>
      <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-2">{title}</div>
      <pre className="glass-card p-3 text-xs font-mono text-gray-300 overflow-x-auto max-h-48 whitespace-pre-wrap break-all">
        {content}
      </pre>
    </div>
  )
}

function StatusBadge({ code, text }: { code: number; text: string }) {
  const getStyle = () => {
    if (code >= 200 && code < 300) return 'text-status-success bg-status-success/10 border-status-success/30'
    if (code >= 300 && code < 400) return 'text-status-info bg-status-info/10 border-status-info/30'
    if (code >= 400 && code < 500) return 'text-status-warning bg-status-warning/10 border-status-warning/30'
    return 'text-status-error bg-status-error/10 border-status-error/30'
  }

  return (
    <span className={`inline-flex items-center gap-2 px-3 py-1.5 rounded-lg font-mono text-sm border ${getStyle()}`}>
      <span className="font-semibold">{code}</span>
      <span className="text-xs opacity-80">{text}</span>
    </span>
  )
}

function getTTFBIndicator(ttfbMs: number | undefined): 'success' | 'warning' | 'error' | null {
  if (!ttfbMs || ttfbMs === 0) return null
  if (ttfbMs < 200) return 'success'
  if (ttfbMs <= 600) return 'warning'
  return 'error'
}

function calculateThroughput(bytesSent: number, bytesReceived: number, durationMs: number): string {
  if (!durationMs || durationMs === 0) return 'N/A'
  const totalBytes = bytesSent + bytesReceived
  const durationSeconds = durationMs / 1000
  const bytesPerSecond = totalBytes / durationSeconds
  return formatBytes(bytesPerSecond) + '/s'
}

function getProtocolVersionFromALPN(alpn: string[] | undefined): 'HTTP/2' | 'HTTP/1.1' | 'Unknown' {
  if (!alpn || alpn.length === 0) return 'Unknown'
  if (alpn.includes('h2')) return 'HTTP/2'
  if (alpn.includes('http/1.1')) return 'HTTP/1.1'
  return 'Unknown'
}

function MetricItem({ label, value, indicator }: { label: string; value: string; indicator?: 'success' | 'warning' | 'error' | null }) {
  const getIndicatorClass = () => {
    switch (indicator) {
      case 'success': return 'text-status-success'
      case 'warning': return 'text-status-warning'
      case 'error': return 'text-status-error'
      default: return 'text-gray-400'
    }
  }

  return (
    <div className="glass-card p-3">
      <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-1">{label}</div>
      <div className={`text-sm font-semibold font-mono ${getIndicatorClass()}`}>{value}</div>
    </div>
  )
}

function ProtocolVersionBadge({ version }: { version: 'HTTP/2' | 'HTTP/1.1' | 'Unknown' }) {
  const getBadgeStyle = () => {
    switch (version) {
      case 'HTTP/2':
        return 'text-status-success bg-status-success/10 border-status-success/30'
      case 'HTTP/1.1':
        return 'text-status-info bg-status-info/10 border-status-info/30'
      default:
        return 'text-gray-400 bg-gray-500/10 border-gray-500/30'
    }
  }

  return (
    <div className="glass-card p-3">
      <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-1">Protocol Version</div>
      <span className={`inline-flex items-center px-2 py-0.5 rounded-md font-mono text-xs border ${getBadgeStyle()}`}>
        {version}
      </span>
    </div>
  )
}

function TimingBar({ flow }: { flow: Flow }) {
  const segments: { label: string; value: number; color: string }[] = []
  let total = 0

  if (flow.tcpHandshakeMs) {
    segments.push({ label: 'TCP Handshake', value: flow.tcpHandshakeMs, color: 'bg-status-info' })
    total += flow.tcpHandshakeMs
  }

  if (flow.tlsHandshakeMs) {
    segments.push({ label: 'TLS Handshake', value: flow.tlsHandshakeMs, color: 'bg-amber-500' })
    total += flow.tlsHandshakeMs
  }

  if (flow.ttfbMs) {
    const processing = flow.ttfbMs - total
    if (processing > 0) {
      segments.push({ label: 'TTFB', value: processing, color: 'bg-status-success' })
      total = flow.ttfbMs
    }
  }

  if (total === 0) {
    return <div className="text-xs text-gray-600">No timing data available</div>
  }

  return (
    <div className="space-y-3">
      {/* Bar */}
      <div className="h-3 bg-void-800 rounded-full overflow-hidden flex">
        {segments.map((seg, i) => (
          <div
            key={i}
            className={`${seg.color} h-full transition-all duration-500`}
            style={{ width: `${(seg.value / total) * 100}%` }}
            title={`${seg.label}: ${seg.value.toFixed(1)}ms`}
          />
        ))}
      </div>

      {/* Legend */}
      <div className="flex flex-wrap gap-4">
        {segments.map((seg, i) => (
          <div key={i} className="flex items-center gap-2">
            <div className={`w-2.5 h-2.5 rounded-full ${seg.color}`} />
            <span className="text-[11px] text-gray-400">
              {seg.label}: <span className="text-white font-mono">{seg.value.toFixed(1)}ms</span>
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

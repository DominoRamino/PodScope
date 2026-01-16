import { Flow } from '../types'
import { X, Download, ArrowRight, Lock, Terminal } from 'lucide-react'

interface FlowDetailProps {
  flow: Flow
  onClose: () => void
  onDownloadPCAP: () => void
  onOpenTerminal?: (podName: string) => void
}

export function FlowDetail({ flow, onClose, onDownloadPCAP, onOpenTerminal }: FlowDetailProps) {
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  }

  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp)
    return date.toLocaleString()
  }

  return (
    <div className="h-full flex flex-col bg-slate-850">
      {/* Header */}
      <div className="bg-slate-800 border-b border-slate-700 px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h2 className="font-semibold text-white">Flow Details</h2>
          <span className="text-xs text-slate-400 font-mono">{flow.id}</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onDownloadPCAP}
            className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
            title="Download PCAP"
          >
            <Download className="w-4 h-4 text-slate-400" />
          </button>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
          >
            <X className="w-4 h-4 text-slate-400" />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        {/* Summary Card */}
        <Section title="Summary">
          <div className="grid grid-cols-2 gap-4">
            <InfoItem label="Protocol" value={flow.protocol} />
            <InfoItem label="Status" value={flow.status} />
            <InfoItem label="Duration" value={`${flow.duration}ms`} />
            <InfoItem label="Timestamp" value={formatTimestamp(flow.timestamp)} />
          </div>
        </Section>

        {/* Connection Info */}
        <Section title="Connection">
          <div className="flex items-center gap-4 p-3 bg-slate-800 rounded-lg">
            <div className="flex-1">
              <div className="text-xs text-slate-400 mb-1">Source</div>
              <div className="font-mono text-sm">{flow.srcIp}:{flow.srcPort}</div>
              {flow.srcPod && (
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-xs text-slate-500">
                    {flow.srcNamespace}/{flow.srcPod}
                  </span>
                  {onOpenTerminal && (
                    <button
                      onClick={() => onOpenTerminal(`${flow.srcNamespace}/${flow.srcPod}`)}
                      className="p-1 hover:bg-slate-700 rounded transition-colors"
                      title="Open terminal"
                    >
                      <Terminal className="w-3 h-3 text-slate-400" />
                    </button>
                  )}
                </div>
              )}
            </div>
            <ArrowRight className="w-5 h-5 text-slate-500" />
            <div className="flex-1">
              <div className="text-xs text-slate-400 mb-1">Destination</div>
              <div className="font-mono text-sm">{flow.dstIp}:{flow.dstPort}</div>
              {flow.dstPod && (
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-xs text-slate-500">
                    {flow.dstNamespace}/{flow.dstPod}
                  </span>
                  {onOpenTerminal && (
                    <button
                      onClick={() => onOpenTerminal(`${flow.dstNamespace}/${flow.dstPod}`)}
                      className="p-1 hover:bg-slate-700 rounded transition-colors"
                      title="Open terminal"
                    >
                      <Terminal className="w-3 h-3 text-slate-400" />
                    </button>
                  )}
                </div>
              )}
              {flow.dstService && (
                <div className="text-xs text-podscope-400 mt-1">{flow.dstService}</div>
              )}
            </div>
          </div>
        </Section>

        {/* Timing */}
        <Section title="Timing">
          <TimingBar flow={flow} />
        </Section>

        {/* Data Transfer */}
        <Section title="Data Transfer">
          <div className="grid grid-cols-2 gap-4">
            <div className="p-3 bg-slate-800 rounded-lg">
              <div className="text-xs text-slate-400 mb-1">Sent</div>
              <div className="text-lg font-semibold text-green-400">
                {formatBytes(flow.bytesSent)}
              </div>
              <div className="text-xs text-slate-500">{flow.packetsSent} packets</div>
            </div>
            <div className="p-3 bg-slate-800 rounded-lg">
              <div className="text-xs text-slate-400 mb-1">Received</div>
              <div className="text-lg font-semibold text-blue-400">
                {formatBytes(flow.bytesReceived)}
              </div>
              <div className="text-xs text-slate-500">{flow.packetsReceived} packets</div>
            </div>
          </div>
        </Section>

        {/* TLS Info */}
        {flow.tls && (
          <Section title="TLS Info">
            <div className="space-y-2">
              <InfoItem label="Version" value={flow.tls.version} />
              {flow.tls.sni && <InfoItem label="SNI" value={flow.tls.sni} />}
              {flow.tls.cipherSuite && <InfoItem label="Cipher Suite" value={flow.tls.cipherSuite} />}
              {flow.tls.alpn && flow.tls.alpn.length > 0 && (
                <InfoItem label="ALPN" value={flow.tls.alpn.join(', ')} />
              )}
              <div className="p-3 bg-yellow-500/10 border border-yellow-500/20 rounded-lg text-sm text-yellow-400">
                <Lock className="inline w-4 h-4 mr-2" />
                Payload is encrypted. Only metadata is available.
              </div>
            </div>
          </Section>
        )}

        {/* HTTP Info */}
        {flow.http && (
          <>
            <Section title="HTTP Request">
              <div className="space-y-3">
                <div className="p-3 bg-slate-800 rounded-lg">
                  <span className="text-green-400 font-medium">{flow.http.method}</span>
                  <span className="ml-2 text-slate-300">{flow.http.url}</span>
                </div>
                {flow.http.host && <InfoItem label="Host" value={flow.http.host} />}
                {flow.http.requestHeaders && Object.keys(flow.http.requestHeaders).length > 0 && (
                  <HeadersTable headers={flow.http.requestHeaders} title="Request Headers" />
                )}
                {flow.http.requestBody && (
                  <div>
                    <div className="text-xs text-slate-400 mb-2">Request Body</div>
                    <pre className="p-3 bg-slate-800 rounded-lg text-xs overflow-x-auto">
                      {flow.http.requestBody}
                    </pre>
                  </div>
                )}
              </div>
            </Section>

            <Section title="HTTP Response">
              <div className="space-y-3">
                <div className="p-3 bg-slate-800 rounded-lg">
                  <StatusBadge code={flow.http.statusCode} text={flow.http.statusText} />
                </div>
                {flow.http.contentType && <InfoItem label="Content-Type" value={flow.http.contentType} />}
                {flow.http.contentLength !== undefined && (
                  <InfoItem label="Content-Length" value={formatBytes(flow.http.contentLength)} />
                )}
                {flow.http.responseHeaders && Object.keys(flow.http.responseHeaders).length > 0 && (
                  <HeadersTable headers={flow.http.responseHeaders} title="Response Headers" />
                )}
                {flow.http.responseBody && (
                  <div>
                    <div className="text-xs text-slate-400 mb-2">Response Body (truncated)</div>
                    <pre className="p-3 bg-slate-800 rounded-lg text-xs overflow-x-auto max-h-48">
                      {flow.http.responseBody}
                    </pre>
                  </div>
                )}
              </div>
            </Section>
          </>
        )}
      </div>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h3 className="text-sm font-medium text-slate-400 mb-3">{title}</h3>
      {children}
    </div>
  )
}

function InfoItem({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex justify-between items-center py-2 border-b border-slate-700/50">
      <span className="text-slate-400 text-sm">{label}</span>
      <span className="text-white font-mono text-sm">{value}</span>
    </div>
  )
}

function HeadersTable({ headers, title }: { headers: Record<string, string>; title: string }) {
  return (
    <div>
      <div className="text-xs text-slate-400 mb-2">{title}</div>
      <div className="bg-slate-800 rounded-lg overflow-hidden">
        {Object.entries(headers).map(([key, value]) => (
          <div key={key} className="flex border-b border-slate-700/50 last:border-0">
            <div className="w-1/3 px-3 py-2 text-xs text-slate-400 bg-slate-800/50 font-mono">
              {key}
            </div>
            <div className="flex-1 px-3 py-2 text-xs text-slate-300 font-mono break-all">
              {value}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function StatusBadge({ code, text }: { code: number; text: string }) {
  const getColor = () => {
    if (code >= 200 && code < 300) return 'text-green-400 bg-green-400/10'
    if (code >= 300 && code < 400) return 'text-blue-400 bg-blue-400/10'
    if (code >= 400 && code < 500) return 'text-yellow-400 bg-yellow-400/10'
    return 'text-red-400 bg-red-400/10'
  }

  return (
    <span className={`px-3 py-1 rounded font-mono text-sm ${getColor()}`}>
      {code} {text}
    </span>
  )
}

function TimingBar({ flow }: { flow: Flow }) {
  const segments = []
  let total = 0

  if (flow.tcpHandshakeMs) {
    segments.push({ label: 'TCP Handshake', value: flow.tcpHandshakeMs, color: 'bg-blue-500' })
    total += flow.tcpHandshakeMs
  }

  if (flow.tlsHandshakeMs) {
    segments.push({ label: 'TLS Handshake', value: flow.tlsHandshakeMs, color: 'bg-yellow-500' })
    total += flow.tlsHandshakeMs
  }

  if (flow.ttfbMs) {
    const processing = flow.ttfbMs - total
    if (processing > 0) {
      segments.push({ label: 'Processing', value: processing, color: 'bg-green-500' })
      total = flow.ttfbMs
    }
  }

  if (total === 0) {
    return <div className="text-sm text-slate-500">No timing data available</div>
  }

  return (
    <div className="space-y-3">
      {/* Bar */}
      <div className="h-6 bg-slate-800 rounded-lg overflow-hidden flex">
        {segments.map((seg, i) => (
          <div
            key={i}
            className={`${seg.color} h-full`}
            style={{ width: `${(seg.value / total) * 100}%` }}
            title={`${seg.label}: ${seg.value.toFixed(1)}ms`}
          />
        ))}
      </div>

      {/* Legend */}
      <div className="flex flex-wrap gap-4">
        {segments.map((seg, i) => (
          <div key={i} className="flex items-center gap-2">
            <div className={`w-3 h-3 rounded ${seg.color}`} />
            <span className="text-xs text-slate-400">
              {seg.label}: {seg.value.toFixed(1)}ms
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

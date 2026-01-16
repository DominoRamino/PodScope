export type Protocol = 'TCP' | 'HTTP' | 'HTTPS' | 'TLS'
export type FlowStatus = 'OPEN' | 'CLOSED' | 'RESET' | 'TIMEOUT'

export interface HTTPInfo {
  method: string
  url: string
  host: string
  statusCode: number
  statusText: string
  requestHeaders?: Record<string, string>
  responseHeaders?: Record<string, string>
  requestBody?: string
  responseBody?: string
  contentType?: string
  contentLength?: number
}

export interface TLSInfo {
  version: string
  sni: string
  cipherSuite: string
  alpn?: string[]
  encrypted: boolean
}

export interface Flow {
  id: string
  timestamp: string
  duration: number

  srcIp: string
  srcPort: number
  srcPod?: string
  srcNamespace?: string

  dstIp: string
  dstPort: number
  dstPod?: string
  dstNamespace?: string
  dstService?: string

  protocol: Protocol
  status: FlowStatus

  bytesSent: number
  bytesReceived: number
  packetsSent: number
  packetsReceived: number

  tcpHandshakeMs?: number
  tlsHandshakeMs?: number
  ttfbMs?: number

  http?: HTTPInfo
  tls?: TLSInfo
}

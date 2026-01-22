import type { Flow, HTTPInfo, TLSInfo, Protocol, FlowStatus } from '../types'

let flowIdCounter = 0

/**
 * Creates a mock TCP Flow object with sensible defaults.
 * All properties can be overridden via the overrides parameter.
 */
export function createMockFlow(overrides?: Partial<Flow>): Flow {
  flowIdCounter++
  const defaultFlow: Flow = {
    id: `flow-${flowIdCounter}`,
    timestamp: new Date().toISOString(),
    duration: 100,

    srcIp: '10.0.0.1',
    srcPort: 45678,
    srcPod: 'client-pod',
    srcNamespace: 'default',

    dstIp: '10.0.0.5',
    dstPort: 80,
    dstPod: 'server-pod',
    dstNamespace: 'default',
    dstService: 'my-service',

    protocol: 'TCP' as Protocol,
    status: 'CLOSED' as FlowStatus,

    bytesSent: 512,
    bytesReceived: 1024,
    packetsSent: 5,
    packetsReceived: 8,

    tcpHandshakeMs: 2,
  }

  return { ...defaultFlow, ...overrides }
}

/**
 * Creates a mock HTTP Flow object with HTTP-specific information.
 * Includes HTTPInfo with method, URL, status, and headers.
 */
export function createMockHTTPFlow(overrides?: Partial<Flow>): Flow {
  const httpInfo: HTTPInfo = {
    method: 'GET',
    url: '/api/users',
    host: 'api.example.com',
    statusCode: 200,
    statusText: '200 OK',
    requestHeaders: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    },
    responseHeaders: {
      'Content-Type': 'application/json',
      'Content-Length': '256',
    },
    contentType: 'application/json',
    contentLength: 256,
  }

  return createMockFlow({
    protocol: 'HTTP' as Protocol,
    dstPort: 80,
    http: httpInfo,
    ttfbMs: 15,
    ...overrides,
  })
}

/**
 * Creates a mock TLS/HTTPS Flow object with TLS-specific information.
 * Includes TLSInfo with version, SNI, and cipher suite.
 */
export function createMockTLSFlow(overrides?: Partial<Flow>): Flow {
  const tlsInfo: TLSInfo = {
    version: 'TLS 1.3',
    sni: 'secure.example.com',
    cipherSuite: 'TLS_AES_256_GCM_SHA384',
    alpn: ['h2', 'http/1.1'],
    encrypted: true,
  }

  return createMockFlow({
    protocol: 'HTTPS' as Protocol,
    dstPort: 443,
    tls: tlsInfo,
    tlsHandshakeMs: 25,
    ...overrides,
  })
}

/**
 * Resets the flow ID counter. Useful for test isolation.
 */
export function resetFlowIdCounter(): void {
  flowIdCounter = 0
}

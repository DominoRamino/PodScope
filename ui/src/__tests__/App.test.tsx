import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi, beforeEach, afterEach, describe, it, expect } from 'vitest'
import App from '../App'
import { createMockWebSocketClass, clearMockWebSocketInstances, type MockWebSocketInstance } from '../test-utils/mockWebSocket'
import { createMockHTTPFlow, createMockFlow, createMockTLSFlow, resetFlowIdCounter } from '../test-utils/testData'

describe('App component initial render', () => {
  let MockWebSocketClass: new (url: string) => MockWebSocketInstance
  let instances: MockWebSocketInstance[]

  beforeEach(() => {
    // Create fresh mock WebSocket class for each test
    const mock = createMockWebSocketClass()
    MockWebSocketClass = mock.MockWebSocketClass
    instances = mock.instances
    vi.stubGlobal('WebSocket', MockWebSocketClass)

    // Mock fetch for stats endpoint
    vi.spyOn(globalThis, 'fetch').mockImplementation((url: RequestInfo | URL) => {
      if (typeof url === 'string' && url.includes('/api/stats')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            flows: 0,
            wsClients: 0,
            pcapSize: 0,
            paused: false,
          }),
        } as Response)
      }
      return Promise.resolve({
        ok: false,
        json: () => Promise.resolve({}),
      } as Response)
    })
  })

  afterEach(() => {
    clearMockWebSocketInstances(instances)
    vi.restoreAllMocks()
  })

  it('renders without crashing', async () => {
    render(<App />)
    // Check that the main title is rendered
    expect(screen.getByText('PodScope')).toBeInTheDocument()
    // Wait for any pending state updates
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0))
    })
  })

  it('shows header elements on initial render', async () => {
    render(<App />)
    // Check header content
    expect(screen.getByText('PodScope')).toBeInTheDocument()
    expect(screen.getByText('Kubernetes Traffic Analyzer')).toBeInTheDocument()
    expect(screen.getByPlaceholderText(/Filter by IP/)).toBeInTheDocument()
    // Wait for any pending state updates
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0))
    })
  })

  it('shows Disconnected status initially before WebSocket connects', async () => {
    render(<App />)
    // Initially should show disconnected (before WebSocket onopen fires)
    expect(screen.getByText('Disconnected')).toBeInTheDocument()
    // Wait for any pending state updates
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0))
    })
  })

  it('displays Live when WebSocket opens', async () => {
    render(<App />)

    // Get the WebSocket instance that was created
    expect(instances.length).toBe(1)
    const ws = instances[0]

    // Simulate WebSocket connection opening
    await act(async () => {
      ws.simulateOpen()
    })

    // Should now show Live
    await waitFor(() => {
      expect(screen.getByText('Live')).toBeInTheDocument()
    })
  })

  it('displays Disconnected when WebSocket closes', async () => {
    render(<App />)

    const ws = instances[0]

    // First open the connection
    await act(async () => {
      ws.simulateOpen()
    })

    // Verify connected
    await waitFor(() => {
      expect(screen.getByText('Live')).toBeInTheDocument()
    })

    // Now close the connection
    await act(async () => {
      ws.simulateClose(1000, 'Normal closure')
    })

    // Should show Disconnected again
    await waitFor(() => {
      expect(screen.getByText('Disconnected')).toBeInTheDocument()
    })
  })

  it('creates WebSocket with correct URL format', async () => {
    render(<App />)

    expect(instances.length).toBe(1)
    const ws = instances[0]

    // Check URL contains the expected path
    expect(ws.url).toContain('/api/flows/ws')
    // Wait for any pending state updates
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0))
    })
  })

  it('shows filter buttons on initial render', async () => {
    render(<App />)
    // Check filter toggles are present - using text match that includes checkmark prefix
    expect(screen.getByText(/HTTP\/HTTPS Only/)).toBeInTheDocument()
    expect(screen.getByText(/Show DNS/)).toBeInTheDocument()
    expect(screen.getByText(/Show All Ports/)).toBeInTheDocument()
    // Wait for any pending state updates
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0))
    })
  })

  it('shows pause button on initial render', async () => {
    render(<App />)
    // Check pause button is present (not paused by default)
    expect(screen.getByRole('button', { name: /pause/i })).toBeInTheDocument()
    // Wait for any pending state updates
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0))
    })
  })

  it('shows download button on initial render', async () => {
    render(<App />)
    // Check download button is present
    expect(screen.getByRole('button', { name: /download pcap/i })).toBeInTheDocument()
    // Wait for any pending state updates
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0))
    })
  })
})

describe('App WebSocket message handling', () => {
  let MockWebSocketClass: new (url: string) => MockWebSocketInstance
  let instances: MockWebSocketInstance[]

  beforeEach(() => {
    // Reset flow ID counter for consistent test data
    resetFlowIdCounter()

    // Create fresh mock WebSocket class for each test
    const mock = createMockWebSocketClass()
    MockWebSocketClass = mock.MockWebSocketClass
    instances = mock.instances
    vi.stubGlobal('WebSocket', MockWebSocketClass)

    // Mock fetch for stats endpoint
    vi.spyOn(globalThis, 'fetch').mockImplementation((url: RequestInfo | URL) => {
      if (typeof url === 'string' && url.includes('/api/stats')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            flows: 0,
            wsClients: 0,
            pcapSize: 0,
            paused: false,
          }),
        } as Response)
      }
      return Promise.resolve({
        ok: false,
        json: () => Promise.resolve({}),
      } as Response)
    })
  })

  afterEach(() => {
    clearMockWebSocketInstances(instances)
    vi.restoreAllMocks()
  })

  it('processes catchup message with array of flows correctly', async () => {
    render(<App />)

    const ws = instances[0]

    // Open WebSocket connection
    await act(async () => {
      ws.simulateOpen()
    })

    // Create multiple flows for catchup
    const flow1 = createMockHTTPFlow({ id: 'catchup-1', srcPod: 'pod-alpha' })
    const flow2 = createMockHTTPFlow({ id: 'catchup-2', srcPod: 'pod-beta' })
    const flow3 = createMockHTTPFlow({ id: 'catchup-3', srcPod: 'pod-gamma' })

    // Simulate catchup message (array of existing flows)
    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [flow1, flow2, flow3],
      })
    })

    // Verify flows appear in the UI
    await waitFor(() => {
      expect(screen.getByText('pod-alpha')).toBeInTheDocument()
      expect(screen.getByText('pod-beta')).toBeInTheDocument()
      expect(screen.getByText('pod-gamma')).toBeInTheDocument()
    })
  })

  it('processes batch message with type: batch correctly', async () => {
    render(<App />)

    const ws = instances[0]

    // Open WebSocket connection
    await act(async () => {
      ws.simulateOpen()
    })

    // Create flows for batch
    const batchFlow1 = createMockHTTPFlow({ id: 'batch-1', srcPod: 'batch-pod-1' })
    const batchFlow2 = createMockHTTPFlow({ id: 'batch-2', srcPod: 'batch-pod-2' })

    // Simulate batch message
    await act(async () => {
      ws.simulateMessage({
        type: 'batch',
        flows: [batchFlow1, batchFlow2],
      })
    })

    // Verify flows appear in the UI
    await waitFor(() => {
      expect(screen.getByText('batch-pod-1')).toBeInTheDocument()
      expect(screen.getByText('batch-pod-2')).toBeInTheDocument()
    })
  })

  it('adds single flow message to flow list', async () => {
    render(<App />)

    const ws = instances[0]

    // Open WebSocket connection
    await act(async () => {
      ws.simulateOpen()
    })

    // Create a single flow (legacy format without type wrapper)
    const singleFlow = createMockHTTPFlow({ id: 'single-1', srcPod: 'single-pod' })

    // Simulate single flow message (backward compatible format)
    await act(async () => {
      ws.simulateMessage(singleFlow)
    })

    // Verify flow appears in the UI
    await waitFor(() => {
      expect(screen.getByText('single-pod')).toBeInTheDocument()
    })
  })

  it('flows appear in UI after message received', async () => {
    render(<App />)

    const ws = instances[0]

    // Open WebSocket connection
    await act(async () => {
      ws.simulateOpen()
    })

    // Create a flow with specific identifying data
    // Note: For HTTP flows, destination shows http.host, not dstPod
    const testFlow = createMockHTTPFlow({
      id: 'ui-test-flow',
      srcPod: 'source-pod-123',
      dstPod: 'destination-pod-456',
      srcIp: '192.168.1.100',
      dstIp: '192.168.1.200',
      http: {
        method: 'GET',
        url: '/api/data',
        host: 'custom-api-host.example.com', // This is what shows in destination
        statusCode: 200,
        statusText: '200 OK',
        requestHeaders: {},
        responseHeaders: {},
      },
    })

    // Send the flow
    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [testFlow],
      })
    })

    // Verify specific flow data appears
    // Source shows srcPod, Destination shows http.host for HTTP flows
    await waitFor(() => {
      expect(screen.getByText('source-pod-123')).toBeInTheDocument()
      expect(screen.getByText('custom-api-host.example.com')).toBeInTheDocument()
    })
  })

  it('updates existing flow when receiving flow with same ID', async () => {
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // Send initial flow
    const initialFlow = createMockHTTPFlow({
      id: 'update-test',
      srcPod: 'original-pod',
      bytesSent: 100,
    })

    await act(async () => {
      ws.simulateMessage(initialFlow)
    })

    await waitFor(() => {
      expect(screen.getByText('original-pod')).toBeInTheDocument()
    })

    // Send updated flow with same ID but different data
    const updatedFlow = createMockHTTPFlow({
      id: 'update-test',
      srcPod: 'updated-pod',
      bytesSent: 500,
    })

    await act(async () => {
      ws.simulateMessage(updatedFlow)
    })

    // The updated pod name should appear, and original should not
    await waitFor(() => {
      expect(screen.getByText('updated-pod')).toBeInTheDocument()
      expect(screen.queryByText('original-pod')).not.toBeInTheDocument()
    })
  })

  it('handles multiple sequential messages correctly', async () => {
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // Send first batch
    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [createMockHTTPFlow({ id: 'seq-1', srcPod: 'first-pod' })],
      })
    })

    await waitFor(() => {
      expect(screen.getByText('first-pod')).toBeInTheDocument()
    })

    // Send second batch
    await act(async () => {
      ws.simulateMessage({
        type: 'batch',
        flows: [createMockHTTPFlow({ id: 'seq-2', srcPod: 'second-pod' })],
      })
    })

    // Both flows should be present
    await waitFor(() => {
      expect(screen.getByText('first-pod')).toBeInTheDocument()
      expect(screen.getByText('second-pod')).toBeInTheDocument()
    })
  })

  it('merges flows from batch with existing flows', async () => {
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // Initial catchup with some flows
    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [
          createMockHTTPFlow({ id: 'merge-1', srcPod: 'existing-pod' }),
        ],
      })
    })

    await waitFor(() => {
      expect(screen.getByText('existing-pod')).toBeInTheDocument()
    })

    // New batch with additional flows
    await act(async () => {
      ws.simulateMessage({
        type: 'batch',
        flows: [
          createMockHTTPFlow({ id: 'merge-2', srcPod: 'new-pod' }),
        ],
      })
    })

    // Both existing and new flows should be present
    await waitFor(() => {
      expect(screen.getByText('existing-pod')).toBeInTheDocument()
      expect(screen.getByText('new-pod')).toBeInTheDocument()
    })
  })

  it('handles empty catchup message gracefully', async () => {
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // Send empty catchup
    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [],
      })
    })

    // App should still be functional
    await waitFor(() => {
      expect(screen.getByText('Live')).toBeInTheDocument()
    })
  })

  it('handles invalid JSON message gracefully without crashing', async () => {
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // Spy on console.error to verify error handling
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    // Send invalid JSON
    await act(async () => {
      if (ws.onmessage) {
        const event = new MessageEvent('message', { data: 'invalid json {{{' })
        ws.onmessage(event)
      }
    })

    // App should still be functional
    await waitFor(() => {
      expect(screen.getByText('Live')).toBeInTheDocument()
    })

    // Verify error was logged
    expect(consoleErrorSpy).toHaveBeenCalled()

    consoleErrorSpy.mockRestore()
  })
})

describe('App flow filtering', () => {
  let MockWebSocketClass: new (url: string) => MockWebSocketInstance
  let instances: MockWebSocketInstance[]

  beforeEach(() => {
    // Reset flow ID counter for consistent test data
    resetFlowIdCounter()

    // Create fresh mock WebSocket class for each test
    const mock = createMockWebSocketClass()
    MockWebSocketClass = mock.MockWebSocketClass
    instances = mock.instances
    vi.stubGlobal('WebSocket', MockWebSocketClass)

    // Mock fetch for stats endpoint
    vi.spyOn(globalThis, 'fetch').mockImplementation((url: RequestInfo | URL) => {
      if (typeof url === 'string' && url.includes('/api/stats')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            flows: 0,
            wsClients: 0,
            pcapSize: 0,
            paused: false,
          }),
        } as Response)
      }
      return Promise.resolve({
        ok: false,
        json: () => Promise.resolve({}),
      } as Response)
    })
  })

  afterEach(() => {
    clearMockWebSocketInstances(instances)
    vi.restoreAllMocks()
  })

  it('HTTP Only filter shows only HTTP/HTTPS protocol flows by default', async () => {
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // Send flows with different protocols
    const httpFlow = createMockHTTPFlow({ id: 'http-1', srcPod: 'http-pod', dstPort: 80 })
    const httpsFlow = createMockTLSFlow({ id: 'https-1', srcPod: 'https-pod', dstPort: 443 })
    const tcpFlow = createMockFlow({ id: 'tcp-1', srcPod: 'tcp-pod', protocol: 'TCP', dstPort: 3306 }) // Non-HTTP port

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [httpFlow, httpsFlow, tcpFlow],
      })
    })

    // HTTP Only is enabled by default - should show HTTP and HTTPS flows
    await waitFor(() => {
      expect(screen.getByText('http-pod')).toBeInTheDocument()
      expect(screen.getByText('https-pod')).toBeInTheDocument()
    })

    // TCP flow on non-HTTP port should NOT be visible
    expect(screen.queryByText('tcp-pod')).not.toBeInTheDocument()
  })

  it('HTTP Only filter shows flows on common HTTP ports regardless of protocol', async () => {
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // TCP flow on port 8080 (common HTTP port) should be shown
    const tcpOn8080 = createMockFlow({ id: 'tcp-8080', srcPod: 'tcp-8080-pod', protocol: 'TCP', dstPort: 8080 })
    // TCP flow on port 3306 (MySQL) should be filtered out
    const tcpOnMySQL = createMockFlow({ id: 'tcp-mysql', srcPod: 'mysql-pod', protocol: 'TCP', dstPort: 3306 })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [tcpOn8080, tcpOnMySQL],
      })
    })

    // HTTP Only filter allows common HTTP ports even with TCP protocol
    await waitFor(() => {
      expect(screen.getByText('tcp-8080-pod')).toBeInTheDocument()
    })
    expect(screen.queryByText('mysql-pod')).not.toBeInTheDocument()
  })

  it('Show All Ports filter shows all protocols when clicked', async () => {
    const user = userEvent.setup()
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // Send flows with different protocols and ports
    const httpFlow = createMockHTTPFlow({ id: 'http-all', srcPod: 'http-all-pod', dstPort: 80 })
    const tcpFlow = createMockFlow({ id: 'tcp-all', srcPod: 'tcp-all-pod', protocol: 'TCP', dstPort: 3306 })
    const dnsFlow = createMockFlow({ id: 'dns-all', srcPod: 'dns-all-pod', protocol: 'TCP', dstPort: 53 })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [httpFlow, tcpFlow, dnsFlow],
      })
    })

    // Initially with HTTP Only, TCP on 3306 and DNS should be hidden
    await waitFor(() => {
      expect(screen.getByText('http-all-pod')).toBeInTheDocument()
    })
    expect(screen.queryByText('tcp-all-pod')).not.toBeInTheDocument()
    expect(screen.queryByText('dns-all-pod')).not.toBeInTheDocument()

    // Click "Show All Ports" button
    const showAllButton = screen.getByText(/Show All Ports/)
    await user.click(showAllButton)

    // Now all flows should be visible
    await waitFor(() => {
      expect(screen.getByText('http-all-pod')).toBeInTheDocument()
      expect(screen.getByText('tcp-all-pod')).toBeInTheDocument()
      expect(screen.getByText('dns-all-pod')).toBeInTheDocument()
    })
  })

  it('text search filters by pod name', async () => {
    const user = userEvent.setup()
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // Send multiple HTTP flows
    const flow1 = createMockHTTPFlow({ id: 'search-1', srcPod: 'alpha-service' })
    const flow2 = createMockHTTPFlow({ id: 'search-2', srcPod: 'beta-service' })
    const flow3 = createMockHTTPFlow({ id: 'search-3', srcPod: 'gamma-pod' })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [flow1, flow2, flow3],
      })
    })

    // All flows visible initially
    await waitFor(() => {
      expect(screen.getByText('alpha-service')).toBeInTheDocument()
      expect(screen.getByText('beta-service')).toBeInTheDocument()
      expect(screen.getByText('gamma-pod')).toBeInTheDocument()
    })

    // Type in search filter
    const searchInput = screen.getByPlaceholderText(/Filter by IP/)
    await user.type(searchInput, 'alpha')

    // Only alpha-service should be visible
    await waitFor(() => {
      expect(screen.getByText('alpha-service')).toBeInTheDocument()
      expect(screen.queryByText('beta-service')).not.toBeInTheDocument()
      expect(screen.queryByText('gamma-pod')).not.toBeInTheDocument()
    })
  })

  it('text search filters by IP address', async () => {
    const user = userEvent.setup()
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    const flow1 = createMockHTTPFlow({ id: 'ip-1', srcPod: 'pod-a', srcIp: '192.168.1.100' })
    const flow2 = createMockHTTPFlow({ id: 'ip-2', srcPod: 'pod-b', srcIp: '10.0.0.50' })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [flow1, flow2],
      })
    })

    await waitFor(() => {
      expect(screen.getByText('pod-a')).toBeInTheDocument()
      expect(screen.getByText('pod-b')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText(/Filter by IP/)
    await user.type(searchInput, '192.168')

    await waitFor(() => {
      expect(screen.getByText('pod-a')).toBeInTheDocument()
      expect(screen.queryByText('pod-b')).not.toBeInTheDocument()
    })
  })

  it('text search filters by HTTP URL', async () => {
    const user = userEvent.setup()
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    const flow1 = createMockHTTPFlow({
      id: 'url-1',
      srcPod: 'api-client',
      http: {
        method: 'GET',
        url: '/api/users/profile',
        host: 'api.example.com',
        statusCode: 200,
        statusText: '200 OK',
        requestHeaders: {},
        responseHeaders: {},
      },
    })
    const flow2 = createMockHTTPFlow({
      id: 'url-2',
      srcPod: 'web-client',
      http: {
        method: 'GET',
        url: '/assets/styles.css',
        host: 'cdn.example.com',
        statusCode: 200,
        statusText: '200 OK',
        requestHeaders: {},
        responseHeaders: {},
      },
    })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [flow1, flow2],
      })
    })

    await waitFor(() => {
      expect(screen.getByText('api-client')).toBeInTheDocument()
      expect(screen.getByText('web-client')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText(/Filter by IP/)
    await user.type(searchInput, '/api/users')

    await waitFor(() => {
      expect(screen.getByText('api-client')).toBeInTheDocument()
      expect(screen.queryByText('web-client')).not.toBeInTheDocument()
    })
  })

  it('text search filters by TLS SNI', async () => {
    const user = userEvent.setup()
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    const flow1 = createMockTLSFlow({
      id: 'sni-1',
      srcPod: 'secure-client-1',
      tls: {
        version: 'TLS 1.3',
        sni: 'api.stripe.com',
        cipherSuite: 'TLS_AES_256_GCM_SHA384',
        encrypted: true,
      },
    })
    const flow2 = createMockTLSFlow({
      id: 'sni-2',
      srcPod: 'secure-client-2',
      tls: {
        version: 'TLS 1.3',
        sni: 'github.com',
        cipherSuite: 'TLS_AES_256_GCM_SHA384',
        encrypted: true,
      },
    })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [flow1, flow2],
      })
    })

    await waitFor(() => {
      expect(screen.getByText('secure-client-1')).toBeInTheDocument()
      expect(screen.getByText('secure-client-2')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText(/Filter by IP/)
    await user.type(searchInput, 'stripe')

    await waitFor(() => {
      expect(screen.getByText('secure-client-1')).toBeInTheDocument()
      expect(screen.queryByText('secure-client-2')).not.toBeInTheDocument()
    })
  })

  it('empty filter state shows all flows (within HTTP Only constraint)', async () => {
    const user = userEvent.setup()
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    const flow1 = createMockHTTPFlow({ id: 'empty-1', srcPod: 'empty-pod-1' })
    const flow2 = createMockHTTPFlow({ id: 'empty-2', srcPod: 'empty-pod-2' })
    const flow3 = createMockHTTPFlow({ id: 'empty-3', srcPod: 'empty-pod-3' })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [flow1, flow2, flow3],
      })
    })

    // All flows visible initially
    await waitFor(() => {
      expect(screen.getByText('empty-pod-1')).toBeInTheDocument()
      expect(screen.getByText('empty-pod-2')).toBeInTheDocument()
      expect(screen.getByText('empty-pod-3')).toBeInTheDocument()
    })

    // Type something then clear it
    const searchInput = screen.getByPlaceholderText(/Filter by IP/)
    await user.type(searchInput, 'something')

    await waitFor(() => {
      expect(screen.queryByText('empty-pod-1')).not.toBeInTheDocument()
    })

    // Clear the filter
    await user.clear(searchInput)

    // All flows should be visible again
    await waitFor(() => {
      expect(screen.getByText('empty-pod-1')).toBeInTheDocument()
      expect(screen.getByText('empty-pod-2')).toBeInTheDocument()
      expect(screen.getByText('empty-pod-3')).toBeInTheDocument()
    })
  })

  it('search is case insensitive', async () => {
    const user = userEvent.setup()
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    const flow = createMockHTTPFlow({ id: 'case-1', srcPod: 'MyTestPod' })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [flow],
      })
    })

    await waitFor(() => {
      expect(screen.getByText('MyTestPod')).toBeInTheDocument()
    })

    // Search with different case
    const searchInput = screen.getByPlaceholderText(/Filter by IP/)
    await user.type(searchInput, 'mytestpod')

    // Should still find the flow
    await waitFor(() => {
      expect(screen.getByText('MyTestPod')).toBeInTheDocument()
    })
  })

  it('clicking HTTP Only button again disables the filter', async () => {
    const user = userEvent.setup()
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // HTTP flow on port 80 and TCP flow on non-HTTP port
    const httpFlow = createMockHTTPFlow({ id: 'toggle-http', srcPod: 'http-toggle-pod' })
    const tcpFlow = createMockFlow({ id: 'toggle-tcp', srcPod: 'tcp-toggle-pod', protocol: 'TCP', dstPort: 3306 })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [httpFlow, tcpFlow],
      })
    })

    // Initially HTTP Only is on, so only HTTP flow is visible
    await waitFor(() => {
      expect(screen.getByText('http-toggle-pod')).toBeInTheDocument()
    })
    expect(screen.queryByText('tcp-toggle-pod')).not.toBeInTheDocument()

    // Click HTTP/HTTPS Only button to toggle it off
    const httpOnlyButton = screen.getByText(/HTTP\/HTTPS Only/)
    await user.click(httpOnlyButton)

    // Now we're in "custom" mode - neither HTTP Only nor All Ports
    // But HTTP flow should still be visible
    await waitFor(() => {
      expect(screen.getByText('http-toggle-pod')).toBeInTheDocument()
    })
  })

  it('DNS flows are hidden by default even on port 53', async () => {
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // Create HTTP flow and DNS flow
    const httpFlow = createMockHTTPFlow({ id: 'dns-test-http', srcPod: 'dns-test-http-pod' })
    const dnsFlow = createMockFlow({
      id: 'dns-test-dns',
      srcPod: 'dns-test-dns-pod',
      protocol: 'TCP',
      dstPort: 53
    })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [httpFlow, dnsFlow],
      })
    })

    // HTTP flow should be visible
    await waitFor(() => {
      expect(screen.getByText('dns-test-http-pod')).toBeInTheDocument()
    })

    // DNS flow should be hidden (showDNS is false by default)
    expect(screen.queryByText('dns-test-dns-pod')).not.toBeInTheDocument()
  })

  it('Show DNS button reveals DNS flows when HTTP Only is active', async () => {
    const user = userEvent.setup()
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    // Create HTTP flow on port 80 and DNS flow on port 53
    // Note: DNS port 53 is not in HTTP_PORTS, so with HTTP Only filter, DNS flows would be hidden
    // but DNS filtering is a separate check
    const httpFlow = createMockHTTPFlow({
      id: 'dns-test-http',
      srcPod: 'dns-test-http-pod',
    })
    const dnsFlowOnHTTPPort = createMockFlow({
      id: 'dns-show',
      srcPod: 'dns-show-pod',
      protocol: 'TCP',
      srcPort: 53, // DNS source port (response from DNS server)
      dstPort: 8080, // HTTP port - this flow would pass HTTP filter
    })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [httpFlow, dnsFlowOnHTTPPort],
      })
    })

    // HTTP flow should be visible
    await waitFor(() => {
      expect(screen.getByText('dns-test-http-pod')).toBeInTheDocument()
    })

    // DNS flow has srcPort 53 but dstPort is HTTP port 8080
    // With HTTP Only filter, this passes the port check but is filtered by DNS check
    expect(screen.queryByText('dns-show-pod')).not.toBeInTheDocument()

    // Click Show DNS button
    const showDNSButton = screen.getByText(/Show DNS/)
    await user.click(showDNSButton)

    // Now DNS flow should be visible
    await waitFor(() => {
      expect(screen.getByText('dns-show-pod')).toBeInTheDocument()
    })
  })

  it('Show All Ports includes DNS flows (DNS filtering skipped)', async () => {
    const user = userEvent.setup()
    render(<App />)

    const ws = instances[0]

    await act(async () => {
      ws.simulateOpen()
    })

    const dnsFlow = createMockFlow({
      id: 'all-ports-dns',
      srcPod: 'all-ports-dns-pod',
      protocol: 'TCP',
      dstPort: 53
    })

    await act(async () => {
      ws.simulateMessage({
        type: 'catchup',
        flows: [dnsFlow],
      })
    })

    // With HTTP Only (default), DNS on port 53 is filtered out
    expect(screen.queryByText('all-ports-dns-pod')).not.toBeInTheDocument()

    // Click "Show All Ports" button
    const showAllButton = screen.getByText(/Show All Ports/)
    await user.click(showAllButton)

    // With Show All Ports, DNS filtering is skipped - DNS flows are shown
    await waitFor(() => {
      expect(screen.getByText('all-ports-dns-pod')).toBeInTheDocument()
    })
  })
})

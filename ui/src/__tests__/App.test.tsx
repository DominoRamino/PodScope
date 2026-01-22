import { render, screen, waitFor, act } from '@testing-library/react'
import { vi, beforeEach, afterEach, describe, it, expect } from 'vitest'
import App from '../App'
import { createMockWebSocketClass, clearMockWebSocketInstances, type MockWebSocketInstance } from '../test-utils/mockWebSocket'
import { createMockHTTPFlow, resetFlowIdCounter } from '../test-utils/testData'

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

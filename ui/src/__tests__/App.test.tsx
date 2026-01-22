import { render, screen, waitFor, act } from '@testing-library/react'
import { vi, beforeEach, afterEach, describe, it, expect } from 'vitest'
import App from '../App'
import { createMockWebSocketClass, clearMockWebSocketInstances, type MockWebSocketInstance } from '../test-utils/mockWebSocket'

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

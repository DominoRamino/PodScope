import { render, screen, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi, beforeEach, describe, it, expect } from 'vitest'
import { FlowList } from '../../components/FlowList'
import { createMockFlow, createMockHTTPFlow, createMockTLSFlow, resetFlowIdCounter } from '../../test-utils/testData'
import type { Flow } from '../../types'

// Default props factory
const createDefaultProps = () => ({
  flows: [] as Flow[],
  selectedId: undefined as string | undefined,
  onSelect: vi.fn(),
})

describe('FlowList component', () => {
  beforeEach(() => {
    resetFlowIdCounter()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('empty state', () => {
    it('shows appropriate message when no flows', async () => {
      const props = createDefaultProps()
      props.flows = []

      render(<FlowList {...props} />)

      // Run timers for ResizeObserver callback
      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('No flows captured yet')).toBeInTheDocument()
    })

    it('shows additional hint text in empty state', async () => {
      const props = createDefaultProps()
      props.flows = []

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('Traffic will appear here in real-time')).toBeInTheDocument()
    })

    it('renders Server icon in empty state', async () => {
      const props = createDefaultProps()
      props.flows = []

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // Server icon from lucide-react has svg with class w-12
      const container = screen.getByText('No flows captured yet').closest('div')
      expect(container).toBeInTheDocument()
      expect(container?.querySelector('svg')).toBeInTheDocument()
    })

    it('does not call onSelect when empty', async () => {
      const props = createDefaultProps()
      props.flows = []

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(props.onSelect).not.toHaveBeenCalled()
    })
  })

  describe('single flow rendering', () => {
    it('displays source pod name', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ srcPod: 'my-client-pod' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('my-client-pod')).toBeInTheDocument()
    })

    it('displays source namespace when available', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ srcPod: 'client-pod', srcNamespace: 'production' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('production')).toBeInTheDocument()
    })

    it('displays source IP:port when no pod name', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ srcPod: undefined, srcIp: '192.168.1.100', srcPort: 54321 })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('192.168.1.100:54321')).toBeInTheDocument()
    })

    it('displays destination from http.host when available', async () => {
      const props = createDefaultProps()
      const flow = createMockHTTPFlow()
      flow.http!.host = 'api.myservice.com'
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('api.myservice.com')).toBeInTheDocument()
    })

    it('displays destination from tls.sni when no http.host', async () => {
      const props = createDefaultProps()
      const flow = createMockTLSFlow()
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // TLS flow has sni: 'secure.example.com'
      expect(screen.getByText('secure.example.com')).toBeInTheDocument()
    })

    it('displays destination service when no http.host or tls.sni', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ dstService: 'backend-service' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('backend-service')).toBeInTheDocument()
    })

    it('displays destination pod when no service', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ dstService: undefined, dstPod: 'backend-pod' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('backend-pod')).toBeInTheDocument()
    })

    it('displays destination IP:port as fallback', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({
        dstService: undefined,
        dstPod: undefined,
        dstIp: '10.0.0.99',
        dstPort: 8080
      })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('10.0.0.99:8080')).toBeInTheDocument()
    })
  })

  describe('timestamp formatting', () => {
    it('displays formatted timestamp', async () => {
      const props = createDefaultProps()
      // Use a specific timestamp
      const flow = createMockFlow({ timestamp: '2026-01-21T14:30:45.123Z' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // formatTime outputs HH:MM:SS.mmm in local time
      // We look for a pattern like XX:XX:XX.XXX
      const timeElement = screen.getByText(/\d{2}:\d{2}:\d{2}\.\d{3}/)
      expect(timeElement).toBeInTheDocument()
    })

    it('displays timestamp with milliseconds', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ timestamp: '2026-01-21T12:00:00.456Z' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // The timestamp should contain .456 for milliseconds (might vary with timezone)
      const timeElement = screen.getByText(/\.\d{3}/)
      expect(timeElement).toBeInTheDocument()
    })
  })

  describe('protocol badge colors', () => {
    it('displays TCP protocol badge with blue color', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ protocol: 'TCP' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const badge = screen.getByText('TCP')
      expect(badge).toBeInTheDocument()
      expect(badge).toHaveClass('text-blue-400')
      expect(badge).toHaveClass('bg-blue-400/10')
    })

    it('displays HTTP protocol badge with green color', async () => {
      const props = createDefaultProps()
      const flow = createMockHTTPFlow()
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const badge = screen.getByText('HTTP')
      expect(badge).toBeInTheDocument()
      expect(badge).toHaveClass('text-green-400')
      expect(badge).toHaveClass('bg-green-400/10')
    })

    it('displays HTTPS protocol badge with yellow color', async () => {
      const props = createDefaultProps()
      const flow = createMockTLSFlow({ protocol: 'HTTPS' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const badge = screen.getByText('HTTPS')
      expect(badge).toBeInTheDocument()
      expect(badge).toHaveClass('text-yellow-400')
      expect(badge).toHaveClass('bg-yellow-400/10')
    })

    it('displays TLS protocol badge with yellow color', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ protocol: 'TLS' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const badge = screen.getByText('TLS')
      expect(badge).toBeInTheDocument()
      expect(badge).toHaveClass('text-yellow-400')
      expect(badge).toHaveClass('bg-yellow-400/10')
    })
  })

  describe('status display', () => {
    it('displays flow status for TCP flow', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ status: 'CLOSED' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('CLOSED')).toBeInTheDocument()
    })

    it('displays HTTP status code when available', async () => {
      const props = createDefaultProps()
      const flow = createMockHTTPFlow()
      flow.http!.statusCode = 404
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('404')).toBeInTheDocument()
    })
  })

  describe('latency display', () => {
    it('displays ttfbMs when available', async () => {
      const props = createDefaultProps()
      const flow = createMockHTTPFlow({ ttfbMs: 25.5 })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('25.5ms')).toBeInTheDocument()
    })

    it('displays tcpHandshakeMs when no ttfbMs', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ ttfbMs: undefined, tcpHandshakeMs: 3.2 })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('3.2ms')).toBeInTheDocument()
    })

    it('displays dash when no latency info', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ ttfbMs: undefined, tcpHandshakeMs: undefined })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('-')).toBeInTheDocument()
    })
  })

  describe('size display', () => {
    it('displays total bytes (sent + received)', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ bytesSent: 512, bytesReceived: 1024 })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // Total is 1536 bytes = 1.50 KB
      expect(screen.getByText('1.50 KB')).toBeInTheDocument()
    })

    it('displays bytes for small sizes', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ bytesSent: 100, bytesReceived: 150 })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // Total is 250 bytes
      expect(screen.getByText('250 B')).toBeInTheDocument()
    })
  })

  describe('HTTP URL display', () => {
    it('displays HTTP method and URL for HTTP flows', async () => {
      const props = createDefaultProps()
      const flow = createMockHTTPFlow()
      flow.http!.method = 'POST'
      flow.http!.url = '/api/users/create'
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      expect(screen.getByText('POST /api/users/create')).toBeInTheDocument()
    })

    it('does not display URL row for root path (/)', async () => {
      const props = createDefaultProps()
      const flow = createMockHTTPFlow()
      flow.http!.url = '/'
      flow.http!.method = 'GET'
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // Should not display "GET /" text
      expect(screen.queryByText('GET /')).not.toBeInTheDocument()
    })
  })

  describe('encryption indicator', () => {
    it('shows lock icon for HTTPS flows', async () => {
      const props = createDefaultProps()
      const flow = createMockTLSFlow({ protocol: 'HTTPS' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // Lock icon from lucide-react - look for the yellow lock
      const row = screen.getByText('HTTPS').closest('[class*="grid"]')
      const lockIcon = row?.querySelector('svg.text-yellow-500')
      expect(lockIcon).toBeInTheDocument()
    })

    it('shows lock icon for TLS flows', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ protocol: 'TLS' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const row = screen.getByText('TLS').closest('[class*="grid"]')
      const lockIcon = row?.querySelector('svg.text-yellow-500')
      expect(lockIcon).toBeInTheDocument()
    })

    it('does not show lock icon for HTTP flows', async () => {
      const props = createDefaultProps()
      const flow = createMockHTTPFlow()
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const row = screen.getByText('HTTP').closest('[class*="grid"]')
      const lockIcon = row?.querySelector('svg.text-yellow-500')
      expect(lockIcon).not.toBeInTheDocument()
    })
  })

  describe('table headers', () => {
    it('displays Time header', () => {
      const props = createDefaultProps()

      render(<FlowList {...props} />)

      expect(screen.getByText('Time')).toBeInTheDocument()
    })

    it('displays Source header', () => {
      const props = createDefaultProps()

      render(<FlowList {...props} />)

      expect(screen.getByText('Source')).toBeInTheDocument()
    })

    it('displays Destination header', () => {
      const props = createDefaultProps()

      render(<FlowList {...props} />)

      expect(screen.getByText('Destination')).toBeInTheDocument()
    })

    it('displays Protocol header', () => {
      const props = createDefaultProps()

      render(<FlowList {...props} />)

      expect(screen.getByText('Protocol')).toBeInTheDocument()
    })

    it('displays Status header', () => {
      const props = createDefaultProps()

      render(<FlowList {...props} />)

      expect(screen.getByText('Status')).toBeInTheDocument()
    })

    it('displays Latency header', () => {
      const props = createDefaultProps()

      render(<FlowList {...props} />)

      expect(screen.getByText('Latency')).toBeInTheDocument()
    })

    it('displays Size header', () => {
      const props = createDefaultProps()

      render(<FlowList {...props} />)

      expect(screen.getByText('Size')).toBeInTheDocument()
    })
  })

  describe('row selection', () => {
    it('calls onSelectFlow with flow object when row is clicked', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
      const props = createDefaultProps()
      const flow = createMockFlow({ id: 'test-flow-123' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // Find and click the row
      const row = screen.getByText('client-pod').closest('[class*="grid"][class*="cursor-pointer"]')
      expect(row).toBeInTheDocument()

      await act(async () => {
        await user.click(row!)
        vi.runAllTimers()
      })

      expect(props.onSelect).toHaveBeenCalledTimes(1)
      expect(props.onSelect).toHaveBeenCalledWith(flow)
    })

    it('passes the correct flow object when row is clicked', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
      const props = createDefaultProps()
      const flow = createMockHTTPFlow({ id: 'http-flow-456', srcPod: 'api-client' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const row = screen.getByText('api-client').closest('[class*="cursor-pointer"]')

      await act(async () => {
        await user.click(row!)
        vi.runAllTimers()
      })

      expect(props.onSelect).toHaveBeenCalledWith(expect.objectContaining({
        id: 'http-flow-456',
        protocol: 'HTTP',
        srcPod: 'api-client'
      }))
    })

    it('selected row has highlighted styling', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ id: 'selected-flow' })
      props.flows = [flow]
      props.selectedId = 'selected-flow'

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // Find the row and check for selected styling
      const row = screen.getByText('client-pod').closest('[class*="grid"][class*="cursor-pointer"]')
      expect(row).toHaveClass('bg-podscope-900/30')
      expect(row).toHaveClass('border-l-2')
      expect(row).toHaveClass('border-l-podscope-500')
    })

    it('non-selected row does not have highlighted styling', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ id: 'some-flow' })
      props.flows = [flow]
      props.selectedId = 'different-flow-id'

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const row = screen.getByText('client-pod').closest('[class*="grid"][class*="cursor-pointer"]')
      expect(row).not.toHaveClass('bg-podscope-900/30')
    })

    it('clicking different rows calls onSelect with respective flows', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
      const props = createDefaultProps()
      const flow1 = createMockFlow({ id: 'flow-1', srcPod: 'pod-alpha' })
      const flow2 = createMockFlow({ id: 'flow-2', srcPod: 'pod-beta' })
      props.flows = [flow1, flow2]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      // Click first row
      const row1 = screen.getByText('pod-alpha').closest('[class*="cursor-pointer"]')
      await act(async () => {
        await user.click(row1!)
        vi.runAllTimers()
      })

      expect(props.onSelect).toHaveBeenLastCalledWith(flow1)

      // Click second row
      const row2 = screen.getByText('pod-beta').closest('[class*="cursor-pointer"]')
      await act(async () => {
        await user.click(row2!)
        vi.runAllTimers()
      })

      expect(props.onSelect).toHaveBeenLastCalledWith(flow2)
      expect(props.onSelect).toHaveBeenCalledTimes(2)
    })
  })

  describe('protocol badge display with colors', () => {
    it('HTTP shows green badge', async () => {
      const props = createDefaultProps()
      const flow = createMockHTTPFlow()
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const badge = screen.getByText('HTTP')
      expect(badge).toHaveClass('text-green-400')
      expect(badge).toHaveClass('bg-green-400/10')
    })

    it('HTTPS shows yellow badge', async () => {
      const props = createDefaultProps()
      const flow = createMockTLSFlow({ protocol: 'HTTPS' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const badge = screen.getByText('HTTPS')
      expect(badge).toHaveClass('text-yellow-400')
      expect(badge).toHaveClass('bg-yellow-400/10')
    })

    it('TCP shows blue badge', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ protocol: 'TCP' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const badge = screen.getByText('TCP')
      expect(badge).toHaveClass('text-blue-400')
      expect(badge).toHaveClass('bg-blue-400/10')
    })

    it('protocol badge has correct rounded and padding styling', async () => {
      const props = createDefaultProps()
      const flow = createMockFlow({ protocol: 'TCP' })
      props.flows = [flow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const badge = screen.getByText('TCP')
      expect(badge).toHaveClass('px-2')
      expect(badge).toHaveClass('py-0.5')
      expect(badge).toHaveClass('rounded')
      expect(badge).toHaveClass('text-xs')
      expect(badge).toHaveClass('font-medium')
    })

    it('displays multiple flows with correct protocol badges', async () => {
      const props = createDefaultProps()
      const httpFlow = createMockHTTPFlow({ id: 'http-1' })
      const tcpFlow = createMockFlow({ id: 'tcp-1', protocol: 'TCP' })
      const httpsFlow = createMockTLSFlow({ id: 'https-1', protocol: 'HTTPS' })
      props.flows = [httpFlow, tcpFlow, httpsFlow]

      render(<FlowList {...props} />)

      await act(async () => {
        vi.runAllTimers()
      })

      const httpBadge = screen.getByText('HTTP')
      const tcpBadge = screen.getByText('TCP')
      const httpsBadge = screen.getByText('HTTPS')

      expect(httpBadge).toHaveClass('text-green-400')
      expect(tcpBadge).toHaveClass('text-blue-400')
      expect(httpsBadge).toHaveClass('text-yellow-400')
    })
  })
})

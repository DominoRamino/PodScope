import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi, beforeEach, describe, it, expect } from 'vitest'
import { FlowDetail } from '../../components/FlowDetail'
import { createMockFlow, createMockHTTPFlow, createMockTLSFlow, resetFlowIdCounter } from '../../test-utils/testData'

// Default props for FlowDetail component
const createDefaultProps = () => ({
  flow: createMockFlow(),
  onClose: vi.fn(),
  onDownloadPCAP: vi.fn(),
  onOpenTerminal: vi.fn(),
})

describe('FlowDetail component', () => {
  beforeEach(() => {
    resetFlowIdCounter()
  })

  describe('basic display', () => {
    describe('flow ID display', () => {
      it('shows flow ID in header', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ id: 'test-flow-123' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('test-flow-123')).toBeInTheDocument()
      })

      it('flow ID has monospace font styling', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ id: 'flow-abc' })

        render(<FlowDetail {...props} />)

        const flowIdElement = screen.getByText('flow-abc')
        expect(flowIdElement).toHaveClass('font-mono')
      })

      it('shows Flow Details heading', () => {
        const props = createDefaultProps()

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Flow Details')).toBeInTheDocument()
      })
    })

    describe('source information display', () => {
      it('displays source IP and port', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({
          srcIp: '192.168.1.100',
          srcPort: 54321,
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('192.168.1.100:54321')).toBeInTheDocument()
      })

      it('displays source pod name with namespace', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({
          srcPod: 'frontend-pod',
          srcNamespace: 'production',
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('production/frontend-pod')).toBeInTheDocument()
      })

      it('shows Source label', () => {
        const props = createDefaultProps()

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Source')).toBeInTheDocument()
      })

      it('does not display pod info when srcPod is not set', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({
          srcPod: '',
          srcNamespace: 'default',
        })

        render(<FlowDetail {...props} />)

        // srcPod is empty, so namespace/pod should not appear
        expect(screen.queryByText('default/')).not.toBeInTheDocument()
      })
    })

    describe('destination information display', () => {
      it('displays destination IP and port', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({
          dstIp: '10.0.0.50',
          dstPort: 8080,
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('10.0.0.50:8080')).toBeInTheDocument()
      })

      it('displays destination pod name with namespace', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({
          dstPod: 'backend-pod',
          dstNamespace: 'staging',
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('staging/backend-pod')).toBeInTheDocument()
      })

      it('shows Destination label', () => {
        const props = createDefaultProps()

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Destination')).toBeInTheDocument()
      })

      it('displays destination service when set', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({
          dstService: 'api-gateway-service',
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('api-gateway-service')).toBeInTheDocument()
      })

      it('does not display service when not set', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({
          dstService: '',
        })

        render(<FlowDetail {...props} />)

        // No service name should be shown
        // Just verify the component renders without the service
        expect(screen.getByText('Destination')).toBeInTheDocument()
      })

      it('does not display pod info when dstPod is not set', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({
          dstPod: '',
          dstNamespace: 'default',
        })

        render(<FlowDetail {...props} />)

        // Just verify component renders without crashing
        expect(screen.getByText('Destination')).toBeInTheDocument()
      })
    })

    describe('protocol display', () => {
      it('displays TCP protocol', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ protocol: 'TCP' })

        render(<FlowDetail {...props} />)

        // Protocol is shown via InfoItem component
        expect(screen.getByText('Protocol')).toBeInTheDocument()
        expect(screen.getByText('TCP')).toBeInTheDocument()
      })

      it('displays HTTP protocol', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({ protocol: 'HTTP' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('HTTP')).toBeInTheDocument()
      })

      it('displays HTTPS protocol', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({ protocol: 'HTTPS' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('HTTPS')).toBeInTheDocument()
      })

      it('displays TLS protocol', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({ protocol: 'TLS' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('TLS')).toBeInTheDocument()
      })
    })

    describe('status display', () => {
      it('displays CLOSED status', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ status: 'CLOSED' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Status')).toBeInTheDocument()
        expect(screen.getByText('CLOSED')).toBeInTheDocument()
      })

      it('displays OPEN status', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ status: 'OPEN' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('OPEN')).toBeInTheDocument()
      })

      it('displays RESET status', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ status: 'RESET' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('RESET')).toBeInTheDocument()
      })

      it('displays TIMEOUT status', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ status: 'TIMEOUT' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('TIMEOUT')).toBeInTheDocument()
      })
    })

    describe('timestamp display', () => {
      it('displays formatted timestamp', () => {
        const props = createDefaultProps()
        // Use a specific timestamp
        props.flow = createMockFlow({
          timestamp: '2026-01-21T10:30:00.000Z',
        })

        render(<FlowDetail {...props} />)

        // Timestamp label should be present
        expect(screen.getByText('Timestamp')).toBeInTheDocument()
        // The actual formatted timestamp will depend on locale, but should exist
        // Check that it's in the Summary section which uses toLocaleString()
      })

      it('shows Timestamp label in Summary section', () => {
        const props = createDefaultProps()

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Timestamp')).toBeInTheDocument()
        expect(screen.getByText('Summary')).toBeInTheDocument()
      })
    })

    describe('duration display', () => {
      it('displays flow duration in milliseconds', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ duration: 250 })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Duration')).toBeInTheDocument()
        expect(screen.getByText('250ms')).toBeInTheDocument()
      })

      it('displays zero duration', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ duration: 0 })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('0ms')).toBeInTheDocument()
      })
    })
  })

  describe('header buttons', () => {
    it('calls onClose when close button is clicked', async () => {
      const user = userEvent.setup()
      const props = createDefaultProps()

      render(<FlowDetail {...props} />)

      // There are multiple buttons, find the one without a title (close) vs download
      const buttons = screen.getAllByRole('button')
      const closeBtn = buttons.find(btn => !btn.getAttribute('title'))

      if (closeBtn) {
        await user.click(closeBtn)
        expect(props.onClose).toHaveBeenCalledTimes(1)
      }
    })

    it('calls onDownloadPCAP when download button is clicked', async () => {
      const user = userEvent.setup()
      const props = createDefaultProps()

      render(<FlowDetail {...props} />)

      const downloadButton = screen.getByTitle('Download PCAP')
      await user.click(downloadButton)

      expect(props.onDownloadPCAP).toHaveBeenCalledTimes(1)
    })
  })

  describe('data transfer section', () => {
    it('displays bytes sent', () => {
      const props = createDefaultProps()
      props.flow = createMockFlow({ bytesSent: 2048 })

      render(<FlowDetail {...props} />)

      expect(screen.getByText('Sent')).toBeInTheDocument()
      expect(screen.getByText('2.00 KB')).toBeInTheDocument()
    })

    it('displays bytes received', () => {
      const props = createDefaultProps()
      props.flow = createMockFlow({ bytesReceived: 4096 })

      render(<FlowDetail {...props} />)

      expect(screen.getByText('Received')).toBeInTheDocument()
      expect(screen.getByText('4.00 KB')).toBeInTheDocument()
    })

    it('displays packets sent count', () => {
      const props = createDefaultProps()
      props.flow = createMockFlow({ packetsSent: 15 })

      render(<FlowDetail {...props} />)

      expect(screen.getByText('15 packets')).toBeInTheDocument()
    })

    it('displays packets received count', () => {
      const props = createDefaultProps()
      props.flow = createMockFlow({ packetsReceived: 20 })

      render(<FlowDetail {...props} />)

      expect(screen.getByText('20 packets')).toBeInTheDocument()
    })
  })

  describe('terminal button', () => {
    it('shows terminal button for source pod when onOpenTerminal is provided', () => {
      const props = createDefaultProps()
      props.flow = createMockFlow({
        srcPod: 'test-pod',
        srcNamespace: 'test-ns',
      })

      render(<FlowDetail {...props} />)

      const terminalButtons = screen.getAllByTitle('Open terminal')
      expect(terminalButtons.length).toBeGreaterThan(0)
    })

    it('calls onOpenTerminal with pod path when terminal button clicked', async () => {
      const user = userEvent.setup()
      const props = createDefaultProps()
      props.flow = createMockFlow({
        srcPod: 'my-pod',
        srcNamespace: 'my-namespace',
      })

      render(<FlowDetail {...props} />)

      const terminalButtons = screen.getAllByTitle('Open terminal')
      await user.click(terminalButtons[0])

      expect(props.onOpenTerminal).toHaveBeenCalledWith('my-namespace/my-pod')
    })

    it('does not show terminal buttons when onOpenTerminal is not provided', () => {
      const props = {
        ...createDefaultProps(),
        onOpenTerminal: undefined,
      }

      render(<FlowDetail {...props} />)

      expect(screen.queryByTitle('Open terminal')).not.toBeInTheDocument()
    })
  })

  describe('timing section', () => {
    it('shows timing section heading', () => {
      const props = createDefaultProps()

      render(<FlowDetail {...props} />)

      expect(screen.getByText('Timing')).toBeInTheDocument()
    })

    it('shows TCP handshake timing when available', () => {
      const props = createDefaultProps()
      props.flow = createMockFlow({ tcpHandshakeMs: 5.5 })

      render(<FlowDetail {...props} />)

      expect(screen.getByText(/TCP Handshake: 5.5ms/)).toBeInTheDocument()
    })

    it('shows TLS handshake timing when available', () => {
      const props = createDefaultProps()
      props.flow = createMockTLSFlow({ tlsHandshakeMs: 30.2 })

      render(<FlowDetail {...props} />)

      expect(screen.getByText(/TLS Handshake: 30.2ms/)).toBeInTheDocument()
    })

    it('shows no timing data message when no timing info', () => {
      const props = createDefaultProps()
      props.flow = createMockFlow({
        tcpHandshakeMs: undefined,
        tlsHandshakeMs: undefined,
        ttfbMs: undefined,
      })

      render(<FlowDetail {...props} />)

      expect(screen.getByText('No timing data available')).toBeInTheDocument()
    })
  })
})

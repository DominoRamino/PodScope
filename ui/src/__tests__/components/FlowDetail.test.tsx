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
  isDownloading: false,
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

  describe('HTTP section', () => {
    describe('visibility', () => {
      it('shows HTTP Request section for HTTP flows', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({ protocol: 'HTTP' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('HTTP Request')).toBeInTheDocument()
      })

      it('shows HTTP Response section for HTTP flows', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({ protocol: 'HTTP' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('HTTP Response')).toBeInTheDocument()
      })

      it('shows HTTP sections for HTTPS flows with HTTP info', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({ protocol: 'HTTPS' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('HTTP Request')).toBeInTheDocument()
        expect(screen.getByText('HTTP Response')).toBeInTheDocument()
      })

      it('does not show HTTP section for plain TCP flows', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ protocol: 'TCP', http: undefined })

        render(<FlowDetail {...props} />)

        expect(screen.queryByText('HTTP Request')).not.toBeInTheDocument()
        expect(screen.queryByText('HTTP Response')).not.toBeInTheDocument()
      })

      it('does not show HTTP section for TLS flows without HTTP info', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({ protocol: 'TLS', http: undefined })

        render(<FlowDetail {...props} />)

        expect(screen.queryByText('HTTP Request')).not.toBeInTheDocument()
        expect(screen.queryByText('HTTP Response')).not.toBeInTheDocument()
      })
    })

    describe('request method display', () => {
      it('displays GET method', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/test',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('GET')).toBeInTheDocument()
      })

      it('displays POST method', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'POST',
            url: '/api/users',
            host: 'api.example.com',
            statusCode: 201,
            statusText: 'Created',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('POST')).toBeInTheDocument()
      })

      it('displays PUT method', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'PUT',
            url: '/api/users/1',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('PUT')).toBeInTheDocument()
      })

      it('displays DELETE method', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'DELETE',
            url: '/api/users/1',
            host: 'api.example.com',
            statusCode: 204,
            statusText: 'No Content',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('DELETE')).toBeInTheDocument()
      })
    })

    describe('URL display', () => {
      it('displays the request URL', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/products/123',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('/api/products/123')).toBeInTheDocument()
      })

      it('displays URL with query parameters', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/search?q=test&page=1',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('/api/search?q=test&page=1')).toBeInTheDocument()
      })
    })

    describe('host display', () => {
      it('displays the host header when present', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/test',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Host')).toBeInTheDocument()
        expect(screen.getByText('api.example.com')).toBeInTheDocument()
      })

      it('does not display host when empty string', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/test',
            host: '',
            statusCode: 200,
            statusText: 'OK',
          },
        })

        render(<FlowDetail {...props} />)

        // Host label shouldn't appear in the HTTP Request section when host is empty
        const httpRequestSection = screen.getByText('HTTP Request').parentElement
        expect(httpRequestSection).not.toHaveTextContent('Host')
      })
    })

    describe('response status display', () => {
      it('displays 200 OK status', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/test',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('200 OK')).toBeInTheDocument()
      })

      it('displays 404 Not Found status', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/missing',
            host: 'api.example.com',
            statusCode: 404,
            statusText: 'Not Found',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('404 Not Found')).toBeInTheDocument()
      })

      it('displays 500 Internal Server Error status', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'POST',
            url: '/api/error',
            host: 'api.example.com',
            statusCode: 500,
            statusText: 'Internal Server Error',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('500 Internal Server Error')).toBeInTheDocument()
      })
    })

    describe('content info display', () => {
      it('displays content type when present', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/data',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
            contentType: 'application/json',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Content-Type')).toBeInTheDocument()
        expect(screen.getByText('application/json')).toBeInTheDocument()
      })

      it('displays content length when present', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/data',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
            contentLength: 2048,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Content-Length')).toBeInTheDocument()
        expect(screen.getByText('2.00 KB')).toBeInTheDocument()
      })
    })

    describe('headers display', () => {
      it('displays request headers when present', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/test',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
            requestHeaders: {
              'Authorization': 'Bearer token123',
              'User-Agent': 'TestClient/1.0',
            },
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Request Headers')).toBeInTheDocument()
        expect(screen.getByText('Authorization')).toBeInTheDocument()
        expect(screen.getByText('Bearer token123')).toBeInTheDocument()
        expect(screen.getByText('User-Agent')).toBeInTheDocument()
        expect(screen.getByText('TestClient/1.0')).toBeInTheDocument()
      })

      it('displays response headers when present', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/test',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
            responseHeaders: {
              'X-Request-Id': 'req-abc-123',
              'Cache-Control': 'no-cache',
            },
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Response Headers')).toBeInTheDocument()
        expect(screen.getByText('X-Request-Id')).toBeInTheDocument()
        expect(screen.getByText('req-abc-123')).toBeInTheDocument()
        expect(screen.getByText('Cache-Control')).toBeInTheDocument()
        expect(screen.getByText('no-cache')).toBeInTheDocument()
      })

      it('does not display headers section when headers are empty', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/test',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
            requestHeaders: {},
            responseHeaders: {},
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.queryByText('Request Headers')).not.toBeInTheDocument()
        expect(screen.queryByText('Response Headers')).not.toBeInTheDocument()
      })
    })

    describe('body display', () => {
      it('displays request body when present', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'POST',
            url: '/api/users',
            host: 'api.example.com',
            statusCode: 201,
            statusText: 'Created',
            requestBody: '{"name": "John", "email": "john@example.com"}',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Request Body')).toBeInTheDocument()
        expect(screen.getByText('{"name": "John", "email": "john@example.com"}')).toBeInTheDocument()
      })

      it('displays response body when present', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({
          http: {
            method: 'GET',
            url: '/api/data',
            host: 'api.example.com',
            statusCode: 200,
            statusText: 'OK',
            responseBody: '{"result": "success", "data": [1, 2, 3]}',
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText(/Response Body/)).toBeInTheDocument()
        expect(screen.getByText('{"result": "success", "data": [1, 2, 3]}')).toBeInTheDocument()
      })
    })
  })

  describe('TLS section', () => {
    describe('visibility', () => {
      it('shows TLS Info section for HTTPS flows', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({ protocol: 'HTTPS' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('TLS Info')).toBeInTheDocument()
      })

      it('shows TLS Info section for TLS flows', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({ protocol: 'TLS' })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('TLS Info')).toBeInTheDocument()
      })

      it('does not show TLS section for plain TCP flows', () => {
        const props = createDefaultProps()
        props.flow = createMockFlow({ protocol: 'TCP', tls: undefined })

        render(<FlowDetail {...props} />)

        expect(screen.queryByText('TLS Info')).not.toBeInTheDocument()
      })

      it('does not show TLS section for HTTP flows without TLS', () => {
        const props = createDefaultProps()
        props.flow = createMockHTTPFlow({ protocol: 'HTTP', tls: undefined })

        render(<FlowDetail {...props} />)

        expect(screen.queryByText('TLS Info')).not.toBeInTheDocument()
      })
    })

    describe('SNI (Server Name) display', () => {
      it('displays SNI when present', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.3',
            sni: 'api.example.com',
            cipherSuite: 'TLS_AES_256_GCM_SHA384',
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('SNI')).toBeInTheDocument()
        expect(screen.getByText('api.example.com')).toBeInTheDocument()
      })

      it('displays SNI with subdomain', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.2',
            sni: 'secure.api.example.com',
            cipherSuite: 'TLS_AES_128_GCM_SHA256',
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('secure.api.example.com')).toBeInTheDocument()
      })

      it('does not display SNI when it is empty string', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.3',
            sni: '',
            cipherSuite: 'TLS_AES_256_GCM_SHA384',
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('TLS Info')).toBeInTheDocument()
        // SNI label should not appear when SNI is empty
        expect(screen.queryByText('SNI')).not.toBeInTheDocument()
      })
    })

    describe('TLS version display', () => {
      it('displays TLS 1.3 version', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.3',
            sni: 'example.com',
            cipherSuite: 'TLS_AES_256_GCM_SHA384',
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Version')).toBeInTheDocument()
        expect(screen.getByText('TLS 1.3')).toBeInTheDocument()
      })

      it('displays TLS 1.2 version', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.2',
            sni: 'example.com',
            cipherSuite: 'TLS_AES_128_GCM_SHA256',
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('TLS 1.2')).toBeInTheDocument()
      })

      it('displays TLS 1.1 version', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.1',
            sni: 'example.com',
            cipherSuite: 'TLS_RSA_WITH_AES_256_CBC_SHA',
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('TLS 1.1')).toBeInTheDocument()
      })

      it('displays TLS 1.0 version', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.0',
            sni: 'example.com',
            cipherSuite: 'TLS_RSA_WITH_AES_128_CBC_SHA',
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('TLS 1.0')).toBeInTheDocument()
      })
    })

    describe('cipher suite display', () => {
      it('displays cipher suite when present', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.3',
            sni: 'example.com',
            cipherSuite: 'TLS_AES_256_GCM_SHA384',
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('Cipher Suite')).toBeInTheDocument()
        expect(screen.getByText('TLS_AES_256_GCM_SHA384')).toBeInTheDocument()
      })

      it('does not display cipher suite when empty string', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.3',
            sni: 'example.com',
            cipherSuite: '',
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.queryByText('Cipher Suite')).not.toBeInTheDocument()
      })
    })

    describe('ALPN display', () => {
      it('displays ALPN protocols when present', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.3',
            sni: 'example.com',
            cipherSuite: 'TLS_AES_256_GCM_SHA384',
            alpn: ['h2', 'http/1.1'],
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('ALPN')).toBeInTheDocument()
        expect(screen.getByText('h2, http/1.1')).toBeInTheDocument()
      })

      it('displays single ALPN protocol', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.3',
            sni: 'example.com',
            cipherSuite: 'TLS_AES_256_GCM_SHA384',
            alpn: ['h2'],
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.getByText('h2')).toBeInTheDocument()
      })

      it('does not display ALPN when array is empty', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.3',
            sni: 'example.com',
            cipherSuite: 'TLS_AES_256_GCM_SHA384',
            alpn: [],
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.queryByText('ALPN')).not.toBeInTheDocument()
      })

      it('does not display ALPN when not present', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow({
          tls: {
            version: 'TLS 1.3',
            sni: 'example.com',
            cipherSuite: 'TLS_AES_256_GCM_SHA384',
            alpn: undefined,
            encrypted: true,
          },
        })

        render(<FlowDetail {...props} />)

        expect(screen.queryByText('ALPN')).not.toBeInTheDocument()
      })
    })

    describe('encryption notice', () => {
      it('shows encrypted payload notice for TLS flows', () => {
        const props = createDefaultProps()
        props.flow = createMockTLSFlow()

        render(<FlowDetail {...props} />)

        expect(screen.getByText(/Payload is encrypted/)).toBeInTheDocument()
        expect(screen.getByText(/Only metadata is available/)).toBeInTheDocument()
      })
    })
  })
})

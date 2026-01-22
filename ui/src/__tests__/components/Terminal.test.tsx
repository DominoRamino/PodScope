import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi, beforeEach, afterEach, describe, it, expect } from 'vitest'
import { Terminal } from '../../components/Terminal'
import { createMockWebSocketClass, clearMockWebSocketInstances } from '../../test-utils/mockWebSocket'

// Mock XTerm.js Terminal class
const mockTerminalInstance = {
  loadAddon: vi.fn(),
  open: vi.fn(),
  writeln: vi.fn(),
  write: vi.fn(),
  onData: vi.fn(),
  dispose: vi.fn(),
  cols: 80,
  rows: 24,
}

vi.mock('@xterm/xterm', () => ({
  Terminal: vi.fn(() => mockTerminalInstance),
}))

// Mock FitAddon
const mockFitAddonInstance = {
  fit: vi.fn(),
}

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: vi.fn(() => mockFitAddonInstance),
}))

// Mock WebLinksAddon
const mockWebLinksAddonInstance = {}

vi.mock('@xterm/addon-web-links', () => ({
  WebLinksAddon: vi.fn(() => mockWebLinksAddonInstance),
}))

// Mock xterm CSS import
vi.mock('@xterm/xterm/css/xterm.css', () => ({}))

// Create mock WebSocket class for capturing instances
const { MockWebSocketClass, instances } = createMockWebSocketClass()

// Store original WebSocket for restoration
let originalWebSocket: typeof WebSocket

// Default props for Terminal component
const createDefaultProps = (): {
  namespace: string
  podName: string
  container?: string
  onClose: ReturnType<typeof vi.fn>
} => ({
  namespace: 'default',
  podName: 'test-pod',
  onClose: vi.fn(),
})

describe('Terminal component', () => {
  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks()
    clearMockWebSocketInstances(instances)

    // Store original and override WebSocket with our mock class
    originalWebSocket = globalThis.WebSocket
    globalThis.WebSocket = MockWebSocketClass as unknown as typeof WebSocket

    // Mock window.location for WebSocket URL construction
    Object.defineProperty(window, 'location', {
      value: {
        protocol: 'http:',
        host: 'localhost:8080',
      },
      writable: true,
    })

    // Reset terminal mock instance functions
    mockTerminalInstance.loadAddon.mockClear()
    mockTerminalInstance.open.mockClear()
    mockTerminalInstance.writeln.mockClear()
    mockTerminalInstance.write.mockClear()
    mockTerminalInstance.onData.mockClear()
    mockTerminalInstance.dispose.mockClear()
    mockFitAddonInstance.fit.mockClear()
  })

  afterEach(() => {
    // Restore original WebSocket without clearing all global stubs
    globalThis.WebSocket = originalWebSocket
  })

  describe('initialization', () => {
    describe('XTerm.js initialization', () => {
      it('creates Terminal instance with correct options', async () => {
        const { Terminal: TerminalMock } = await import('@xterm/xterm')
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        expect(TerminalMock).toHaveBeenCalledWith(
          expect.objectContaining({
            cursorBlink: true,
            fontSize: 14,
            fontFamily: 'Menlo, Monaco, "Courier New", monospace',
            allowProposedApi: true,
          })
        )
      })

      it('loads FitAddon onto terminal', async () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        expect(mockTerminalInstance.loadAddon).toHaveBeenCalledWith(mockFitAddonInstance)
      })

      it('loads WebLinksAddon onto terminal', async () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        expect(mockTerminalInstance.loadAddon).toHaveBeenCalledWith(mockWebLinksAddonInstance)
      })

      it('opens terminal in container element', () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        expect(mockTerminalInstance.open).toHaveBeenCalled()
        // The argument should be an HTMLDivElement (the container ref)
        const openArg = mockTerminalInstance.open.mock.calls[0][0]
        expect(openArg).toBeInstanceOf(HTMLDivElement)
      })

      it('calls fit() after terminal initialization', () => {
        vi.useFakeTimers()
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        // fit() is called via setTimeout(..., 0)
        vi.runAllTimers()

        expect(mockFitAddonInstance.fit).toHaveBeenCalled()
        vi.useRealTimers()
      })
    })

    describe('container rendering', () => {
      it('renders container div for terminal', () => {
        const props = createDefaultProps()

        const { container } = render(<Terminal {...props} />)

        // The terminal container div has class flex-1 and overflow-hidden
        const terminalDiv = container.querySelector('.flex-1.overflow-hidden')
        expect(terminalDiv).toBeInTheDocument()
      })

      it('renders with proper styling classes', () => {
        const props = createDefaultProps()

        const { container } = render(<Terminal {...props} />)

        // Main container has bg-slate-800 and border classes
        const mainContainer = container.querySelector('.bg-slate-800')
        expect(mainContainer).toBeInTheDocument()
        expect(mainContainer).toHaveClass('flex', 'flex-col', 'h-full')
      })

      it('renders header section', () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        expect(screen.getByText('Terminal')).toBeInTheDocument()
      })

      it('displays namespace and pod name in header', () => {
        const props = createDefaultProps()
        props.namespace = 'production'
        props.podName = 'my-app-pod'

        render(<Terminal {...props} />)

        expect(screen.getByText('production/my-app-pod')).toBeInTheDocument()
      })
    })
  })

  describe('WebSocket connection', () => {
    describe('URL construction', () => {
      it('creates WebSocket with correct URL including namespace parameter', () => {
        const props = createDefaultProps()
        props.namespace = 'test-namespace'

        render(<Terminal {...props} />)

        expect(instances.length).toBe(1)
        expect(instances[0].url).toContain('namespace=test-namespace')
      })

      it('creates WebSocket with correct URL including pod parameter', () => {
        const props = createDefaultProps()
        props.podName = 'my-pod'

        render(<Terminal {...props} />)

        expect(instances.length).toBe(1)
        expect(instances[0].url).toContain('pod=my-pod')
      })

      it('uses ws:// protocol for http: location', () => {
        Object.defineProperty(window, 'location', {
          value: {
            protocol: 'http:',
            host: 'localhost:8080',
          },
          writable: true,
        })
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        expect(instances.length).toBe(1)
        expect(instances[0].url).toMatch(/^ws:\/\//)
      })

      it('uses wss:// protocol for https: location', () => {
        Object.defineProperty(window, 'location', {
          value: {
            protocol: 'https:',
            host: 'secure.example.com',
          },
          writable: true,
        })
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        expect(instances.length).toBe(1)
        expect(instances[0].url).toMatch(/^wss:\/\//)
      })

      it('constructs URL with correct host', () => {
        Object.defineProperty(window, 'location', {
          value: {
            protocol: 'http:',
            host: 'myhost:3000',
          },
          writable: true,
        })
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        expect(instances[0].url).toContain('myhost:3000')
      })

      it('uses /api/terminal/ws endpoint path', () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        expect(instances[0].url).toContain('/api/terminal/ws')
      })

      it('includes container parameter when provided', () => {
        const props = createDefaultProps()
        props.container = 'my-container'

        render(<Terminal {...props} />)

        expect(instances[0].url).toContain('container=my-container')
      })

      it('does not include container parameter when not provided', () => {
        const props = createDefaultProps()
        // container is undefined/not set

        render(<Terminal {...props} />)

        expect(instances[0].url).not.toContain('container=')
      })
    })

    describe('WebSocket events', () => {
      it('writes connection message on WebSocket open', () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        // Simulate WebSocket open
        instances[0].simulateOpen()

        expect(mockTerminalInstance.writeln).toHaveBeenCalledWith(
          expect.stringContaining('Connected to agent terminal')
        )
      })

      it('writes pod info on WebSocket open', () => {
        const props = createDefaultProps()
        props.namespace = 'prod'
        props.podName = 'backend-pod'

        render(<Terminal {...props} />)

        instances[0].simulateOpen()

        expect(mockTerminalInstance.writeln).toHaveBeenCalledWith(
          expect.stringContaining('prod/backend-pod')
        )
      })

      it('sends resize message on WebSocket open', () => {
        vi.useFakeTimers()
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        // Need to advance timers for fit() to be called
        vi.runAllTimers()

        // Simulate WebSocket open
        instances[0].readyState = 1 // WebSocket.OPEN
        instances[0].simulateOpen()

        expect(instances[0].send).toHaveBeenCalledWith(
          expect.stringContaining('"type":"resize"')
        )
        vi.useRealTimers()
      })

      it('writes error message on WebSocket error', () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        instances[0].simulateError()

        expect(mockTerminalInstance.writeln).toHaveBeenCalledWith(
          expect.stringContaining('WebSocket error')
        )
      })

      it('writes close message on WebSocket close', () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        instances[0].simulateClose()

        expect(mockTerminalInstance.writeln).toHaveBeenCalledWith(
          expect.stringContaining('Connection closed')
        )
      })

      it('writes received text messages to terminal', () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        instances[0].simulateMessage('Hello from server')

        expect(mockTerminalInstance.write).toHaveBeenCalledWith('Hello from server')
      })
    })

    describe('terminal input handling', () => {
      it('registers onData callback for terminal input', () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        expect(mockTerminalInstance.onData).toHaveBeenCalled()
      })

      it('sends input data to WebSocket when connected', () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        // Get the onData callback
        const onDataCallback = mockTerminalInstance.onData.mock.calls[0][0]

        // Set WebSocket to OPEN state
        instances[0].readyState = 1

        // Simulate typing
        onDataCallback('ls -la')

        expect(instances[0].send).toHaveBeenCalledWith(
          JSON.stringify({ type: 'input', data: 'ls -la' })
        )
      })

      it('does not send input when WebSocket is not open', () => {
        const props = createDefaultProps()

        render(<Terminal {...props} />)

        const onDataCallback = mockTerminalInstance.onData.mock.calls[0][0]

        // WebSocket is in CONNECTING state (readyState = 0)
        instances[0].readyState = 0

        onDataCallback('test input')

        expect(instances[0].send).not.toHaveBeenCalled()
      })
    })
  })

  describe('close button', () => {
    it('renders close button', () => {
      const props = createDefaultProps()

      render(<Terminal {...props} />)

      const closeButton = screen.getByTitle('Close terminal')
      expect(closeButton).toBeInTheDocument()
    })

    it('calls onClose callback when close button is clicked', async () => {
      const user = userEvent.setup()
      const props = createDefaultProps()

      render(<Terminal {...props} />)

      const closeButton = screen.getByTitle('Close terminal')
      await user.click(closeButton)

      expect(props.onClose).toHaveBeenCalledTimes(1)
    })

    it('close button has hover styling', () => {
      const props = createDefaultProps()

      render(<Terminal {...props} />)

      const closeButton = screen.getByTitle('Close terminal')
      expect(closeButton).toHaveClass('hover:bg-slate-700')
    })
  })

  describe('maximize/minimize functionality', () => {
    it('renders maximize button when onToggleMaximize is provided', () => {
      const props = {
        ...createDefaultProps(),
        onToggleMaximize: vi.fn(),
        isMaximized: false,
      }

      render(<Terminal {...props} />)

      const maximizeButton = screen.getByTitle('Maximize')
      expect(maximizeButton).toBeInTheDocument()
    })

    it('does not render maximize button when onToggleMaximize is not provided', () => {
      const props = createDefaultProps()
      // onToggleMaximize is undefined

      render(<Terminal {...props} />)

      expect(screen.queryByTitle('Maximize')).not.toBeInTheDocument()
      expect(screen.queryByTitle('Minimize')).not.toBeInTheDocument()
    })

    it('shows Minimize title when isMaximized is true', () => {
      const props = {
        ...createDefaultProps(),
        onToggleMaximize: vi.fn(),
        isMaximized: true,
      }

      render(<Terminal {...props} />)

      expect(screen.getByTitle('Minimize')).toBeInTheDocument()
    })

    it('calls onToggleMaximize when maximize button is clicked', async () => {
      const user = userEvent.setup()
      const props = {
        ...createDefaultProps(),
        onToggleMaximize: vi.fn(),
        isMaximized: false,
      }

      render(<Terminal {...props} />)

      const maximizeButton = screen.getByTitle('Maximize')
      await user.click(maximizeButton)

      expect(props.onToggleMaximize).toHaveBeenCalledTimes(1)
    })
  })

  describe('cleanup on unmount', () => {
    it('disposes terminal on unmount', () => {
      const props = createDefaultProps()

      const { unmount } = render(<Terminal {...props} />)

      unmount()

      expect(mockTerminalInstance.dispose).toHaveBeenCalled()
    })

    it('closes WebSocket on unmount', () => {
      const props = createDefaultProps()

      const { unmount } = render(<Terminal {...props} />)

      unmount()

      expect(instances[0].close).toHaveBeenCalled()
    })
  })
})

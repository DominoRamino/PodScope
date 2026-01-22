import { vi, type Mock } from 'vitest'

/**
 * Interface for the mock WebSocket instance with helper methods
 */
export interface MockWebSocketInstance {
  // Standard WebSocket properties
  url: string
  readyState: number
  onopen: ((event: Event) => void) | null
  onclose: ((event: CloseEvent) => void) | null
  onmessage: ((event: MessageEvent) => void) | null
  onerror: ((event: Event) => void) | null

  // Standard WebSocket methods (spied)
  send: Mock
  close: Mock

  // Helper methods for testing
  simulateMessage: (data: unknown) => void
  simulateOpen: () => void
  simulateClose: (code?: number, reason?: string) => void
  simulateError: (message?: string) => void
}

// WebSocket readyState constants
const CONNECTING = 0
const OPEN = 1
const CLOSING = 2
const CLOSED = 3

/**
 * Creates a mock WebSocket instance with helper methods for testing.
 *
 * Unlike the global MockWebSocket in vitest.setup.ts, this factory creates
 * controllable instances where you manually trigger events instead of
 * auto-connecting on construction.
 *
 * @param url - The WebSocket URL (defaults to 'ws://localhost:8080')
 * @returns MockWebSocketInstance with spy methods and simulation helpers
 *
 * @example
 * ```ts
 * const ws = createMockWebSocket('ws://localhost:8080/api/flows/ws')
 *
 * // Simulate connection opening
 * ws.simulateOpen()
 *
 * // Simulate receiving a message
 * ws.simulateMessage({ type: 'flow', data: mockFlow })
 *
 * // Verify send was called
 * expect(ws.send).toHaveBeenCalledWith('{"type": "ping"}')
 *
 * // Simulate close
 * ws.simulateClose(1000, 'Normal closure')
 * ```
 */
export function createMockWebSocket(url: string = 'ws://localhost:8080'): MockWebSocketInstance {
  const instance: MockWebSocketInstance = {
    url,
    readyState: CONNECTING,
    onopen: null,
    onclose: null,
    onmessage: null,
    onerror: null,

    send: vi.fn(),
    close: vi.fn(() => {
      instance.readyState = CLOSED
    }),

    /**
     * Simulates receiving a message from the server.
     * If data is an object, it will be JSON-stringified.
     * If data is a string, it will be sent as-is.
     *
     * @param data - The message data (object will be JSON-stringified)
     */
    simulateMessage(data: unknown): void {
      if (instance.onmessage) {
        const messageData = typeof data === 'string' ? data : JSON.stringify(data)
        const event = new MessageEvent('message', { data: messageData })
        instance.onmessage(event)
      }
    },

    /**
     * Simulates the WebSocket connection opening.
     * Sets readyState to OPEN and triggers onopen callback.
     */
    simulateOpen(): void {
      instance.readyState = OPEN
      if (instance.onopen) {
        const event = new Event('open')
        instance.onopen(event)
      }
    },

    /**
     * Simulates the WebSocket connection closing.
     * Sets readyState to CLOSED and triggers onclose callback.
     *
     * @param code - Close code (default: 1000 for normal closure)
     * @param reason - Close reason (default: empty string)
     */
    simulateClose(code: number = 1000, reason: string = ''): void {
      instance.readyState = CLOSED
      if (instance.onclose) {
        const event = new CloseEvent('close', { code, reason, wasClean: code === 1000 })
        instance.onclose(event)
      }
    },

    /**
     * Simulates a WebSocket error.
     * Triggers onerror callback.
     *
     * @param message - Optional error message
     */
    simulateError(message: string = 'WebSocket error'): void {
      if (instance.onerror) {
        const event = new Event('error')
        // Add custom property for error details
        Object.defineProperty(event, 'message', { value: message })
        instance.onerror(event)
      }
    },
  }

  return instance
}

/**
 * Creates a mock WebSocket constructor that captures created instances.
 * Useful for testing components that create their own WebSocket connections.
 *
 * @returns Object with the mock constructor and array of created instances
 *
 * @example
 * ```ts
 * const { MockWebSocketClass, instances } = createMockWebSocketClass()
 * vi.stubGlobal('WebSocket', MockWebSocketClass)
 *
 * // Component creates WebSocket internally
 * render(<MyComponent />)
 *
 * // Access the created instance
 * const ws = instances[0]
 * ws.simulateOpen()
 * ws.simulateMessage({ type: 'data', payload: {...} })
 * ```
 */
export function createMockWebSocketClass(): {
  MockWebSocketClass: new (url: string) => MockWebSocketInstance
  instances: MockWebSocketInstance[]
} {
  const instances: MockWebSocketInstance[] = []

  class MockWebSocketClass implements MockWebSocketInstance {
    static CONNECTING = CONNECTING
    static OPEN = OPEN
    static CLOSING = CLOSING
    static CLOSED = CLOSED

    url: string
    readyState: number = CONNECTING
    onopen: ((event: Event) => void) | null = null
    onclose: ((event: CloseEvent) => void) | null = null
    onmessage: ((event: MessageEvent) => void) | null = null
    onerror: ((event: Event) => void) | null = null
    send: Mock
    close: Mock

    constructor(url: string) {
      this.url = url
      this.send = vi.fn()
      this.close = vi.fn(() => {
        this.readyState = CLOSED
      })
      instances.push(this)
    }

    simulateMessage(data: unknown): void {
      if (this.onmessage) {
        const messageData = typeof data === 'string' ? data : JSON.stringify(data)
        const event = new MessageEvent('message', { data: messageData })
        this.onmessage(event)
      }
    }

    simulateOpen(): void {
      this.readyState = OPEN
      if (this.onopen) {
        const event = new Event('open')
        this.onopen(event)
      }
    }

    simulateClose(code: number = 1000, reason: string = ''): void {
      this.readyState = CLOSED
      if (this.onclose) {
        const event = new CloseEvent('close', { code, reason, wasClean: code === 1000 })
        this.onclose(event)
      }
    }

    simulateError(message: string = 'WebSocket error'): void {
      if (this.onerror) {
        const event = new Event('error')
        Object.defineProperty(event, 'message', { value: message })
        this.onerror(event)
      }
    }
  }

  return {
    MockWebSocketClass: MockWebSocketClass as unknown as new (url: string) => MockWebSocketInstance,
    instances,
  }
}

/**
 * Clears the instances array from a createMockWebSocketClass() result.
 * Useful for test cleanup between test cases.
 *
 * @param instances - The instances array to clear
 */
export function clearMockWebSocketInstances(instances: MockWebSocketInstance[]): void {
  instances.length = 0
}

// Export constants for test assertions
export const WebSocketReadyState = {
  CONNECTING,
  OPEN,
  CLOSING,
  CLOSED,
} as const

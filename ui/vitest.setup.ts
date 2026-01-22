import '@testing-library/jest-dom'
import { cleanup } from '@testing-library/react'
import { afterEach, vi } from 'vitest'

// Cleanup after each test
afterEach(() => {
  cleanup()
})

// WebSocket mock class
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  url: string
  readyState: number = MockWebSocket.CONNECTING
  onopen: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null

  constructor(url: string) {
    this.url = url
    // Simulate async connection
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN
      if (this.onopen) {
        this.onopen(new Event('open'))
      }
    }, 0)
  }

  send = vi.fn()
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent('close'))
    }
  })
}

// Replace global WebSocket with mock
vi.stubGlobal('WebSocket', MockWebSocket)

// window.matchMedia mock for responsive components
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(), // deprecated
    removeListener: vi.fn(), // deprecated
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// URL.createObjectURL mock for blob downloads
URL.createObjectURL = vi.fn(() => 'blob:mock-url')
URL.revokeObjectURL = vi.fn()

// ResizeObserver mock for virtualized lists (@tanstack/react-virtual)
// The callback needs to be called with entry data for the virtualizer to measure
class MockResizeObserver {
  private callback: ResizeObserverCallback
  private observedElements: Set<Element> = new Set()

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback
  }

  observe(element: Element) {
    this.observedElements.add(element)
    // Immediately trigger callback with mock dimensions
    const entry = {
      target: element,
      contentRect: {
        width: 800,
        height: 500,
        top: 0,
        left: 0,
        bottom: 500,
        right: 800,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      },
      borderBoxSize: [{ inlineSize: 800, blockSize: 500 }],
      contentBoxSize: [{ inlineSize: 800, blockSize: 500 }],
      devicePixelContentBoxSize: [{ inlineSize: 800, blockSize: 500 }],
    } as unknown as ResizeObserverEntry

    // Call callback async to match real behavior
    setTimeout(() => {
      this.callback([entry], this)
    }, 0)
  }

  unobserve(element: Element) {
    this.observedElements.delete(element)
  }

  disconnect() {
    this.observedElements.clear()
  }
}
vi.stubGlobal('ResizeObserver', MockResizeObserver)

// Mock element measurement for virtualized lists
// The virtual list needs getBoundingClientRect to return a height > 0
const originalGetBoundingClientRect = Element.prototype.getBoundingClientRect
Element.prototype.getBoundingClientRect = function () {
  const result = originalGetBoundingClientRect.call(this)
  // If this element has overflow-y-auto class (virtual scroll container), return a usable height
  if (this.classList?.contains('overflow-y-auto')) {
    return {
      ...result,
      height: 500, // Virtual container height
      width: 800,
      top: 0,
      left: 0,
      bottom: 500,
      right: 800,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    }
  }
  return result
}

// Mock scrollHeight and clientHeight for virtual scroll containers
Object.defineProperty(HTMLElement.prototype, 'scrollHeight', {
  get() {
    if (this.classList?.contains('overflow-y-auto')) {
      return 500
    }
    return 0
  },
})

Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
  get() {
    if (this.classList?.contains('overflow-y-auto')) {
      return 500
    }
    return 0
  },
})

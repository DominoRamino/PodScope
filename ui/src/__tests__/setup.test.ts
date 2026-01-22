/**
 * Smoke tests to verify the testing setup works correctly.
 * These tests validate that Vitest, React Testing Library, jest-dom matchers,
 * and mock utilities are all functioning as expected.
 */
import { describe, it, expect, test } from 'vitest'
import { render, screen } from '@testing-library/react'
import React from 'react'

// Import mock utilities to verify they are importable and functional
import {
  createMockFlow,
  createMockHTTPFlow,
  createMockTLSFlow,
  resetFlowIdCounter,
} from '../test-utils/testData'
import {
  createMockWebSocket,
  createMockWebSocketClass,
  clearMockWebSocketInstances,
  WebSocketReadyState,
} from '../test-utils/mockWebSocket'

describe('Vitest Setup', () => {
  it('runs a simple expect assertion', () => {
    expect(1 + 1).toBe(2)
  })

  it('handles boolean assertions', () => {
    expect(true).toBe(true)
    expect(false).toBe(false)
  })

  it('handles string assertions', () => {
    expect('hello').toBe('hello')
    expect('hello world').toContain('world')
  })

  it('handles array assertions', () => {
    expect([1, 2, 3]).toEqual([1, 2, 3])
    expect([1, 2, 3]).toHaveLength(3)
    expect([1, 2, 3]).toContain(2)
  })

  it('handles object assertions', () => {
    const obj = { a: 1, b: 2 }
    expect(obj).toEqual({ a: 1, b: 2 })
    expect(obj).toHaveProperty('a')
    expect(obj).toHaveProperty('a', 1)
  })
})

describe('React Testing Library', () => {
  it('renders a simple div element', () => {
    render(React.createElement('div', { 'data-testid': 'test-div' }, 'Hello World'))
    const element = screen.getByTestId('test-div')
    expect(element).toBeDefined()
    expect(element.textContent).toBe('Hello World')
  })

  it('renders nested elements', () => {
    render(
      React.createElement('div', { 'data-testid': 'parent' },
        React.createElement('span', { 'data-testid': 'child' }, 'Child content')
      )
    )
    const parent = screen.getByTestId('parent')
    const child = screen.getByTestId('child')
    expect(parent).toBeDefined()
    expect(child).toBeDefined()
    expect(child.textContent).toBe('Child content')
  })

  it('finds elements by text content', () => {
    render(React.createElement('button', null, 'Click me'))
    const button = screen.getByText('Click me')
    expect(button.tagName).toBe('BUTTON')
  })

  it('finds elements by role', () => {
    render(React.createElement('button', { type: 'submit' }, 'Submit'))
    const button = screen.getByRole('button', { name: 'Submit' })
    expect(button).toBeDefined()
  })
})

describe('jest-dom matchers', () => {
  it('uses toBeInTheDocument matcher', () => {
    render(React.createElement('div', { 'data-testid': 'in-dom' }, 'I am in the document'))
    const element = screen.getByTestId('in-dom')
    expect(element).toBeInTheDocument()
  })

  it('uses toHaveTextContent matcher', () => {
    render(React.createElement('p', { 'data-testid': 'text-content' }, 'Some text here'))
    const element = screen.getByTestId('text-content')
    expect(element).toHaveTextContent('Some text here')
    expect(element).toHaveTextContent(/text/i)
  })

  it('uses toBeVisible matcher', () => {
    render(React.createElement('div', { 'data-testid': 'visible-div' }, 'Visible'))
    const element = screen.getByTestId('visible-div')
    expect(element).toBeVisible()
  })

  it('uses toHaveAttribute matcher', () => {
    render(React.createElement('input', { 'data-testid': 'input', type: 'text', placeholder: 'Enter text' }))
    const element = screen.getByTestId('input')
    expect(element).toHaveAttribute('type', 'text')
    expect(element).toHaveAttribute('placeholder', 'Enter text')
  })

  it('uses toHaveClass matcher', () => {
    render(React.createElement('div', { 'data-testid': 'with-class', className: 'my-class another-class' }))
    const element = screen.getByTestId('with-class')
    expect(element).toHaveClass('my-class')
    expect(element).toHaveClass('another-class')
  })
})

describe('Mock Utilities - testData', () => {
  beforeEach(() => {
    resetFlowIdCounter()
  })

  it('imports createMockFlow successfully', () => {
    expect(createMockFlow).toBeTypeOf('function')
  })

  it('creates a basic TCP flow', () => {
    const flow = createMockFlow()
    expect(flow).toBeDefined()
    expect(flow.id).toBe('flow-1')
    expect(flow.protocol).toBe('TCP')
    expect(flow.srcIp).toBeDefined()
    expect(flow.dstIp).toBeDefined()
  })

  it('creates flows with auto-incrementing IDs', () => {
    const flow1 = createMockFlow()
    const flow2 = createMockFlow()
    expect(flow1.id).toBe('flow-1')
    expect(flow2.id).toBe('flow-2')
  })

  it('allows overriding flow properties', () => {
    const flow = createMockFlow({ srcIp: '192.168.1.1', protocol: 'HTTP' })
    expect(flow.srcIp).toBe('192.168.1.1')
    expect(flow.protocol).toBe('HTTP')
  })

  it('imports createMockHTTPFlow successfully', () => {
    expect(createMockHTTPFlow).toBeTypeOf('function')
  })

  it('creates an HTTP flow with HTTP info', () => {
    const flow = createMockHTTPFlow()
    expect(flow.protocol).toBe('HTTP')
    expect(flow.dstPort).toBe(80)
    expect(flow.http).toBeDefined()
    expect(flow.http?.method).toBe('GET')
    expect(flow.http?.url).toBe('/api/users')
  })

  it('imports createMockTLSFlow successfully', () => {
    expect(createMockTLSFlow).toBeTypeOf('function')
  })

  it('creates a TLS/HTTPS flow with TLS info', () => {
    const flow = createMockTLSFlow()
    expect(flow.protocol).toBe('HTTPS')
    expect(flow.dstPort).toBe(443)
    expect(flow.tls).toBeDefined()
    expect(flow.tls?.sni).toBe('secure.example.com')
    expect(flow.tls?.version).toBe('TLS 1.3')
  })

  it('imports resetFlowIdCounter successfully', () => {
    expect(resetFlowIdCounter).toBeTypeOf('function')
    const flow1 = createMockFlow()
    expect(flow1.id).toBe('flow-1')
    resetFlowIdCounter()
    const flow2 = createMockFlow()
    expect(flow2.id).toBe('flow-1') // Counter reset
  })
})

describe('Mock Utilities - mockWebSocket', () => {
  it('imports createMockWebSocket successfully', () => {
    expect(createMockWebSocket).toBeTypeOf('function')
  })

  it('creates a mock WebSocket with default URL', () => {
    const ws = createMockWebSocket()
    expect(ws.url).toBe('ws://localhost:8080')
    expect(ws.readyState).toBe(WebSocketReadyState.CONNECTING)
  })

  it('creates a mock WebSocket with custom URL', () => {
    const ws = createMockWebSocket('ws://custom:9090')
    expect(ws.url).toBe('ws://custom:9090')
  })

  it('has simulateOpen helper method', () => {
    const ws = createMockWebSocket()
    expect(ws.simulateOpen).toBeTypeOf('function')

    let openCalled = false
    ws.onopen = () => {
      openCalled = true
    }
    ws.simulateOpen()

    expect(ws.readyState).toBe(WebSocketReadyState.OPEN)
    expect(openCalled).toBe(true)
  })

  it('has simulateMessage helper method', () => {
    const ws = createMockWebSocket()
    expect(ws.simulateMessage).toBeTypeOf('function')

    let receivedData: unknown = null
    ws.onmessage = (event) => {
      receivedData = JSON.parse(event.data)
    }
    ws.simulateMessage({ type: 'test', value: 123 })

    expect(receivedData).toEqual({ type: 'test', value: 123 })
  })

  it('has simulateClose helper method', () => {
    const ws = createMockWebSocket()
    expect(ws.simulateClose).toBeTypeOf('function')

    let closeCode: number | null = null
    ws.onclose = (event) => {
      closeCode = event.code
    }
    ws.simulateClose(1000, 'Normal closure')

    expect(ws.readyState).toBe(WebSocketReadyState.CLOSED)
    expect(closeCode).toBe(1000)
  })

  it('has send and close spy methods', () => {
    const ws = createMockWebSocket()
    expect(ws.send).toBeDefined()
    expect(ws.close).toBeDefined()

    ws.send('test message')
    expect(ws.send).toHaveBeenCalledWith('test message')

    ws.close()
    expect(ws.close).toHaveBeenCalled()
  })

  it('imports createMockWebSocketClass successfully', () => {
    expect(createMockWebSocketClass).toBeTypeOf('function')
  })

  it('creates mock WebSocket class that captures instances', () => {
    const { MockWebSocketClass, instances } = createMockWebSocketClass()
    expect(instances).toHaveLength(0)

    const ws = new MockWebSocketClass('ws://test:8080')
    expect(instances).toHaveLength(1)
    expect(ws.url).toBe('ws://test:8080')
  })

  it('imports clearMockWebSocketInstances successfully', () => {
    expect(clearMockWebSocketInstances).toBeTypeOf('function')

    const { MockWebSocketClass, instances } = createMockWebSocketClass()
    new MockWebSocketClass('ws://test:8080')
    expect(instances).toHaveLength(1)

    clearMockWebSocketInstances(instances)
    expect(instances).toHaveLength(0)
  })

  it('imports WebSocketReadyState constants', () => {
    expect(WebSocketReadyState).toBeDefined()
    expect(WebSocketReadyState.CONNECTING).toBe(0)
    expect(WebSocketReadyState.OPEN).toBe(1)
    expect(WebSocketReadyState.CLOSING).toBe(2)
    expect(WebSocketReadyState.CLOSED).toBe(3)
  })
})

describe('Global Mocks from vitest.setup.ts', () => {
  test('WebSocket is globally mocked', () => {
    expect(WebSocket).toBeDefined()
    const ws = new WebSocket('ws://test:8080')
    expect(ws).toBeDefined()
    expect(ws.url).toBe('ws://test:8080')
  })

  test('window.matchMedia is mocked', () => {
    expect(window.matchMedia).toBeDefined()
    const result = window.matchMedia('(min-width: 768px)')
    expect(result).toBeDefined()
    expect(result.matches).toBe(false)
    expect(result.media).toBe('(min-width: 768px)')
  })

  test('URL.createObjectURL is mocked', () => {
    expect(URL.createObjectURL).toBeDefined()
    const blob = new Blob(['test'], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    expect(url).toBe('blob:mock-url')
  })

  test('URL.revokeObjectURL is mocked', () => {
    expect(URL.revokeObjectURL).toBeDefined()
    // Should not throw
    URL.revokeObjectURL('blob:mock-url')
  })
})

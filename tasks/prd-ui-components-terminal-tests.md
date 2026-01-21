# PRD: UI Components - Terminal Tests

## Introduction

Add tests for the `Terminal.tsx` component which provides an interactive terminal using XTerm.js and WebSocket communication. The terminal allows executing commands in the agent's ephemeral container. Tests focus on WebSocket message protocol, resize handling, and component lifecycle.

## Goals

- Test Terminal component initialization and XTerm.js integration
- Verify WebSocket connection and message protocol
- Test terminal resize handling and debouncing
- Verify close and maximize callbacks
- Test component cleanup on unmount

## User Stories

### US-001: Test Terminal Initialization
**Description:** As a developer, I want tests for Terminal component initialization to verify XTerm.js setup.

**Acceptance Criteria:**
- [ ] Initialization tests pass:
  - Component renders container div
  - XTerm Terminal instance created
  - FitAddon attached for auto-sizing
  - WebLinksAddon attached for clickable links
  - Theme applied (dark background)
- [ ] `npm test -- --run "Terminal.*init"` passes

---

### US-002: Test WebSocket Connection
**Description:** As a developer, I want tests for Terminal WebSocket connection lifecycle.

**Acceptance Criteria:**
- [ ] WebSocket connection tests pass:
  - WebSocket created with correct URL
  - URL includes namespace and pod parameters
  - Connection message displayed on open
  - Error message displayed on error
  - Reconnection not attempted (single session)
- [ ] `npm test -- --run "Terminal.*WebSocket|Terminal.*connect"` passes

---

### US-003: Test Terminal Input
**Description:** As a developer, I want tests for keyboard input being sent over WebSocket.

**Acceptance Criteria:**
- [ ] Input tests pass:
  - Keyboard input captured by XTerm
  - Input sent via WebSocket as `{"type": "input", "data": "..."}`
  - Special keys (Enter, Backspace) handled
  - WebSocket `send` called with correct format
- [ ] `npm test -- --run "Terminal.*input"` passes

---

### US-004: Test Terminal Output
**Description:** As a developer, I want tests for WebSocket messages being displayed in terminal.

**Acceptance Criteria:**
- [ ] Output tests pass:
  - WebSocket message with `type: "output"` written to terminal
  - Binary data handled correctly
  - Output displayed in XTerm instance
- [ ] `npm test -- --run "Terminal.*output"` passes

---

### US-005: Test Terminal Resize
**Description:** As a developer, I want tests for terminal resize handling.

**Acceptance Criteria:**
- [ ] Resize tests pass:
  - Container size change triggers resize
  - Resize message sent: `{"type": "resize", "cols": N, "rows": N}`
  - Resize debounced (100ms) to prevent spam
  - FitAddon.fit() called on resize
- [ ] `npm test -- --run "Terminal.*resize"` passes

---

### US-006: Test Close Callback
**Description:** As a developer, I want tests for Terminal close button functionality.

**Acceptance Criteria:**
- [ ] Close tests pass:
  - Close button visible in header
  - Click calls `onClose` callback
  - WebSocket closed on component close
  - XTerm disposed on unmount
- [ ] `npm test -- --run "Terminal.*close"` passes

---

### US-007: Test Maximize Toggle
**Description:** As a developer, I want tests for Terminal maximize/minimize functionality.

**Acceptance Criteria:**
- [ ] Maximize tests pass:
  - Maximize button visible
  - Click calls `onToggleMaximize` callback
  - Button icon changes based on maximized state
  - Resize triggered after maximize/minimize
- [ ] `npm test -- --run "Terminal.*maximize"` passes

---

### US-008: Test Header Display
**Description:** As a developer, I want tests for Terminal header showing pod information.

**Acceptance Criteria:**
- [ ] Header tests pass:
  - Namespace displayed
  - Pod name displayed
  - Container name displayed (if provided)
  - Connection status indicator shown
- [ ] `npm test -- --run "Terminal.*header"` passes

---

## Functional Requirements

- FR-1: Create `ui/src/__tests__/components/Terminal.test.tsx` with 8 tests
- FR-2: Mock XTerm.js Terminal class and addons
- FR-3: Mock WebSocket with message simulation
- FR-4: Use fake timers for debounce testing
- FR-5: Test component cleanup on unmount

### XTerm Mock Pattern
```typescript
import { vi } from 'vitest'

// Mock XTerm.js
vi.mock('@xterm/xterm', () => ({
  Terminal: vi.fn().mockImplementation(() => ({
    open: vi.fn(),
    write: vi.fn(),
    onData: vi.fn(),
    onResize: vi.fn(),
    dispose: vi.fn(),
    cols: 80,
    rows: 24,
  })),
}))

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: vi.fn().mockImplementation(() => ({
    fit: vi.fn(),
    proposeDimensions: vi.fn().mockReturnValue({ cols: 80, rows: 24 }),
  })),
}))
```

### WebSocket Message Simulation
```typescript
test('sends resize message on terminal resize', async () => {
  const { mockWS } = createMockWebSocket()
  vi.stubGlobal('WebSocket', vi.fn(() => mockWS))
  vi.useFakeTimers()

  render(<Terminal namespace="default" pod="test-pod" onClose={() => {}} />)

  // Simulate resize
  act(() => {
    window.dispatchEvent(new Event('resize'))
    vi.advanceTimersByTime(150) // Past 100ms debounce
  })

  expect(mockWS.send).toHaveBeenCalledWith(
    expect.stringContaining('"type":"resize"')
  )

  vi.useRealTimers()
})
```

## Non-Goals

- No testing of actual shell command execution
- No testing of XTerm.js internal rendering
- No E2E testing with real Kubernetes exec

## Technical Considerations

- XTerm.js requires DOM container - use `{ container: document.body }` or similar
- Terminal resize uses ResizeObserver - may need polyfill in jsdom
- WebSocket binary frames may need ArrayBuffer handling
- Component uses `useEffect` cleanup - verify disposal

## Success Metrics

- All 8 tests pass
- Coverage >70% for Terminal.tsx
- Tests complete in under 3 seconds
- No memory leaks from XTerm instances

## Open Questions

- Should ResizeObserver be mocked or polyfilled?

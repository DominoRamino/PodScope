# PRD: UI Components - FlowList and FlowDetail Tests

## Introduction

Add tests for the `FlowList.tsx` virtualized table component and `FlowDetail.tsx` detail panel component. FlowList uses `@tanstack/react-virtual` for performance with large datasets. FlowDetail displays comprehensive flow information including timing visualization and protocol-specific sections.

## Goals

- Test FlowList rendering with various flow counts
- Verify FlowList row click selection behavior
- Test FlowDetail conditional rendering based on protocol
- Verify timing bar calculation and display
- Test memoization prevents unnecessary re-renders

## User Stories

### US-001: Test FlowList Empty State
**Description:** As a developer, I want tests for FlowList empty state rendering.

**Acceptance Criteria:**
- [ ] Empty state tests pass:
  - Shows appropriate message when no flows
  - No table rows rendered
  - Container still renders (for layout)
- [ ] `npm test -- --run "FlowList.*empty"` passes

---

### US-002: Test FlowList Single Flow Rendering
**Description:** As a developer, I want tests for FlowList with a single flow to verify row content.

**Acceptance Criteria:**
- [ ] Single flow render tests pass:
  - Protocol badge displayed with correct color
  - Source and destination shown
  - Timestamp formatted correctly
  - Duration/latency shown
  - Bytes sent/received formatted
- [ ] `npm test -- --run "FlowList.*single"` passes

---

### US-003: Test FlowList Multiple Flows
**Description:** As a developer, I want tests for FlowList with multiple flows to verify list behavior.

**Acceptance Criteria:**
- [ ] Multiple flows tests pass:
  - All flows rendered (or virtualized)
  - Flows in correct order
  - Each row has unique key
  - Scrolling works (virtual list)
- [ ] `npm test -- --run "FlowList.*multiple"` passes

---

### US-004: Test FlowList Row Selection
**Description:** As a developer, I want tests for row click selection behavior.

**Acceptance Criteria:**
- [ ] Row selection tests pass:
  - Click on row calls `onSelectFlow` with flow object
  - Selected row has highlighted styling
  - Clicking different row updates selection
  - Selection state managed by parent (prop-driven)
- [ ] `npm test -- --run "FlowList.*select|FlowList.*click"` passes

---

### US-005: Test FlowList Protocol Display
**Description:** As a developer, I want tests for protocol badge display variations.

**Acceptance Criteria:**
- [ ] Protocol display tests pass:
  - HTTP shows green badge
  - HTTPS shows yellow badge with lock indicator
  - TLS shows yellow badge
  - TCP shows blue badge
  - Badge text matches protocol name
- [ ] `npm test -- --run "FlowList.*protocol"` passes

---

### US-006: Test FlowList Status Display
**Description:** As a developer, I want tests for HTTP status code and connection status display.

**Acceptance Criteria:**
- [ ] Status display tests pass:
  - HTTP flow shows status code (200, 404, 500, etc.)
  - Status code color-coded (green/yellow/red)
  - Non-HTTP shows connection status (CLOSED, RESET, TIMEOUT)
- [ ] `npm test -- --run "FlowList.*status"` passes

---

### US-007: Test FlowDetail Basic Display
**Description:** As a developer, I want tests for FlowDetail basic flow information display.

**Acceptance Criteria:**
- [ ] Basic display tests pass:
  - Flow ID shown
  - Source IP:port and pod name displayed
  - Destination IP:port and pod name displayed
  - Protocol and status shown
  - Timestamp displayed
  - Duration shown in ms
- [ ] `npm test -- --run "FlowDetail.*basic|FlowDetail.*display"` passes

---

### US-008: Test FlowDetail Data Transfer Stats
**Description:** As a developer, I want tests for bytes sent/received display.

**Acceptance Criteria:**
- [ ] Data transfer tests pass:
  - Bytes sent formatted (KB, MB, etc.)
  - Bytes received formatted
  - Packets sent count shown
  - Packets received count shown
- [ ] `npm test -- --run "FlowDetail.*data|FlowDetail.*bytes"` passes

---

### US-009: Test FlowDetail Timing Bar
**Description:** As a developer, I want tests for the visual timing bar component.

**Acceptance Criteria:**
- [ ] Timing bar tests pass:
  - TCP handshake segment calculated correctly
  - TLS handshake segment shown for HTTPS/TLS
  - Data transfer segment shown
  - Segment widths proportional to time
  - Total bar represents total duration
- [ ] `npm test -- --run "FlowDetail.*timing"` passes

---

### US-010: Test FlowDetail HTTP Section
**Description:** As a developer, I want tests for HTTP-specific information display.

**Acceptance Criteria:**
- [ ] HTTP section tests pass:
  - Section visible only for HTTP/HTTPS flows
  - Request method displayed (GET, POST, etc.)
  - URL displayed
  - Status code with badge color
  - Response headers table shown
  - Content-Type displayed
- [ ] `npm test -- --run "FlowDetail.*HTTP|FlowDetail.*http"` passes

---

### US-011: Test FlowDetail TLS Section
**Description:** As a developer, I want tests for TLS-specific information display.

**Acceptance Criteria:**
- [ ] TLS section tests pass:
  - Section visible only for HTTPS/TLS flows
  - SNI (Server Name) displayed
  - TLS version shown
  - Cipher suite shown if available
- [ ] `npm test -- --run "FlowDetail.*TLS|FlowDetail.*tls"` passes

---

### US-012: Test FlowDetail Action Buttons
**Description:** As a developer, I want tests for FlowDetail action buttons.

**Acceptance Criteria:**
- [ ] Action button tests pass:
  - Close button calls `onClose` callback
  - Download PCAP button calls `onDownloadPCAP` with flow
  - Terminal button calls `onOpenTerminal` with pod info
  - Buttons disabled when appropriate (no pod for terminal)
- [ ] `npm test -- --run "FlowDetail.*button|FlowDetail.*action"` passes

---

## Functional Requirements

- FR-1: Create `ui/src/__tests__/components/FlowList.test.tsx` with 10 tests
- FR-2: Create `ui/src/__tests__/components/FlowDetail.test.tsx` with 12 tests
- FR-3: Use mock flow data from test utilities
- FR-4: Test conditional rendering with different flow types
- FR-5: Verify memoization with React Testing Library

### Test Pattern for Conditional Rendering
```typescript
import { render, screen } from '@testing-library/react'
import FlowDetail from '../../components/FlowDetail'
import { createMockHTTPFlow, createMockTLSFlow } from '../test-utils/testData'

test('shows HTTP section for HTTP flows', () => {
  const httpFlow = createMockHTTPFlow()
  render(<FlowDetail flow={httpFlow} onClose={() => {}} />)

  expect(screen.getByText(/GET/)).toBeInTheDocument()
  expect(screen.getByText(/200/)).toBeInTheDocument()
})

test('shows TLS section for HTTPS flows', () => {
  const tlsFlow = createMockTLSFlow()
  render(<FlowDetail flow={tlsFlow} onClose={() => {}} />)

  expect(screen.getByText(/TLS 1.3/)).toBeInTheDocument()
  expect(screen.getByText(/secure.example.com/)).toBeInTheDocument()
})
```

## Non-Goals

- No testing of virtualization performance
- No testing of scroll behavior in detail
- No visual regression testing

## Technical Considerations

- FlowList uses `@tanstack/react-virtual` - may need container with height
- FlowRow is memoized with custom comparison - test with React.memo behavior
- Timing bar uses CSS percentage widths - test calculated values not pixels

## Success Metrics

- All 22 tests pass (10 FlowList + 12 FlowDetail)
- Coverage >80% for both components
- Tests complete in under 5 seconds
- Memoization tests verify render count

## Open Questions

- Should virtualization be tested with specific scroll positions?

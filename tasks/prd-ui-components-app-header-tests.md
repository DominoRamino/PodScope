# PRD: UI Components - App and Header Tests

## Introduction

Add integration tests for the main `App.tsx` component and `Header.tsx` component. These tests cover WebSocket connection handling, flow filtering logic, pause/resume functionality, and header controls. Tests use React Testing Library with mocked WebSocket and fetch.

## Goals

- Test App component WebSocket connection lifecycle
- Verify flow filtering logic (HTTP, DNS, search)
- Test Header component controls and state display
- Verify pause toggle and PCAP download functionality

## User Stories

### US-001: Test App Initial Render
**Description:** As a developer, I want tests for App component initial render and connection status.

**Acceptance Criteria:**
- [ ] Initial render tests pass:
  - App renders without crashing
  - Shows "Connecting..." or connection indicator initially
  - Displays "Live" when WebSocket opens
  - Displays "Disconnected" when WebSocket closes
- [ ] `npm test -- --run "App.*render|App.*connection"` passes

---

### US-002: Test WebSocket Message Handling
**Description:** As a developer, I want tests for WebSocket message parsing to verify flows are correctly processed.

**Acceptance Criteria:**
- [ ] WebSocket message tests pass:
  - Catchup message (array of flows) processed correctly
  - Batch message with `type: "batch"` processed correctly
  - Single flow message adds to list
  - Flows appear in UI after message received
- [ ] Messages when paused still update flow list
- [ ] `npm test -- --run "App.*WebSocket|App.*message"` passes

---

### US-003: Test Flow Filtering
**Description:** As a developer, I want tests for flow filtering logic in App component.

**Acceptance Criteria:**
- [ ] Filter tests pass:
  - HTTP Only filter shows only HTTP/HTTPS protocol flows
  - All Ports filter shows all protocols
  - DNS filter excludes/includes port 53 traffic
  - Text search filters by pod name, IP, URL, SNI
  - Combined filters work together (AND logic)
- [ ] Empty filter state shows all flows
- [ ] `npm test -- --run "App.*filter"` passes

---

### US-004: Test Pause Toggle
**Description:** As a developer, I want tests for pause toggle functionality and API interaction.

**Acceptance Criteria:**
- [ ] Pause toggle tests pass:
  - Pause button calls `POST /api/pause`
  - UI updates to show "Resume" when paused
  - API response updates internal state
  - Flows continue updating when paused (only PCAP stops)
- [ ] `npm test -- --run "App.*pause"` passes

---

### US-005: Test PCAP Download
**Description:** As a developer, I want tests for PCAP download functionality.

**Acceptance Criteria:**
- [ ] PCAP download tests pass:
  - Download button triggers fetch to `/api/pcap/download`
  - Filter parameters included in URL
  - Blob response triggers file download
  - Filename includes timestamp
- [ ] `npm test -- --run "App.*download|App.*PCAP"` passes

---

### US-006: Test Header Connection Status
**Description:** As a developer, I want tests for Header component connection status display.

**Acceptance Criteria:**
- [ ] Header status tests pass:
  - Green indicator when connected
  - Red indicator when disconnected
  - Flow count displayed
  - PCAP size formatted and displayed
- [ ] `npm test -- --run "Header.*status|Header.*connection"` passes

---

### US-007: Test Header Search and Filters
**Description:** As a developer, I want tests for Header search input and filter toggles.

**Acceptance Criteria:**
- [ ] Search and filter tests pass:
  - Search input updates on type
  - Search calls `onFilterChange` callback
  - HTTP Only toggle works
  - DNS toggle works
  - Filter toggles are mutually exclusive where appropriate
- [ ] `npm test -- --run "Header.*search|Header.*filter"` passes

---

### US-008: Test Header Pause and Download Buttons
**Description:** As a developer, I want tests for Header pause button and download button.

**Acceptance Criteria:**
- [ ] Button tests pass:
  - Pause button shows "Pause" when not paused
  - Pause button shows "Resume" when paused
  - Pause button calls `onTogglePause` callback
  - Download button calls `onDownloadPCAP` callback
- [ ] `npm test -- --run "Header.*button|Header.*pause"` passes

---

### US-009: Test BPF Filter Input
**Description:** As a developer, I want tests for BPF filter input and submission in Header.

**Acceptance Criteria:**
- [ ] BPF filter tests pass:
  - Input accepts filter text
  - Apply button calls API `POST /api/bpf-filter`
  - Error response shows alert/error message
  - Clear button resets filter
- [ ] `npm test -- --run "Header.*BPF|Header.*bpf"` passes

---

### US-010: Test PCAP Reset
**Description:** As a developer, I want tests for PCAP reset confirmation and action.

**Acceptance Criteria:**
- [ ] PCAP reset tests pass:
  - Reset button shows confirmation dialog
  - Confirm calls `POST /api/pcap/reset`
  - Cancel does not call API
  - Success updates UI state
- [ ] `npm test -- --run "Header.*reset"` passes

---

## Functional Requirements

- FR-1: Create `ui/src/__tests__/App.test.tsx` with 12 test functions
- FR-2: Create `ui/src/__tests__/components/Header.test.tsx` with 14 test functions
- FR-3: Use React Testing Library `render`, `screen`, `fireEvent`, `waitFor`
- FR-4: Mock WebSocket using test utilities from setup PRD
- FR-5: Mock fetch for API calls
- FR-6: Use `userEvent` for realistic user interactions

### Test Pattern for WebSocket
```typescript
import { render, screen, waitFor } from '@testing-library/react'
import { vi } from 'vitest'
import App from '../App'
import { createMockWebSocket } from '../test-utils/mockWebSocket'

test('displays Live when WebSocket connects', async () => {
  const { mockWS, simulateOpen } = createMockWebSocket()
  vi.stubGlobal('WebSocket', vi.fn(() => mockWS))

  render(<App />)

  simulateOpen()

  await waitFor(() => {
    expect(screen.getByText(/live/i)).toBeInTheDocument()
  })
})
```

### Test Pattern for Fetch Mock
```typescript
beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve({ paused: false }),
  }))
})
```

## Non-Goals

- No E2E testing with real WebSocket server
- No visual regression testing
- No performance testing

## Technical Considerations

- App uses `useRef` for pause state to avoid stale closures - test actual behavior
- WebSocket reconnection logic may need timeout mocking
- Filter logic is memoized with `useMemo` - verify recalculation on change

## Success Metrics

- All 26 tests pass (12 App + 14 Header)
- Coverage >80% for App.tsx and Header.tsx
- Tests complete in under 5 seconds
- No flaky tests due to timing issues

## Open Questions

- Should WebSocket reconnection be tested, or just initial connection?

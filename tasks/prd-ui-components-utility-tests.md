# PRD: UI Components - Utility Function Tests

## Introduction

Add unit tests for utility functions extracted from React components. These are pure functions with no React dependencies, making them ideal for fast, isolated testing. Functions include formatters (bytes, time), color helpers (protocol, status), and string parsers (pod names).

## Goals

- Test all formatting functions with various input ranges
- Verify color coding functions return correct Tailwind classes
- Test string parsing edge cases
- Achieve 100% coverage on utility functions

## User Stories

### US-001: Test Byte Formatting
**Description:** As a developer, I want tests for `formatBytes()` to verify correct human-readable byte formatting.

**Acceptance Criteria:**
- [ ] All formatBytes tests pass:
  - 0 returns "0 B"
  - Values under 1KB show bytes (e.g., "512 B")
  - 1024 returns "1.00 KB"
  - Values in KB range formatted correctly
  - Values in MB range formatted correctly
  - Values in GB range formatted correctly
  - Decimal precision is 2 places
- [ ] `npm test -- --run formatBytes` passes

---

### US-002: Test Time Formatting
**Description:** As a developer, I want tests for `formatTime()` to verify correct time display with millisecond precision.

**Acceptance Criteria:**
- [ ] All formatTime tests pass:
  - Formats as HH:MM:SS.mmm
  - Pads hours, minutes, seconds with leading zeros
  - Pads milliseconds to 3 digits
  - Handles midnight (00:00:00.000)
  - Handles end of day (23:59:59.999)
- [ ] `npm test -- --run formatTime` passes

---

### US-003: Test Protocol Color Coding
**Description:** As a developer, I want tests for `getProtocolColor()` to verify correct Tailwind classes for each protocol.

**Acceptance Criteria:**
- [ ] All getProtocolColor tests pass:
  - HTTP returns green color classes
  - HTTPS returns yellow/amber color classes
  - TLS returns yellow/amber color classes
  - TCP returns blue color classes
  - Unknown protocol has fallback color
- [ ] Returns both text and background color classes
- [ ] `npm test -- --run getProtocolColor` passes

---

### US-004: Test Status Color Coding
**Description:** As a developer, I want tests for `getStatusColor()` to verify correct colors for HTTP status codes and connection status.

**Acceptance Criteria:**
- [ ] HTTP status code color tests pass:
  - 2xx (200, 201, 204) returns green
  - 3xx (301, 302, 304) returns blue
  - 4xx (400, 401, 403, 404) returns yellow/amber
  - 5xx (500, 502, 503) returns red
- [ ] Connection status color tests pass:
  - CLOSED returns green
  - RESET returns red
  - TIMEOUT returns yellow
  - OPEN returns blue
- [ ] `npm test -- --run getStatusColor` passes

---

### US-005: Test Pod Name Parsing
**Description:** As a developer, I want tests for `parsePodName()` to verify correct namespace/name extraction.

**Acceptance Criteria:**
- [ ] Pod name parsing tests pass:
  - "namespace/podname" extracts both correctly
  - "podname" (no slash) defaults namespace to "default"
  - Empty string handled gracefully
  - Multiple slashes handled (takes first segment as namespace)
- [ ] `npm test -- --run parsePodName` passes

---

## Functional Requirements

- FR-1: Create `ui/src/__tests__/utils.test.ts` with 20 test functions
- FR-2: Extract utility functions to `ui/src/utils.ts` if not already separate
- FR-3: Use `describe` blocks to group related tests
- FR-4: Use `test.each` for table-driven tests where appropriate
- FR-5: No React or DOM dependencies in these tests

### Test Structure
```typescript
import { describe, test, expect } from 'vitest'
import { formatBytes, formatTime, getProtocolColor, getStatusColor, parsePodName } from '../utils'

describe('formatBytes', () => {
  test.each([
    [0, '0 B'],
    [512, '512 B'],
    [1024, '1.00 KB'],
    [1536, '1.50 KB'],
    [1048576, '1.00 MB'],
    [1073741824, '1.00 GB'],
  ])('formatBytes(%i) returns %s', (input, expected) => {
    expect(formatBytes(input)).toBe(expected)
  })
})

describe('getStatusColor', () => {
  test.each([
    [200, 'green'],
    [201, 'green'],
    [301, 'blue'],
    [404, 'yellow'],
    [500, 'red'],
  ])('getStatusColor(%i) contains %s', (code, color) => {
    expect(getStatusColor(code)).toContain(color)
  })
})
```

## Non-Goals

- No React component testing (separate PRDs)
- No DOM manipulation testing
- No async/WebSocket testing

## Technical Considerations

- Functions may need to be extracted from components to a shared utils file
- Color classes should match Tailwind CSS class names
- Time formatting should use local timezone handling from Date object

### Files Containing Utility Functions
- `ui/src/components/Header.tsx` - formatBytes (lines 41-47)
- `ui/src/components/FlowList.tsx` - formatTime, getProtocolColor, getStatusColor
- `ui/src/App.tsx` - parsePodName (lines 187-196)

## Success Metrics

- All 20 tests pass
- Coverage report shows 100% for utility functions
- Tests complete in under 1 second
- No DOM or React dependencies in test file

## Open Questions

- Should utility functions be extracted to a separate `utils.ts` file or tested inline?

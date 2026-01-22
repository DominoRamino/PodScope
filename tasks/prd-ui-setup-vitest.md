# PRD: UI Setup - Vitest Testing Framework

## Introduction

Set up Vitest testing framework for the React UI in the `ui/` directory. This is a prerequisite for all UI component tests. Vitest provides native Vite integration, fast execution, and Jest-compatible APIs. This PRD covers dependency installation, configuration, and test utilities.

## Goals

- Install Vitest and React Testing Library dependencies
- Configure Vitest for React/TypeScript with jsdom environment
- Create test setup file with common mocks (WebSocket, fetch)
- Create test utilities for mock data generation
- Verify setup with a simple smoke test

## User Stories

### US-001: Install Testing Dependencies
**Description:** As a developer, I want testing dependencies added to package.json so I can write and run tests.

**Acceptance Criteria:**
- [ ] Dependencies added to `ui/package.json` devDependencies:
  - `vitest` ^1.2.0
  - `@testing-library/react` ^14.1.2
  - `@testing-library/jest-dom` ^6.2.0
  - `@testing-library/user-event` ^14.5.2
  - `jsdom` ^23.2.0
  - `@vitest/coverage-v8` ^1.2.0
- [ ] Scripts added to package.json:
  - `"test": "vitest"`
  - `"test:coverage": "vitest run --coverage"`
- [ ] `npm install` completes without errors

---

### US-002: Create Vitest Configuration
**Description:** As a developer, I want a Vitest config file that works with the existing Vite setup.

**Acceptance Criteria:**
- [ ] `ui/vitest.config.ts` created with:
  - React plugin enabled
  - jsdom test environment
  - Setup file reference
  - Test file pattern: `src/**/*.test.{ts,tsx}`
  - Coverage configuration for v8 provider
- [ ] Config extends existing Vite configuration where appropriate
- [ ] TypeScript types work correctly in test files

---

### US-003: Create Test Setup File
**Description:** As a developer, I want a setup file with common mocks so tests don't need to repeat boilerplate.

**Acceptance Criteria:**
- [ ] `ui/vitest.setup.ts` created with:
  - `@testing-library/jest-dom` matchers imported
  - `cleanup()` called after each test
  - WebSocket mock class with `send`, `close`, `onmessage`, `onopen`, `onclose`
  - `window.matchMedia` mock for responsive components
  - `URL.createObjectURL` mock for blob downloads
- [ ] Setup file imported in vitest.config.ts

---

### US-004: Create Test Data Utilities
**Description:** As a developer, I want factory functions for creating mock Flow objects so tests have consistent data.

**Acceptance Criteria:**
- [ ] `ui/src/test-utils/testData.ts` created with:
  - `createMockFlow(overrides?)` - basic TCP flow
  - `createMockHTTPFlow(overrides?)` - flow with HTTP info
  - `createMockTLSFlow(overrides?)` - flow with TLS/SNI info
  - All functions return properly typed `Flow` objects
- [ ] Factory functions allow partial overrides for flexibility
- [ ] TypeScript types match `ui/src/types.ts` definitions

---

### US-005: Create WebSocket Mock Utility
**Description:** As a developer, I want a WebSocket mock helper for testing real-time components.

**Acceptance Criteria:**
- [ ] `ui/src/test-utils/mockWebSocket.ts` created with:
  - `createMockWebSocket()` factory function
  - `simulateMessage(data)` helper to trigger onmessage
  - `simulateOpen()` helper to trigger onopen
  - `simulateClose()` helper to trigger onclose
  - Spy methods for `send` and `close`
- [ ] Mock can be injected into components via `vi.stubGlobal`

---

### US-006: Verify Setup with Smoke Test
**Description:** As a developer, I want a simple smoke test to verify the testing setup works correctly.

**Acceptance Criteria:**
- [ ] `ui/src/__tests__/setup.test.ts` created with:
  - Test that Vitest runs (simple expect)
  - Test that React Testing Library works (render a div)
  - Test that jest-dom matchers work (toBeInTheDocument)
  - Test that mock utilities are importable
- [ ] `npm test` runs and passes
- [ ] `npm run test:coverage` generates coverage report

---

## Functional Requirements

- FR-1: Add 6 devDependencies to `ui/package.json`
- FR-2: Create `ui/vitest.config.ts` configuration file
- FR-3: Create `ui/vitest.setup.ts` setup file
- FR-4: Create `ui/src/test-utils/testData.ts` with flow factories
- FR-5: Create `ui/src/test-utils/mockWebSocket.ts` with WebSocket mock
- FR-6: Create `ui/src/__tests__/setup.test.ts` smoke test

## Non-Goals

- No actual component tests (separate PRDs)
- No E2E testing setup (Playwright, Cypress)
- No visual regression testing

## Technical Considerations

### vitest.config.ts Structure
```typescript
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./vitest.setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/**/*.test.{ts,tsx}', 'src/test-utils/**']
    }
  }
})
```

### vitest.setup.ts Structure
```typescript
import '@testing-library/jest-dom'
import { cleanup } from '@testing-library/react'
import { afterEach, vi } from 'vitest'

afterEach(() => cleanup())

// Mock WebSocket
class MockWebSocket {
  static OPEN = 1
  static CLOSED = 3
  readyState = MockWebSocket.OPEN
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  send = vi.fn()
  close = vi.fn()
}
vi.stubGlobal('WebSocket', MockWebSocket)

// Mock matchMedia
vi.stubGlobal('matchMedia', vi.fn().mockImplementation(query => ({
  matches: false,
  media: query,
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
})))

// Mock URL.createObjectURL
vi.stubGlobal('URL', {
  ...URL,
  createObjectURL: vi.fn(() => 'blob:mock-url'),
  revokeObjectURL: vi.fn(),
})
```

## Success Metrics

- `npm test` runs without configuration errors
- Smoke test passes
- Coverage report generated with `npm run test:coverage`
- Setup completes in under 30 seconds

## Open Questions

None - this is a standard Vitest + React Testing Library setup.

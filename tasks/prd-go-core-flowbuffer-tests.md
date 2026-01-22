# PRD: Go Core - FlowBuffer Tests

## Introduction

Add comprehensive unit tests for the `FlowRingBuffer` data structure in `pkg/hub/flowbuffer.go`. This is a pure data structure with no external dependencies, making it an ideal starting point for test coverage. The ring buffer provides O(1) insertion and lookup for network flow storage.

## Goals

- Achieve 100% test coverage for `pkg/hub/flowbuffer.go`
- Verify correct circular buffer behavior including wrap-around and eviction
- Ensure thread-safety with concurrent access patterns
- Validate index map consistency during insertions and evictions

## User Stories

### US-001: Test Buffer Initialization
**Description:** As a developer, I want tests for `NewFlowRingBuffer()` to ensure buffers are created with correct capacity settings.

**Acceptance Criteria:**
- [ ] All capacity initialization tests pass:
  - Default capacity (10000) when 0 passed
  - Custom capacity respected when specified
  - Environment variable `MAX_FLOWS` override works
- [ ] `go test -v ./pkg/hub/... -run TestNewFlowRingBuffer` passes

---

### US-002: Test Flow Addition and Updates
**Description:** As a developer, I want tests for `Add()` to verify correct insertion, update detection, and eviction behavior.

**Acceptance Criteria:**
- [ ] All Add() tests pass:
  - New flow returns `true`
  - Update to existing flow (same ID) returns `false`
  - Oldest flow evicted when at capacity
  - Index map updated correctly on eviction
- [ ] `go test -v ./pkg/hub/... -run TestAdd` passes

---

### US-003: Test Flow Retrieval Methods
**Description:** As a developer, I want tests for `GetAll()`, `GetRecent()`, and `Get()` to verify correct ordering and lookup behavior.

**Acceptance Criteria:**
- [ ] All retrieval tests pass:
  - `GetAll()` returns empty slice for empty buffer
  - `GetAll()` returns flows in chronological order (oldest first)
  - `GetAll()` maintains correct order after wrap-around/eviction
  - `GetRecent(n)` returns newest-first ordering
  - `GetRecent(n)` returns all flows when n > size
  - `Get(id)` returns correct flow for existing ID
  - `Get(id)` returns nil for non-existing ID
- [ ] `go test -v ./pkg/hub/... -run "TestGetAll|TestGetRecent|TestGet"` passes

---

### US-004: Test Buffer Clear and State Reset
**Description:** As a developer, I want tests for `Clear()` to verify complete state reset.

**Acceptance Criteria:**
- [ ] Clear() test passes:
  - Resets head to 0
  - Resets size to 0
  - Clears index map
  - Subsequent `GetAll()` returns empty slice
- [ ] `go test -v ./pkg/hub/... -run TestClear` passes

---

## Functional Requirements

- FR-1: Create `pkg/hub/flowbuffer_test.go` with 14 test functions
- FR-2: Use table-driven tests where appropriate for multiple input scenarios
- FR-3: Include helper function to create mock `protocol.Flow` structs
- FR-4: Tests must not depend on external systems or timing

## Non-Goals

- No benchmark tests in this PRD (can be added later)
- No fuzz testing
- No concurrent stress testing (basic thread-safety only)

## Technical Considerations

- Import `github.com/podscope/podscope/pkg/protocol` for Flow struct
- Use `t.Parallel()` where tests are independent
- Consider using subtests (`t.Run()`) for related test cases

## Success Metrics

- All 14 tests pass
- `go test -cover ./pkg/hub/...` shows >95% coverage for flowbuffer.go
- Tests complete in under 1 second

## Open Questions

None - this is a straightforward pure function testing task.

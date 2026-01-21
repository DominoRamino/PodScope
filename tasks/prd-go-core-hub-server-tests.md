# PRD: Go Core - Hub Server HTTP Handler Tests

## Introduction

Add comprehensive HTTP handler tests for `pkg/hub/server.go` using Go's `httptest` package. The Hub server exposes REST API endpoints for flow management, PCAP operations, pause control, and BPF filter configuration. These tests verify the API contract without requiring a running server.

## Goals

- Test all HTTP API endpoints using `httptest.NewRecorder()`
- Verify correct HTTP status codes and response formats
- Test error handling for invalid requests
- Ensure pause state affects PCAP storage behavior

## User Stories

### US-001: Test Health Endpoint
**Description:** As a developer, I want tests for `GET /api/health` to verify the health check response format.

**Acceptance Criteria:**
- [ ] Health endpoint tests pass:
  - Returns HTTP 200 OK
  - Response includes `status: "healthy"`
  - Response includes `sessionId`
  - Response includes current `bpfFilter` (empty string if not set)
- [ ] `go test -v ./pkg/hub/... -run TestHandleHealth` passes

---

### US-002: Test Flows Endpoint
**Description:** As a developer, I want tests for `/api/flows` GET and POST to verify flow storage and retrieval.

**Acceptance Criteria:**
- [ ] Flows GET tests pass:
  - Empty buffer returns empty JSON array
  - Returns all stored flows in JSON format
- [ ] Flows POST tests pass:
  - Valid flow JSON returns 201 Created
  - Invalid JSON returns 400 Bad Request
  - Flow is stored in buffer after POST
- [ ] Method not allowed (PUT/DELETE) returns 405
- [ ] `go test -v ./pkg/hub/... -run TestHandleFlows` passes

---

### US-003: Test Pause Endpoint
**Description:** As a developer, I want tests for `/api/pause` to verify pause state management.

**Acceptance Criteria:**
- [ ] Pause GET tests pass:
  - Returns `{"paused": false}` initially
  - Returns current pause state
- [ ] Pause POST tests pass:
  - `{"paused": true}` sets pause to true
  - `{"paused": false}` sets pause to false
  - Empty body toggles pause state
- [ ] `go test -v ./pkg/hub/... -run TestHandlePause` passes

---

### US-004: Test BPF Filter Endpoint
**Description:** As a developer, I want tests for `/api/bpf-filter` to verify BPF filter management and validation.

**Acceptance Criteria:**
- [ ] BPF filter GET tests pass:
  - Returns empty filter by default
  - Returns current filter after set
- [ ] BPF filter POST tests pass:
  - Valid BPF syntax accepted (e.g., "tcp port 80")
  - Invalid BPF syntax returns 400 with error message
  - Empty string clears filter
  - Missing filter field returns 400
- [ ] `go test -v ./pkg/hub/... -run TestHandleBPFFilter` passes

---

### US-005: Test PCAP Upload Endpoint
**Description:** As a developer, I want tests for `POST /api/pcap/upload` to verify PCAP data storage.

**Acceptance Criteria:**
- [ ] PCAP upload tests pass:
  - Valid binary data returns 200 OK
  - Uses `X-Agent-ID` header for agent identification
  - Defaults to "unknown" agent if header missing
  - Data silently dropped when paused (still returns 200)
  - GET method returns 405
- [ ] `go test -v ./pkg/hub/... -run TestHandlePCAPUpload` passes

---

### US-006: Test Stats Endpoint
**Description:** As a developer, I want tests for `GET /api/stats` to verify statistics reporting.

**Acceptance Criteria:**
- [ ] Stats endpoint tests pass:
  - Returns `flowCount` (number of flows)
  - Returns `paused` state
  - Returns `pcapSize` (bytes stored)
  - Returns `wsClients` count
- [ ] `go test -v ./pkg/hub/... -run TestHandleStats` passes

---

### US-007: Test Agent Registration and PCAP Reset
**Description:** As a developer, I want tests for agent registration and PCAP reset endpoints.

**Acceptance Criteria:**
- [ ] Agent registration tests pass:
  - Valid agent JSON returns 200
  - Invalid JSON returns 400
- [ ] PCAP reset tests pass:
  - POST clears PCAP buffer
  - Returns success response
- [ ] `go test -v ./pkg/hub/... -run "TestHandleAgents|TestHandlePCAPReset"` passes

---

## Functional Requirements

- FR-1: Create `pkg/hub/server_test.go` with 28 test functions
- FR-2: Create `setupTestServer(t *testing.T) *Server` helper function
- FR-3: Use `httptest.NewRecorder()` for response capture
- FR-4: Use `httptest.NewRequest()` for request creation
- FR-5: Use `t.TempDir()` for PCAP buffer directory

### Test Helper Pattern
```go
func setupTestServer(t *testing.T) *Server {
    return &Server{
        sessionID:  "test-session-123",
        flowBuffer: NewFlowRingBuffer(100),
        pcapBuffer: NewPCAPBuffer(t.TempDir(), 1024*1024),
        wsClients:  make(map[*websocket.Conn]bool),
        bpfFilter:  "",
        paused:     false,
    }
}
```

## Non-Goals

- No WebSocket handler tests (requires WebSocket client mocking)
- No terminal WebSocket tests (requires Kubernetes exec)
- No load/performance testing

## Technical Considerations

- Import `net/http/httptest` for test infrastructure
- JSON responses should be validated with struct unmarshaling
- BPF filter validation may require libpcap (skip if unavailable)

## Success Metrics

- All 28 tests pass
- `go test -cover ./pkg/hub/...` shows >70% coverage for server.go
- Tests complete in under 5 seconds

## Open Questions

- Should BPF validation tests be skipped if libpcap is unavailable?

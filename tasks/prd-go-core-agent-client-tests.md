# PRD: Go Core - Agent Hub Client Tests

## Introduction

Add unit tests for the Agent's Hub communication client in `pkg/agent/client.go`. This client handles HTTP communication between the capture agent and the Hub server, including flow submission, PCAP uploads, heartbeats, and BPF filter updates. Tests use `httptest.NewServer()` to mock the Hub.

## Goals

- Test Hub client initialization and URL construction
- Verify connection and agent registration flow
- Test flow and PCAP data queuing with channel management
- Verify heartbeat mechanism and BPF filter update detection
- Test disconnection detection and callback

## User Stories

### US-001: Test Client Initialization
**Description:** As a developer, I want tests for `NewHubClient()` to verify correct URL construction and port translation.

**Acceptance Criteria:**
- [ ] Initialization tests pass:
  - Port 9090 (gRPC) translated to 8080 (HTTP)
  - URL scheme is HTTP
  - Base URL constructed correctly
  - Agent info stored correctly
- [ ] `go test -v ./pkg/agent/... -run TestNewHubClient` passes

---

### US-002: Test Connection and Registration
**Description:** As a developer, I want tests for `Connect()` to verify health check and agent registration.

**Acceptance Criteria:**
- [ ] Connection tests pass:
  - Successful health check (`GET /api/health` returns 200) marks connected
  - Failed health check (500/timeout) returns error
  - Agent registration (`POST /api/agents`) called after health check
  - Connection state updated correctly
- [ ] `go test -v ./pkg/agent/... -run TestConnect` passes

---

### US-003: Test Flow Queuing
**Description:** As a developer, I want tests for `SendFlow()` to verify non-blocking flow queuing behavior.

**Acceptance Criteria:**
- [ ] SendFlow tests pass:
  - Flow added to channel when space available
  - Returns nil (no error) on success
  - Channel full scenario handled gracefully (returns error)
  - Flow sent to Hub via `POST /api/flows` with correct JSON
- [ ] `go test -v ./pkg/agent/... -run TestSendFlow` passes

---

### US-004: Test PCAP Chunk Queuing
**Description:** As a developer, I want tests for `SendPCAPChunk()` to verify PCAP data queuing with data copy.

**Acceptance Criteria:**
- [ ] SendPCAPChunk tests pass:
  - Data copied (not referenced) before queuing
  - Returns nil on success
  - Channel full scenario returns error
  - PCAP sent to Hub via `POST /api/pcap/upload` with `X-Agent-ID` header
- [ ] `go test -v ./pkg/agent/... -run TestSendPCAPChunk` passes

---

### US-005: Test Heartbeat and BPF Filter Updates
**Description:** As a developer, I want tests for heartbeat mechanism to verify BPF filter detection and failure tracking.

**Acceptance Criteria:**
- [ ] Heartbeat tests pass:
  - Heartbeat request sent to health endpoint
  - New BPF filter in response triggers `UpdateBPFFilter()` on capturer
  - Empty filter clears current filter
  - Consecutive failures tracked (increments counter)
  - After maxFailures (3), `onDisconnect` callback invoked
- [ ] `go test -v ./pkg/agent/... -run TestSendHeartbeat` passes

---

### US-006: Test Connection State and Cleanup
**Description:** As a developer, I want tests for `IsConnected()` and `Close()` to verify state management and cleanup.

**Acceptance Criteria:**
- [ ] State tests pass:
  - `IsConnected()` returns current connection state
  - `Close()` sets connected to false
  - `Close()` waits for worker goroutines (WaitGroup)
  - Channels closed after Close()
- [ ] `go test -v ./pkg/agent/... -run "TestIsConnected|TestClose"` passes

---

## Functional Requirements

- FR-1: Create `pkg/agent/client_test.go` with 16 test functions
- FR-2: Use `httptest.NewServer()` to mock Hub API endpoints
- FR-3: Create mock `Capturer` interface for `SetBPFFilter` testing
- FR-4: Test channel behavior (capacity 1000 for flows, 100 for PCAP)
- FR-5: Use `t.Cleanup()` for server shutdown

### Mock Hub Server Pattern
```go
func setupMockHub(t *testing.T) *httptest.Server {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": "healthy",
            "bpfFilter": "",
        })
    })
    mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    mux.HandleFunc("/api/flows", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusCreated)
    })
    return httptest.NewServer(mux)
}
```

## Non-Goals

- No testing of actual network latency or timeouts
- No testing of TLS/HTTPS connections
- No stress testing of channel capacity

## Technical Considerations

- Client uses HTTP despite gRPC port (9090 → 8080 translation)
- Heartbeat interval is configurable but defaults to 5 seconds
- `onDisconnect` callback may be nil (check before calling)

## Success Metrics

- All 16 tests pass
- `go test -cover ./pkg/agent/...` shows >70% coverage for client.go
- Tests complete in under 3 seconds

## Open Questions

None.

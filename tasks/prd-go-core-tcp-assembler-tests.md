# PRD: Go Core - TCP Assembler Tests

## Introduction

Add comprehensive unit tests for the TCP stream reassembly logic in `pkg/agent/assembler.go`. This file contains critical protocol detection and parsing functions including TLS SNI extraction, HTTP method detection, and bidirectional flow key normalization. Tests will use real TLS/HTTP captures as fixtures.

## Goals

- Test all pure functions in `pkg/agent/assembler.go`
- Verify correct bidirectional flow key normalization
- Ensure accurate protocol detection (HTTP, TLS, HTTPS, TCP)
- Validate TLS ClientHello SNI extraction with real capture data
- Test HTTP request/response parsing

## User Stories

### US-001: Test Flow Key Normalization
**Description:** As a developer, I want tests for `flowKey()` to ensure bidirectional flows produce identical keys regardless of direction.

**Acceptance Criteria:**
- [ ] All flowKey tests pass:
  - Source IP < Dest IP produces consistent key
  - Source IP > Dest IP produces same key as reversed
  - Same IP with different ports sorts by port
  - A→B and B→A produce identical keys
- [ ] `go test -v ./pkg/agent/... -run TestFlowKey` passes

---

### US-002: Test HTTP Method Detection
**Description:** As a developer, I want tests for `isHTTPMethod()` to verify correct detection of HTTP request/response patterns.

**Acceptance Criteria:**
- [ ] All isHTTPMethod tests pass:
  - Detects GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH, CONNECT
  - Detects HTTP/ response prefix
  - Rejects non-HTTP payloads (binary data, TLS, etc.)
  - Verifies case sensitivity behavior
- [ ] `go test -v ./pkg/agent/... -run TestIsHTTPMethod` passes

---

### US-003: Test Protocol Detection
**Description:** As a developer, I want tests for `detectProtocol()` to verify correct identification of application protocols from payload and port.

**Acceptance Criteria:**
- [ ] All detectProtocol tests pass:
  - TLS ClientHello (0x16 0x03) → ProtocolTLS
  - HTTP methods → ProtocolHTTP
  - Port 443/8443 → ProtocolHTTPS
  - Unknown → ProtocolTCP (fallback)
- [ ] `go test -v ./pkg/agent/... -run TestDetectProtocol` passes

---

### US-004: Test TLS SNI Extraction
**Description:** As a developer, I want tests for `extractSNI()` using real TLS ClientHello captures to verify accurate Server Name Indication extraction.

**Acceptance Criteria:**
- [ ] All extractSNI tests pass:
  - Extracts SNI from valid ClientHello (use real Wireshark capture)
  - Returns empty string when no SNI extension present
  - Handles truncated/malformed data gracefully (no panic)
  - Parses TLS 1.2 and TLS 1.3 ClientHello formats
- [ ] Test fixtures include real TLS captures as byte slices
- [ ] `go test -v ./pkg/agent/... -run TestExtractSNI` passes

---

### US-005: Test HTTP Parsing
**Description:** As a developer, I want tests for HTTP request/response parsing to verify correct extraction of method, URL, status, and headers.

**Acceptance Criteria:**
- [ ] All parseHTTP tests pass:
  - Parses GET/POST requests with headers
  - Extracts method, URL, Host header
  - Parses response status code and headers
  - Handles partial/incomplete data without panic
- [ ] `go test -v ./pkg/agent/... -run TestParseHTTP` passes

---

## Functional Requirements

- FR-1: Create `pkg/agent/assembler_test.go` with 18 test functions
- FR-2: Include real TLS ClientHello byte fixtures captured from Wireshark
- FR-3: Include sample HTTP request/response byte fixtures
- FR-4: Export internal functions if needed for testing (or use same package)
- FR-5: Use table-driven tests for method/protocol detection

## Non-Goals

- No testing of ProcessPacket() flow state machine (requires gopacket mocking)
- No testing of cleanupLoop() goroutine timing
- No integration tests with actual network capture

## Technical Considerations

- `flowKey()`, `isHTTPMethod()`, `extractSNI()` are package-private - tests must be in `package agent`
- TLS fixture should be a real ClientHello from `openssl s_client` or Wireshark
- HTTP fixtures should include headers and body portions

### Sample TLS ClientHello Fixture

```go
// Captured from: openssl s_client -connect example.com:443 -servername example.com
var tlsClientHelloExampleCom = []byte{
    0x16, 0x03, 0x01, // TLS record: Handshake, TLS 1.0
    // ... full ClientHello with SNI extension for "example.com"
}
```

## Success Metrics

- All 18 tests pass
- `go test -cover ./pkg/agent/...` shows >80% coverage for assembler.go
- No panics on malformed input data
- Tests complete in under 2 seconds

## Open Questions

None - fixtures will be created inline with tests.

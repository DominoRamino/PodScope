# PRD: Go Core - PCAP Encoding Tests

## Introduction

Add unit tests for PCAP binary format encoding in both `pkg/agent/capture.go` and `pkg/hub/pcap.go`. These tests verify that the PCAP global headers and packet headers are correctly encoded in libpcap format, ensuring compatibility with Wireshark and other PCAP analysis tools.

## Goals

- Verify PCAP global header encoding matches libpcap specification
- Verify PCAP packet header encoding with correct timestamps and lengths
- Ensure binary format is little-endian as required by libpcap
- Test file-based PCAP storage and multi-agent file merging

## User Stories

### US-001: Test Agent PCAP Header Encoding
**Description:** As a developer, I want tests for PCAP header encoding in `pkg/agent/capture.go` to ensure captured packets are Wireshark-compatible.

**Acceptance Criteria:**
- [ ] All agent PCAP header tests pass:
  - Magic number is 0xa1b2c3d4 (little-endian: d4 c3 b2 a1)
  - Version is 2.4 (major=2, minor=4)
  - Snaplen is 65535
  - Link type is 1 (Ethernet)
  - Total header size is exactly 24 bytes
- [ ] `go test -v ./pkg/agent/... -run TestWritePCAPHeader` passes

---

### US-002: Test Agent PCAP Packet Encoding
**Description:** As a developer, I want tests for PCAP packet encoding to verify correct timestamp and length fields.

**Acceptance Criteria:**
- [ ] All agent PCAP packet tests pass:
  - Timestamp seconds encoded correctly
  - Timestamp microseconds computed correctly
  - Included length matches actual data length
  - Original length field set correctly
  - Packet data appended after 16-byte header
- [ ] `go test -v ./pkg/agent/... -run TestWritePCAPPacket` passes

---

### US-003: Test Hub PCAP Header Encoding
**Description:** As a developer, I want tests for PCAP header encoding in `pkg/hub/pcap.go` to ensure stored PCAP files are valid.

**Acceptance Criteria:**
- [ ] All hub PCAP header tests pass:
  - Magic number correct
  - Version 2.4
  - Header exactly 24 bytes
- [ ] `go test -v ./pkg/hub/... -run TestWritePCAPHeader` passes

---

### US-004: Test Hub PCAP Packet Encoding
**Description:** As a developer, I want tests for hub-side PCAP packet encoding and timestamp handling.

**Acceptance Criteria:**
- [ ] All hub PCAP packet tests pass:
  - Timestamp seconds from Unix time
  - Microseconds computed from nanoseconds
  - Included length correct
  - Original length correct
  - Packet header is 16 bytes
- [ ] `go test -v ./pkg/hub/... -run TestWritePCAPPacket` passes

---

### US-005: Test PCAP File Operations
**Description:** As a developer, I want tests for PCAP file storage and merging to verify multi-agent capture handling.

**Acceptance Criteria:**
- [ ] All PCAP file operation tests pass:
  - `Write()` creates agent-specific PCAP file with header
  - Subsequent writes append packets (no duplicate header)
  - `GetSessionPCAP()` merges multiple agent files
  - Merged output has single global header
  - Per-agent headers (bytes 0-23) skipped during merge
- [ ] `go test -v ./pkg/hub/... -run "TestWrite|TestGetSessionPCAP"` passes

---

## Functional Requirements

- FR-1: Create `pkg/agent/capture_test.go` with 8 test functions
- FR-2: Create `pkg/hub/pcap_test.go` with 10 test functions
- FR-3: Use `t.TempDir()` for file-based tests
- FR-4: Verify binary output byte-by-byte against PCAP specification
- FR-5: Include helper to validate PCAP magic number and version

## Non-Goals

- No testing of actual packet capture (requires root/NET_RAW)
- No testing of BPF filter application
- No performance/benchmark tests

## Technical Considerations

- PCAP format reference: https://wiki.wireshark.org/Development/LibpcapFileFormat
- Global header: 24 bytes (magic, version major/minor, thiszone, sigfigs, snaplen, network)
- Packet header: 16 bytes (ts_sec, ts_usec, incl_len, orig_len)
- All fields are little-endian

### PCAP Global Header Structure
```
Bytes 0-3:   Magic number (0xa1b2c3d4)
Bytes 4-5:   Version major (2)
Bytes 6-7:   Version minor (4)
Bytes 8-11:  Thiszone (0)
Bytes 12-15: Sigfigs (0)
Bytes 16-19: Snaplen (65535)
Bytes 20-23: Network (1 = Ethernet)
```

## Success Metrics

- All 18 tests pass across both files
- PCAP output can be opened by Wireshark without errors
- Tests complete in under 2 seconds

## Open Questions

None.

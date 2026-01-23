package agent

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/google/gopacket"
)

// mockPacket implements gopacket.Packet for testing
type mockPacket struct {
	data      []byte
	timestamp time.Time
}

func (m *mockPacket) Data() []byte {
	return m.data
}

func (m *mockPacket) Metadata() *gopacket.PacketMetadata {
	return &gopacket.PacketMetadata{
		CaptureInfo: gopacket.CaptureInfo{
			Timestamp: m.timestamp,
		},
	}
}

// Implement remaining gopacket.Packet interface methods with no-op implementations
func (m *mockPacket) String() string                          { return "" }
func (m *mockPacket) Dump() string                            { return "" }
func (m *mockPacket) Layers() []gopacket.Layer                { return nil }
func (m *mockPacket) Layer(gopacket.LayerType) gopacket.Layer { return nil }
func (m *mockPacket) LayerClass(gopacket.LayerClass) gopacket.Layer {
	return nil
}
func (m *mockPacket) LinkLayer() gopacket.LinkLayer           { return nil }
func (m *mockPacket) NetworkLayer() gopacket.NetworkLayer     { return nil }
func (m *mockPacket) TransportLayer() gopacket.TransportLayer { return nil }
func (m *mockPacket) ApplicationLayer() gopacket.ApplicationLayer {
	return nil
}
func (m *mockPacket) ErrorLayer() gopacket.ErrorLayer { return nil }

// TestWritePCAPHeader_MagicNumber verifies the PCAP magic number is correct
func TestWritePCAPHeader_MagicNumber(t *testing.T) {
	c := &Capturer{}

	c.writePCAPHeader()

	data := c.pcapBuffer.Bytes()
	if len(data) < 4 {
		t.Fatalf("Expected at least 4 bytes for magic number, got %d", len(data))
	}

	// Magic number should be 0xa1b2c3d4 in little-endian (d4 c3 b2 a1)
	magic := binary.LittleEndian.Uint32(data[0:4])
	expected := uint32(0xa1b2c3d4)
	if magic != expected {
		t.Errorf("Expected magic number 0x%08x, got 0x%08x", expected, magic)
	}
}

// TestWritePCAPHeader_Version verifies the PCAP version is 2.4
func TestWritePCAPHeader_Version(t *testing.T) {
	c := &Capturer{}

	c.writePCAPHeader()

	data := c.pcapBuffer.Bytes()
	if len(data) < 8 {
		t.Fatalf("Expected at least 8 bytes for version fields, got %d", len(data))
	}

	// Version major should be 2 (bytes 4-5, little-endian)
	versionMajor := binary.LittleEndian.Uint16(data[4:6])
	if versionMajor != 2 {
		t.Errorf("Expected version major 2, got %d", versionMajor)
	}

	// Version minor should be 4 (bytes 6-7, little-endian)
	versionMinor := binary.LittleEndian.Uint16(data[6:8])
	if versionMinor != 4 {
		t.Errorf("Expected version minor 4, got %d", versionMinor)
	}
}

// TestWritePCAPHeader_Snaplen verifies the snaplen is 65535
func TestWritePCAPHeader_Snaplen(t *testing.T) {
	c := &Capturer{}

	c.writePCAPHeader()

	data := c.pcapBuffer.Bytes()
	if len(data) < 20 {
		t.Fatalf("Expected at least 20 bytes for snaplen field, got %d", len(data))
	}

	// Snaplen is at bytes 16-19 (little-endian)
	snaplen := binary.LittleEndian.Uint32(data[16:20])
	expected := uint32(65535)
	if snaplen != expected {
		t.Errorf("Expected snaplen %d, got %d", expected, snaplen)
	}
}

// TestWritePCAPHeader_LinkType verifies the link type is 1 (Ethernet)
func TestWritePCAPHeader_LinkType(t *testing.T) {
	c := &Capturer{}

	c.writePCAPHeader()

	data := c.pcapBuffer.Bytes()
	if len(data) < 24 {
		t.Fatalf("Expected at least 24 bytes for link type field, got %d", len(data))
	}

	// Link type is at bytes 20-23 (little-endian)
	linkType := binary.LittleEndian.Uint32(data[20:24])
	expected := uint32(1) // Ethernet
	if linkType != expected {
		t.Errorf("Expected link type %d (Ethernet), got %d", expected, linkType)
	}
}

// TestWritePCAPHeader_TotalSize verifies the header is exactly 24 bytes
func TestWritePCAPHeader_TotalSize(t *testing.T) {
	c := &Capturer{}

	c.writePCAPHeader()

	data := c.pcapBuffer.Bytes()
	expected := 24
	if len(data) != expected {
		t.Errorf("Expected header size %d bytes, got %d", expected, len(data))
	}
}

// TestWritePCAPHeader_FullStructure verifies the complete header byte-by-byte
func TestWritePCAPHeader_FullStructure(t *testing.T) {
	c := &Capturer{}

	c.writePCAPHeader()

	data := c.pcapBuffer.Bytes()
	expected := []byte{
		0xd4, 0xc3, 0xb2, 0xa1, // Magic number (little-endian)
		0x02, 0x00, // Version major
		0x04, 0x00, // Version minor
		0x00, 0x00, 0x00, 0x00, // Timezone
		0x00, 0x00, 0x00, 0x00, // Sigfigs
		0xff, 0xff, 0x00, 0x00, // Snaplen (65535)
		0x01, 0x00, 0x00, 0x00, // Link type (Ethernet)
	}

	if !bytes.Equal(data, expected) {
		t.Errorf("Header mismatch\nExpected: %v\nGot:      %v", expected, data)
	}
}

// TestWritePCAPHeader_TimezoneZero verifies timezone is zero
func TestWritePCAPHeader_TimezoneZero(t *testing.T) {
	c := &Capturer{}

	c.writePCAPHeader()

	data := c.pcapBuffer.Bytes()
	if len(data) < 12 {
		t.Fatalf("Expected at least 12 bytes for timezone field, got %d", len(data))
	}

	// Timezone is at bytes 8-11 (little-endian)
	timezone := binary.LittleEndian.Uint32(data[8:12])
	if timezone != 0 {
		t.Errorf("Expected timezone 0, got %d", timezone)
	}
}

// TestWritePCAPHeader_SigfigsZero verifies sigfigs is zero
func TestWritePCAPHeader_SigfigsZero(t *testing.T) {
	c := &Capturer{}

	c.writePCAPHeader()

	data := c.pcapBuffer.Bytes()
	if len(data) < 16 {
		t.Fatalf("Expected at least 16 bytes for sigfigs field, got %d", len(data))
	}

	// Sigfigs is at bytes 12-15 (little-endian)
	sigfigs := binary.LittleEndian.Uint32(data[12:16])
	if sigfigs != 0 {
		t.Errorf("Expected sigfigs 0, got %d", sigfigs)
	}
}

// =====================================================================
// Tests for writePCAPPacket (PCAP packet header encoding)
// =====================================================================

// TestWritePCAPPacket_TimestampSeconds verifies timestamp seconds encoded correctly
func TestWritePCAPPacket_TimestampSeconds(t *testing.T) {
	c := &Capturer{}

	// Create a packet with a specific timestamp
	ts := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
	packet := &mockPacket{
		data:      []byte{0x01, 0x02, 0x03, 0x04},
		timestamp: ts,
	}

	c.writePCAPPacket(packet)

	data := c.pcapBuffer.Bytes()
	if len(data) < 4 {
		t.Fatalf("Expected at least 4 bytes for timestamp seconds, got %d", len(data))
	}

	// Timestamp seconds is at bytes 0-3 (little-endian)
	tsSec := binary.LittleEndian.Uint32(data[0:4])
	expected := uint32(ts.Unix())
	if tsSec != expected {
		t.Errorf("Expected timestamp seconds %d, got %d", expected, tsSec)
	}
}

// TestWritePCAPPacket_TimestampMicroseconds verifies timestamp microseconds computed correctly
func TestWritePCAPPacket_TimestampMicroseconds(t *testing.T) {
	c := &Capturer{}

	// Create a packet with nanoseconds that translate to specific microseconds
	// 123456789 nanoseconds = 123456 microseconds (integer division)
	ts := time.Date(2024, 6, 15, 10, 30, 45, 123456789, time.UTC)
	packet := &mockPacket{
		data:      []byte{0x01, 0x02, 0x03, 0x04},
		timestamp: ts,
	}

	c.writePCAPPacket(packet)

	data := c.pcapBuffer.Bytes()
	if len(data) < 8 {
		t.Fatalf("Expected at least 8 bytes for timestamp fields, got %d", len(data))
	}

	// Timestamp microseconds is at bytes 4-7 (little-endian)
	tsUsec := binary.LittleEndian.Uint32(data[4:8])
	expected := uint32(123456) // 123456789 / 1000 = 123456
	if tsUsec != expected {
		t.Errorf("Expected timestamp microseconds %d, got %d", expected, tsUsec)
	}
}

// TestWritePCAPPacket_IncludedLength verifies included length matches actual data length
func TestWritePCAPPacket_IncludedLength(t *testing.T) {
	c := &Capturer{}

	packetData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	packet := &mockPacket{
		data:      packetData,
		timestamp: time.Now(),
	}

	c.writePCAPPacket(packet)

	data := c.pcapBuffer.Bytes()
	if len(data) < 12 {
		t.Fatalf("Expected at least 12 bytes for included length field, got %d", len(data))
	}

	// Included length is at bytes 8-11 (little-endian)
	inclLen := binary.LittleEndian.Uint32(data[8:12])
	expected := uint32(len(packetData))
	if inclLen != expected {
		t.Errorf("Expected included length %d, got %d", expected, inclLen)
	}
}

// TestWritePCAPPacket_OriginalLength verifies original length field set correctly
func TestWritePCAPPacket_OriginalLength(t *testing.T) {
	c := &Capturer{}

	packetData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	packet := &mockPacket{
		data:      packetData,
		timestamp: time.Now(),
	}

	c.writePCAPPacket(packet)

	data := c.pcapBuffer.Bytes()
	if len(data) < 16 {
		t.Fatalf("Expected at least 16 bytes for original length field, got %d", len(data))
	}

	// Original length is at bytes 12-15 (little-endian)
	origLen := binary.LittleEndian.Uint32(data[12:16])
	expected := uint32(len(packetData))
	if origLen != expected {
		t.Errorf("Expected original length %d, got %d", expected, origLen)
	}
}

// TestWritePCAPPacket_DataAppendedAfterHeader verifies packet data appended after 16-byte header
func TestWritePCAPPacket_DataAppendedAfterHeader(t *testing.T) {
	c := &Capturer{}

	packetData := []byte{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe}
	packet := &mockPacket{
		data:      packetData,
		timestamp: time.Now(),
	}

	c.writePCAPPacket(packet)

	data := c.pcapBuffer.Bytes()

	// Total should be 16 bytes header + packet data length
	expectedTotal := 16 + len(packetData)
	if len(data) != expectedTotal {
		t.Fatalf("Expected total length %d, got %d", expectedTotal, len(data))
	}

	// Packet data should start at byte 16
	actualData := data[16:]
	if !bytes.Equal(actualData, packetData) {
		t.Errorf("Packet data mismatch\nExpected: %v\nGot:      %v", packetData, actualData)
	}
}

// TestWritePCAPPacket_HeaderSize verifies packet header is exactly 16 bytes
func TestWritePCAPPacket_HeaderSize(t *testing.T) {
	c := &Capturer{}

	// Use empty data to verify header size alone
	packet := &mockPacket{
		data:      []byte{},
		timestamp: time.Now(),
	}

	c.writePCAPPacket(packet)

	data := c.pcapBuffer.Bytes()
	expectedHeaderSize := 16
	if len(data) != expectedHeaderSize {
		t.Errorf("Expected packet header size %d bytes, got %d", expectedHeaderSize, len(data))
	}
}

// TestWritePCAPPacket_ZeroTimestamp verifies handling of zero timestamp (Unix epoch)
func TestWritePCAPPacket_ZeroTimestamp(t *testing.T) {
	c := &Capturer{}

	// Unix epoch: January 1, 1970 00:00:00 UTC
	ts := time.Unix(0, 0)
	packet := &mockPacket{
		data:      []byte{0x01},
		timestamp: ts,
	}

	c.writePCAPPacket(packet)

	data := c.pcapBuffer.Bytes()
	if len(data) < 8 {
		t.Fatalf("Expected at least 8 bytes for timestamp fields, got %d", len(data))
	}

	tsSec := binary.LittleEndian.Uint32(data[0:4])
	tsUsec := binary.LittleEndian.Uint32(data[4:8])

	if tsSec != 0 {
		t.Errorf("Expected timestamp seconds 0, got %d", tsSec)
	}
	if tsUsec != 0 {
		t.Errorf("Expected timestamp microseconds 0, got %d", tsUsec)
	}
}

// TestWritePCAPPacket_LargePacket verifies handling of large packet data
func TestWritePCAPPacket_LargePacket(t *testing.T) {
	c := &Capturer{}

	// Create a large packet (1500 bytes - typical MTU)
	packetData := make([]byte, 1500)
	for i := range packetData {
		packetData[i] = byte(i % 256)
	}

	packet := &mockPacket{
		data:      packetData,
		timestamp: time.Now(),
	}

	c.writePCAPPacket(packet)

	data := c.pcapBuffer.Bytes()

	// Verify included and original lengths
	inclLen := binary.LittleEndian.Uint32(data[8:12])
	origLen := binary.LittleEndian.Uint32(data[12:16])

	if inclLen != 1500 {
		t.Errorf("Expected included length 1500, got %d", inclLen)
	}
	if origLen != 1500 {
		t.Errorf("Expected original length 1500, got %d", origLen)
	}

	// Verify packet data integrity
	actualData := data[16:]
	if !bytes.Equal(actualData, packetData) {
		t.Error("Large packet data mismatch")
	}
}

// TestWritePCAPPacket_MultiplePackets verifies multiple packets written sequentially
func TestWritePCAPPacket_MultiplePackets(t *testing.T) {
	c := &Capturer{}

	packet1Data := []byte{0x01, 0x02, 0x03}
	packet1 := &mockPacket{
		data:      packet1Data,
		timestamp: time.Unix(1000, 0),
	}

	packet2Data := []byte{0x04, 0x05}
	packet2 := &mockPacket{
		data:      packet2Data,
		timestamp: time.Unix(2000, 0),
	}

	c.writePCAPPacket(packet1)
	c.writePCAPPacket(packet2)

	data := c.pcapBuffer.Bytes()

	// Expected total: (16 + 3) + (16 + 2) = 37 bytes
	expectedTotal := (16 + 3) + (16 + 2)
	if len(data) != expectedTotal {
		t.Fatalf("Expected total length %d, got %d", expectedTotal, len(data))
	}

	// Verify first packet timestamp (bytes 0-3)
	ts1 := binary.LittleEndian.Uint32(data[0:4])
	if ts1 != 1000 {
		t.Errorf("Expected first packet timestamp 1000, got %d", ts1)
	}

	// Verify first packet data (bytes 16-18)
	if !bytes.Equal(data[16:19], packet1Data) {
		t.Errorf("First packet data mismatch")
	}

	// Second packet starts at byte 19
	// Verify second packet timestamp (bytes 19-22)
	ts2 := binary.LittleEndian.Uint32(data[19:23])
	if ts2 != 2000 {
		t.Errorf("Expected second packet timestamp 2000, got %d", ts2)
	}

	// Verify second packet data (bytes 35-36)
	if !bytes.Equal(data[35:37], packet2Data) {
		t.Errorf("Second packet data mismatch")
	}
}

// =====================================================================
// Tests for BPF Filter behavior
// =====================================================================

// TestSetBPFFilter_StoresDefaultFilter verifies SetBPFFilter stores the filter as default
func TestSetBPFFilter_StoresDefaultFilter(t *testing.T) {
	c := &Capturer{}

	hubExclusion := "not (host 10.96.0.100 and (port 8080 or port 9090))"
	c.SetBPFFilter(hubExclusion)

	if c.bpfFilter != hubExclusion {
		t.Errorf("Expected bpfFilter %q, got %q", hubExclusion, c.bpfFilter)
	}
	if c.defaultBPFFilter != hubExclusion {
		t.Errorf("Expected defaultBPFFilter %q, got %q", hubExclusion, c.defaultBPFFilter)
	}
}

// TestBuildCombinedFilter_EmptyUserFilter_ReturnsDefaultFilter verifies empty user filter uses default
func TestBuildCombinedFilter_EmptyUserFilter_ReturnsDefaultFilter(t *testing.T) {
	c := &Capturer{}
	defaultFilter := "not (host 10.96.0.100 and (port 8080 or port 9090))"
	c.SetBPFFilter(defaultFilter)

	combined := c.BuildCombinedFilter("")

	if combined != defaultFilter {
		t.Errorf("Expected %q, got %q", defaultFilter, combined)
	}
}

// TestBuildCombinedFilter_UserFilter_CombinesWithDefault verifies user filter is combined with default
func TestBuildCombinedFilter_UserFilter_CombinesWithDefault(t *testing.T) {
	c := &Capturer{}
	defaultFilter := "not (host 10.96.0.100 and (port 8080 or port 9090))"
	c.SetBPFFilter(defaultFilter)

	userFilter := "tcp port 80"
	combined := c.BuildCombinedFilter(userFilter)

	// Should combine: (user filter) and (default filter)
	expected := "(tcp port 80) and (not (host 10.96.0.100 and (port 8080 or port 9090)))"
	if combined != expected {
		t.Errorf("Expected %q, got %q", expected, combined)
	}
}

// TestBuildCombinedFilter_ComplexUserFilter_CombinesCorrectly verifies complex filters combine
func TestBuildCombinedFilter_ComplexUserFilter_CombinesCorrectly(t *testing.T) {
	c := &Capturer{}
	defaultFilter := "not (host 10.96.0.100 and (port 8080 or port 9090))"
	c.SetBPFFilter(defaultFilter)

	userFilter := "tcp port 80 or tcp port 443"
	combined := c.BuildCombinedFilter(userFilter)

	// Should wrap user filter in parens to preserve logic
	expected := "(tcp port 80 or tcp port 443) and (not (host 10.96.0.100 and (port 8080 or port 9090)))"
	if combined != expected {
		t.Errorf("Expected %q, got %q", expected, combined)
	}
}

// TestBuildCombinedFilter_NoDefaultFilter_ReturnsUserFilter verifies user filter returned when no default
func TestBuildCombinedFilter_NoDefaultFilter_ReturnsUserFilter(t *testing.T) {
	c := &Capturer{}
	// No default filter set

	userFilter := "tcp port 80"
	combined := c.BuildCombinedFilter(userFilter)

	if combined != userFilter {
		t.Errorf("Expected %q, got %q", userFilter, combined)
	}
}

// TestBuildCombinedFilter_NoDefaultNoUser_ReturnsEmpty verifies empty when both empty
func TestBuildCombinedFilter_NoDefaultNoUser_ReturnsEmpty(t *testing.T) {
	c := &Capturer{}

	combined := c.BuildCombinedFilter("")

	if combined != "" {
		t.Errorf("Expected empty string, got %q", combined)
	}
}

// TestBuildCombinedFilter_PreservesHubExclusion verifies hub exclusion always included
func TestBuildCombinedFilter_PreservesHubExclusion(t *testing.T) {
	c := &Capturer{}
	hubExclusion := "not (host 10.96.0.100 and (port 8080 or port 9090))"
	c.SetBPFFilter(hubExclusion)

	testCases := []struct {
		name       string
		userFilter string
	}{
		{"not port 53", "not port 53"},
		{"http/https only", "tcp port 80 or tcp port 443"},
		{"tcp syn only", "tcp[tcpflags] & tcp-syn != 0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			combined := c.BuildCombinedFilter(tc.userFilter)

			// Combined filter MUST contain the hub exclusion
			if !containsSubstring(combined, "host 10.96.0.100") {
				t.Errorf("Combined filter missing hub IP exclusion: %q", combined)
			}
			if !containsSubstring(combined, "port 8080") {
				t.Errorf("Combined filter missing port 8080 exclusion: %q", combined)
			}
			if !containsSubstring(combined, "port 9090") {
				t.Errorf("Combined filter missing port 9090 exclusion: %q", combined)
			}
		})
	}
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

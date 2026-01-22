package hub

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
	"time"
)

// TestWritePCAPHeader_MagicNumber tests that the PCAP magic number is correct
func TestWritePCAPHeader_MagicNumber(t *testing.T) {
	var buf bytes.Buffer
	err := writePCAPHeader(&buf)
	if err != nil {
		t.Fatalf("writePCAPHeader() error = %v", err)
	}

	data := buf.Bytes()
	if len(data) < 4 {
		t.Fatalf("header too short: got %d bytes, want at least 4", len(data))
	}

	// Magic number is 0xa1b2c3d4, written little-endian as d4 c3 b2 a1
	magic := binary.LittleEndian.Uint32(data[0:4])
	want := uint32(0xa1b2c3d4)
	if magic != want {
		t.Errorf("magic number = 0x%08x, want 0x%08x", magic, want)
	}

	// Verify byte order (little-endian)
	if data[0] != 0xd4 || data[1] != 0xc3 || data[2] != 0xb2 || data[3] != 0xa1 {
		t.Errorf("magic number bytes = [%02x %02x %02x %02x], want [d4 c3 b2 a1]",
			data[0], data[1], data[2], data[3])
	}
}

// TestWritePCAPHeader_Version tests that the PCAP version is 2.4
func TestWritePCAPHeader_Version(t *testing.T) {
	var buf bytes.Buffer
	err := writePCAPHeader(&buf)
	if err != nil {
		t.Fatalf("writePCAPHeader() error = %v", err)
	}

	data := buf.Bytes()
	if len(data) < 8 {
		t.Fatalf("header too short: got %d bytes, want at least 8", len(data))
	}

	// Version major at offset 4-5 (uint16)
	major := binary.LittleEndian.Uint16(data[4:6])
	if major != 2 {
		t.Errorf("version major = %d, want 2", major)
	}

	// Version minor at offset 6-7 (uint16)
	minor := binary.LittleEndian.Uint16(data[6:8])
	if minor != 4 {
		t.Errorf("version minor = %d, want 4", minor)
	}
}

// TestWritePCAPHeader_Snaplen tests that the snaplen is 65535
func TestWritePCAPHeader_Snaplen(t *testing.T) {
	var buf bytes.Buffer
	err := writePCAPHeader(&buf)
	if err != nil {
		t.Fatalf("writePCAPHeader() error = %v", err)
	}

	data := buf.Bytes()
	if len(data) < 20 {
		t.Fatalf("header too short: got %d bytes, want at least 20", len(data))
	}

	// Snaplen at offset 16-19 (after magic, version, timezone, sigfigs)
	// Layout: magic(4) + major(2) + minor(2) + thiszone(4) + sigfigs(4) + snaplen(4)
	snaplen := binary.LittleEndian.Uint32(data[16:20])
	want := uint32(65535)
	if snaplen != want {
		t.Errorf("snaplen = %d, want %d", snaplen, want)
	}
}

// TestWritePCAPHeader_LinkType tests that the link type is 1 (Ethernet)
func TestWritePCAPHeader_LinkType(t *testing.T) {
	var buf bytes.Buffer
	err := writePCAPHeader(&buf)
	if err != nil {
		t.Fatalf("writePCAPHeader() error = %v", err)
	}

	data := buf.Bytes()
	if len(data) < 24 {
		t.Fatalf("header too short: got %d bytes, want at least 24", len(data))
	}

	// Link type at offset 20-23 (after snaplen)
	linkType := binary.LittleEndian.Uint32(data[20:24])
	want := uint32(1) // Ethernet
	if linkType != want {
		t.Errorf("link type = %d, want %d (Ethernet)", linkType, want)
	}
}

// TestWritePCAPHeader_Size tests that the header is exactly 24 bytes
func TestWritePCAPHeader_Size(t *testing.T) {
	var buf bytes.Buffer
	err := writePCAPHeader(&buf)
	if err != nil {
		t.Fatalf("writePCAPHeader() error = %v", err)
	}

	if buf.Len() != 24 {
		t.Errorf("header size = %d bytes, want 24", buf.Len())
	}
}

// TestWritePCAPHeader_TimezoneAndSigfigs tests timezone and sigfigs are zero
func TestWritePCAPHeader_TimezoneAndSigfigs(t *testing.T) {
	var buf bytes.Buffer
	err := writePCAPHeader(&buf)
	if err != nil {
		t.Fatalf("writePCAPHeader() error = %v", err)
	}

	data := buf.Bytes()
	if len(data) < 16 {
		t.Fatalf("header too short: got %d bytes, want at least 16", len(data))
	}

	// Timezone (thiszone) at offset 8-11 (int32)
	timezone := int32(binary.LittleEndian.Uint32(data[8:12]))
	if timezone != 0 {
		t.Errorf("timezone = %d, want 0", timezone)
	}

	// Sigfigs at offset 12-15 (uint32)
	sigfigs := binary.LittleEndian.Uint32(data[12:16])
	if sigfigs != 0 {
		t.Errorf("sigfigs = %d, want 0", sigfigs)
	}
}

// TestWritePCAPHeader_FullStructure tests the complete header structure
func TestWritePCAPHeader_FullStructure(t *testing.T) {
	var buf bytes.Buffer
	err := writePCAPHeader(&buf)
	if err != nil {
		t.Fatalf("writePCAPHeader() error = %v", err)
	}

	data := buf.Bytes()
	if len(data) != 24 {
		t.Fatalf("header size = %d bytes, want 24", len(data))
	}

	// Parse the full structure
	type pcapHeader struct {
		MagicNumber  uint32
		VersionMajor uint16
		VersionMinor uint16
		ThisZone     int32
		SigFigs      uint32
		SnapLen      uint32
		Network      uint32
	}

	var header pcapHeader
	reader := bytes.NewReader(data)
	err = binary.Read(reader, binary.LittleEndian, &header)
	if err != nil {
		t.Fatalf("failed to parse header: %v", err)
	}

	// Verify all fields
	if header.MagicNumber != 0xa1b2c3d4 {
		t.Errorf("MagicNumber = 0x%08x, want 0xa1b2c3d4", header.MagicNumber)
	}
	if header.VersionMajor != 2 {
		t.Errorf("VersionMajor = %d, want 2", header.VersionMajor)
	}
	if header.VersionMinor != 4 {
		t.Errorf("VersionMinor = %d, want 4", header.VersionMinor)
	}
	if header.ThisZone != 0 {
		t.Errorf("ThisZone = %d, want 0", header.ThisZone)
	}
	if header.SigFigs != 0 {
		t.Errorf("SigFigs = %d, want 0", header.SigFigs)
	}
	if header.SnapLen != 65535 {
		t.Errorf("SnapLen = %d, want 65535", header.SnapLen)
	}
	if header.Network != 1 {
		t.Errorf("Network = %d, want 1 (Ethernet)", header.Network)
	}
}

// TestWritePCAPHeader_NoError tests that writePCAPHeader doesn't return an error
func TestWritePCAPHeader_NoError(t *testing.T) {
	var buf bytes.Buffer
	err := writePCAPHeader(&buf)
	if err != nil {
		t.Errorf("writePCAPHeader() returned unexpected error: %v", err)
	}
}

// TestWritePCAPPacket_TimestampSeconds tests that timestamp seconds are from Unix time
func TestWritePCAPPacket_TimestampSeconds(t *testing.T) {
	var buf bytes.Buffer
	data := []byte("test packet data")
	timestamp := time.Date(2024, 6, 15, 10, 30, 45, 123456789, time.UTC)

	err := WritePCAPPacket(&buf, data, timestamp)
	if err != nil {
		t.Fatalf("WritePCAPPacket() error = %v", err)
	}

	result := buf.Bytes()
	if len(result) < 4 {
		t.Fatalf("output too short: got %d bytes, want at least 4", len(result))
	}

	// Timestamp seconds at offset 0-3 (uint32)
	tsSec := binary.LittleEndian.Uint32(result[0:4])
	wantSec := uint32(timestamp.Unix())
	if tsSec != wantSec {
		t.Errorf("timestamp seconds = %d, want %d", tsSec, wantSec)
	}
}

// TestWritePCAPPacket_TimestampMicroseconds tests that microseconds are computed correctly
func TestWritePCAPPacket_TimestampMicroseconds(t *testing.T) {
	var buf bytes.Buffer
	data := []byte("test packet data")
	// Use a timestamp with specific nanoseconds: 123456789 ns = 123456 us + 789 ns
	timestamp := time.Date(2024, 6, 15, 10, 30, 45, 123456789, time.UTC)

	err := WritePCAPPacket(&buf, data, timestamp)
	if err != nil {
		t.Fatalf("WritePCAPPacket() error = %v", err)
	}

	result := buf.Bytes()
	if len(result) < 8 {
		t.Fatalf("output too short: got %d bytes, want at least 8", len(result))
	}

	// Timestamp microseconds at offset 4-7 (uint32)
	tsUsec := binary.LittleEndian.Uint32(result[4:8])
	// Expected: nanoseconds / 1000 = 123456789 / 1000 = 123456 (integer division)
	wantUsec := uint32(123456)
	if tsUsec != wantUsec {
		t.Errorf("timestamp microseconds = %d, want %d", tsUsec, wantUsec)
	}
}

// TestWritePCAPPacket_MicrosecondsPrecision tests various microsecond values
func TestWritePCAPPacket_MicrosecondsPrecision(t *testing.T) {
	tests := []struct {
		name      string
		nanos     int
		wantUsec  uint32
	}{
		{"zero nanoseconds", 0, 0},
		{"999 nanoseconds rounds to 0", 999, 0},
		{"1000 nanoseconds = 1 usec", 1000, 1},
		{"500000000 ns = 500000 usec", 500000000, 500000},
		{"999999999 ns = 999999 usec", 999999999, 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			data := []byte("test")
			timestamp := time.Date(2024, 1, 1, 0, 0, 0, tt.nanos, time.UTC)

			err := WritePCAPPacket(&buf, data, timestamp)
			if err != nil {
				t.Fatalf("WritePCAPPacket() error = %v", err)
			}

			result := buf.Bytes()
			tsUsec := binary.LittleEndian.Uint32(result[4:8])
			if tsUsec != tt.wantUsec {
				t.Errorf("timestamp microseconds = %d, want %d", tsUsec, tt.wantUsec)
			}
		})
	}
}

// TestWritePCAPPacket_IncludedLength tests that included length is correct
func TestWritePCAPPacket_IncludedLength(t *testing.T) {
	tests := []struct {
		name     string
		dataLen  int
	}{
		{"empty data", 0},
		{"small packet", 16},
		{"typical packet", 100},
		{"MTU size packet", 1500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			data := make([]byte, tt.dataLen)
			timestamp := time.Now()

			err := WritePCAPPacket(&buf, data, timestamp)
			if err != nil {
				t.Fatalf("WritePCAPPacket() error = %v", err)
			}

			result := buf.Bytes()
			if len(result) < 12 {
				t.Fatalf("output too short: got %d bytes, want at least 12", len(result))
			}

			// Included length at offset 8-11 (uint32)
			inclLen := binary.LittleEndian.Uint32(result[8:12])
			wantLen := uint32(tt.dataLen)
			if inclLen != wantLen {
				t.Errorf("included length = %d, want %d", inclLen, wantLen)
			}
		})
	}
}

// TestWritePCAPPacket_OriginalLength tests that original length is correct
func TestWritePCAPPacket_OriginalLength(t *testing.T) {
	tests := []struct {
		name     string
		dataLen  int
	}{
		{"empty data", 0},
		{"small packet", 16},
		{"typical packet", 100},
		{"MTU size packet", 1500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			data := make([]byte, tt.dataLen)
			timestamp := time.Now()

			err := WritePCAPPacket(&buf, data, timestamp)
			if err != nil {
				t.Fatalf("WritePCAPPacket() error = %v", err)
			}

			result := buf.Bytes()
			if len(result) < 16 {
				t.Fatalf("output too short: got %d bytes, want at least 16", len(result))
			}

			// Original length at offset 12-15 (uint32)
			origLen := binary.LittleEndian.Uint32(result[12:16])
			wantLen := uint32(tt.dataLen)
			if origLen != wantLen {
				t.Errorf("original length = %d, want %d", origLen, wantLen)
			}
		})
	}
}

// TestWritePCAPPacket_HeaderSize tests that packet header is exactly 16 bytes
func TestWritePCAPPacket_HeaderSize(t *testing.T) {
	var buf bytes.Buffer
	data := []byte("test packet data")
	timestamp := time.Now()

	err := WritePCAPPacket(&buf, data, timestamp)
	if err != nil {
		t.Fatalf("WritePCAPPacket() error = %v", err)
	}

	// Total output should be 16-byte header + data length
	expectedLen := 16 + len(data)
	if buf.Len() != expectedLen {
		t.Errorf("output size = %d bytes, want %d (16 byte header + %d data)", buf.Len(), expectedLen, len(data))
	}
}

// TestWritePCAPPacket_DataAppended tests that packet data is appended after header
func TestWritePCAPPacket_DataAppended(t *testing.T) {
	var buf bytes.Buffer
	data := []byte("test packet data 12345")
	timestamp := time.Now()

	err := WritePCAPPacket(&buf, data, timestamp)
	if err != nil {
		t.Fatalf("WritePCAPPacket() error = %v", err)
	}

	result := buf.Bytes()
	if len(result) < 16+len(data) {
		t.Fatalf("output too short: got %d bytes, want %d", len(result), 16+len(data))
	}

	// Data should be at offset 16 (after header)
	packetData := result[16:]
	if !bytes.Equal(packetData, data) {
		t.Errorf("packet data = %q, want %q", packetData, data)
	}
}

// TestWritePCAPPacket_ZeroTimestamp tests Unix epoch timestamp handling
func TestWritePCAPPacket_ZeroTimestamp(t *testing.T) {
	var buf bytes.Buffer
	data := []byte("test")
	timestamp := time.Unix(0, 0) // Unix epoch

	err := WritePCAPPacket(&buf, data, timestamp)
	if err != nil {
		t.Fatalf("WritePCAPPacket() error = %v", err)
	}

	result := buf.Bytes()
	tsSec := binary.LittleEndian.Uint32(result[0:4])
	tsUsec := binary.LittleEndian.Uint32(result[4:8])

	if tsSec != 0 {
		t.Errorf("timestamp seconds = %d, want 0", tsSec)
	}
	if tsUsec != 0 {
		t.Errorf("timestamp microseconds = %d, want 0", tsUsec)
	}
}

// TestWritePCAPPacket_FullStructure tests the complete packet record structure
func TestWritePCAPPacket_FullStructure(t *testing.T) {
	var buf bytes.Buffer
	data := []byte("hello world packet")
	timestamp := time.Date(2024, 12, 25, 12, 0, 0, 500000000, time.UTC) // 500000 usec

	err := WritePCAPPacket(&buf, data, timestamp)
	if err != nil {
		t.Fatalf("WritePCAPPacket() error = %v", err)
	}

	result := buf.Bytes()
	if len(result) != 16+len(data) {
		t.Fatalf("output size = %d bytes, want %d", len(result), 16+len(data))
	}

	// Parse the full packet header structure
	type pcapPacketHeader struct {
		TsSec   uint32
		TsUsec  uint32
		InclLen uint32
		OrigLen uint32
	}

	var header pcapPacketHeader
	reader := bytes.NewReader(result[:16])
	err = binary.Read(reader, binary.LittleEndian, &header)
	if err != nil {
		t.Fatalf("failed to parse packet header: %v", err)
	}

	// Verify all header fields
	wantSec := uint32(timestamp.Unix())
	if header.TsSec != wantSec {
		t.Errorf("TsSec = %d, want %d", header.TsSec, wantSec)
	}
	if header.TsUsec != 500000 {
		t.Errorf("TsUsec = %d, want 500000", header.TsUsec)
	}
	if header.InclLen != uint32(len(data)) {
		t.Errorf("InclLen = %d, want %d", header.InclLen, len(data))
	}
	if header.OrigLen != uint32(len(data)) {
		t.Errorf("OrigLen = %d, want %d", header.OrigLen, len(data))
	}

	// Verify data
	if !bytes.Equal(result[16:], data) {
		t.Errorf("packet data mismatch")
	}
}

// TestWritePCAPPacket_MultiplePackets tests writing multiple packets sequentially
func TestWritePCAPPacket_MultiplePackets(t *testing.T) {
	var buf bytes.Buffer
	packets := []struct {
		data      []byte
		timestamp time.Time
	}{
		{[]byte("first packet"), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{[]byte("second packet data"), time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC)},
		{[]byte("third"), time.Date(2024, 1, 1, 0, 0, 2, 0, time.UTC)},
	}

	for _, p := range packets {
		err := WritePCAPPacket(&buf, p.data, p.timestamp)
		if err != nil {
			t.Fatalf("WritePCAPPacket() error = %v", err)
		}
	}

	result := buf.Bytes()

	// Verify total size
	expectedSize := 0
	for _, p := range packets {
		expectedSize += 16 + len(p.data)
	}
	if len(result) != expectedSize {
		t.Errorf("total size = %d, want %d", len(result), expectedSize)
	}

	// Verify each packet
	offset := 0
	for i, p := range packets {
		// Read included length to find packet boundary
		inclLen := binary.LittleEndian.Uint32(result[offset+8 : offset+12])
		if inclLen != uint32(len(p.data)) {
			t.Errorf("packet %d: included length = %d, want %d", i, inclLen, len(p.data))
		}

		// Verify data
		packetData := result[offset+16 : offset+16+int(inclLen)]
		if !bytes.Equal(packetData, p.data) {
			t.Errorf("packet %d: data mismatch", i)
		}

		offset += 16 + len(p.data)
	}
}

// TestWritePCAPPacket_NoError tests that WritePCAPPacket doesn't return an error
func TestWritePCAPPacket_NoError(t *testing.T) {
	var buf bytes.Buffer
	data := []byte("test packet")
	timestamp := time.Now()

	err := WritePCAPPacket(&buf, data, timestamp)
	if err != nil {
		t.Errorf("WritePCAPPacket() returned unexpected error: %v", err)
	}
}

// ===== PCAPBuffer Write() Tests =====

// TestWrite_CreatesAgentFile tests that Write() creates a new agent-specific PCAP file
func TestWrite_CreatesAgentFile(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	// Write some packet data (not a PCAP header)
	packetData := []byte("packet data here")
	err := pb.Write("agent-001", packetData)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify the file was created
	filePath := dir + "/agent-agent-001.pcap"
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// File should contain PCAP header (24 bytes) + packet data
	expectedSize := int64(24 + len(packetData))
	if info.Size() != expectedSize {
		t.Errorf("file size = %d, want %d", info.Size(), expectedSize)
	}
}

// TestWrite_FileHasPCAPHeader tests that new files start with PCAP global header
func TestWrite_FileHasPCAPHeader(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	packetData := []byte("test packet")
	err := pb.Write("agent-002", packetData)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Read the file
	filePath := dir + "/agent-agent-002.pcap"
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(data) < 24 {
		t.Fatalf("file too short: got %d bytes, want at least 24", len(data))
	}

	// Verify PCAP magic number at start
	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != 0xa1b2c3d4 {
		t.Errorf("PCAP magic = 0x%08x, want 0xa1b2c3d4", magic)
	}

	// Verify version 2.4
	major := binary.LittleEndian.Uint16(data[4:6])
	minor := binary.LittleEndian.Uint16(data[6:8])
	if major != 2 || minor != 4 {
		t.Errorf("PCAP version = %d.%d, want 2.4", major, minor)
	}
}

// TestWrite_AppendsWithoutDuplicateHeader tests subsequent writes append packets without new header
func TestWrite_AppendsWithoutDuplicateHeader(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	packet1 := []byte("first packet")
	packet2 := []byte("second packet")

	// Write first packet - creates file with header
	err := pb.Write("agent-003", packet1)
	if err != nil {
		t.Fatalf("Write() first error = %v", err)
	}

	// Write second packet - should append without new header
	err = pb.Write("agent-003", packet2)
	if err != nil {
		t.Fatalf("Write() second error = %v", err)
	}

	// Read the file
	filePath := dir + "/agent-agent-003.pcap"
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Should have: header(24) + packet1 + packet2
	expectedSize := 24 + len(packet1) + len(packet2)
	if len(data) != expectedSize {
		t.Errorf("file size = %d, want %d", len(data), expectedSize)
	}

	// Verify only ONE PCAP header (check for magic at start, not in data section)
	// The header is only at bytes 0-23
	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != 0xa1b2c3d4 {
		t.Errorf("PCAP magic = 0x%08x, want 0xa1b2c3d4", magic)
	}

	// Verify both packets are in the file
	if !bytes.Contains(data[24:], packet1) {
		t.Error("file does not contain packet1")
	}
	if !bytes.Contains(data[24:], packet2) {
		t.Error("file does not contain packet2")
	}
}

// TestWrite_StripsIncomingPCAPHeader tests that incoming data with PCAP header is stripped
func TestWrite_StripsIncomingPCAPHeader(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	// Create data with a PCAP header prefix (little-endian magic)
	pcapHeader := []byte{
		0xd4, 0xc3, 0xb2, 0xa1, // magic (little-endian)
		0x02, 0x00, 0x04, 0x00, // version 2.4
		0x00, 0x00, 0x00, 0x00, // timezone
		0x00, 0x00, 0x00, 0x00, // sigfigs
		0xff, 0xff, 0x00, 0x00, // snaplen 65535
		0x01, 0x00, 0x00, 0x00, // network (ethernet)
	}
	packetData := []byte("actual packet data")
	dataWithHeader := append(pcapHeader, packetData...)

	err := pb.Write("agent-004", dataWithHeader)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Read the file
	filePath := dir + "/agent-agent-004.pcap"
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Should have: our header(24) + packetData (without the duplicate header)
	expectedSize := 24 + len(packetData)
	if len(data) != expectedSize {
		t.Errorf("file size = %d, want %d (incoming header should be stripped)", len(data), expectedSize)
	}
}

// TestWrite_MultipleAgents tests that separate files are created for each agent
func TestWrite_MultipleAgents(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	// Write data from two different agents
	err := pb.Write("agent-a", []byte("agent a data"))
	if err != nil {
		t.Fatalf("Write() agent-a error = %v", err)
	}

	err = pb.Write("agent-b", []byte("agent b data"))
	if err != nil {
		t.Fatalf("Write() agent-b error = %v", err)
	}

	// Verify two separate files exist
	_, err = os.Stat(dir + "/agent-agent-a.pcap")
	if err != nil {
		t.Errorf("agent-a file not found: %v", err)
	}

	_, err = os.Stat(dir + "/agent-agent-b.pcap")
	if err != nil {
		t.Errorf("agent-b file not found: %v", err)
	}
}

// TestWrite_UpdatesTotalSize tests that totalSize is correctly updated
func TestWrite_UpdatesTotalSize(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	// Initially size should be 0
	if pb.Size() != 0 {
		t.Errorf("initial size = %d, want 0", pb.Size())
	}

	// Write some data
	packet1 := []byte("packet one data")
	err := pb.Write("agent-005", packet1)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Size() returns totalSize which tracks packet data written (not file size with headers)
	// The implementation only adds the data bytes to totalSize, not the PCAP header
	expectedSize := int64(len(packet1))
	if pb.Size() != expectedSize {
		t.Errorf("size after first write = %d, want %d", pb.Size(), expectedSize)
	}

	// Write more data
	packet2 := []byte("packet two")
	err = pb.Write("agent-005", packet2)
	if err != nil {
		t.Fatalf("Write() second error = %v", err)
	}

	// Size should now include both packets (data only, no headers)
	expectedSize = int64(len(packet1) + len(packet2))
	if pb.Size() != expectedSize {
		t.Errorf("size after second write = %d, want %d", pb.Size(), expectedSize)
	}
}

// ===== PCAPBuffer GetSessionPCAP() Tests =====

// TestGetSessionPCAP_EmptyBuffer tests GetSessionPCAP with no data
func TestGetSessionPCAP_EmptyBuffer(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	data, err := pb.GetSessionPCAP()
	if err != nil {
		t.Fatalf("GetSessionPCAP() error = %v", err)
	}

	// Should return just the global header (24 bytes)
	if len(data) != 24 {
		t.Errorf("empty buffer PCAP size = %d, want 24", len(data))
	}

	// Verify magic number
	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != 0xa1b2c3d4 {
		t.Errorf("magic = 0x%08x, want 0xa1b2c3d4", magic)
	}
}

// TestGetSessionPCAP_SingleAgent tests merging with single agent
func TestGetSessionPCAP_SingleAgent(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	packetData := []byte("single agent packet")
	err := pb.Write("agent-single", packetData)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	merged, err := pb.GetSessionPCAP()
	if err != nil {
		t.Fatalf("GetSessionPCAP() error = %v", err)
	}

	// Should have: header(24) + packetData
	expectedSize := 24 + len(packetData)
	if len(merged) != expectedSize {
		t.Errorf("merged size = %d, want %d", len(merged), expectedSize)
	}

	// Verify header
	magic := binary.LittleEndian.Uint32(merged[0:4])
	if magic != 0xa1b2c3d4 {
		t.Errorf("magic = 0x%08x, want 0xa1b2c3d4", magic)
	}

	// Verify packet data is present
	if !bytes.Equal(merged[24:], packetData) {
		t.Errorf("packet data mismatch")
	}
}

// TestGetSessionPCAP_MergesMultipleAgents tests merging files from multiple agents
func TestGetSessionPCAP_MergesMultipleAgents(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	packet1 := []byte("agent1-packet")
	packet2 := []byte("agent2-packet")

	err := pb.Write("agent-1", packet1)
	if err != nil {
		t.Fatalf("Write() agent-1 error = %v", err)
	}

	err = pb.Write("agent-2", packet2)
	if err != nil {
		t.Fatalf("Write() agent-2 error = %v", err)
	}

	merged, err := pb.GetSessionPCAP()
	if err != nil {
		t.Fatalf("GetSessionPCAP() error = %v", err)
	}

	// Should have: header(24) + packet1 + packet2
	expectedSize := 24 + len(packet1) + len(packet2)
	if len(merged) != expectedSize {
		t.Errorf("merged size = %d, want %d", len(merged), expectedSize)
	}

	// Verify single header
	magic := binary.LittleEndian.Uint32(merged[0:4])
	if magic != 0xa1b2c3d4 {
		t.Errorf("magic = 0x%08x, want 0xa1b2c3d4", magic)
	}

	// Verify both packets are in the merged data
	if !bytes.Contains(merged[24:], packet1) {
		t.Error("merged data does not contain packet1")
	}
	if !bytes.Contains(merged[24:], packet2) {
		t.Error("merged data does not contain packet2")
	}
}

// TestGetSessionPCAP_SingleGlobalHeader tests that merged output has only one global header
func TestGetSessionPCAP_SingleGlobalHeader(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	// Write to 3 different agents
	err := pb.Write("agent-x", []byte("packet-x"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	err = pb.Write("agent-y", []byte("packet-y"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	err = pb.Write("agent-z", []byte("packet-z"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	merged, err := pb.GetSessionPCAP()
	if err != nil {
		t.Fatalf("GetSessionPCAP() error = %v", err)
	}

	// Count occurrences of PCAP magic number (little-endian: d4 c3 b2 a1)
	magicLE := []byte{0xd4, 0xc3, 0xb2, 0xa1}
	count := bytes.Count(merged, magicLE)

	// Should have exactly one magic number (at the beginning)
	if count != 1 {
		t.Errorf("magic number count = %d, want 1 (should be single global header)", count)
	}

	// Verify magic is at position 0
	if !bytes.HasPrefix(merged, magicLE) {
		t.Error("merged data does not start with PCAP magic number")
	}
}

// TestGetSessionPCAP_SkipsPerAgentHeaders tests that per-agent headers (bytes 0-23) are skipped
func TestGetSessionPCAP_SkipsPerAgentHeaders(t *testing.T) {
	dir := t.TempDir()
	pb := NewPCAPBuffer(dir, 1024*1024)
	defer pb.Close()

	// Each agent file has 24-byte header + data
	// The merged output should skip these per-agent headers
	packet1 := []byte("PACKET_ONE__")  // 12 bytes
	packet2 := []byte("PACKET_TWO__")  // 12 bytes

	err := pb.Write("agent-p", packet1)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	err = pb.Write("agent-q", packet2)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify agent files have headers
	data1, _ := os.ReadFile(dir + "/agent-agent-p.pcap")
	data2, _ := os.ReadFile(dir + "/agent-agent-q.pcap")

	if len(data1) != 24+len(packet1) {
		t.Errorf("agent-p file size = %d, want %d", len(data1), 24+len(packet1))
	}
	if len(data2) != 24+len(packet2) {
		t.Errorf("agent-q file size = %d, want %d", len(data2), 24+len(packet2))
	}

	// Get merged PCAP
	merged, err := pb.GetSessionPCAP()
	if err != nil {
		t.Fatalf("GetSessionPCAP() error = %v", err)
	}

	// Merged should have: global header (24) + packet1 data + packet2 data
	// NOT: global header (24) + agent1 header (24) + packet1 + agent2 header (24) + packet2
	expectedSize := 24 + len(packet1) + len(packet2)
	if len(merged) != expectedSize {
		t.Errorf("merged size = %d, want %d (per-agent headers should be skipped)", len(merged), expectedSize)
	}

	// Verify packets are present without their original headers
	mergedContent := merged[24:] // skip global header
	if !bytes.Contains(mergedContent, packet1) {
		t.Error("merged data missing packet1")
	}
	if !bytes.Contains(mergedContent, packet2) {
		t.Error("merged data missing packet2")
	}
}

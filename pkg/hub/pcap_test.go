package hub

import (
	"bytes"
	"encoding/binary"
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

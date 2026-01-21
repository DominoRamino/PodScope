package hub

import (
	"bytes"
	"encoding/binary"
	"testing"
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

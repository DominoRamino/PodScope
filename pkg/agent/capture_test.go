package agent

import (
	"bytes"
	"encoding/binary"
	"testing"
)

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

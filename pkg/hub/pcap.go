package hub

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// PCAP magic number for microsecond resolution
	pcapMagicNumber = 0xa1b2c3d4
	// PCAP version
	pcapVersionMajor = 2
	pcapVersionMinor = 4
	// Ethernet link type
	pcapLinkTypeEthernet = 1
	// Maximum snapshot length
	pcapSnaplen = 65535
)

// PCAPBuffer manages PCAP data storage
type PCAPBuffer struct {
	dir       string
	maxSize   int64
	mutex     sync.RWMutex
	agents    map[string]*agentBuffer
	totalSize int64
}

// agentBuffer stores PCAP data from a single agent
type agentBuffer struct {
	agentID  string
	filePath string
	file     *os.File
	size     int64
}

// NewPCAPBuffer creates a new PCAP buffer
func NewPCAPBuffer(dir string, maxSize int64) *PCAPBuffer {
	return &PCAPBuffer{
		dir:     dir,
		maxSize: maxSize,
		agents:  make(map[string]*agentBuffer),
	}
}

// Write writes PCAP data from an agent
func (p *PCAPBuffer) Write(agentID string, data []byte) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Get or create agent buffer
	ab, exists := p.agents[agentID]
	if !exists {
		filePath := filepath.Join(p.dir, fmt.Sprintf("agent-%s.pcap", agentID))
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to create pcap file: %w", err)
		}

		// Write PCAP global header for new files
		var headerBuf bytes.Buffer
		if err := writePCAPHeader(&headerBuf); err != nil {
			file.Close()
			return fmt.Errorf("failed to write pcap header: %w", err)
		}
		if _, err := file.Write(headerBuf.Bytes()); err != nil {
			file.Close()
			return fmt.Errorf("failed to write pcap header to file: %w", err)
		}

		ab = &agentBuffer{
			agentID:  agentID,
			filePath: filePath,
			file:     file,
			size:     int64(headerBuf.Len()), // Account for header size
		}
		p.agents[agentID] = ab
	}

	// Strip PCAP header from data if present (agent's first chunk includes header)
	// PCAP magic number is 0xd4c3b2a1 (little-endian) or 0xa1b2c3d4 (big-endian)
	dataToWrite := data
	if len(data) >= 24 &&
		((data[0] == 0xd4 && data[1] == 0xc3 && data[2] == 0xb2 && data[3] == 0xa1) ||
			(data[0] == 0xa1 && data[1] == 0xb2 && data[2] == 0xc3 && data[3] == 0xd4)) {
		// This chunk has a PCAP header, skip it
		dataToWrite = data[24:]
	}

	// Write data
	n, err := ab.file.Write(dataToWrite)
	if err != nil {
		return fmt.Errorf("failed to write pcap data: %w", err)
	}

	ab.size += int64(n)
	p.totalSize += int64(n)

	return nil
}

// Size returns the total size of PCAP data
func (p *PCAPBuffer) Size() int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.totalSize
}

// GetSessionPCAP returns all PCAP data for the session merged into a single file
func (p *PCAPBuffer) GetSessionPCAP() ([]byte, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var buf bytes.Buffer

	// Write PCAP global header
	if err := writePCAPHeader(&buf); err != nil {
		return nil, err
	}

	// Merge all agent PCAP files
	for _, ab := range p.agents {
		if err := ab.file.Sync(); err != nil {
			continue
		}

		// Read from the file
		data, err := os.ReadFile(ab.filePath)
		if err != nil {
			continue
		}

		// Each agent file has a 24-byte PCAP global header at the start
		// Skip it and append only the packet records
		if len(data) > 24 {
			buf.Write(data[24:])
		}
	}

	return buf.Bytes(), nil
}

// GetStreamPCAP returns PCAP data for a specific stream
func (p *PCAPBuffer) GetStreamPCAP(streamID string) ([]byte, error) {
	// For MVP, we return all data - stream filtering would require packet parsing
	// TODO: Implement stream-specific filtering
	return p.GetSessionPCAP()
}

// Reset clears all PCAP data and deletes files
func (p *PCAPBuffer) Reset() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Close and delete all agent files
	for _, ab := range p.agents {
		if ab.file != nil {
			ab.file.Close()
		}
		// Delete the file
		if err := os.Remove(ab.filePath); err != nil {
			// Log but don't fail if file doesn't exist
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete pcap file: %w", err)
			}
		}
	}

	// Clear agents map and reset total size
	p.agents = make(map[string]*agentBuffer)
	p.totalSize = 0

	return nil
}

// Close closes all open files
func (p *PCAPBuffer) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, ab := range p.agents {
		if ab.file != nil {
			ab.file.Close()
		}
	}

	return nil
}

// writePCAPHeader writes the global PCAP header
func writePCAPHeader(w io.Writer) error {
	header := struct {
		MagicNumber  uint32
		VersionMajor uint16
		VersionMinor uint16
		ThisZone     int32
		SigFigs      uint32
		SnapLen      uint32
		Network      uint32
	}{
		MagicNumber:  pcapMagicNumber,
		VersionMajor: pcapVersionMajor,
		VersionMinor: pcapVersionMinor,
		ThisZone:     0,
		SigFigs:      0,
		SnapLen:      pcapSnaplen,
		Network:      pcapLinkTypeEthernet,
	}

	return binary.Write(w, binary.LittleEndian, &header)
}

// WritePCAPPacket writes a packet to a PCAP file
func WritePCAPPacket(w io.Writer, data []byte, timestamp time.Time) error {
	ts := timestamp.Unix()
	usec := timestamp.UnixMicro() - ts*1000000

	header := struct {
		TsSec   uint32
		TsUsec  uint32
		InclLen uint32
		OrigLen uint32
	}{
		TsSec:   uint32(ts),
		TsUsec:  uint32(usec),
		InclLen: uint32(len(data)),
		OrigLen: uint32(len(data)),
	}

	if err := binary.Write(w, binary.LittleEndian, &header); err != nil {
		return err
	}

	_, err := w.Write(data)
	return err
}

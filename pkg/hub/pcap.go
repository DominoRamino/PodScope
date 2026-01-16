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

		ab = &agentBuffer{
			agentID:  agentID,
			filePath: filePath,
			file:     file,
		}
		p.agents[agentID] = ab
	}

	// Write data
	n, err := ab.file.Write(data)
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

		// Skip the global header from each agent file and append packets
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

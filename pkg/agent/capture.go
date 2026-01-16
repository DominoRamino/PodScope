package agent

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/podscope/podscope/pkg/protocol"
)

const (
	// SnapLen is the maximum bytes to capture per packet
	SnapLen = 65535
	// BufferSize is the buffer size for packet capture
	BufferSize = 8 * 1024 * 1024 // 8MB
	// FlushInterval is how often to flush PCAP data to hub
	FlushInterval = 500 * time.Millisecond
)

// Capturer handles packet capture and analysis
type Capturer struct {
	iface            string
	bpfFilter        string
	defaultBPFFilter string // Initial filter to reset to when clearing
	handle           *pcap.Handle
	hubClient        *HubClient
	agentInfo        *protocol.AgentInfo

	// Packet buffer for PCAP
	pcapBuffer  bytes.Buffer
	pcapMutex   sync.Mutex

	// TCP stream reassembly
	assembler   *TCPAssembler

	// Stats
	stats       CaptureStats
	statsMutex  sync.RWMutex
}

// CaptureStats holds capture statistics
type CaptureStats struct {
	PacketsCaptured  uint64
	BytesCaptured    uint64
	TCPPackets       uint64
	UDPPackets       uint64
	HTTPRequests     uint64
	TLSHandshakes    uint64
	Errors           uint64
}

// NewCapturer creates a new packet capturer
func NewCapturer(iface string, agentInfo *protocol.AgentInfo, hubClient *HubClient) *Capturer {
	c := &Capturer{
		iface:     iface,
		hubClient: hubClient,
		agentInfo: agentInfo,
	}

	// Initialize TCP assembler with agent info for pod name population
	c.assembler = NewTCPAssembler(c.onFlowComplete, agentInfo)

	return c
}

// SetBPFFilter sets the BPF filter for capture
func (c *Capturer) SetBPFFilter(filter string) {
	c.bpfFilter = filter
	c.defaultBPFFilter = filter // Store as default for reset
}

// UpdateBPFFilter updates the BPF filter on a running capture
func (c *Capturer) UpdateBPFFilter(filter string) error {
	if c.handle == nil {
		return fmt.Errorf("capture not running")
	}

	// If empty string, reset to default filter
	targetFilter := filter
	if filter == "" {
		targetFilter = c.defaultBPFFilter
		log.Printf("Empty filter received - resetting to default")
	}

	// Check if filter actually changed
	if c.bpfFilter == targetFilter {
		return nil // No change needed
	}

	log.Printf("====================================")
	log.Printf("UPDATING BPF FILTER:")
	log.Printf("  Old: %s", c.bpfFilter)
	log.Printf("  New: %s", targetFilter)
	log.Printf("====================================")

	// Apply new filter to running handle
	if err := c.handle.SetBPFFilter(targetFilter); err != nil {
		log.Printf("ERROR: Failed to update BPF filter: %v", err)
		return fmt.Errorf("failed to update BPF filter: %w", err)
	}

	// Store new filter
	c.bpfFilter = targetFilter
	log.Printf("SUCCESS: BPF filter updated on running capture")

	return nil
}

// Start begins packet capture
func (c *Capturer) Start(ctx context.Context) error {
	// Open the device
	handle, err := pcap.OpenLive(c.iface, SnapLen, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("failed to open interface %s: %w", c.iface, err)
	}
	c.handle = handle

	// Set BPF filter if specified
	if c.bpfFilter != "" {
		log.Printf("Applying BPF filter to pcap handle: %s", c.bpfFilter)
		if err := handle.SetBPFFilter(c.bpfFilter); err != nil {
			log.Printf("ERROR: Failed to set BPF filter: %v", err)
			return fmt.Errorf("failed to set BPF filter: %w", err)
		}
		log.Printf("SUCCESS: BPF filter applied to pcap handle")
	} else {
		log.Printf("WARNING: No BPF filter set - capturing ALL traffic!")
	}

	// Write PCAP header
	c.writePCAPHeader()

	// Start PCAP flush goroutine
	go c.flushLoop(ctx)

	// Start packet processing
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetSource.NoCopy = true

	log.Printf("Starting capture on interface %s", c.iface)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping capture...")
			c.handle.Close()
			return nil
		case packet, ok := <-packetSource.Packets():
			if !ok {
				return nil
			}
			c.processPacket(packet)
		}
	}
}

// processPacket handles a single captured packet
func (c *Capturer) processPacket(packet gopacket.Packet) {
	c.statsMutex.Lock()
	c.stats.PacketsCaptured++
	c.stats.BytesCaptured += uint64(len(packet.Data()))
	c.statsMutex.Unlock()

	// Write to PCAP buffer
	c.writePCAPPacket(packet)

	// Parse network layers
	networkLayer := packet.NetworkLayer()
	if networkLayer == nil {
		return
	}

	transportLayer := packet.TransportLayer()
	if transportLayer == nil {
		return
	}

	// DEBUG: Log DNS packets that get through BPF filter
	switch tl := transportLayer.(type) {
	case *layers.TCP:
		if tl.SrcPort == 53 || tl.DstPort == 53 {
			log.Printf("WARNING: DNS TCP packet captured despite BPF filter! %d->%d", tl.SrcPort, tl.DstPort)
		}
	case *layers.UDP:
		if tl.SrcPort == 53 || tl.DstPort == 53 {
			log.Printf("WARNING: DNS UDP packet captured despite BPF filter! %d->%d", tl.SrcPort, tl.DstPort)
		}
	}

	switch transportLayer.LayerType() {
	case layers.LayerTypeTCP:
		c.statsMutex.Lock()
		c.stats.TCPPackets++
		c.statsMutex.Unlock()
		c.processTCPPacket(packet)
	case layers.LayerTypeUDP:
		c.statsMutex.Lock()
		c.stats.UDPPackets++
		c.statsMutex.Unlock()
		// UDP processing can be added later
	}
}

// processTCPPacket processes a TCP packet
func (c *Capturer) processTCPPacket(packet gopacket.Packet) {
	tcp := packet.TransportLayer().(*layers.TCP)
	networkLayer := packet.NetworkLayer()

	var srcIP, dstIP string
	switch ip := networkLayer.(type) {
	case *layers.IPv4:
		srcIP = ip.SrcIP.String()
		dstIP = ip.DstIP.String()
	case *layers.IPv6:
		srcIP = ip.SrcIP.String()
		dstIP = ip.DstIP.String()
	}

	// Feed to TCP assembler for stream reconstruction
	c.assembler.ProcessPacket(
		srcIP, dstIP,
		uint16(tcp.SrcPort), uint16(tcp.DstPort),
		tcp, packet.Metadata().Timestamp,
		packet.ApplicationLayer(),
	)
}

// writePCAPHeader writes the PCAP global header to the buffer
func (c *Capturer) writePCAPHeader() {
	c.pcapMutex.Lock()
	defer c.pcapMutex.Unlock()

	// Write proper PCAP global header (24 bytes)
	// Magic number (0xa1b2c3d4), version 2.4, timezone 0, sigfigs 0, snaplen 65535, linktype 1 (ethernet)
	header := []byte{
		0xd4, 0xc3, 0xb2, 0xa1, // Magic number (little-endian)
		0x02, 0x00,             // Version major
		0x04, 0x00,             // Version minor
		0x00, 0x00, 0x00, 0x00, // Timezone
		0x00, 0x00, 0x00, 0x00, // Sigfigs
		0xff, 0xff, 0x00, 0x00, // Snaplen (65535)
		0x01, 0x00, 0x00, 0x00, // Link type (Ethernet)
	}
	c.pcapBuffer.Write(header)
}

// writePCAPPacket writes a packet to the PCAP buffer with proper PCAP packet header
func (c *Capturer) writePCAPPacket(packet gopacket.Packet) {
	c.pcapMutex.Lock()
	defer c.pcapMutex.Unlock()

	data := packet.Data()
	ts := packet.Metadata().Timestamp

	// Write PCAP packet header (16 bytes)
	tsSec := uint32(ts.Unix())
	tsUsec := uint32(ts.Nanosecond() / 1000)
	inclLen := uint32(len(data))
	origLen := uint32(len(data))

	// Write header fields in little-endian
	c.pcapBuffer.Write([]byte{
		byte(tsSec), byte(tsSec >> 8), byte(tsSec >> 16), byte(tsSec >> 24),
		byte(tsUsec), byte(tsUsec >> 8), byte(tsUsec >> 16), byte(tsUsec >> 24),
		byte(inclLen), byte(inclLen >> 8), byte(inclLen >> 16), byte(inclLen >> 24),
		byte(origLen), byte(origLen >> 8), byte(origLen >> 16), byte(origLen >> 24),
	})

	// Write packet data
	c.pcapBuffer.Write(data)
}

// flushLoop periodically flushes PCAP data to the hub
func (c *Capturer) flushLoop(ctx context.Context) {
	ticker := time.NewTicker(FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.flushPCAP()
			return
		case <-ticker.C:
			c.flushPCAP()
		}
	}
}

// flushPCAP sends buffered PCAP data to the hub
func (c *Capturer) flushPCAP() {
	c.pcapMutex.Lock()
	if c.pcapBuffer.Len() == 0 {
		c.pcapMutex.Unlock()
		return
	}
	data := make([]byte, c.pcapBuffer.Len())
	copy(data, c.pcapBuffer.Bytes())
	c.pcapBuffer.Reset()
	c.pcapMutex.Unlock()

	if c.hubClient != nil {
		if err := c.hubClient.SendPCAPChunk(data); err != nil {
			log.Printf("Failed to send PCAP chunk: %v", err)
		}
	}
}

// onFlowComplete is called when a TCP flow is complete
func (c *Capturer) onFlowComplete(flow *protocol.Flow) {
	if c.hubClient != nil {
		if err := c.hubClient.SendFlow(flow); err != nil {
			log.Printf("Failed to send flow: %v", err)
		}
	}

	// Update stats
	c.statsMutex.Lock()
	if flow.HTTP != nil {
		c.stats.HTTPRequests++
	}
	if flow.TLS != nil {
		c.stats.TLSHandshakes++
	}
	c.statsMutex.Unlock()
}

// Stats returns current capture statistics
func (c *Capturer) Stats() CaptureStats {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()
	return c.stats
}

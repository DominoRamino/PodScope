package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/podscope/podscope/pkg/protocol"
)

// HubClient manages connection to the Hub via HTTP
type HubClient struct {
	hubURL    string
	agentInfo *protocol.AgentInfo
	client    *http.Client
	ctx       context.Context
	cancel    context.CancelFunc

	// Flow streaming
	flowChan chan *protocol.Flow
	flowWg   sync.WaitGroup

	// PCAP streaming
	pcapChan chan []byte
	pcapWg   sync.WaitGroup

	// Connection state
	connected bool
	connMutex sync.RWMutex

	// Capturer reference for BPF filter updates
	capturer       *Capturer
	lastBPFFilter  string
	bpfFilterMutex sync.RWMutex
}

// NewHubClient creates a new Hub client
func NewHubClient(address string, agentInfo *protocol.AgentInfo) *HubClient {
	ctx, cancel := context.WithCancel(context.Background())

	// Convert gRPC address to HTTP (port 9090 -> 8080)
	// The Hub runs HTTP on 8080 and gRPC on 9090
	hubURL := fmt.Sprintf("http://%s", address)
	// Replace port 9090 with 8080 for HTTP
	hubURL = hubURL[:len(hubURL)-4] + "8080"

	return &HubClient{
		hubURL:    hubURL,
		agentInfo: agentInfo,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		ctx:      ctx,
		cancel:   cancel,
		flowChan: make(chan *protocol.Flow, 1000),
		pcapChan: make(chan []byte, 100),
	}
}

// SetCapturer sets the capturer reference for BPF filter updates
func (c *HubClient) SetCapturer(capturer *Capturer) {
	c.capturer = capturer
}

// Connect establishes connection to the Hub
func (c *HubClient) Connect() error {
	// Test connection with health check
	resp, err := c.client.Get(c.hubURL + "/api/health")
	if err != nil {
		return fmt.Errorf("failed to connect to hub: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hub health check failed: %d", resp.StatusCode)
	}

	c.connMutex.Lock()
	c.connected = true
	c.connMutex.Unlock()

	log.Printf("Connected to Hub at %s", c.hubURL)

	// Register agent
	if err := c.registerAgent(); err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}

	// Start background workers
	c.startFlowStreamer()
	c.startPCAPStreamer()
	c.startHeartbeat()

	return nil
}

// registerAgent registers this agent with the Hub
func (c *HubClient) registerAgent() error {
	data, err := json.Marshal(c.agentInfo)
	if err != nil {
		return err
	}

	resp, err := c.client.Post(c.hubURL+"/api/agents", "application/json", bytes.NewReader(data))
	if err != nil {
		// Non-fatal for MVP - just log
		log.Printf("Agent registration request failed: %v", err)
	} else {
		resp.Body.Close()
	}

	log.Printf("Agent registered: %s (%s/%s)",
		c.agentInfo.ID, c.agentInfo.Namespace, c.agentInfo.PodName)
	return nil
}

// startFlowStreamer starts the flow streaming goroutine
func (c *HubClient) startFlowStreamer() {
	c.flowWg.Add(1)
	go func() {
		defer c.flowWg.Done()
		c.flowStreamLoop()
	}()
}

// flowStreamLoop handles flow streaming
func (c *HubClient) flowStreamLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case flow := <-c.flowChan:
			c.sendFlowToHub(flow)
		}
	}
}

// sendFlowToHub sends a flow event to the Hub via HTTP POST
func (c *HubClient) sendFlowToHub(flow *protocol.Flow) {
	c.connMutex.RLock()
	connected := c.connected
	c.connMutex.RUnlock()

	if !connected {
		log.Printf("Not connected, dropping flow: %s", flow.ID)
		return
	}

	data, err := json.Marshal(flow)
	if err != nil {
		log.Printf("Failed to marshal flow: %v", err)
		return
	}

	resp, err := c.client.Post(c.hubURL+"/api/flows", "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("Failed to send flow to hub: %v", err)
		return
	}
	resp.Body.Close()

	log.Printf("Flow: %s %s:%d -> %s:%d [%s]",
		flow.Protocol, flow.SrcIP, flow.SrcPort, flow.DstIP, flow.DstPort, flow.Status)
}

// startPCAPStreamer starts the PCAP streaming goroutine
func (c *HubClient) startPCAPStreamer() {
	c.pcapWg.Add(1)
	go func() {
		defer c.pcapWg.Done()
		c.pcapStreamLoop()
	}()
}

// pcapStreamLoop handles PCAP data streaming
func (c *HubClient) pcapStreamLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case data := <-c.pcapChan:
			c.sendPCAPToHub(data)
		}
	}
}

// sendPCAPToHub sends PCAP data to the Hub via HTTP POST
func (c *HubClient) sendPCAPToHub(data []byte) {
	c.connMutex.RLock()
	connected := c.connected
	c.connMutex.RUnlock()

	if !connected {
		return
	}

	req, err := http.NewRequest("POST", c.hubURL+"/api/pcap/upload", bytes.NewReader(data))
	if err != nil {
		log.Printf("Failed to create PCAP upload request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Agent-ID", c.agentInfo.ID)

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("Failed to send PCAP to hub: %v", err)
		return
	}
	resp.Body.Close()

	log.Printf("Sent %d bytes of PCAP data to Hub", len(data))
}

// startHeartbeat starts the heartbeat goroutine
func (c *HubClient) startHeartbeat() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				c.sendHeartbeat()
			}
		}
	}()
}

// sendHeartbeat sends a heartbeat to the Hub
func (c *HubClient) sendHeartbeat() {
	c.connMutex.RLock()
	connected := c.connected
	c.connMutex.RUnlock()

	if !connected {
		return
	}

	// GET request as heartbeat - check for BPF filter updates
	resp, err := c.client.Get(c.hubURL + "/api/health")
	if err != nil {
		log.Printf("Heartbeat failed: %v", err)
		return
	}
	defer resp.Body.Close()

	// Parse response to check for BPF filter updates
	var healthResp struct {
		Status    string `json:"status"`
		SessionID string `json:"sessionId"`
		BPFFilter string `json:"bpfFilter"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		log.Printf("Failed to parse heartbeat response: %v", err)
		return
	}

	// Check if BPF filter has changed (including empty string to reset)
	c.bpfFilterMutex.Lock()
	lastFilter := c.lastBPFFilter
	c.bpfFilterMutex.Unlock()

	if healthResp.BPFFilter != lastFilter {
		if healthResp.BPFFilter == "" {
			log.Printf("BPF filter cleared by hub - will reset to default on next capture")
		} else {
			log.Printf("BPF filter update detected from hub: %s", healthResp.BPFFilter)
		}

		// Apply the new filter if we have a capturer reference
		if c.capturer != nil {
			if err := c.capturer.UpdateBPFFilter(healthResp.BPFFilter); err != nil {
				log.Printf("Failed to update BPF filter: %v", err)
			} else {
				c.bpfFilterMutex.Lock()
				c.lastBPFFilter = healthResp.BPFFilter
				c.bpfFilterMutex.Unlock()
			}
		} else {
			log.Printf("WARNING: Cannot update BPF filter - no capturer reference")
		}
	}
}

// SendFlow queues a flow for sending to the Hub
func (c *HubClient) SendFlow(flow *protocol.Flow) error {
	select {
	case c.flowChan <- flow:
		return nil
	default:
		return fmt.Errorf("flow channel full")
	}
}

// SendPCAPChunk queues PCAP data for sending to the Hub
func (c *HubClient) SendPCAPChunk(data []byte) error {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	select {
	case c.pcapChan <- dataCopy:
		return nil
	default:
		return fmt.Errorf("pcap channel full")
	}
}

// Close closes the connection to the Hub
func (c *HubClient) Close() error {
	c.cancel()

	c.connMutex.Lock()
	c.connected = false
	c.connMutex.Unlock()

	// Wait for streamers to finish
	c.flowWg.Wait()
	c.pcapWg.Wait()

	return nil
}

// IsConnected returns whether the client is connected
func (c *HubClient) IsConnected() bool {
	c.connMutex.RLock()
	defer c.connMutex.RUnlock()
	return c.connected
}

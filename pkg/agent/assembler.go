package agent

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/uuid"
	"github.com/podscope/podscope/pkg/protocol"
)

const (
	// MaxBodySize is the maximum request/response body to capture
	MaxBodySize = 1024 // 1KB for MVP
	// FlowTimeout is how long to keep incomplete flows
	FlowTimeout = 30 * time.Second
)

// TCPAssembler reassembles TCP streams
type TCPAssembler struct {
	flows          map[string]*TCPFlow
	mutex          sync.RWMutex
	onFlowComplete func(*protocol.Flow)

	// Agent info for populating pod names
	agentPodName   string
	agentNamespace string
	agentPodIP     string

	// Hub info for agent traffic tagging
	hubIP string
}

// TCPFlow represents a TCP connection
type TCPFlow struct {
	ID          string
	SrcIP       string
	SrcPort     uint16
	DstIP       string
	DstPort     uint16
	StartTime   time.Time
	LastSeen    time.Time

	// State tracking
	SYNSeen     bool
	SYNACKSeen  bool
	FINSeen     bool
	RSTSeen     bool

	// Timing
	SYNTime     time.Time
	SYNACKTime  time.Time
	FirstDataTime time.Time

	// Data buffers
	ClientData  bytes.Buffer
	ServerData  bytes.Buffer

	// Counters
	PacketsSent   uint32
	PacketsRecv   uint32
	BytesSent     uint64
	BytesReceived uint64

	// Parsed data
	HTTP        *protocol.HTTPInfo
	TLS         *protocol.TLSInfo
	Protocol    protocol.Protocol
}

// NewTCPAssembler creates a new TCP stream assembler
func NewTCPAssembler(onComplete func(*protocol.Flow), agentInfo *protocol.AgentInfo) *TCPAssembler {
	a := &TCPAssembler{
		flows:          make(map[string]*TCPFlow),
		onFlowComplete: onComplete,
	}

	// Store agent info for populating pod names in flows
	if agentInfo != nil {
		a.agentPodName = agentInfo.PodName
		a.agentNamespace = agentInfo.Namespace
		a.agentPodIP = agentInfo.PodIP
	}

	// Start cleanup goroutine
	go a.cleanupLoop()

	return a
}

// SetHubIP sets the Hub IP for agent traffic tagging.
// This allows identifying flows that are agent->Hub communication.
func (a *TCPAssembler) SetHubIP(hubIP string) {
	a.hubIP = hubIP
}

// isAgentTraffic checks if a flow is agent-to-Hub communication.
// Returns true and the traffic type if this is agent traffic.
func (a *TCPAssembler) isAgentTraffic(flow *TCPFlow) (bool, string) {
	// Need both pod IP and hub IP to identify agent traffic
	if a.agentPodIP == "" || a.hubIP == "" {
		return false, ""
	}

	// Check if this is traffic from pod to hub on agent ports
	isFromPodToHub := flow.SrcIP == a.agentPodIP && flow.DstIP == a.hubIP
	isFromHubToPod := flow.SrcIP == a.hubIP && flow.DstIP == a.agentPodIP

	if !isFromPodToHub && !isFromHubToPod {
		return false, ""
	}

	// Check if it's on agent communication ports (8080 or 9090)
	isAgentPort := flow.DstPort == 8080 || flow.DstPort == 9090 ||
		flow.SrcPort == 8080 || flow.SrcPort == 9090

	if !isAgentPort {
		return false, ""
	}

	// Determine traffic type from HTTP path if available
	if flow.HTTP != nil && flow.HTTP.URL != "" {
		switch {
		case strings.HasPrefix(flow.HTTP.URL, "/api/health"):
			return true, "health"
		case strings.HasPrefix(flow.HTTP.URL, "/api/flows"):
			return true, "flow"
		case strings.HasPrefix(flow.HTTP.URL, "/api/pcap"):
			return true, "pcap"
		case strings.HasPrefix(flow.HTTP.URL, "/api/agents"):
			return true, "registration"
		}
	}

	// It's agent traffic but we couldn't determine the specific type
	return true, "unknown"
}

// flowKey generates a unique key for a TCP flow
func flowKey(srcIP, dstIP string, srcPort, dstPort uint16) string {
	// Normalize so A->B and B->A use same key
	if srcIP < dstIP || (srcIP == dstIP && srcPort < dstPort) {
		return fmt.Sprintf("%s:%d-%s:%d", srcIP, srcPort, dstIP, dstPort)
	}
	return fmt.Sprintf("%s:%d-%s:%d", dstIP, dstPort, srcIP, srcPort)
}

// ProcessPacket processes a TCP packet
func (a *TCPAssembler) ProcessPacket(srcIP, dstIP string, srcPort, dstPort uint16, tcp *layers.TCP, timestamp time.Time, appLayer gopacket.ApplicationLayer) {
	key := flowKey(srcIP, dstIP, srcPort, dstPort)

	a.mutex.Lock()
	flow, exists := a.flows[key]
	if !exists {
		flow = &TCPFlow{
			ID:        uuid.New().String()[:8],
			SrcIP:     srcIP,
			SrcPort:   srcPort,
			DstIP:     dstIP,
			DstPort:   dstPort,
			StartTime: timestamp,
			Protocol:  protocol.ProtocolTCP,
		}
		a.flows[key] = flow
	}
	flow.LastSeen = timestamp
	a.mutex.Unlock()

	// Track TCP state
	isFromClient := srcIP == flow.SrcIP && srcPort == flow.SrcPort

	if tcp.SYN && !tcp.ACK {
		flow.SYNSeen = true
		flow.SYNTime = timestamp
	}

	if tcp.SYN && tcp.ACK {
		flow.SYNACKSeen = true
		flow.SYNACKTime = timestamp
	}

	if tcp.FIN {
		flow.FINSeen = true
	}

	if tcp.RST {
		flow.RSTSeen = true
		a.completeFlow(key, flow)
		return
	}

	// Track payload
	if appLayer != nil && len(appLayer.Payload()) > 0 {
		payload := appLayer.Payload()

		if flow.FirstDataTime.IsZero() {
			flow.FirstDataTime = timestamp
		}

		if isFromClient {
			flow.PacketsSent++
			flow.BytesSent += uint64(len(payload))
			flow.ClientData.Write(payload)
		} else {
			flow.PacketsRecv++
			flow.BytesReceived += uint64(len(payload))
			flow.ServerData.Write(payload)
		}

		// Try to detect protocol from first data packet
		if flow.Protocol == protocol.ProtocolTCP {
			flow.Protocol = a.detectProtocol(payload, dstPort)
		}

		// Parse protocol-specific data
		a.parsePayload(flow)
	}

	// Complete flow on FIN
	if flow.FINSeen && flow.SYNACKSeen {
		a.completeFlow(key, flow)
	}
}

// detectProtocol tries to detect the application protocol
func (a *TCPAssembler) detectProtocol(payload []byte, dstPort uint16) protocol.Protocol {
	// Check for TLS ClientHello
	if len(payload) > 5 && payload[0] == 0x16 && payload[1] == 0x03 {
		return protocol.ProtocolTLS
	}

	// Check for HTTP
	if isHTTPMethod(payload) {
		return protocol.ProtocolHTTP
	}

	// Common HTTPS ports
	if dstPort == 443 || dstPort == 8443 {
		return protocol.ProtocolHTTPS
	}

	return protocol.ProtocolTCP
}

// isHTTPMethod checks if payload starts with an HTTP method
func isHTTPMethod(payload []byte) bool {
	methods := []string{"GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH ", "CONNECT "}
	for _, method := range methods {
		if bytes.HasPrefix(payload, []byte(method)) {
			return true
		}
	}
	// Check for HTTP response
	if bytes.HasPrefix(payload, []byte("HTTP/")) {
		return true
	}
	return false
}

// parsePayload parses application layer data
func (a *TCPAssembler) parsePayload(flow *TCPFlow) {
	switch flow.Protocol {
	case protocol.ProtocolHTTP:
		a.parseHTTP(flow)
	case protocol.ProtocolTLS, protocol.ProtocolHTTPS:
		a.parseTLS(flow)
	}
}

// parseHTTP parses HTTP request/response
func (a *TCPAssembler) parseHTTP(flow *TCPFlow) {
	// Parse request from client data (only if not already parsed)
	if flow.HTTP == nil && flow.ClientData.Len() > 0 {
		req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(flow.ClientData.Bytes())))
		if err == nil {
			flow.HTTP = &protocol.HTTPInfo{
				Method:         req.Method,
				URL:            req.URL.String(),
				Host:           req.Host,
				RequestHeaders: make(map[string]string),
			}

			// Copy headers
			for k, v := range req.Header {
				flow.HTTP.RequestHeaders[k] = strings.Join(v, ", ")
			}

			flow.Protocol = protocol.ProtocolHTTP
		}
	}

	// Parse response from server data (only if request parsed and response not yet parsed)
	if flow.HTTP != nil && flow.HTTP.StatusCode == 0 && flow.ServerData.Len() > 0 {
		resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(flow.ServerData.Bytes())), nil)
		if err == nil {
			flow.HTTP.StatusCode = resp.StatusCode
			flow.HTTP.StatusText = resp.Status
			flow.HTTP.ResponseHeaders = make(map[string]string)
			flow.HTTP.ContentType = resp.Header.Get("Content-Type")
			flow.HTTP.ContentLength = resp.ContentLength

			for k, v := range resp.Header {
				flow.HTTP.ResponseHeaders[k] = strings.Join(v, ", ")
			}
		}
	}
}

// parseTLS parses TLS ClientHello to extract SNI and cipher suites
func (a *TCPAssembler) parseTLS(flow *TCPFlow) {
	if flow.TLS != nil {
		return // Already parsed
	}

	data := flow.ClientData.Bytes()
	if len(data) < 6 {
		return
	}

	// Check for TLS record
	if data[0] != 0x16 { // Handshake
		return
	}

	flow.TLS = &protocol.TLSInfo{
		Encrypted: true,
	}

	// Parse TLS version
	switch {
	case data[1] == 0x03 && data[2] == 0x03:
		flow.TLS.Version = "TLS 1.2"
	case data[1] == 0x03 && data[2] == 0x01:
		flow.TLS.Version = "TLS 1.0"
	case data[1] == 0x03 && data[2] == 0x02:
		flow.TLS.Version = "TLS 1.1"
	default:
		flow.TLS.Version = fmt.Sprintf("TLS %d.%d", data[1], data[2])
	}

	// Extract SNI and cipher suites from ClientHello
	tlsInfo := extractTLSClientHelloInfo(data)
	if tlsInfo.SNI != "" {
		flow.TLS.SNI = tlsInfo.SNI
	}
	if len(tlsInfo.CipherSuites) > 0 {
		flow.TLS.CipherSuites = tlsInfo.CipherSuites
	}

	if flow.Protocol == protocol.ProtocolTLS {
		flow.Protocol = protocol.ProtocolHTTPS
	}
}

// tlsClientHelloInfo holds parsed TLS ClientHello data
type tlsClientHelloInfo struct {
	SNI          string
	CipherSuites []uint16
}

// extractTLSClientHelloInfo extracts SNI and cipher suites from TLS ClientHello
func extractTLSClientHelloInfo(data []byte) tlsClientHelloInfo {
	result := tlsClientHelloInfo{}

	// Skip TLS record header (5 bytes) and handshake header (4 bytes)
	if len(data) < 43 {
		return result
	}

	// ClientHello starts at offset 5
	// Skip: handshake type (1), length (3), version (2), random (32)
	offset := 5 + 1 + 3 + 2 + 32

	if len(data) <= offset {
		return result
	}

	// Session ID length
	sessionIDLen := int(data[offset])
	offset += 1 + sessionIDLen

	if len(data) <= offset+2 {
		return result
	}

	// Cipher suites length (2 bytes, big-endian)
	cipherSuitesLen := int(data[offset])<<8 | int(data[offset+1])
	offset += 2

	// Extract cipher suites before skipping
	// Each cipher suite is 2 bytes (big-endian)
	if len(data) >= offset+cipherSuitesLen {
		numCipherSuites := cipherSuitesLen / 2
		result.CipherSuites = make([]uint16, 0, numCipherSuites)
		for i := 0; i < cipherSuitesLen; i += 2 {
			cipherID := uint16(data[offset+i])<<8 | uint16(data[offset+i+1])
			result.CipherSuites = append(result.CipherSuites, cipherID)
		}
	}
	offset += cipherSuitesLen

	if len(data) <= offset+1 {
		return result
	}

	// Compression methods length
	compMethodsLen := int(data[offset])
	offset += 1 + compMethodsLen

	if len(data) <= offset+2 {
		return result
	}

	// Extensions length
	extensionsLen := int(data[offset])<<8 | int(data[offset+1])
	offset += 2

	end := offset + extensionsLen
	if end > len(data) {
		end = len(data)
	}

	// Parse extensions
	for offset < end-4 {
		extType := int(data[offset])<<8 | int(data[offset+1])
		extLen := int(data[offset+2])<<8 | int(data[offset+3])
		offset += 4

		if extType == 0 { // Server Name extension
			if offset+2 < len(data) {
				// Skip list length (2 bytes)
				listLen := int(data[offset])<<8 | int(data[offset+1])
				offset += 2

				if offset+3 < len(data) && listLen > 0 {
					nameType := data[offset]
					nameLen := int(data[offset+1])<<8 | int(data[offset+2])
					offset += 3

					if nameType == 0 && offset+nameLen <= len(data) {
						result.SNI = string(data[offset : offset+nameLen])
					}
				}
			}
		}

		offset += extLen
	}

	return result
}

// extractSNI extracts Server Name Indication from TLS ClientHello (legacy wrapper)
func extractSNI(data []byte) string {
	return extractTLSClientHelloInfo(data).SNI
}

// completeFlow marks a flow as complete and sends it
func (a *TCPAssembler) completeFlow(key string, flow *TCPFlow) {
	a.mutex.Lock()
	delete(a.flows, key)
	a.mutex.Unlock()

	// Build final Flow struct
	f := &protocol.Flow{
		ID:            flow.ID,
		Timestamp:     flow.StartTime,
		Duration:      flow.LastSeen.Sub(flow.StartTime).Milliseconds(),
		SrcIP:         flow.SrcIP,
		SrcPort:       flow.SrcPort,
		DstIP:         flow.DstIP,
		DstPort:       flow.DstPort,
		Protocol:      flow.Protocol,
		BytesSent:     flow.BytesSent,
		BytesReceived: flow.BytesReceived,
		PacketsSent:   flow.PacketsSent,
		PacketsRecv:   flow.PacketsRecv,
		HTTP:          flow.HTTP,
		TLS:           flow.TLS,
	}

	// Populate pod names based on agent info
	// The agent is injected into a specific pod, so we know its IP
	// All traffic on the pod's network interface involves this pod
	log.Printf("DEBUG: completeFlow - agentPodName=%q agentPodIP=%q flow.SrcIP=%s flow.DstIP=%s flow.SrcPort=%d flow.DstPort=%d",
		a.agentPodName, a.agentPodIP, flow.SrcIP, flow.DstIP, flow.SrcPort, flow.DstPort)

	if a.agentPodName != "" && a.agentNamespace != "" {
		// Since the agent runs in the target pod's network namespace,
		// all captured traffic is to/from this pod
		// Try to match IPs, but also use the pod info as a fallback
		if a.agentPodIP != "" {
			if flow.SrcIP == a.agentPodIP {
				f.SrcPod = a.agentPodName
				f.SrcNamespace = a.agentNamespace
				log.Printf("DEBUG: Matched SrcIP=%s to pod %s/%s", flow.SrcIP, a.agentNamespace, a.agentPodName)
			}
			if flow.DstIP == a.agentPodIP {
				f.DstPod = a.agentPodName
				f.DstNamespace = a.agentNamespace
				log.Printf("DEBUG: Matched DstIP=%s to pod %s/%s", flow.DstIP, a.agentNamespace, a.agentPodName)
			}
		}
		// If neither IP matched but we have agent info, the source is likely our pod
		// (since we're capturing on the pod's interface, outgoing traffic has our IP as source)
		if f.SrcPod == "" && f.DstPod == "" && a.agentPodName != "" {
			// For outgoing connections (high src port), source is our pod
			// For incoming connections (listening on low port), dest is our pod
			if flow.SrcPort > 1024 {
				f.SrcPod = a.agentPodName
				f.SrcNamespace = a.agentNamespace
				log.Printf("DEBUG: Fallback - assigned SrcPod=%s/%s (high src port %d)", a.agentNamespace, a.agentPodName, flow.SrcPort)
			} else {
				f.DstPod = a.agentPodName
				f.DstNamespace = a.agentNamespace
				log.Printf("DEBUG: Fallback - assigned DstPod=%s/%s (low src port %d)", a.agentNamespace, a.agentPodName, flow.SrcPort)
			}
		}
	}

	log.Printf("DEBUG: Final flow - SrcPod=%q DstPod=%q", f.SrcPod, f.DstPod)

	// Tag agent traffic for filtering
	if isAgent, trafficType := a.isAgentTraffic(flow); isAgent {
		f.IsAgentTraffic = true
		f.AgentTrafficType = trafficType
		log.Printf("DEBUG: Tagged as agent traffic - type=%s", trafficType)
	}

	// Calculate timing
	if flow.SYNSeen && flow.SYNACKSeen {
		f.TCPHandshakeMs = flow.SYNACKTime.Sub(flow.SYNTime).Seconds() * 1000
	}

	// Set status
	if flow.RSTSeen {
		f.Status = protocol.StatusReset
	} else if flow.FINSeen {
		f.Status = protocol.StatusClosed
	} else {
		f.Status = protocol.StatusTimeout
	}

	// Notify callback
	if a.onFlowComplete != nil {
		a.onFlowComplete(f)
	}
}

// cleanupLoop removes stale flows
func (a *TCPAssembler) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		a.mutex.Lock()
		now := time.Now()
		for key, flow := range a.flows {
			if now.Sub(flow.LastSeen) > FlowTimeout {
				delete(a.flows, key)
				// Complete the flow with timeout
				go a.completeFlow(key, flow)
			}
		}
		a.mutex.Unlock()
	}
}

package protocol

import (
	"time"
)

// Protocol represents the detected protocol
type Protocol string

const (
	ProtocolTCP   Protocol = "TCP"
	ProtocolHTTP  Protocol = "HTTP"
	ProtocolHTTPS Protocol = "HTTPS"
	ProtocolTLS   Protocol = "TLS"
)

// FlowStatus represents the status of a flow
type FlowStatus string

const (
	StatusOpen       FlowStatus = "OPEN"
	StatusClosed     FlowStatus = "CLOSED"
	StatusReset      FlowStatus = "RESET"
	StatusTimeout    FlowStatus = "TIMEOUT"
)

// Flow represents a captured network flow
type Flow struct {
	ID           string     `json:"id"`
	Timestamp    time.Time  `json:"timestamp"`
	Duration     int64      `json:"duration"` // milliseconds

	// Source info
	SrcIP        string     `json:"srcIp"`
	SrcPort      uint16     `json:"srcPort"`
	SrcPod       string     `json:"srcPod,omitempty"`
	SrcNamespace string     `json:"srcNamespace,omitempty"`

	// Destination info
	DstIP        string     `json:"dstIp"`
	DstPort      uint16     `json:"dstPort"`
	DstPod       string     `json:"dstPod,omitempty"`
	DstNamespace string     `json:"dstNamespace,omitempty"`
	DstService   string     `json:"dstService,omitempty"`

	// Protocol info
	Protocol     Protocol   `json:"protocol"`
	Status       FlowStatus `json:"status"`

	// Size metrics
	BytesSent     uint64     `json:"bytesSent"`
	BytesReceived uint64     `json:"bytesReceived"`
	PacketsSent   uint32     `json:"packetsSent"`
	PacketsRecv   uint32     `json:"packetsReceived"`

	// Timing
	TCPHandshakeMs  float64 `json:"tcpHandshakeMs,omitempty"`
	TLSHandshakeMs  float64 `json:"tlsHandshakeMs,omitempty"`
	TimeToFirstByte float64 `json:"ttfbMs,omitempty"`

	// HTTP info (plaintext only)
	HTTP *HTTPInfo `json:"http,omitempty"`

	// TLS info
	TLS *TLSInfo `json:"tls,omitempty"`

	// Agent traffic identification (for filtering noise from captures)
	IsAgentTraffic   bool   `json:"isAgentTraffic,omitempty"`
	AgentTrafficType string `json:"agentTrafficType,omitempty"` // "health", "flow", "pcap", "registration"
}

// HTTPInfo contains HTTP request/response information
type HTTPInfo struct {
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Host            string            `json:"host"`
	StatusCode      int               `json:"statusCode"`
	StatusText      string            `json:"statusText"`
	RequestHeaders  map[string]string `json:"requestHeaders,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`
	RequestBody     string            `json:"requestBody,omitempty"`  // Truncated
	ResponseBody    string            `json:"responseBody,omitempty"` // Truncated
	ContentType     string            `json:"contentType,omitempty"`
	ContentLength   int64             `json:"contentLength,omitempty"`
}

// TLSInfo contains TLS handshake information
type TLSInfo struct {
	Version       string   `json:"version"`
	SNI           string   `json:"sni"`
	CipherSuite   string   `json:"cipherSuite"`
	ALPN          []string `json:"alpn,omitempty"`
	Encrypted     bool     `json:"encrypted"`
}

// AgentInfo identifies a capture agent
type AgentInfo struct {
	ID        string `json:"id"`
	PodName   string `json:"podName"`
	Namespace string `json:"namespace"`
	PodIP     string `json:"podIp"`
	NodeName  string `json:"nodeName"`
}

// FlowEvent is sent from agent to hub
type FlowEvent struct {
	Agent *AgentInfo `json:"agent"`
	Flow  *Flow      `json:"flow"`
}

// PCAPChunk is raw PCAP data sent from agent to hub
type PCAPChunk struct {
	AgentID   string `json:"agentId"`
	Timestamp int64  `json:"timestamp"`
	Data      []byte `json:"data"`
}

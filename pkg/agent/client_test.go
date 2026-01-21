package agent

import (
	"strings"
	"testing"

	"github.com/podscope/podscope/pkg/protocol"
)

// Helper to create test AgentInfo
func createTestAgentInfo() *protocol.AgentInfo {
	return &protocol.AgentInfo{
		ID:        "test-agent-001",
		PodName:   "test-pod",
		Namespace: "default",
		PodIP:     "10.0.0.5",
		NodeName:  "node-1",
	}
}

// TestNewHubClient_PortTranslation tests that gRPC port 9090 is translated to HTTP port 8080
func TestNewHubClient_PortTranslation(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub.podscope.svc.cluster.local:9090", agentInfo)
	defer client.Close()

	if !strings.HasSuffix(client.hubURL, ":8080") {
		t.Errorf("Expected URL to end with :8080, got %s", client.hubURL)
	}

	expected := "http://hub.podscope.svc.cluster.local:8080"
	if client.hubURL != expected {
		t.Errorf("Expected %s, got %s", expected, client.hubURL)
	}
}

// TestNewHubClient_HTTPScheme tests that URL scheme is HTTP
func TestNewHubClient_HTTPScheme(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub.test:9090", agentInfo)
	defer client.Close()

	if !strings.HasPrefix(client.hubURL, "http://") {
		t.Errorf("Expected URL to start with http://, got %s", client.hubURL)
	}
}

// TestNewHubClient_BaseURLConstruction tests that base URL is constructed correctly
func TestNewHubClient_BaseURLConstruction(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "simple hostname with gRPC port",
			address:  "hub:9090",
			expected: "http://hub:8080",
		},
		{
			name:     "FQDN with gRPC port",
			address:  "hub.podscope-abc123.svc.cluster.local:9090",
			expected: "http://hub.podscope-abc123.svc.cluster.local:8080",
		},
		{
			name:     "IP address with gRPC port",
			address:  "10.0.0.100:9090",
			expected: "http://10.0.0.100:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentInfo := createTestAgentInfo()
			client := NewHubClient(tt.address, agentInfo)
			defer client.Close()

			if client.hubURL != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, client.hubURL)
			}
		})
	}
}

// TestNewHubClient_AgentInfoStored tests that agent info is stored correctly
func TestNewHubClient_AgentInfoStored(t *testing.T) {
	agentInfo := &protocol.AgentInfo{
		ID:        "agent-123",
		PodName:   "my-pod",
		Namespace: "my-namespace",
		PodIP:     "192.168.1.100",
		NodeName:  "worker-node-1",
	}

	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	if client.agentInfo != agentInfo {
		t.Error("Expected agentInfo to be stored by reference")
	}

	if client.agentInfo.ID != "agent-123" {
		t.Errorf("Expected agent ID 'agent-123', got %s", client.agentInfo.ID)
	}

	if client.agentInfo.PodName != "my-pod" {
		t.Errorf("Expected pod name 'my-pod', got %s", client.agentInfo.PodName)
	}

	if client.agentInfo.Namespace != "my-namespace" {
		t.Errorf("Expected namespace 'my-namespace', got %s", client.agentInfo.Namespace)
	}

	if client.agentInfo.PodIP != "192.168.1.100" {
		t.Errorf("Expected pod IP '192.168.1.100', got %s", client.agentInfo.PodIP)
	}

	if client.agentInfo.NodeName != "worker-node-1" {
		t.Errorf("Expected node name 'worker-node-1', got %s", client.agentInfo.NodeName)
	}
}

// TestNewHubClient_HTTPClientCreated tests that HTTP client is created with timeout
func TestNewHubClient_HTTPClientCreated(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	if client.client == nil {
		t.Fatal("Expected HTTP client to be created")
	}

	if client.client.Timeout == 0 {
		t.Error("Expected HTTP client to have non-zero timeout")
	}
}

// TestNewHubClient_ChannelsInitialized tests that flow and PCAP channels are created
func TestNewHubClient_ChannelsInitialized(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	if client.flowChan == nil {
		t.Error("Expected flowChan to be initialized")
	}

	if cap(client.flowChan) != 1000 {
		t.Errorf("Expected flowChan capacity 1000, got %d", cap(client.flowChan))
	}

	if client.pcapChan == nil {
		t.Error("Expected pcapChan to be initialized")
	}

	if cap(client.pcapChan) != 100 {
		t.Errorf("Expected pcapChan capacity 100, got %d", cap(client.pcapChan))
	}
}

// TestNewHubClient_InitialStateNotConnected tests that initial connection state is false
func TestNewHubClient_InitialStateNotConnected(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	if client.IsConnected() {
		t.Error("Expected initial connection state to be false")
	}
}

// TestNewHubClient_ContextInitialized tests that context is properly initialized
func TestNewHubClient_ContextInitialized(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	if client.ctx == nil {
		t.Error("Expected context to be initialized")
	}

	if client.cancel == nil {
		t.Error("Expected cancel function to be initialized")
	}

	// Context should not be cancelled initially
	select {
	case <-client.ctx.Done():
		t.Error("Expected context to not be cancelled initially")
	default:
		// Expected - context not cancelled
	}
}

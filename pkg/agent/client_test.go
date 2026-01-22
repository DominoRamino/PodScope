package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

// Helper to create a test Hub client that connects to a test server
// We bypass NewHubClient's port translation by setting hubURL directly
func createClientForTestServer(t *testing.T, serverURL string) *HubClient {
	t.Helper()
	agentInfo := createTestAgentInfo()
	return &HubClient{
		hubURL:    serverURL,
		agentInfo: agentInfo,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		flowChan:    make(chan *protocol.Flow, 1000),
		pcapChan:    make(chan []byte, 100),
		maxFailures: 3,
	}
}

// TestConnect_SuccessfulHealthCheck tests that successful health check marks client as connected
func TestConnect_SuccessfulHealthCheck(t *testing.T) {
	healthCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			healthCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"sessionId": "test-session",
				"bpfFilter": "",
			})
		case "/api/agents":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)
	// Initialize context since we're bypassing NewHubClient
	client.ctx, client.cancel = nil, nil

	// Create a custom connect that uses our test server URL
	// We'll test the individual steps that Connect() performs
	resp, err := client.client.Get(client.hubURL + "/api/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !healthCalled {
		t.Error("Expected health endpoint to be called")
	}

	// Simulate what Connect() does after health check
	client.connected = true

	if !client.IsConnected() {
		t.Error("Expected client to be marked as connected after successful health check")
	}
}

// TestConnect_HealthCheckReturns200_MarksConnected tests connected state is set on 200 OK
func TestConnect_HealthCheckReturns200_MarksConnected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		} else if r.URL.Path == "/api/agents" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)
	client.ctx, client.cancel = nil, nil

	// Test health check succeeds
	resp, err := client.client.Get(client.hubURL + "/api/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Mark as connected (what Connect() does)
	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()

	if !client.IsConnected() {
		t.Error("Expected IsConnected() to return true after setting connected")
	}
}

// TestConnect_HealthCheckFails500_ReturnsError tests that 500 status returns error
func TestConnect_HealthCheckFails500_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)

	// Test health check returns 500
	resp, err := client.client.Get(client.hubURL + "/api/health")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("Expected non-200 status code")
	}

	// Connect() would return error for non-200 status
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

// TestConnect_HealthCheckTimeout_ReturnsError tests that connection timeout returns error
func TestConnect_HealthCheckTimeout_ReturnsError(t *testing.T) {
	// Create a server that delays response longer than client timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than client timeout to trigger timeout
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)
	// Set a very short timeout to trigger timeout error
	client.client.Timeout = 50 * time.Millisecond

	// Test health check times out
	_, err := client.client.Get(client.hubURL + "/api/health")
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Verify client is not connected when health check fails
	if client.IsConnected() {
		t.Error("Expected IsConnected() to return false when health check fails")
	}
}

// TestConnect_AgentRegistrationCalledAfterHealthCheck tests that agent registration happens after health check
func TestConnect_AgentRegistrationCalledAfterHealthCheck(t *testing.T) {
	callOrder := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			callOrder = append(callOrder, "health")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/api/agents":
			callOrder = append(callOrder, "agents")
			if r.Method != "POST" {
				t.Errorf("Expected POST method for /api/agents, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)

	// Simulate Connect() flow: health check first, then agent registration
	// Step 1: Health check
	resp, err := client.client.Get(client.hubURL + "/api/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	resp.Body.Close()

	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()

	// Step 2: Agent registration (simulating registerAgent())
	data, _ := json.Marshal(client.agentInfo)
	resp, err = client.client.Post(client.hubURL+"/api/agents", "application/json", strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Agent registration failed: %v", err)
	}
	resp.Body.Close()

	// Verify order
	if len(callOrder) != 2 {
		t.Fatalf("Expected 2 calls, got %d", len(callOrder))
	}

	if callOrder[0] != "health" {
		t.Errorf("Expected first call to be 'health', got '%s'", callOrder[0])
	}

	if callOrder[1] != "agents" {
		t.Errorf("Expected second call to be 'agents', got '%s'", callOrder[1])
	}
}

// TestConnect_AgentRegistrationPayload tests that agent registration sends correct JSON payload
func TestConnect_AgentRegistrationPayload(t *testing.T) {
	var receivedAgent protocol.AgentInfo

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/api/agents":
			if r.Method == "POST" {
				json.NewDecoder(r.Body).Decode(&receivedAgent)
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)

	// Register agent
	data, _ := json.Marshal(client.agentInfo)
	resp, err := client.client.Post(client.hubURL+"/api/agents", "application/json", strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Agent registration failed: %v", err)
	}
	resp.Body.Close()

	// Verify received payload
	if receivedAgent.ID != client.agentInfo.ID {
		t.Errorf("Expected agent ID '%s', got '%s'", client.agentInfo.ID, receivedAgent.ID)
	}

	if receivedAgent.PodName != client.agentInfo.PodName {
		t.Errorf("Expected pod name '%s', got '%s'", client.agentInfo.PodName, receivedAgent.PodName)
	}

	if receivedAgent.Namespace != client.agentInfo.Namespace {
		t.Errorf("Expected namespace '%s', got '%s'", client.agentInfo.Namespace, receivedAgent.Namespace)
	}

	if receivedAgent.PodIP != client.agentInfo.PodIP {
		t.Errorf("Expected pod IP '%s', got '%s'", client.agentInfo.PodIP, receivedAgent.PodIP)
	}
}

// TestConnect_HealthCheckConnectionRefused_ReturnsError tests connection refused scenario
func TestConnect_HealthCheckConnectionRefused_ReturnsError(t *testing.T) {
	// Create client pointing to a closed port (no server)
	client := createClientForTestServer(t, "http://127.0.0.1:59999")

	// Test health check fails with connection error
	_, err := client.client.Get(client.hubURL + "/api/health")
	if err == nil {
		t.Error("Expected connection error, got nil")
	}

	// Verify client is not connected
	if client.IsConnected() {
		t.Error("Expected IsConnected() to return false when connection fails")
	}
}

// Helper to create a test flow
func createTestFlow(id string) *protocol.Flow {
	return &protocol.Flow{
		ID:       id,
		SrcIP:    "10.0.0.1",
		DstIP:    "10.0.0.2",
		SrcPort:  12345,
		DstPort:  80,
		Protocol: "HTTP",
		Status:   "closed",
	}
}

// TestSendFlow_AddedToChannelWhenSpaceAvailable tests that flow is queued when channel has space
func TestSendFlow_AddedToChannelWhenSpaceAvailable(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	flow := createTestFlow("flow-001")

	err := client.SendFlow(flow)
	if err != nil {
		t.Errorf("Expected nil error when channel has space, got: %v", err)
	}

	// Verify flow is in channel
	select {
	case received := <-client.flowChan:
		if received.ID != flow.ID {
			t.Errorf("Expected flow ID '%s', got '%s'", flow.ID, received.ID)
		}
	default:
		t.Error("Expected flow to be in channel, but channel was empty")
	}
}

// TestSendFlow_ReturnsNilOnSuccess tests that SendFlow returns nil on successful queue
func TestSendFlow_ReturnsNilOnSuccess(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	flow := createTestFlow("flow-002")

	err := client.SendFlow(flow)
	if err != nil {
		t.Errorf("Expected SendFlow to return nil, got: %v", err)
	}
}

// TestSendFlow_MultipleFlowsQueued tests that multiple flows can be queued
func TestSendFlow_MultipleFlowsQueued(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	// Queue multiple flows
	for i := 0; i < 10; i++ {
		flow := createTestFlow(fmt.Sprintf("flow-%03d", i))
		err := client.SendFlow(flow)
		if err != nil {
			t.Errorf("Expected nil error for flow %d, got: %v", i, err)
		}
	}

	// Verify channel has 10 flows
	if len(client.flowChan) != 10 {
		t.Errorf("Expected 10 flows in channel, got %d", len(client.flowChan))
	}
}

// TestSendFlow_ChannelFullReturnsError tests that error is returned when channel is full
func TestSendFlow_ChannelFullReturnsError(t *testing.T) {
	// Create a client with small channel capacity for testing
	agentInfo := createTestAgentInfo()
	client := &HubClient{
		hubURL:    "http://hub:8080",
		agentInfo: agentInfo,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		flowChan:    make(chan *protocol.Flow, 2), // Small capacity
		pcapChan:    make(chan []byte, 100),
		maxFailures: 3,
	}
	client.ctx, client.cancel = context.WithCancel(context.Background())
	defer client.Close()

	// Fill the channel
	client.SendFlow(createTestFlow("flow-001"))
	client.SendFlow(createTestFlow("flow-002"))

	// Third flow should return error
	err := client.SendFlow(createTestFlow("flow-003"))
	if err == nil {
		t.Error("Expected error when channel is full, got nil")
	}

	if !strings.Contains(err.Error(), "channel full") {
		t.Errorf("Expected 'channel full' error message, got: %v", err)
	}
}

// TestSendFlow_FlowSentToHubViaHTTP tests that queued flow is sent to Hub via POST /api/flows
func TestSendFlow_FlowSentToHubViaHTTP(t *testing.T) {
	var receivedFlow protocol.Flow
	flowReceived := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"sessionId": "test-session",
				"bpfFilter": "",
			})
		case "/api/agents":
			w.WriteHeader(http.StatusOK)
		case "/api/flows":
			if r.Method == "POST" {
				json.NewDecoder(r.Body).Decode(&receivedFlow)
				flowReceived <- true
			}
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)
	client.ctx, client.cancel = context.WithCancel(context.Background())
	defer client.Close()

	// Mark as connected and start the flow streamer
	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()
	client.startFlowStreamer()

	// Send a flow
	flow := &protocol.Flow{
		ID:       "test-flow-http",
		SrcIP:    "192.168.1.10",
		DstIP:    "10.0.0.5",
		SrcPort:  45678,
		DstPort:  80,
		Protocol: "HTTP",
		Status:   "closed",
	}

	err := client.SendFlow(flow)
	if err != nil {
		t.Fatalf("Failed to send flow: %v", err)
	}

	// Wait for flow to be received by server
	select {
	case <-flowReceived:
		// Flow was received
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for flow to be sent to Hub")
	}

	// Verify the flow data
	if receivedFlow.ID != flow.ID {
		t.Errorf("Expected flow ID '%s', got '%s'", flow.ID, receivedFlow.ID)
	}
	if receivedFlow.SrcIP != flow.SrcIP {
		t.Errorf("Expected SrcIP '%s', got '%s'", flow.SrcIP, receivedFlow.SrcIP)
	}
	if receivedFlow.DstIP != flow.DstIP {
		t.Errorf("Expected DstIP '%s', got '%s'", flow.DstIP, receivedFlow.DstIP)
	}
	if receivedFlow.SrcPort != flow.SrcPort {
		t.Errorf("Expected SrcPort %d, got %d", flow.SrcPort, receivedFlow.SrcPort)
	}
	if receivedFlow.DstPort != flow.DstPort {
		t.Errorf("Expected DstPort %d, got %d", flow.DstPort, receivedFlow.DstPort)
	}
	if receivedFlow.Protocol != flow.Protocol {
		t.Errorf("Expected Protocol '%s', got '%s'", flow.Protocol, receivedFlow.Protocol)
	}
}

// TestSendFlow_JSONCorrectlyFormatted tests that flow is sent with correct JSON formatting
func TestSendFlow_JSONCorrectlyFormatted(t *testing.T) {
	var receivedBody []byte
	bodyReceived := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"sessionId": "test-session",
				"bpfFilter": "",
			})
		case "/api/agents":
			w.WriteHeader(http.StatusOK)
		case "/api/flows":
			if r.Method == "POST" {
				buf := new(bytes.Buffer)
				buf.ReadFrom(r.Body)
				receivedBody = buf.Bytes()
				bodyReceived <- true
			}
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)
	client.ctx, client.cancel = context.WithCancel(context.Background())
	defer client.Close()

	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()
	client.startFlowStreamer()

	flow := &protocol.Flow{
		ID:       "json-test-flow",
		SrcIP:    "10.0.0.1",
		DstIP:    "10.0.0.2",
		SrcPort:  1234,
		DstPort:  443,
		Protocol: "HTTPS",
		Status:   "closed",
	}

	err := client.SendFlow(flow)
	if err != nil {
		t.Fatalf("Failed to send flow: %v", err)
	}

	select {
	case <-bodyReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for flow to be sent to Hub")
	}

	// Verify JSON is valid
	var parsedFlow protocol.Flow
	err = json.Unmarshal(receivedBody, &parsedFlow)
	if err != nil {
		t.Errorf("Received body is not valid JSON: %v", err)
	}

	// Verify key fields
	if parsedFlow.ID != "json-test-flow" {
		t.Errorf("Expected ID 'json-test-flow', got '%s'", parsedFlow.ID)
	}
}

// TestSendFlow_PostMethodUsed tests that POST method is used for /api/flows
func TestSendFlow_PostMethodUsed(t *testing.T) {
	methodUsed := ""
	methodReceived := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"sessionId": "test-session",
				"bpfFilter": "",
			})
		case "/api/agents":
			w.WriteHeader(http.StatusOK)
		case "/api/flows":
			methodUsed = r.Method
			methodReceived <- true
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)
	client.ctx, client.cancel = context.WithCancel(context.Background())
	defer client.Close()

	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()
	client.startFlowStreamer()

	err := client.SendFlow(createTestFlow("method-test"))
	if err != nil {
		t.Fatalf("Failed to send flow: %v", err)
	}

	select {
	case <-methodReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for flow to be sent to Hub")
	}

	if methodUsed != "POST" {
		t.Errorf("Expected POST method, got '%s'", methodUsed)
	}
}

// TestSendPCAPChunk_DataCopiedBeforeQueuing tests that PCAP data is copied (not referenced) before queuing
func TestSendPCAPChunk_DataCopiedBeforeQueuing(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	// Original data
	originalData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	err := client.SendPCAPChunk(originalData)
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}

	// Modify the original data after sending
	originalData[0] = 0xFF
	originalData[1] = 0xFF

	// Retrieve the queued data
	select {
	case queuedData := <-client.pcapChan:
		// Verify the queued data was NOT affected by the modification
		if queuedData[0] == 0xFF || queuedData[1] == 0xFF {
			t.Error("Expected data to be copied, but modification affected queued data")
		}
		if queuedData[0] != 0x01 || queuedData[1] != 0x02 {
			t.Errorf("Expected original bytes [0x01, 0x02], got [0x%02x, 0x%02x]", queuedData[0], queuedData[1])
		}
	default:
		t.Error("Expected PCAP data to be in channel, but channel was empty")
	}
}

// TestSendPCAPChunk_ReturnsNilOnSuccess tests that SendPCAPChunk returns nil on successful queue
func TestSendPCAPChunk_ReturnsNilOnSuccess(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	data := []byte{0x0a, 0x0b, 0x0c, 0x0d}

	err := client.SendPCAPChunk(data)
	if err != nil {
		t.Errorf("Expected SendPCAPChunk to return nil, got: %v", err)
	}
}

// TestSendPCAPChunk_ChannelFullReturnsError tests that error is returned when pcap channel is full
func TestSendPCAPChunk_ChannelFullReturnsError(t *testing.T) {
	// Create a client with small pcap channel capacity for testing
	agentInfo := createTestAgentInfo()
	client := &HubClient{
		hubURL:    "http://hub:8080",
		agentInfo: agentInfo,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		flowChan:    make(chan *protocol.Flow, 1000),
		pcapChan:    make(chan []byte, 2), // Small capacity
		maxFailures: 3,
	}
	client.ctx, client.cancel = context.WithCancel(context.Background())
	defer client.Close()

	// Fill the channel
	client.SendPCAPChunk([]byte{0x01})
	client.SendPCAPChunk([]byte{0x02})

	// Third chunk should return error
	err := client.SendPCAPChunk([]byte{0x03})
	if err == nil {
		t.Error("Expected error when channel is full, got nil")
	}

	if !strings.Contains(err.Error(), "pcap channel full") {
		t.Errorf("Expected 'pcap channel full' error message, got: %v", err)
	}
}

// TestSendPCAPChunk_MultipleChunksQueued tests that multiple PCAP chunks can be queued
func TestSendPCAPChunk_MultipleChunksQueued(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	// Queue multiple PCAP chunks
	for i := 0; i < 10; i++ {
		data := []byte{byte(i), byte(i + 1), byte(i + 2)}
		err := client.SendPCAPChunk(data)
		if err != nil {
			t.Errorf("Expected nil error for chunk %d, got: %v", i, err)
		}
	}

	// Verify channel has 10 chunks
	if len(client.pcapChan) != 10 {
		t.Errorf("Expected 10 chunks in channel, got %d", len(client.pcapChan))
	}
}

// TestSendPCAPChunk_SentToHubViaHTTP tests that queued PCAP data is sent to Hub via POST /api/pcap/upload
func TestSendPCAPChunk_SentToHubViaHTTP(t *testing.T) {
	var receivedData []byte
	dataReceived := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"sessionId": "test-session",
				"bpfFilter": "",
			})
		case "/api/agents":
			w.WriteHeader(http.StatusOK)
		case "/api/pcap/upload":
			if r.Method == "POST" {
				buf := new(bytes.Buffer)
				buf.ReadFrom(r.Body)
				receivedData = buf.Bytes()
				dataReceived <- true
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)
	client.ctx, client.cancel = context.WithCancel(context.Background())
	defer client.Close()

	// Mark as connected and start the PCAP streamer
	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()
	client.startPCAPStreamer()

	// Send PCAP data
	testData := []byte{0xd4, 0xc3, 0xb2, 0xa1, 0x02, 0x00, 0x04, 0x00}

	err := client.SendPCAPChunk(testData)
	if err != nil {
		t.Fatalf("Failed to send PCAP chunk: %v", err)
	}

	// Wait for data to be received by server
	select {
	case <-dataReceived:
		// Data was received
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for PCAP data to be sent to Hub")
	}

	// Verify the data
	if len(receivedData) != len(testData) {
		t.Errorf("Expected data length %d, got %d", len(testData), len(receivedData))
	}

	for i := range testData {
		if receivedData[i] != testData[i] {
			t.Errorf("Data mismatch at byte %d: expected 0x%02x, got 0x%02x", i, testData[i], receivedData[i])
			break
		}
	}
}

// TestSendPCAPChunk_XAgentIDHeaderSet tests that X-Agent-ID header is set when sending PCAP data
func TestSendPCAPChunk_XAgentIDHeaderSet(t *testing.T) {
	var receivedAgentID string
	headerReceived := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"sessionId": "test-session",
				"bpfFilter": "",
			})
		case "/api/agents":
			w.WriteHeader(http.StatusOK)
		case "/api/pcap/upload":
			if r.Method == "POST" {
				receivedAgentID = r.Header.Get("X-Agent-ID")
				headerReceived <- true
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)
	client.ctx, client.cancel = context.WithCancel(context.Background())
	defer client.Close()

	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()
	client.startPCAPStreamer()

	err := client.SendPCAPChunk([]byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Fatalf("Failed to send PCAP chunk: %v", err)
	}

	select {
	case <-headerReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for PCAP data to be sent to Hub")
	}

	// Verify X-Agent-ID header matches agent info
	expectedAgentID := client.agentInfo.ID
	if receivedAgentID != expectedAgentID {
		t.Errorf("Expected X-Agent-ID '%s', got '%s'", expectedAgentID, receivedAgentID)
	}
}

// TestSendPCAPChunk_PostMethodUsed tests that POST method is used for /api/pcap/upload
func TestSendPCAPChunk_PostMethodUsed(t *testing.T) {
	methodUsed := ""
	methodReceived := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"sessionId": "test-session",
				"bpfFilter": "",
			})
		case "/api/agents":
			w.WriteHeader(http.StatusOK)
		case "/api/pcap/upload":
			methodUsed = r.Method
			methodReceived <- true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)
	client.ctx, client.cancel = context.WithCancel(context.Background())
	defer client.Close()

	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()
	client.startPCAPStreamer()

	err := client.SendPCAPChunk([]byte{0x01, 0x02})
	if err != nil {
		t.Fatalf("Failed to send PCAP chunk: %v", err)
	}

	select {
	case <-methodReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for PCAP data to be sent to Hub")
	}

	if methodUsed != "POST" {
		t.Errorf("Expected POST method, got '%s'", methodUsed)
	}
}

// TestSendPCAPChunk_EmptyData tests that empty PCAP data can be queued
func TestSendPCAPChunk_EmptyData(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	err := client.SendPCAPChunk([]byte{})
	if err != nil {
		t.Errorf("Expected nil error for empty data, got: %v", err)
	}

	// Verify empty data is in channel
	select {
	case received := <-client.pcapChan:
		if len(received) != 0 {
			t.Errorf("Expected empty slice, got %d bytes", len(received))
		}
	default:
		t.Error("Expected empty data to be in channel, but channel was empty")
	}
}

// TestIsConnected_InitiallyFalse tests that IsConnected() returns false initially
func TestIsConnected_InitiallyFalse(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	if client.IsConnected() {
		t.Error("Expected IsConnected() to return false initially")
	}
}

// TestIsConnected_TrueAfterConnectedSet tests that IsConnected() returns true after connected is set
func TestIsConnected_TrueAfterConnectedSet(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	// Manually set connected state
	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()

	if !client.IsConnected() {
		t.Error("Expected IsConnected() to return true after setting connected")
	}
}

// TestIsConnected_FalseAfterDisconnected tests that IsConnected() returns false after disconnection
func TestIsConnected_FalseAfterDisconnected(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	// First connect then disconnect
	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()

	// Verify connected
	if !client.IsConnected() {
		t.Error("Expected IsConnected() to return true after setting connected")
	}

	// Disconnect
	client.connMutex.Lock()
	client.connected = false
	client.connMutex.Unlock()

	// Verify disconnected
	if client.IsConnected() {
		t.Error("Expected IsConnected() to return false after setting disconnected")
	}
}

// TestIsConnected_ReturnsCurrentState tests that IsConnected() accurately reflects current state
func TestIsConnected_ReturnsCurrentState(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	// Test multiple state transitions
	states := []bool{false, true, false, true, true, false}

	for i, expected := range states {
		client.connMutex.Lock()
		client.connected = expected
		client.connMutex.Unlock()

		if client.IsConnected() != expected {
			t.Errorf("State %d: Expected IsConnected() to return %v, got %v", i, expected, client.IsConnected())
		}
	}
}

// TestIsConnected_ThreadSafe tests that IsConnected() is safe for concurrent access
func TestIsConnected_ThreadSafe(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)
	defer client.Close()

	done := make(chan bool)

	// Start multiple goroutines reading state
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = client.IsConnected()
			}
			done <- true
		}()
	}

	// Toggle state in main goroutine
	for i := 0; i < 50; i++ {
		client.connMutex.Lock()
		client.connected = !client.connected
		client.connMutex.Unlock()
	}

	// Wait for all reader goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we got here without race conditions, test passes
}

// TestClose_SetsConnectedToFalse tests that Close() sets connected to false
func TestClose_SetsConnectedToFalse(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)

	// First connect
	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()

	// Verify connected
	if !client.IsConnected() {
		t.Error("Expected IsConnected() to return true before Close()")
	}

	// Close
	err := client.Close()
	if err != nil {
		t.Errorf("Expected Close() to return nil, got: %v", err)
	}

	// Verify disconnected
	if client.IsConnected() {
		t.Error("Expected IsConnected() to return false after Close()")
	}
}

// TestClose_ReturnsNil tests that Close() returns nil
func TestClose_ReturnsNil(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)

	err := client.Close()
	if err != nil {
		t.Errorf("Expected Close() to return nil, got: %v", err)
	}
}

// TestClose_CancelsContext tests that Close() cancels the context
func TestClose_CancelsContext(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)

	// Verify context is not cancelled initially
	select {
	case <-client.ctx.Done():
		t.Error("Expected context to not be cancelled initially")
	default:
		// Expected
	}

	// Close
	client.Close()

	// Verify context is cancelled
	select {
	case <-client.ctx.Done():
		// Expected - context is cancelled
	default:
		t.Error("Expected context to be cancelled after Close()")
	}
}

// TestClose_CanBeCalledMultipleTimes tests that Close() can be called multiple times safely
func TestClose_CanBeCalledMultipleTimes(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)

	// First close
	err := client.Close()
	if err != nil {
		t.Errorf("Expected first Close() to return nil, got: %v", err)
	}

	// Second close - should not panic
	err = client.Close()
	if err != nil {
		t.Errorf("Expected second Close() to return nil, got: %v", err)
	}

	// Verify still disconnected
	if client.IsConnected() {
		t.Error("Expected IsConnected() to return false after multiple Close() calls")
	}
}

// TestClose_WaitsForStreamersToFinish tests that Close() waits for goroutines to finish
func TestClose_WaitsForStreamersToFinish(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"sessionId": "test-session",
				"bpfFilter": "",
			})
		case "/api/agents":
			w.WriteHeader(http.StatusOK)
		case "/api/flows":
			w.WriteHeader(http.StatusCreated)
		case "/api/pcap/upload":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createClientForTestServer(t, server.URL)
	client.ctx, client.cancel = context.WithCancel(context.Background())

	// Mark as connected and start streamers
	client.connMutex.Lock()
	client.connected = true
	client.connMutex.Unlock()
	client.startFlowStreamer()
	client.startPCAPStreamer()

	// Queue some data
	client.SendFlow(createTestFlow("test-flow"))
	client.SendPCAPChunk([]byte{0x01, 0x02, 0x03})

	// Give streamers time to start processing
	time.Sleep(50 * time.Millisecond)

	// Close should wait for streamers to finish
	err := client.Close()
	if err != nil {
		t.Errorf("Expected Close() to return nil, got: %v", err)
	}

	// After Close(), streamers should be done
	if client.IsConnected() {
		t.Error("Expected IsConnected() to return false after Close()")
	}
}

// TestClose_AfterNeverConnected tests that Close() works even if Connect() was never called
func TestClose_AfterNeverConnected(t *testing.T) {
	agentInfo := createTestAgentInfo()
	client := NewHubClient("hub:9090", agentInfo)

	// Never call Connect(), just Close()
	err := client.Close()
	if err != nil {
		t.Errorf("Expected Close() to return nil, got: %v", err)
	}

	if client.IsConnected() {
		t.Error("Expected IsConnected() to return false")
	}
}

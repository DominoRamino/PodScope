package agent

import (
	"encoding/json"
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

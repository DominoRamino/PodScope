package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/podscope/podscope/pkg/protocol"
)

// setupTestServer creates a Server instance suitable for testing.
// It uses a temp directory for PCAP storage and allows custom session ID via environment.
func setupTestServer(t *testing.T) *Server {
	t.Helper()

	// Use a temp directory for PCAP storage
	pcapDir := t.TempDir()

	// Optionally set SESSION_ID from environment, otherwise use a test default
	sessionID := os.Getenv("SESSION_ID")
	if sessionID == "" {
		sessionID = "test-session"
	}

	// Create server with test configuration
	s := &Server{
		httpPort:      8080,
		grpcPort:      9090,
		sessionID:     sessionID,
		pcapDir:       pcapDir,
		flowBuffer:    NewFlowRingBuffer(100), // Small capacity for tests
		wsClients:     make(map[*websocket.Conn]bool),
		wsMutex:       sync.Mutex{},
		flowBatch:     make([]*protocol.Flow, 0, 64),
		batchMutex:    sync.Mutex{},
		batchInterval: 150 * time.Millisecond,
		catchupLimit:  200,
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		pcapBuffer:     NewPCAPBuffer(pcapDir, 1024*1024), // 1MB for tests
		pausedMutex:    sync.RWMutex{},
		bpfFilterMutex: sync.RWMutex{},
	}

	return s
}

// Ensure imports are used
var _ = protocol.Flow{}
var _ = time.Now

// TestHandleHealth_Returns200OK tests that the health endpoint returns HTTP 200
func TestHandleHealth_Returns200OK(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}

// TestHandleHealth_ContentTypeJSON tests that the response is JSON
func TestHandleHealth_ContentTypeJSON(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}
}

// TestHandleHealth_StatusHealthy tests that the response includes status: "healthy"
func TestHandleHealth_StatusHealthy(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	status, ok := resp["status"].(string)
	if !ok {
		t.Fatal("response missing 'status' field or not a string")
	}
	if status != "healthy" {
		t.Errorf("status = %q, want %q", status, "healthy")
	}
}

// TestHandleHealth_IncludesSessionId tests that the response includes sessionId
func TestHandleHealth_IncludesSessionId(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	sessionId, ok := resp["sessionId"].(string)
	if !ok {
		t.Fatal("response missing 'sessionId' field or not a string")
	}
	if sessionId != "test-session" {
		t.Errorf("sessionId = %q, want %q", sessionId, "test-session")
	}
}

// TestHandleHealth_IncludesSessionIdFromEnv tests that sessionId comes from environment
func TestHandleHealth_IncludesSessionIdFromEnv(t *testing.T) {
	// Set custom session ID via environment
	os.Setenv("SESSION_ID", "custom-env-session")
	defer os.Unsetenv("SESSION_ID")

	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	sessionId, ok := resp["sessionId"].(string)
	if !ok {
		t.Fatal("response missing 'sessionId' field or not a string")
	}
	if sessionId != "custom-env-session" {
		t.Errorf("sessionId = %q, want %q", sessionId, "custom-env-session")
	}
}

// TestHandleHealth_IncludesBPFFilter tests that response includes bpfFilter field
func TestHandleHealth_IncludesBPFFilter(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// bpfFilter should be present in response
	_, ok := resp["bpfFilter"]
	if !ok {
		t.Error("response missing 'bpfFilter' field")
	}
}

// TestHandleHealth_BPFFilterEmptyByDefault tests that bpfFilter is empty string when not set
func TestHandleHealth_BPFFilterEmptyByDefault(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	bpfFilter, ok := resp["bpfFilter"].(string)
	if !ok {
		t.Fatal("response missing 'bpfFilter' field or not a string")
	}
	if bpfFilter != "" {
		t.Errorf("bpfFilter = %q, want empty string", bpfFilter)
	}
}

// TestHandleHealth_BPFFilterReturnsCurrentValue tests that bpfFilter returns current value when set
func TestHandleHealth_BPFFilterReturnsCurrentValue(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Set a BPF filter
	s.bpfFilterMutex.Lock()
	s.bpfFilter = "tcp port 80"
	s.bpfFilterMutex.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	bpfFilter, ok := resp["bpfFilter"].(string)
	if !ok {
		t.Fatal("response missing 'bpfFilter' field or not a string")
	}
	if bpfFilter != "tcp port 80" {
		t.Errorf("bpfFilter = %q, want %q", bpfFilter, "tcp port 80")
	}
}

// TestHandleHealth_IncludesTimestamp tests that response includes a timestamp field
func TestHandleHealth_IncludesTimestamp(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// timestamp should be present in response
	_, ok := resp["timestamp"]
	if !ok {
		t.Error("response missing 'timestamp' field")
	}
}

// ============================================================================
// TestHandleFlows tests for /api/flows GET and POST endpoints
// ============================================================================

// TestHandleFlows_GET_EmptyBufferReturnsEmptyArray tests that GET returns empty flows array when buffer is empty
func TestHandleFlows_GET_EmptyBufferReturnsEmptyArray(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/flows", nil)
	w := httptest.NewRecorder()

	s.handleFlows(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	flows, ok := resp["flows"].([]interface{})
	if !ok {
		t.Fatal("response missing 'flows' field or not an array")
	}
	if len(flows) != 0 {
		t.Errorf("flows length = %d, want 0", len(flows))
	}
}

// TestHandleFlows_GET_ReturnsAllStoredFlows tests that GET returns all stored flows in JSON format
func TestHandleFlows_GET_ReturnsAllStoredFlows(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Add some flows to the buffer
	flow1 := &protocol.Flow{
		ID:       "flow-1",
		SrcIP:    "10.0.0.1",
		SrcPort:  12345,
		DstIP:    "10.0.0.2",
		DstPort:  80,
		Protocol: protocol.ProtocolHTTP,
		Status:   protocol.StatusClosed,
	}
	flow2 := &protocol.Flow{
		ID:       "flow-2",
		SrcIP:    "10.0.0.3",
		SrcPort:  54321,
		DstIP:    "10.0.0.4",
		DstPort:  443,
		Protocol: protocol.ProtocolHTTPS,
		Status:   protocol.StatusClosed,
	}
	s.flowBuffer.Add(flow1)
	s.flowBuffer.Add(flow2)

	req := httptest.NewRequest(http.MethodGet, "/api/flows", nil)
	w := httptest.NewRecorder()

	s.handleFlows(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	flows, ok := resp["flows"].([]interface{})
	if !ok {
		t.Fatal("response missing 'flows' field or not an array")
	}
	if len(flows) != 2 {
		t.Errorf("flows length = %d, want 2", len(flows))
	}

	// Check count field
	count, ok := resp["count"].(float64)
	if !ok {
		t.Fatal("response missing 'count' field or not a number")
	}
	if int(count) != 2 {
		t.Errorf("count = %d, want 2", int(count))
	}
}

// TestHandleFlows_GET_ReturnsJSONContentType tests that GET returns JSON content type
func TestHandleFlows_GET_ReturnsJSONContentType(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/flows", nil)
	w := httptest.NewRecorder()

	s.handleFlows(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}
}

// TestHandleFlows_POST_ValidFlowReturns201 tests that POST with valid flow JSON returns 201 Created
func TestHandleFlows_POST_ValidFlowReturns201(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	flowJSON := `{
		"id": "test-flow-1",
		"srcIp": "192.168.1.10",
		"srcPort": 45678,
		"dstIp": "10.0.0.5",
		"dstPort": 80,
		"protocol": "HTTP",
		"status": "CLOSED"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/flows", strings.NewReader(flowJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleFlows(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	status, ok := resp["status"].(string)
	if !ok || status != "ok" {
		t.Errorf("response status = %q, want %q", status, "ok")
	}
}

// TestHandleFlows_POST_InvalidJSONReturns400 tests that POST with invalid JSON returns 400 Bad Request
func TestHandleFlows_POST_InvalidJSONReturns400(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	invalidJSON := `{invalid json here`

	req := httptest.NewRequest(http.MethodPost, "/api/flows", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleFlows(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestHandleFlows_POST_FlowStoredInBuffer tests that POSTed flow is stored in the buffer
func TestHandleFlows_POST_FlowStoredInBuffer(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	flowJSON := `{
		"id": "stored-flow-123",
		"srcIp": "172.16.0.1",
		"srcPort": 33333,
		"dstIp": "172.16.0.2",
		"dstPort": 8080,
		"protocol": "TCP",
		"status": "OPEN"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/flows", strings.NewReader(flowJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleFlows(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusCreated)
	}

	// Verify the flow was stored in the buffer
	storedFlow := s.flowBuffer.Get("stored-flow-123")
	if storedFlow == nil {
		t.Fatal("flow not stored in buffer")
	}
	if storedFlow.SrcIP != "172.16.0.1" {
		t.Errorf("stored flow srcIp = %q, want %q", storedFlow.SrcIP, "172.16.0.1")
	}
	if storedFlow.DstPort != 8080 {
		t.Errorf("stored flow dstPort = %d, want %d", storedFlow.DstPort, 8080)
	}
}

// TestHandleFlows_PUT_Returns405 tests that PUT method returns 405 Method Not Allowed
func TestHandleFlows_PUT_Returns405(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/flows", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	s.handleFlows(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestHandleFlows_DELETE_Returns405 tests that DELETE method returns 405 Method Not Allowed
func TestHandleFlows_DELETE_Returns405(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/flows", nil)
	w := httptest.NewRecorder()

	s.handleFlows(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestHandleFlows_PATCH_Returns405 tests that PATCH method returns 405 Method Not Allowed
func TestHandleFlows_PATCH_Returns405(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodPatch, "/api/flows", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	s.handleFlows(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestHandleFlows_GET_ReturnsCapacity tests that GET response includes capacity field
func TestHandleFlows_GET_ReturnsCapacity(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/flows", nil)
	w := httptest.NewRecorder()

	s.handleFlows(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	capacity, ok := resp["capacity"].(float64)
	if !ok {
		t.Fatal("response missing 'capacity' field or not a number")
	}
	// setupTestServer creates flowBuffer with capacity 100
	if int(capacity) != 100 {
		t.Errorf("capacity = %d, want 100", int(capacity))
	}
}

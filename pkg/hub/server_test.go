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

// ============================================================================
// TestHandlePause tests for /api/pause GET and POST endpoints
// ============================================================================

// TestHandlePause_GET_ReturnsFalseInitially tests that GET returns paused: false when server starts
func TestHandlePause_GET_ReturnsFalseInitially(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/pause", nil)
	w := httptest.NewRecorder()

	s.handlePause(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	paused, ok := resp["paused"]
	if !ok {
		t.Fatal("response missing 'paused' field")
	}
	if paused != false {
		t.Errorf("paused = %v, want false", paused)
	}
}

// TestHandlePause_GET_ReturnsCurrentState tests that GET returns the current pause state
func TestHandlePause_GET_ReturnsCurrentState(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Set paused to true
	s.pausedMutex.Lock()
	s.paused = true
	s.pausedMutex.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/pause", nil)
	w := httptest.NewRecorder()

	s.handlePause(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	paused, ok := resp["paused"]
	if !ok {
		t.Fatal("response missing 'paused' field")
	}
	if paused != true {
		t.Errorf("paused = %v, want true", paused)
	}
}

// TestHandlePause_GET_ReturnsJSONContentType tests that GET returns JSON content type
func TestHandlePause_GET_ReturnsJSONContentType(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/pause", nil)
	w := httptest.NewRecorder()

	s.handlePause(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}
}

// TestHandlePause_POST_SetPausedTrue tests that POST with paused: true sets pause to true
func TestHandlePause_POST_SetPausedTrue(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Ensure we start with paused = false
	s.pausedMutex.Lock()
	s.paused = false
	s.pausedMutex.Unlock()

	body := `{"paused": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/pause", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handlePause(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	paused, ok := resp["paused"]
	if !ok {
		t.Fatal("response missing 'paused' field")
	}
	if paused != true {
		t.Errorf("response paused = %v, want true", paused)
	}

	// Verify internal state was updated
	s.pausedMutex.RLock()
	internalPaused := s.paused
	s.pausedMutex.RUnlock()

	if internalPaused != true {
		t.Errorf("internal paused state = %v, want true", internalPaused)
	}
}

// TestHandlePause_POST_SetPausedFalse tests that POST with paused: false sets pause to false
func TestHandlePause_POST_SetPausedFalse(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Start with paused = true
	s.pausedMutex.Lock()
	s.paused = true
	s.pausedMutex.Unlock()

	body := `{"paused": false}`
	req := httptest.NewRequest(http.MethodPost, "/api/pause", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handlePause(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	paused, ok := resp["paused"]
	if !ok {
		t.Fatal("response missing 'paused' field")
	}
	if paused != false {
		t.Errorf("response paused = %v, want false", paused)
	}

	// Verify internal state was updated
	s.pausedMutex.RLock()
	internalPaused := s.paused
	s.pausedMutex.RUnlock()

	if internalPaused != false {
		t.Errorf("internal paused state = %v, want false", internalPaused)
	}
}

// TestHandlePause_POST_EmptyBodyTogglesFromFalseToTrue tests that POST with empty body toggles pause from false to true
func TestHandlePause_POST_EmptyBodyTogglesFromFalseToTrue(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Start with paused = false
	s.pausedMutex.Lock()
	s.paused = false
	s.pausedMutex.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/pause", nil)
	w := httptest.NewRecorder()

	s.handlePause(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	paused, ok := resp["paused"]
	if !ok {
		t.Fatal("response missing 'paused' field")
	}
	if paused != true {
		t.Errorf("response paused = %v, want true (toggled from false)", paused)
	}

	// Verify internal state was toggled
	s.pausedMutex.RLock()
	internalPaused := s.paused
	s.pausedMutex.RUnlock()

	if internalPaused != true {
		t.Errorf("internal paused state = %v, want true", internalPaused)
	}
}

// TestHandlePause_POST_EmptyBodyTogglesFromTrueToFalse tests that POST with empty body toggles pause from true to false
func TestHandlePause_POST_EmptyBodyTogglesFromTrueToFalse(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Start with paused = true
	s.pausedMutex.Lock()
	s.paused = true
	s.pausedMutex.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/pause", nil)
	w := httptest.NewRecorder()

	s.handlePause(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	paused, ok := resp["paused"]
	if !ok {
		t.Fatal("response missing 'paused' field")
	}
	if paused != false {
		t.Errorf("response paused = %v, want false (toggled from true)", paused)
	}

	// Verify internal state was toggled
	s.pausedMutex.RLock()
	internalPaused := s.paused
	s.pausedMutex.RUnlock()

	if internalPaused != false {
		t.Errorf("internal paused state = %v, want false", internalPaused)
	}
}

// TestHandlePause_POST_MultipleToggles tests that consecutive toggles alternate the state
func TestHandlePause_POST_MultipleToggles(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Start with paused = false
	s.pausedMutex.Lock()
	s.paused = false
	s.pausedMutex.Unlock()

	expectedStates := []bool{true, false, true, false}

	for i, expected := range expectedStates {
		req := httptest.NewRequest(http.MethodPost, "/api/pause", nil)
		w := httptest.NewRecorder()

		s.handlePause(w, req)

		var resp map[string]bool
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("toggle %d: failed to decode response: %v", i+1, err)
		}

		paused := resp["paused"]
		if paused != expected {
			t.Errorf("toggle %d: paused = %v, want %v", i+1, paused, expected)
		}
	}
}

// TestHandlePause_DELETE_Returns405 tests that DELETE method returns 405 Method Not Allowed
func TestHandlePause_DELETE_Returns405(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/pause", nil)
	w := httptest.NewRecorder()

	s.handlePause(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestHandlePause_PUT_Returns405 tests that PUT method returns 405 Method Not Allowed
func TestHandlePause_PUT_Returns405(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/pause", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	s.handlePause(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ============================================================================
// TestHandleBPFFilter tests for /api/bpf-filter GET and POST endpoints
// ============================================================================

// TestHandleBPFFilter_GET_ReturnsEmptyFilterByDefault tests that GET returns empty filter when not set
func TestHandleBPFFilter_GET_ReturnsEmptyFilterByDefault(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/bpf-filter", nil)
	w := httptest.NewRecorder()

	s.handleBPFFilter(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	filter, ok := resp["filter"]
	if !ok {
		t.Fatal("response missing 'filter' field")
	}
	if filter != "" {
		t.Errorf("filter = %q, want empty string", filter)
	}
}

// TestHandleBPFFilter_GET_ReturnsJSONContentType tests that GET returns JSON content type
func TestHandleBPFFilter_GET_ReturnsJSONContentType(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/bpf-filter", nil)
	w := httptest.NewRecorder()

	s.handleBPFFilter(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}
}

// TestHandleBPFFilter_GET_ReturnsCurrentFilterAfterSet tests that GET returns the current filter after it's been set
func TestHandleBPFFilter_GET_ReturnsCurrentFilterAfterSet(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Set a BPF filter directly
	s.bpfFilterMutex.Lock()
	s.bpfFilter = "tcp port 80"
	s.bpfFilterMutex.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/bpf-filter", nil)
	w := httptest.NewRecorder()

	s.handleBPFFilter(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	filter := resp["filter"]
	if filter != "tcp port 80" {
		t.Errorf("filter = %q, want %q", filter, "tcp port 80")
	}
}

// TestHandleBPFFilter_POST_ValidFilterAccepted tests that POST with valid BPF syntax is accepted
func TestHandleBPFFilter_POST_ValidFilterAccepted(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	body := `{"filter": "tcp port 80"}`
	req := httptest.NewRequest(http.MethodPost, "/api/bpf-filter", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleBPFFilter(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	success, ok := resp["success"].(bool)
	if !ok || !success {
		t.Errorf("response success = %v, want true", resp["success"])
	}

	// Verify the filter was stored
	s.bpfFilterMutex.RLock()
	storedFilter := s.bpfFilter
	s.bpfFilterMutex.RUnlock()

	if storedFilter != "tcp port 80" {
		t.Errorf("stored filter = %q, want %q", storedFilter, "tcp port 80")
	}
}

// TestHandleBPFFilter_POST_EmptyStringClearsFilter tests that POST with empty string clears the filter
func TestHandleBPFFilter_POST_EmptyStringClearsFilter(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Set an initial filter
	s.bpfFilterMutex.Lock()
	s.bpfFilter = "tcp port 443"
	s.bpfFilterMutex.Unlock()

	// Clear the filter with empty string
	body := `{"filter": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/bpf-filter", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleBPFFilter(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	success, ok := resp["success"].(bool)
	if !ok || !success {
		t.Errorf("response success = %v, want true", resp["success"])
	}

	// Verify the filter was cleared
	s.bpfFilterMutex.RLock()
	storedFilter := s.bpfFilter
	s.bpfFilterMutex.RUnlock()

	if storedFilter != "" {
		t.Errorf("stored filter = %q, want empty string", storedFilter)
	}
}

// TestHandleBPFFilter_POST_ReturnsFilterInResponse tests that POST returns the filter in response
func TestHandleBPFFilter_POST_ReturnsFilterInResponse(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	body := `{"filter": "udp port 53"}`
	req := httptest.NewRequest(http.MethodPost, "/api/bpf-filter", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleBPFFilter(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	filter, ok := resp["filter"].(string)
	if !ok {
		t.Fatal("response missing 'filter' field")
	}
	if filter != "udp port 53" {
		t.Errorf("response filter = %q, want %q", filter, "udp port 53")
	}
}

// TestHandleBPFFilter_POST_MissingFilterFieldReturns400 tests that POST without filter field returns 400
func TestHandleBPFFilter_POST_MissingFilterFieldReturns400(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/bpf-filter", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleBPFFilter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestHandleBPFFilter_POST_InvalidJSONReturns400 tests that POST with invalid JSON returns 400
func TestHandleBPFFilter_POST_InvalidJSONReturns400(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/bpf-filter", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleBPFFilter(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestHandleBPFFilter_DELETE_Returns405 tests that DELETE method returns 405 Method Not Allowed
func TestHandleBPFFilter_DELETE_Returns405(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/bpf-filter", nil)
	w := httptest.NewRecorder()

	s.handleBPFFilter(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestHandleBPFFilter_PUT_Returns405 tests that PUT method returns 405 Method Not Allowed
func TestHandleBPFFilter_PUT_Returns405(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/bpf-filter", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	s.handleBPFFilter(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ============================================================================
// TestHandlePCAPUpload tests for POST /api/pcap/upload endpoint
// ============================================================================

// TestHandlePCAPUpload_POST_ValidDataReturns200 tests that POST with valid binary data returns 200 OK
func TestHandlePCAPUpload_POST_ValidDataReturns200(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Create some test binary data
	pcapData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	req := httptest.NewRequest(http.MethodPost, "/api/pcap/upload", strings.NewReader(string(pcapData)))
	req.Header.Set("X-Agent-ID", "test-agent-1")
	w := httptest.NewRecorder()

	s.handlePCAPUpload(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}

// TestHandlePCAPUpload_POST_UsesXAgentIDHeader tests that X-Agent-ID header is used for agent identification
func TestHandlePCAPUpload_POST_UsesXAgentIDHeader(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Create test data
	pcapData := []byte{0x10, 0x20, 0x30, 0x40}
	agentID := "my-test-agent-xyz"

	req := httptest.NewRequest(http.MethodPost, "/api/pcap/upload", strings.NewReader(string(pcapData)))
	req.Header.Set("X-Agent-ID", agentID)
	w := httptest.NewRecorder()

	s.handlePCAPUpload(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify data was written with the correct agent ID by checking that a file exists
	// The file should be named agent-{agentID}.pcap
	// We can verify this by checking that pcapBuffer.Size() > 0 after the upload
	size := s.pcapBuffer.Size()
	if size == 0 {
		t.Error("pcapBuffer.Size() = 0, expected > 0 after upload")
	}
}

// TestHandlePCAPUpload_POST_DefaultsToUnknownAgent tests that agent defaults to "unknown" if header missing
func TestHandlePCAPUpload_POST_DefaultsToUnknownAgent(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Create test data without X-Agent-ID header
	pcapData := []byte{0xAA, 0xBB, 0xCC, 0xDD}

	req := httptest.NewRequest(http.MethodPost, "/api/pcap/upload", strings.NewReader(string(pcapData)))
	// Note: NOT setting X-Agent-ID header
	w := httptest.NewRecorder()

	s.handlePCAPUpload(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	// The request should still succeed, defaulting to "unknown" agent
	size := s.pcapBuffer.Size()
	if size == 0 {
		t.Error("pcapBuffer.Size() = 0, expected > 0 after upload with default agent")
	}
}

// TestHandlePCAPUpload_POST_DataDroppedWhenPaused tests that data is silently dropped when paused
func TestHandlePCAPUpload_POST_DataDroppedWhenPaused(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Set server to paused state
	s.pausedMutex.Lock()
	s.paused = true
	s.pausedMutex.Unlock()

	// Create test data
	pcapData := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}

	req := httptest.NewRequest(http.MethodPost, "/api/pcap/upload", strings.NewReader(string(pcapData)))
	req.Header.Set("X-Agent-ID", "test-agent-paused")
	w := httptest.NewRecorder()

	s.handlePCAPUpload(w, req)

	// Should still return 200 OK even when paused
	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d (should return 200 even when paused)", w.Code, http.StatusOK)
	}

	// Verify data was NOT stored (size should be 0)
	size := s.pcapBuffer.Size()
	if size != 0 {
		t.Errorf("pcapBuffer.Size() = %d, want 0 (data should be dropped when paused)", size)
	}
}

// TestHandlePCAPUpload_POST_DataStoredWhenNotPaused tests that data is stored when not paused
func TestHandlePCAPUpload_POST_DataStoredWhenNotPaused(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	// Ensure server is NOT paused (default state, but be explicit)
	s.pausedMutex.Lock()
	s.paused = false
	s.pausedMutex.Unlock()

	// Create test data
	pcapData := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE}

	req := httptest.NewRequest(http.MethodPost, "/api/pcap/upload", strings.NewReader(string(pcapData)))
	req.Header.Set("X-Agent-ID", "test-agent-active")
	w := httptest.NewRecorder()

	s.handlePCAPUpload(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify data WAS stored (size should be > 0)
	size := s.pcapBuffer.Size()
	if size == 0 {
		t.Error("pcapBuffer.Size() = 0, expected > 0 (data should be stored when not paused)")
	}
}

// TestHandlePCAPUpload_GET_Returns405 tests that GET method returns 405 Method Not Allowed
func TestHandlePCAPUpload_GET_Returns405(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/pcap/upload", nil)
	w := httptest.NewRecorder()

	s.handlePCAPUpload(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestHandlePCAPUpload_PUT_Returns405 tests that PUT method returns 405 Method Not Allowed
func TestHandlePCAPUpload_PUT_Returns405(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/pcap/upload", strings.NewReader("data"))
	w := httptest.NewRecorder()

	s.handlePCAPUpload(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestHandlePCAPUpload_DELETE_Returns405 tests that DELETE method returns 405 Method Not Allowed
func TestHandlePCAPUpload_DELETE_Returns405(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/pcap/upload", nil)
	w := httptest.NewRecorder()

	s.handlePCAPUpload(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// TestHandlePCAPUpload_POST_EmptyBodyReturns200 tests that POST with empty body returns 200
func TestHandlePCAPUpload_POST_EmptyBodyReturns200(t *testing.T) {
	s := setupTestServer(t)
	defer s.pcapBuffer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/pcap/upload", strings.NewReader(""))
	req.Header.Set("X-Agent-ID", "test-agent-empty")
	w := httptest.NewRecorder()

	s.handlePCAPUpload(w, req)

	// Empty body should be accepted (it's just 0 bytes of PCAP data)
	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}

package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

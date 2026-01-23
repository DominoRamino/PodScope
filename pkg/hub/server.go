package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/gorilla/websocket"
	"github.com/podscope/podscope/pkg/protocol"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

// Server is the Hub server that aggregates traffic from agents
type Server struct {
	httpPort  int
	grpcPort  int
	sessionID string
	pcapDir   string

	// Flow storage - bounded ring buffer
	flowBuffer *FlowRingBuffer

	// WebSocket clients
	wsClients   map[*websocket.Conn]bool
	wsMutex     sync.Mutex
	wsUpgrader  websocket.Upgrader

	// WebSocket batching
	flowBatch     []*protocol.Flow
	batchMutex    sync.Mutex
	batchTicker   *time.Ticker
	batchInterval time.Duration
	catchupLimit  int

	// PCAP storage
	pcapBuffer  *PCAPBuffer

	// Pause state - when true, PCAP data is not stored
	paused      bool
	pausedMutex sync.RWMutex

	// BPF filter - can be updated dynamically
	bpfFilter      string
	bpfFilterMutex sync.RWMutex

	// Kubernetes client for terminal exec (initialized lazily)
	k8sClient     kubernetes.Interface
	k8sRestConfig *rest.Config
}

// NewServer creates a new Hub server
func NewServer(httpPort, grpcPort int) *Server {
	sessionID := os.Getenv("SESSION_ID")
	if sessionID == "" {
		sessionID = "local"
	}

	pcapDir := os.Getenv("PCAP_DIR")
	if pcapDir == "" {
		pcapDir = "/data/pcap"
	}

	// Read batching configuration from environment
	batchIntervalMs := getEnvIntServer("WS_BATCH_INTERVAL_MS", 150)
	catchupLimit := getEnvIntServer("WS_CATCHUP_LIMIT", 200)

	s := &Server{
		httpPort:   httpPort,
		grpcPort:   grpcPort,
		sessionID:  sessionID,
		pcapDir:    pcapDir,
		flowBuffer: NewFlowRingBuffer(0), // Uses MAX_FLOWS env or default 10000
		wsClients:     make(map[*websocket.Conn]bool),
		flowBatch:     make([]*protocol.Flow, 0, 64),
		batchInterval: time.Duration(batchIntervalMs) * time.Millisecond,
		catchupLimit:  catchupLimit,
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for local development
			},
		},
		pcapBuffer: NewPCAPBuffer(pcapDir, 50*1024*1024), // 50MB rolling buffer
	}

	// Start batch ticker for WebSocket batching
	s.batchTicker = time.NewTicker(s.batchInterval)
	go s.batchBroadcastLoop()

	return s
}

// getEnvIntServer reads an integer from environment variable with a default value.
func getEnvIntServer(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}


// Start starts the Hub server
func (s *Server) Start(ctx context.Context) error {
	// Ensure PCAP directory exists
	if err := os.MkdirAll(s.pcapDir, 0755); err != nil {
		return fmt.Errorf("failed to create pcap directory: %w", err)
	}

	// Start HTTP server
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/flows", s.handleFlows)
	mux.HandleFunc("/api/flows/ws", s.handleFlowsWebSocket)
	mux.HandleFunc("/api/pcap", s.handleDownloadPCAP)
	mux.HandleFunc("/api/pcap/upload", s.handlePCAPUpload)
	mux.HandleFunc("/api/pcap/reset", s.handlePCAPReset)
	mux.HandleFunc("/api/pcap/", s.handleDownloadStreamPCAP)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/pause", s.handlePause)
	mux.HandleFunc("/api/bpf-filter", s.handleBPFFilter)
	mux.HandleFunc("/api/terminal/ws", s.handleTerminalWebSocket)
	mux.HandleFunc("/api/ai/anthropic", s.handleAnthropicProxy)

	// Serve static UI files
	mux.Handle("/", http.FileServer(http.Dir("/app/ui")))

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.httpPort),
		Handler: mux,
	}

	// Start gRPC server for agent connections
	grpcServer, err := s.startGRPCServer()
	if err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	// Handle shutdown
	go func() {
		<-ctx.Done()
		log.Println("Shutting down servers...")
		httpServer.Shutdown(context.Background())
		grpcServer.GracefulStop()
	}()

	log.Printf("Hub server starting - HTTP: %d, gRPC: %d", s.httpPort, s.grpcPort)

	return httpServer.ListenAndServe()
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get current BPF filter
	s.bpfFilterMutex.RLock()
	currentFilter := s.bpfFilter
	s.bpfFilterMutex.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"sessionId": s.sessionID,
		"timestamp": time.Now().UTC(),
		"bpfFilter": currentFilter,
	})
}

// handleFlows handles GET (list flows) and POST (add flow)
func (s *Server) handleFlows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		flows := s.flowBuffer.GetAll()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"flows":    flows,
			"count":    len(flows),
			"capacity": s.flowBuffer.Capacity(),
		})

	case http.MethodPost:
		var flow protocol.Flow
		if err := json.NewDecoder(r.Body).Decode(&flow); err != nil {
			http.Error(w, "Invalid flow data", http.StatusBadRequest)
			return
		}

		s.AddFlow(&flow)
		log.Printf("Received flow: %s %s:%d -> %s:%d [%s]",
			flow.Protocol, flow.SrcIP, flow.SrcPort, flow.DstIP, flow.DstPort, flow.Status)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePCAPUpload receives PCAP data from agents
func (s *Server) handlePCAPUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		agentID = "unknown"
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	if err := s.AddPCAPData(agentID, data); err != nil {
		http.Error(w, "Failed to store PCAP", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handlePCAPReset handles PCAP reset requests
func (s *Server) handlePCAPReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Reset the PCAP buffer
	if err := s.pcapBuffer.Reset(); err != nil {
		log.Printf("Failed to reset PCAP buffer: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to reset PCAP: %v", err),
		})
		return
	}

	log.Printf("PCAP buffer reset successfully")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "PCAP buffer reset successfully",
	})
}

// handleAgents handles agent registration
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var agent protocol.AgentInfo
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, "Invalid agent data", http.StatusBadRequest)
		return
	}

	log.Printf("Agent connected: %s (%s/%s)", agent.ID, agent.Namespace, agent.PodName)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

// handlePause handles pause/resume requests
func (s *Server) handlePause(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Return current pause state
		s.pausedMutex.RLock()
		paused := s.paused
		s.pausedMutex.RUnlock()

		json.NewEncoder(w).Encode(map[string]bool{"paused": paused})

	case http.MethodPost:
		// Toggle or set pause state
		var req struct {
			Paused *bool `json:"paused"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// If no body, toggle the state
			s.pausedMutex.Lock()
			s.paused = !s.paused
			paused := s.paused
			s.pausedMutex.Unlock()

			log.Printf("Capture %s (toggled)", map[bool]string{true: "paused", false: "resumed"}[paused])
			json.NewEncoder(w).Encode(map[string]bool{"paused": paused})
			return
		}

		// Set to specific state
		if req.Paused != nil {
			s.pausedMutex.Lock()
			s.paused = *req.Paused
			paused := s.paused
			s.pausedMutex.Unlock()

			log.Printf("Capture %s", map[bool]string{true: "paused", false: "resumed"}[paused])
			json.NewEncoder(w).Encode(map[string]bool{"paused": paused})
		} else {
			http.Error(w, "Missing 'paused' field", http.StatusBadRequest)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBPFFilter handles BPF filter GET/UPDATE requests
// validateBPFFilter validates a BPF filter expression using libpcap
func validateBPFFilter(filter string) error {
	if filter == "" {
		return nil // Empty filter is valid (resets to default)
	}

	// Try to compile the filter using libpcap
	// We use a dummy inactive handle just for validation
	inactive, err := pcap.NewInactiveHandle("lo")
	if err != nil {
		// If we can't create inactive handle, try to just compile the BPF
		// This is a basic syntax check
		_, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, 65535, filter)
		if err != nil {
			return fmt.Errorf("invalid BPF syntax: %w", err)
		}
		return nil
	}
	defer inactive.CleanUp()

	// Activate the handle
	handle, err := inactive.Activate()
	if err != nil {
		// If activation fails, fall back to compile check
		_, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, 65535, filter)
		if err != nil {
			return fmt.Errorf("invalid BPF syntax: %w", err)
		}
		return nil
	}
	defer handle.Close()

	// Try to set the filter on the handle
	if err := handle.SetBPFFilter(filter); err != nil {
		return fmt.Errorf("invalid BPF syntax: %w", err)
	}

	return nil
}

func (s *Server) handleBPFFilter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Return current BPF filter
		s.bpfFilterMutex.RLock()
		filter := s.bpfFilter
		s.bpfFilterMutex.RUnlock()

		json.NewEncoder(w).Encode(map[string]string{"filter": filter})

	case http.MethodPost:
		// Update BPF filter
		var req struct {
			Filter *string `json:"filter"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if req.Filter == nil {
			http.Error(w, "Missing 'filter' field", http.StatusBadRequest)
			return
		}

		// Validate the filter before applying
		if err := validateBPFFilter(*req.Filter); err != nil {
			log.Printf("BPF filter validation failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
				"message": "Invalid BPF filter syntax",
			})
			return
		}

		// Store the user's filter unchanged
		// The agent is responsible for combining with its own hub exclusion
		s.bpfFilterMutex.Lock()
		s.bpfFilter = *req.Filter
		s.bpfFilterMutex.Unlock()

		log.Printf("BPF filter updated to: %s", *req.Filter)
		log.Printf("Filter will be applied to agents on next heartbeat")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"filter":  *req.Filter,
			"message": "BPF filter will be applied on next heartbeat",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFlowsWebSocket handles WebSocket connections for live flow updates
func (s *Server) handleFlowsWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	s.wsMutex.Lock()
	s.wsClients[conn] = true
	s.wsMutex.Unlock()

	defer func() {
		s.wsMutex.Lock()
		delete(s.wsClients, conn)
		s.wsMutex.Unlock()
	}()

	// Send initial catch-up with limited flows (most recent)
	initialFlows := s.flowBuffer.GetRecent(s.catchupLimit)
	catchUpMsg := map[string]interface{}{
		"type":    "catchup",
		"flows":   initialFlows,
		"total":   s.flowBuffer.Size(),
		"hasMore": s.flowBuffer.Size() > s.catchupLimit,
	}

	if err := conn.WriteJSON(catchUpMsg); err != nil {
		log.Printf("WebSocket catch-up error: %v", err)
		return
	}

	// Keep connection alive and handle incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// handleDownloadPCAP handles PCAP file downloads for the entire session
func (s *Server) handleDownloadPCAP(w http.ResponseWriter, r *http.Request) {
	// Parse filter parameters from query string
	query := r.URL.Query()
	onlyHTTP := query.Get("onlyHTTP") == "true"
	includeDNS := query.Get("includeDNS") == "true"
	allPorts := query.Get("allPorts") == "true"
	searchText := query.Get("search")

	// Log filter parameters
	if onlyHTTP || !includeDNS || allPorts || searchText != "" {
		log.Printf("PCAP download with filters: onlyHTTP=%v includeDNS=%v allPorts=%v search=%q",
			onlyHTTP, includeDNS, allPorts, searchText)
	}

	// TODO: Implement packet filtering based on these parameters
	// For now, return all packets (filtering happens in UI view)
	pcapData, err := s.pcapBuffer.GetSessionPCAP()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate PCAP: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.tcpdump.pcap")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=podscope-%s.pcap", s.sessionID))
	w.Write(pcapData)
}

// handleDownloadStreamPCAP handles PCAP file downloads for a specific stream
func (s *Server) handleDownloadStreamPCAP(w http.ResponseWriter, r *http.Request) {
	// Extract stream ID from URL path
	streamID := r.URL.Path[len("/api/pcap/"):]
	if streamID == "" {
		http.Error(w, "Stream ID required", http.StatusBadRequest)
		return
	}

	// Parse filter parameters from query string
	query := r.URL.Query()
	onlyHTTP := query.Get("onlyHTTP") == "true"
	includeDNS := query.Get("includeDNS") == "true"
	allPorts := query.Get("allPorts") == "true"
	searchText := query.Get("search")

	// Log filter parameters
	if onlyHTTP || !includeDNS || allPorts || searchText != "" {
		log.Printf("Stream PCAP download with filters: stream=%s onlyHTTP=%v includeDNS=%v allPorts=%v search=%q",
			streamID, onlyHTTP, includeDNS, allPorts, searchText)
	}

	// TODO: Implement packet filtering based on these parameters
	// For now, return all packets (filtering happens in UI view)
	pcapData, err := s.pcapBuffer.GetStreamPCAP(streamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate PCAP: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.tcpdump.pcap")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=stream-%s.pcap", streamID))
	w.Write(pcapData)
}

// handleStats returns capture statistics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	flowCount := s.flowBuffer.Size()

	s.wsMutex.Lock()
	clientCount := len(s.wsClients)
	s.wsMutex.Unlock()

	s.pausedMutex.RLock()
	paused := s.paused
	s.pausedMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"flows":        flowCount,
		"flowCapacity": s.flowBuffer.Capacity(),
		"wsClients":    clientCount,
		"pcapSize":     s.pcapBuffer.Size(),
		"sessionId":    s.sessionID,
		"uptime":       time.Now().UTC(),
		"paused":       paused,
	})
}

// AddFlow adds a new flow and queues it for batched WebSocket broadcast
func (s *Server) AddFlow(flow *protocol.Flow) {
	s.flowBuffer.Add(flow)

	// Queue for batched broadcast instead of immediate send
	s.queueFlowForBroadcast(flow)
}

// queueFlowForBroadcast adds a flow to the batch queue
func (s *Server) queueFlowForBroadcast(flow *protocol.Flow) {
	s.batchMutex.Lock()
	s.flowBatch = append(s.flowBatch, flow)
	s.batchMutex.Unlock()
}

// batchBroadcastLoop runs the batching timer
func (s *Server) batchBroadcastLoop() {
	for range s.batchTicker.C {
		s.flushBatch()
	}
}

// flushBatch sends all queued flows as a single batch message
func (s *Server) flushBatch() {
	s.batchMutex.Lock()
	if len(s.flowBatch) == 0 {
		s.batchMutex.Unlock()
		return
	}
	batch := s.flowBatch
	s.flowBatch = make([]*protocol.Flow, 0, 64)
	s.batchMutex.Unlock()

	s.broadcastBatch(batch)
}

// broadcastBatch sends a batch of flows to all connected WebSocket clients
func (s *Server) broadcastBatch(flows []*protocol.Flow) {
	if len(flows) == 0 {
		return
	}

	message := map[string]interface{}{
		"type":  "batch",
		"flows": flows,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal batch: %v", err)
		return
	}

	s.wsMutex.Lock()
	defer s.wsMutex.Unlock()

	for conn := range s.wsClients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("WebSocket write error: %v", err)
			conn.Close()
			delete(s.wsClients, conn)
		}
	}
}

// AddPCAPData adds raw PCAP data from an agent
func (s *Server) AddPCAPData(agentID string, data []byte) error {
	// Check if paused - don't store PCAP data when paused
	s.pausedMutex.RLock()
	paused := s.paused
	s.pausedMutex.RUnlock()

	if paused {
		return nil // Silently drop PCAP data when paused
	}

	return s.pcapBuffer.Write(agentID, data)
}

// initK8sClient initializes the Kubernetes client for terminal exec
func (s *Server) initK8sClient() error {
	if s.k8sClient != nil {
		return nil // Already initialized
	}

	// Try in-cluster config first (when running inside k8s)
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Warning: Failed to get in-cluster config: %v", err)
		return fmt.Errorf("kubernetes client not available: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	s.k8sClient = clientset
	s.k8sRestConfig = config
	log.Println("Kubernetes client initialized for terminal support")
	return nil
}

// getAgentContainer finds the podscope agent ephemeral container in a pod
func (s *Server) getAgentContainer(ctx context.Context, namespace, podName string) (string, error) {
	pod, err := s.k8sClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	for _, ec := range pod.Spec.EphemeralContainers {
		if strings.HasPrefix(ec.Name, "podscope-agent") {
			return ec.Name, nil
		}
	}

	return "", fmt.Errorf("no podscope agent container found in pod %s/%s", namespace, podName)
}

// handleTerminalWebSocket handles WebSocket connections for terminal exec
func (s *Server) handleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("Terminal WebSocket request received: %s", r.URL.String())

	// Initialize k8s client if needed
	if err := s.initK8sClient(); err != nil {
		log.Printf("ERROR: Terminal k8s client init failed: %v", err)
		http.Error(w, fmt.Sprintf("Terminal not available: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Get query parameters
	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("pod")
	container := r.URL.Query().Get("container")

	log.Printf("Terminal request: namespace=%s pod=%s container=%s", namespace, podName, container)

	if namespace == "" || podName == "" {
		http.Error(w, "namespace and pod parameters required", http.StatusBadRequest)
		return
	}

	// If no container specified, find the agent container
	if container == "" {
		log.Printf("Looking for agent container in pod %s/%s...", namespace, podName)
		var err error
		container, err = s.getAgentContainer(r.Context(), namespace, podName)
		if err != nil {
			log.Printf("ERROR: Failed to find agent container: %v", err)
			http.Error(w, fmt.Sprintf("failed to find agent container: %v", err), http.StatusNotFound)
			return
		}
		log.Printf("Found agent container: %s", container)
	}

	log.Printf("Upgrading to WebSocket for terminal session...")
	// Upgrade to WebSocket
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("Terminal session started: %s/%s container=%s", namespace, podName, container)

	// Create the exec request
	log.Printf("Creating exec request for /bin/sh...")
	req := s.k8sClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{"/bin/sh"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	log.Printf("Creating SPDY executor for URL: %s", req.URL())
	exec, err := remotecommand.NewSPDYExecutor(s.k8sRestConfig, http.MethodPost, req.URL())
	if err != nil {
		log.Printf("Failed to create SPDY executor: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nFailed to create executor: %v\r\n", err)))
		return
	}

	// Create terminal bridge
	log.Printf("Creating WebSocket terminal bridge...")
	terminal := NewWebSocketTerminal(conn)
	defer terminal.Close()

	// Set initial terminal size
	terminal.sizeCh <- remotecommand.TerminalSize{Width: 80, Height: 24}

	// Execute and stream
	log.Printf("Starting exec stream...")
	ctx := r.Context()
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             terminal,
		Stdout:            terminal,
		Stderr:            terminal,
		Tty:               true,
		TerminalSizeQueue: terminal,
	})

	if err != nil {
		log.Printf("Terminal exec error: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nSession ended: %v\r\n", err)))
	}

	log.Printf("Terminal session ended: %s/%s", namespace, podName)
}

// handleAnthropicProxy proxies requests to the Anthropic API to avoid CORS issues
func (s *Server) handleAnthropicProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		APIKey  string `json:"apiKey"`
		System  string `json:"system"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.APIKey == "" || req.Message == "" {
		http.Error(w, "Missing apiKey or message", http.StatusBadRequest)
		return
	}

	// Build Anthropic API request
	anthropicReq := map[string]interface{}{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 100,
		"system":     req.System,
		"messages": []map[string]string{
			{"role": "user", "content": req.Message},
		},
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	// Make request to Anthropic API
	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", strings.NewReader(string(body)))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", req.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("Anthropic API error: %v", err)
		http.Error(w, fmt.Sprintf("API request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Anthropic API returned %d: %s", resp.StatusCode, string(respBody))
		http.Error(w, fmt.Sprintf("Anthropic API error: %s", string(respBody)), resp.StatusCode)
		return
	}

	// Parse the response to extract content
	var anthropicResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		http.Error(w, "Failed to parse API response", http.StatusInternalServerError)
		return
	}

	// Extract text content
	content := ""
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			content = c.Text
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"content": content})
}

package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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
	httpPort    int
	grpcPort    int
	sessionID   string
	pcapDir     string

	// Flow storage
	flows      []*protocol.Flow
	flowsMutex sync.RWMutex

	// WebSocket clients
	wsClients   map[*websocket.Conn]bool
	wsMutex     sync.Mutex
	wsUpgrader  websocket.Upgrader

	// PCAP storage
	pcapBuffer  *PCAPBuffer

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

	return &Server{
		httpPort:  httpPort,
		grpcPort:  grpcPort,
		sessionID: sessionID,
		pcapDir:   pcapDir,
		flows:     make([]*protocol.Flow, 0),
		wsClients: make(map[*websocket.Conn]bool),
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for local development
			},
		},
		pcapBuffer: NewPCAPBuffer(pcapDir, 100*1024*1024), // 100MB buffer
	}
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
	mux.HandleFunc("/api/pcap/", s.handleDownloadStreamPCAP)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/terminal/ws", s.handleTerminalWebSocket)

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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"sessionId": s.sessionID,
		"timestamp": time.Now().UTC(),
	})
}

// handleFlows handles GET (list flows) and POST (add flow)
func (s *Server) handleFlows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.flowsMutex.RLock()
		defer s.flowsMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"flows": s.flows,
			"count": len(s.flows),
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

	// Send existing flows
	s.flowsMutex.RLock()
	for _, flow := range s.flows {
		if err := conn.WriteJSON(flow); err != nil {
			s.flowsMutex.RUnlock()
			return
		}
	}
	s.flowsMutex.RUnlock()

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
	s.flowsMutex.RLock()
	flowCount := len(s.flows)
	s.flowsMutex.RUnlock()

	s.wsMutex.Lock()
	clientCount := len(s.wsClients)
	s.wsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"flows":        flowCount,
		"wsClients":    clientCount,
		"pcapSize":     s.pcapBuffer.Size(),
		"sessionId":    s.sessionID,
		"uptime":       time.Now().UTC(),
	})
}

// AddFlow adds a new flow and broadcasts it to WebSocket clients
func (s *Server) AddFlow(flow *protocol.Flow) {
	s.flowsMutex.Lock()
	s.flows = append(s.flows, flow)
	s.flowsMutex.Unlock()

	// Broadcast to WebSocket clients
	s.broadcastFlow(flow)
}

// broadcastFlow sends a flow to all connected WebSocket clients
func (s *Server) broadcastFlow(flow *protocol.Flow) {
	s.wsMutex.Lock()
	defer s.wsMutex.Unlock()

	for conn := range s.wsClients {
		if err := conn.WriteJSON(flow); err != nil {
			log.Printf("WebSocket write error: %v", err)
			conn.Close()
			delete(s.wsClients, conn)
		}
	}
}

// AddPCAPData adds raw PCAP data from an agent
func (s *Server) AddPCAPData(agentID string, data []byte) error {
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
	// Initialize k8s client if needed
	if err := s.initK8sClient(); err != nil {
		http.Error(w, fmt.Sprintf("Terminal not available: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Get query parameters
	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("pod")
	container := r.URL.Query().Get("container")

	if namespace == "" || podName == "" {
		http.Error(w, "namespace and pod parameters required", http.StatusBadRequest)
		return
	}

	// If no container specified, find the agent container
	if container == "" {
		var err error
		container, err = s.getAgentContainer(r.Context(), namespace, podName)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to find agent container: %v", err), http.StatusNotFound)
			return
		}
	}

	// Upgrade to WebSocket
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("Terminal session started: %s/%s container=%s", namespace, podName, container)

	// Create the exec request
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

	exec, err := remotecommand.NewSPDYExecutor(s.k8sRestConfig, http.MethodPost, req.URL())
	if err != nil {
		log.Printf("Failed to create SPDY executor: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nFailed to create executor: %v\r\n", err)))
		return
	}

	// Create terminal bridge
	terminal := NewWebSocketTerminal(conn)
	defer terminal.Close()

	// Set initial terminal size
	terminal.sizeCh <- remotecommand.TerminalSize{Width: 80, Height: 24}

	// Execute and stream
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

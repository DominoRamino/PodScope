package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/podscope/podscope/pkg/agent"
	"github.com/podscope/podscope/pkg/protocol"
)

func main() {
	// Parse configuration from environment
	hubAddress := os.Getenv("HUB_ADDRESS")
	if hubAddress == "" {
		hubAddress = "localhost:9090"
	}

	podName := os.Getenv("POD_NAME")
	podNamespace := os.Getenv("POD_NAMESPACE")
	podIP := os.Getenv("POD_IP")
	sessionID := os.Getenv("SESSION_ID")
	iface := os.Getenv("INTERFACE")
	if iface == "" {
		iface = "eth0"
	}

	// Generate agent ID
	agentID := uuid.New().String()[:8]

	log.Printf("PodScope Agent starting...")
	log.Printf("  Agent ID: %s", agentID)
	log.Printf("  Session: %s", sessionID)
	log.Printf("  Pod: %s/%s (%s)", podNamespace, podName, podIP)
	log.Printf("  Interface: %s", iface)
	log.Printf("  Hub: %s", hubAddress)

	// Create agent info
	agentInfo := &protocol.AgentInfo{
		ID:        agentID,
		PodName:   podName,
		Namespace: podNamespace,
		PodIP:     podIP,
	}

	// Create Hub client
	hubClient := agent.NewHubClient(hubAddress, agentInfo)

	// Connect to Hub with retry
	var connected bool
	for i := 0; i < 30; i++ {
		if err := hubClient.Connect(); err != nil {
			log.Printf("Failed to connect to Hub (attempt %d/30): %v", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		connected = true
		break
	}

	if !connected {
		log.Fatal("Failed to connect to Hub after 30 attempts")
	}

	defer hubClient.Close()

	// Create capturer
	capturer := agent.NewCapturer(iface, agentInfo, hubClient)

	// Set BPF filter to exclude traffic to the Hub (prevent feedback loop)
	bpfFilter := buildHubExclusionFilter(hubAddress)
	if bpfFilter != "" {
		log.Printf("  BPF Filter: %s", bpfFilter)
		capturer.SetBPFFilter(bpfFilter)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Start capture
	log.Println("Starting packet capture...")
	if err := capturer.Start(ctx); err != nil {
		log.Fatalf("Capture error: %v", err)
	}

	// Print final stats
	stats := capturer.Stats()
	log.Printf("Capture complete. Stats:")
	log.Printf("  Packets: %d", stats.PacketsCaptured)
	log.Printf("  Bytes: %d", stats.BytesCaptured)
	log.Printf("  TCP: %d", stats.TCPPackets)
	log.Printf("  HTTP: %d", stats.HTTPRequests)
	log.Printf("  TLS: %d", stats.TLSHandshakes)
}

// buildHubExclusionFilter creates a BPF filter to exclude traffic to/from the Hub
func buildHubExclusionFilter(hubAddress string) string {
	// Parse the hub address to get host and port
	// Hub address format: "podscope-hub.namespace.svc.cluster.local:9090"
	// But we actually connect to port 8080 for HTTP

	host := hubAddress
	if idx := strings.LastIndex(hubAddress, ":"); idx != -1 {
		host = hubAddress[:idx]
	}

	// Resolve the hub hostname to IP
	ips, err := net.LookupIP(host)
	if err != nil {
		log.Printf("Warning: could not resolve hub hostname %s: %v", host, err)
		// Fallback: exclude traffic to common podscope ports
		return "not (port 8080 or port 9090)"
	}

	if len(ips) == 0 {
		log.Printf("Warning: no IPs found for hub hostname %s", host)
		return "not (port 8080 or port 9090)"
	}

	hubIP := ips[0].String()
	log.Printf("  Hub IP: %s", hubIP)

	// Exclude all traffic to/from the hub IP on ports 8080 and 9090
	return fmt.Sprintf("not (host %s and (port 8080 or port 9090))", hubIP)
}

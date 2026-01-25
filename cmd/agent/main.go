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

var (
	// Version information - set at build time via ldflags
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
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

	log.Printf("====================================")
	log.Printf("PodScope Agent starting...")
	log.Printf("  Version: %s", Version)
	log.Printf("  Built: %s", BuildDate)
	log.Printf("  Commit: %s", GitCommit)
	log.Printf("====================================")
	log.Printf("  Agent ID: %s", agentID)
	log.Printf("  Session: %s", sessionID)
	log.Printf("  Pod: %s/%s", podNamespace, podName)
	log.Printf("  Pod IP: %q", podIP) // Use %q to show if empty
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

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set disconnect callback to trigger graceful shutdown when hub goes away
	hubClient.SetOnDisconnect(func() {
		log.Println("Hub disconnected, initiating graceful shutdown...")
		cancel()
	})

	// Create capturer
	capturer := agent.NewCapturer(iface, agentInfo, hubClient)

	// Link capturer to hub client for dynamic BPF filter updates
	hubClient.SetCapturer(capturer)

	// Set BPF filter to exclude agent->Hub traffic only (prevent feedback loop)
	// Uses source IP constraint to avoid filtering legitimate pod traffic to other 8080/9090 services
	bpfFilter, hubIP := buildHubExclusionFilter(hubAddress, podIP)
	if bpfFilter != "" {
		log.Printf("====================================")
		log.Printf("APPLYING BPF FILTER:")
		log.Printf("  %s", bpfFilter)
		log.Printf("====================================")
		capturer.SetBPFFilter(bpfFilter)
	} else {
		log.Printf("WARNING: No BPF filter set! All traffic will be captured!")
	}

	// Set hub IP for agent traffic tagging in assembler
	if hubIP != "" {
		capturer.SetHubIP(hubIP)
		log.Printf("  Hub IP set for agent traffic tagging: %s", hubIP)
	}

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

// buildHubExclusionFilter creates a BPF filter to exclude agent->Hub traffic.
// It uses source IP constraint to avoid filtering legitimate pod traffic to other services on 8080/9090.
// Returns the BPF filter string and the resolved Hub IP (for use in flow tagging).
func buildHubExclusionFilter(hubAddress, podIP string) (filter string, hubIP string) {
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
		// Minimal fallback - exclude traffic to/from pod IP on agent ports
		// This is less precise (may filter legitimate traffic) but prevents feedback loop
		if podIP != "" {
			log.Printf("  Using pod IP constraint for fallback filter")
			return fmt.Sprintf("not (host %s and (tcp port 8080 or tcp port 9090))", podIP), ""
		}
		// Capture everything if we can't identify agent traffic
		log.Printf("  No pod IP available, capturing all traffic")
		return "", ""
	}

	if len(ips) == 0 {
		log.Printf("Warning: no IPs found for hub hostname %s", host)
		if podIP != "" {
			return fmt.Sprintf("not (host %s and (tcp port 8080 or tcp port 9090))", podIP), ""
		}
		return "", ""
	}

	hubIP = ips[0].String()
	log.Printf("  Hub IP: %s", hubIP)
	log.Printf("  Pod IP: %s", podIP)

	// Precise filter: exclude all traffic BETWEEN pod and hub on agent ports (both directions)
	// This preserves legitimate traffic from the target pod to other services on 8080/9090
	if podIP != "" {
		// Using "host A and host B" matches packets where either endpoint is A and the other is B
		// This covers both pod→hub and hub→pod directions
		filter = fmt.Sprintf("not (host %s and host %s and (tcp port 8080 or tcp port 9090))", podIP, hubIP)
		return filter, hubIP
	}

	// Fallback to current behavior if no podIP (less precise but still works)
	filter = fmt.Sprintf("not (host %s and (port 8080 or port 9090))", hubIP)
	return filter, hubIP
}

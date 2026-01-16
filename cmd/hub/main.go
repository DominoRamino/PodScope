package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/podscope/podscope/pkg/hub"
)

func main() {
	// Parse configuration from environment
	httpPort := getEnvInt("HTTP_PORT", 8080)
	grpcPort := getEnvInt("GRPC_PORT", 9090)

	// Create server
	server := hub.NewServer(httpPort, grpcPort)

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

	// Start server
	log.Println("Starting PodScope Hub...")
	if err := server.Start(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

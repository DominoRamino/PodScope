package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/podscope/podscope/pkg/k8s"
	"github.com/spf13/cobra"
)

// staleSessionMaxAge is how old a session must be before it's considered stale
// and eligible for automatic cleanup. Set to 1 hour to avoid cleaning up
// sessions that are still in use but just older.
const staleSessionMaxAge = 1 * time.Hour

var (
	namespace        string
	labelSelector    string
	podName          string
	allNamespaces    bool
	forcePrivileged  bool
	hubPort          int
	uiPort           int
	targetContainer  string
	anthropicAPIKey  string
)

var tapCmd = &cobra.Command{
	Use:   "tap",
	Short: "Start capturing traffic from pods",
	Long: `Start a capture session targeting specific pods.

Examples:
  # Capture from all pods with label app=frontend in default namespace
  podscope tap -n default -l app=frontend

  # Capture from a specific pod
  podscope tap -n default --pod my-pod-abc123

  # Capture from all pods in all namespaces (requires cluster-wide permissions)
  podscope tap -A`,
	RunE: runTap,
}

func init() {
	tapCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Target namespace")
	tapCmd.Flags().StringVarP(&labelSelector, "selector", "l", "", "Label selector to filter pods (e.g., app=frontend)")
	tapCmd.Flags().StringVar(&podName, "pod", "", "Specific pod name to target")
	tapCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Target all namespaces")
	tapCmd.Flags().BoolVar(&forcePrivileged, "force-privileged", false, "Force privileged mode for the capture agent")
	tapCmd.Flags().IntVar(&hubPort, "hub-port", 8080, "Port for the Hub gRPC server")
	tapCmd.Flags().IntVar(&uiPort, "ui-port", 8899, "Local port for the UI (via port-forward)")
	tapCmd.Flags().StringVarP(&targetContainer, "target", "t", "", "Container to share process namespace with (defaults to first container)")
	tapCmd.Flags().StringVar(&anthropicAPIKey, "anthropic-api-key", "", "Anthropic API key for AI features (can also use ANTHROPIC_API_KEY env var)")
}

func runTap(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling with context cancellation
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Signal handler goroutine - cancels context on first signal, force exits on second
	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived %v, shutting down gracefully...\n", sig)
		fmt.Println("(Press Ctrl+C again to force exit)")
		cancel()

		// Second signal = force exit
		<-sigChan
		fmt.Println("\nForce exit!")
		os.Exit(1)
	}()

	fmt.Println("Starting PodScope capture session...")

	// Create Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Clean up any stale sessions from previous ungraceful exits
	if cleaned, err := k8sClient.CleanupStaleNamespaces(ctx, staleSessionMaxAge); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup stale sessions: %v\n", err)
	} else if cleaned > 0 {
		fmt.Printf("Cleaned up %d stale session(s)\n", cleaned)
	}

	// Clean up orphaned RBAC resources from sessions where namespace was deleted but RBAC remained
	if err := k8sClient.CleanupOrphanedRBAC(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup orphaned RBAC resources: %v\n", err)
	}

	// Resolve Anthropic API key (flag takes precedence over env var)
	apiKey := anthropicAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	// Create session manager with options
	sessionOpts := k8s.SessionOptions{
		AnthropicAPIKey: apiKey,
	}
	session, err := k8s.NewSession(k8sClient, sessionOpts)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Cleanup on exit with timeout
	defer func() {
		fmt.Println("\nCleaning up session resources...")
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()

		if err := session.Cleanup(cleanupCtx); err != nil {
			if cleanupCtx.Err() == context.DeadlineExceeded {
				fmt.Fprintf(os.Stderr, "Warning: cleanup timed out after 30 seconds\n")
			} else {
				fmt.Fprintf(os.Stderr, "Warning: cleanup failed: %v\n", err)
			}
		}
		fmt.Println("Cleanup complete.")
	}()

	// Start the session (creates namespace, deploys hub)
	fmt.Println("Creating session namespace and deploying Hub...")
	if err := session.Start(ctx); err != nil {
		if ctx.Err() == context.Canceled {
			return nil // User requested shutdown
		}
		return fmt.Errorf("failed to start session: %w", err)
	}

	// Find target pods
	var targetPods []k8s.PodTarget
	if podName != "" {
		targetPods, err = k8sClient.GetPodByName(ctx, namespace, podName)
	} else if labelSelector != "" {
		if allNamespaces {
			targetPods, err = k8sClient.GetPodsBySelector(ctx, "", labelSelector)
		} else {
			targetPods, err = k8sClient.GetPodsBySelector(ctx, namespace, labelSelector)
		}
	} else {
		return fmt.Errorf("must specify either --pod or -l/--selector")
	}

	if err != nil {
		return fmt.Errorf("failed to find target pods: %w", err)
	}

	if len(targetPods) == 0 {
		return fmt.Errorf("no pods found matching criteria")
	}

	fmt.Printf("Found %d target pod(s):\n", len(targetPods))
	for _, pod := range targetPods {
		fmt.Printf("  - %s/%s\n", pod.Namespace, pod.Name)
	}

	// Inject capture agents into target pods
	fmt.Println("\nInjecting capture agents...")
	for _, pod := range targetPods {
		// Check if shutdown was requested
		if ctx.Err() != nil {
			return nil
		}
		if err := session.InjectAgent(ctx, pod, forcePrivileged, targetContainer); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to inject agent into %s/%s: %v\n",
				pod.Namespace, pod.Name, err)
			continue
		}
		fmt.Printf("  Injected agent into %s/%s\n", pod.Namespace, pod.Name)
	}

	// Start port-forward to Hub
	fmt.Printf("\nStarting port-forward to Hub UI on localhost:%d...\n", uiPort)
	activePort, err := session.StartPortForward(ctx, uiPort)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil // User requested shutdown
		}
		return fmt.Errorf("failed to start port-forward: %w", err)
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("PodScope is running!\n")
	fmt.Printf("UI available at: http://localhost:%d\n", activePort)
	fmt.Printf("Press Ctrl+C to stop and cleanup\n")
	fmt.Printf("========================================\n\n")

	// Wait for context cancellation (triggered by signal handler)
	<-ctx.Done()

	return nil
}

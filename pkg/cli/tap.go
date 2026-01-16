package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/podscope/podscope/pkg/k8s"
	"github.com/spf13/cobra"
)

var (
	namespace       string
	labelSelector   string
	podName         string
	allNamespaces   bool
	forcePrivileged bool
	hubPort         int
	uiPort          int
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
}

func runTap(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for cleanup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Starting PodScope capture session...")

	// Create Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create session manager
	session, err := k8s.NewSession(k8sClient)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Cleanup on exit
	defer func() {
		fmt.Println("\nCleaning up session resources...")
		if err := session.Cleanup(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cleanup failed: %v\n", err)
		}
		fmt.Println("Cleanup complete.")
	}()

	// Start the session (creates namespace, deploys hub)
	fmt.Println("Creating session namespace and deploying Hub...")
	if err := session.Start(ctx); err != nil {
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
		if err := session.InjectAgent(ctx, pod, forcePrivileged); err != nil {
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
		return fmt.Errorf("failed to start port-forward: %w", err)
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("PodScope is running!\n")
	fmt.Printf("UI available at: http://localhost:%d\n", activePort)
	fmt.Printf("Press Ctrl+C to stop and cleanup\n")
	fmt.Printf("========================================\n\n")

	// Wait for interrupt
	select {
	case <-sigChan:
		fmt.Println("\nReceived interrupt signal...")
	case <-ctx.Done():
	}

	return nil
}

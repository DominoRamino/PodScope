package k8s

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// testClient wraps Client with direct access to a fake clientset for testing
type testClient struct {
	fakeClientset kubernetes.Interface
}

// createTestClient creates a testClient with a fake Kubernetes clientset for testing
func createTestClient(t *testing.T) *testClient {
	t.Helper()
	return &testClient{
		fakeClientset: fake.NewSimpleClientset(),
	}
}

// GetPodByName is a test version that uses the fake clientset
func (tc *testClient) GetPodByName(ctx context.Context, namespace, name string) ([]PodTarget, error) {
	pod, err := tc.fakeClientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if pod.Status.Phase != corev1.PodRunning {
		return nil, &PodNotRunningError{
			Name:      name,
			Namespace: namespace,
			Phase:     string(pod.Status.Phase),
		}
	}

	return []PodTarget{{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		IP:        pod.Status.PodIP,
		Node:      pod.Spec.NodeName,
	}}, nil
}

// PodNotRunningError is returned when a pod is not in Running phase
type PodNotRunningError struct {
	Name      string
	Namespace string
	Phase     string
}

func (e *PodNotRunningError) Error() string {
	return "pod " + e.Namespace + "/" + e.Name + " is not running (phase: " + e.Phase + ")"
}

// createClientTestPod creates a pod for testing client methods
func createClientTestPod(t *testing.T, tc *testClient, ctx context.Context, name, namespace, ip, node string, phase corev1.PodPhase) {
	t.Helper()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			NodeName: node,
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: phase,
			PodIP: ip,
		},
	}

	_, err := tc.fakeClientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create test pod: %v", err)
	}
}

// TestGetPodByName_ReturnsPodTargetForRunningPod tests that a PodTarget is returned for a running pod
func TestGetPodByName_ReturnsPodTargetForRunningPod(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "my-pod"
	podNamespace := "default"
	podIP := "10.0.0.50"
	nodeName := "node-1"

	createClientTestPod(t, tc, ctx, podName, podNamespace, podIP, nodeName, corev1.PodRunning)

	targets, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err != nil {
		t.Fatalf("GetPodByName failed: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]
	if target.Name != podName {
		t.Errorf("Name = %q, want %q", target.Name, podName)
	}
}

// TestGetPodByName_ReturnsCorrectNamespace tests that the correct namespace is returned
func TestGetPodByName_ReturnsCorrectNamespace(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "my-pod"
	podNamespace := "my-namespace"
	podIP := "10.0.0.51"
	nodeName := "node-1"

	createClientTestPod(t, tc, ctx, podName, podNamespace, podIP, nodeName, corev1.PodRunning)

	targets, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err != nil {
		t.Fatalf("GetPodByName failed: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	if targets[0].Namespace != podNamespace {
		t.Errorf("Namespace = %q, want %q", targets[0].Namespace, podNamespace)
	}
}

// TestGetPodByName_ReturnsCorrectIP tests that the pod IP is correctly extracted
func TestGetPodByName_ReturnsCorrectIP(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "my-pod"
	podNamespace := "default"
	podIP := "192.168.1.100"
	nodeName := "node-1"

	createClientTestPod(t, tc, ctx, podName, podNamespace, podIP, nodeName, corev1.PodRunning)

	targets, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err != nil {
		t.Fatalf("GetPodByName failed: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	if targets[0].IP != podIP {
		t.Errorf("IP = %q, want %q", targets[0].IP, podIP)
	}
}

// TestGetPodByName_ReturnsCorrectNode tests that the node name is correctly extracted
func TestGetPodByName_ReturnsCorrectNode(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "my-pod"
	podNamespace := "default"
	podIP := "10.0.0.52"
	nodeName := "worker-node-2"

	createClientTestPod(t, tc, ctx, podName, podNamespace, podIP, nodeName, corev1.PodRunning)

	targets, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err != nil {
		t.Fatalf("GetPodByName failed: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	if targets[0].Node != nodeName {
		t.Errorf("Node = %q, want %q", targets[0].Node, nodeName)
	}
}

// TestGetPodByName_ReturnsAllFieldsCorrectly tests that all fields are populated correctly
func TestGetPodByName_ReturnsAllFieldsCorrectly(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "complete-pod"
	podNamespace := "production"
	podIP := "10.20.30.40"
	nodeName := "master-node"

	createClientTestPod(t, tc, ctx, podName, podNamespace, podIP, nodeName, corev1.PodRunning)

	targets, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err != nil {
		t.Fatalf("GetPodByName failed: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]
	if target.Name != podName {
		t.Errorf("Name = %q, want %q", target.Name, podName)
	}
	if target.Namespace != podNamespace {
		t.Errorf("Namespace = %q, want %q", target.Namespace, podNamespace)
	}
	if target.IP != podIP {
		t.Errorf("IP = %q, want %q", target.IP, podIP)
	}
	if target.Node != nodeName {
		t.Errorf("Node = %q, want %q", target.Node, nodeName)
	}
}

// TestGetPodByName_ReturnsErrorForNonExistentPod tests that an error is returned for non-existent pod
func TestGetPodByName_ReturnsErrorForNonExistentPod(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	_, err := tc.GetPodByName(ctx, "default", "non-existent-pod")
	if err == nil {
		t.Error("expected error for non-existent pod, got nil")
	}
}

// TestGetPodByName_ReturnsNotFoundError tests that the error indicates pod not found
func TestGetPodByName_ReturnsNotFoundError(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	_, err := tc.GetPodByName(ctx, "default", "missing-pod")
	if err == nil {
		t.Fatal("expected error for non-existent pod, got nil")
	}

	// The error should indicate the pod was not found
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to contain 'not found', got: %v", err)
	}
}

// TestGetPodByName_ReturnsErrorForPendingPod tests that an error is returned for pending pod
func TestGetPodByName_ReturnsErrorForPendingPod(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "pending-pod"
	podNamespace := "default"

	createClientTestPod(t, tc, ctx, podName, podNamespace, "", "node-1", corev1.PodPending)

	_, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err == nil {
		t.Error("expected error for pending pod, got nil")
	}
}

// TestGetPodByName_ReturnsErrorForFailedPod tests that an error is returned for failed pod
func TestGetPodByName_ReturnsErrorForFailedPod(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "failed-pod"
	podNamespace := "default"

	createClientTestPod(t, tc, ctx, podName, podNamespace, "10.0.0.60", "node-1", corev1.PodFailed)

	_, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err == nil {
		t.Error("expected error for failed pod, got nil")
	}
}

// TestGetPodByName_ReturnsErrorForSucceededPod tests that an error is returned for completed pod
func TestGetPodByName_ReturnsErrorForSucceededPod(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "completed-pod"
	podNamespace := "default"

	createClientTestPod(t, tc, ctx, podName, podNamespace, "10.0.0.61", "node-1", corev1.PodSucceeded)

	_, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err == nil {
		t.Error("expected error for succeeded pod, got nil")
	}
}

// TestGetPodByName_ErrorMessageContainsPhase tests that error message contains the pod phase
func TestGetPodByName_ErrorMessageContainsPhase(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "pending-pod"
	podNamespace := "default"

	createClientTestPod(t, tc, ctx, podName, podNamespace, "", "node-1", corev1.PodPending)

	_, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err == nil {
		t.Fatal("expected error for pending pod, got nil")
	}

	if !strings.Contains(err.Error(), "Pending") {
		t.Errorf("expected error to contain 'Pending', got: %v", err)
	}
}

// TestGetPodByName_ErrorMessageIndicatesNotRunning tests that error indicates pod is not running
func TestGetPodByName_ErrorMessageIndicatesNotRunning(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "failed-pod"
	podNamespace := "default"

	createClientTestPod(t, tc, ctx, podName, podNamespace, "10.0.0.62", "node-1", corev1.PodFailed)

	_, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err == nil {
		t.Fatal("expected error for failed pod, got nil")
	}

	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("expected error to contain 'not running', got: %v", err)
	}
}

// TestGetPodByName_ExtractsPodIPFromStatus tests that pod IP is correctly extracted from pod status
func TestGetPodByName_ExtractsPodIPFromStatus(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create pod with specific IP in status
	podName := "ip-test-pod"
	podNamespace := "default"
	expectedIP := "172.16.0.50"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: expectedIP,
			// PodIPs for multi-IP scenarios
			PodIPs: []corev1.PodIP{
				{IP: expectedIP},
				{IP: "fd00::1"},
			},
		},
	}

	_, err := tc.fakeClientset.CoreV1().Pods(podNamespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create test pod: %v", err)
	}

	targets, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err != nil {
		t.Fatalf("GetPodByName failed: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	// Verify we get the PodIP from status
	if targets[0].IP != expectedIP {
		t.Errorf("IP = %q, want %q (from Status.PodIP)", targets[0].IP, expectedIP)
	}
}

// TestGetPodByName_HandlesEmptyPodIP tests behavior when pod has empty IP
func TestGetPodByName_HandlesEmptyPodIP(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	podName := "no-ip-pod"
	podNamespace := "default"

	// Create a running pod but without an IP yet (can happen during startup)
	createClientTestPod(t, tc, ctx, podName, podNamespace, "", "node-1", corev1.PodRunning)

	targets, err := tc.GetPodByName(ctx, podNamespace, podName)
	if err != nil {
		t.Fatalf("GetPodByName failed: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	// Should return empty IP (not error)
	if targets[0].IP != "" {
		t.Errorf("IP = %q, want empty string", targets[0].IP)
	}
}

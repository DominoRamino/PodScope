package k8s

import (
	"context"
	"strings"
	"testing"
	"time"

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

// GetPodsBySelector is a test version that uses the fake clientset
func (tc *testClient) GetPodsBySelector(ctx context.Context, namespace, selector string) ([]PodTarget, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}

	var pods *corev1.PodList
	var err error

	if namespace == "" {
		pods, err = tc.fakeClientset.CoreV1().Pods("").List(ctx, listOptions)
	} else {
		pods, err = tc.fakeClientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	}

	if err != nil {
		return nil, err
	}

	var targets []PodTarget
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			targets = append(targets, PodTarget{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				IP:        pod.Status.PodIP,
				Node:      pod.Spec.NodeName,
			})
		}
	}

	return targets, nil
}

// createClientTestPodWithLabels creates a pod with labels for testing selector methods
func createClientTestPodWithLabels(t *testing.T, tc *testClient, ctx context.Context, name, namespace, ip, node string, phase corev1.PodPhase, labels map[string]string) {
	t.Helper()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
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

// TestGetPodsBySelector_ReturnsMatchingPods tests that pods matching the label selector are returned
func TestGetPodsBySelector_ReturnsMatchingPods(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"app": "nginx", "env": "prod"}
	createClientTestPodWithLabels(t, tc, ctx, "nginx-1", "default", "10.0.0.1", "node-1", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "nginx-2", "default", "10.0.0.2", "node-2", corev1.PodRunning, labels)

	targets, err := tc.GetPodsBySelector(ctx, "default", "app=nginx")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}
}

// TestGetPodsBySelector_ReturnsAllMatchingFields tests that all PodTarget fields are populated
func TestGetPodsBySelector_ReturnsAllMatchingFields(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"app": "web"}
	createClientTestPodWithLabels(t, tc, ctx, "web-pod", "production", "192.168.1.50", "worker-1", corev1.PodRunning, labels)

	targets, err := tc.GetPodsBySelector(ctx, "production", "app=web")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	target := targets[0]
	if target.Name != "web-pod" {
		t.Errorf("Name = %q, want %q", target.Name, "web-pod")
	}
	if target.Namespace != "production" {
		t.Errorf("Namespace = %q, want %q", target.Namespace, "production")
	}
	if target.IP != "192.168.1.50" {
		t.Errorf("IP = %q, want %q", target.IP, "192.168.1.50")
	}
	if target.Node != "worker-1" {
		t.Errorf("Node = %q, want %q", target.Node, "worker-1")
	}
}

// TestGetPodsBySelector_EmptyNamespaceSearchesAllNamespaces tests that empty namespace searches across all namespaces
func TestGetPodsBySelector_EmptyNamespaceSearchesAllNamespaces(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"tier": "frontend"}

	// Create pods in different namespaces
	createClientTestPodWithLabels(t, tc, ctx, "frontend-1", "namespace-a", "10.0.0.1", "node-1", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "frontend-2", "namespace-b", "10.0.0.2", "node-2", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "frontend-3", "namespace-c", "10.0.0.3", "node-1", corev1.PodRunning, labels)

	// Search with empty namespace
	targets, err := tc.GetPodsBySelector(ctx, "", "tier=frontend")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if len(targets) != 3 {
		t.Errorf("expected 3 targets across all namespaces, got %d", len(targets))
	}
}

// TestGetPodsBySelector_EmptyNamespaceFindsPodsInMultipleNamespaces tests namespace diversity in results
func TestGetPodsBySelector_EmptyNamespaceFindsPodsInMultipleNamespaces(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"role": "api"}

	createClientTestPodWithLabels(t, tc, ctx, "api-1", "dev", "10.0.0.1", "node-1", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "api-2", "staging", "10.0.0.2", "node-2", corev1.PodRunning, labels)

	targets, err := tc.GetPodsBySelector(ctx, "", "role=api")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	// Verify we have pods from different namespaces
	namespaces := make(map[string]bool)
	for _, target := range targets {
		namespaces[target.Namespace] = true
	}

	if len(namespaces) != 2 {
		t.Errorf("expected pods from 2 namespaces, got %d", len(namespaces))
	}
}

// TestGetPodsBySelector_SpecificNamespaceLimitsScope tests that specifying namespace limits search
func TestGetPodsBySelector_SpecificNamespaceLimitsScope(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"component": "database"}

	// Create pods in different namespaces
	createClientTestPodWithLabels(t, tc, ctx, "db-1", "ns-target", "10.0.0.1", "node-1", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "db-2", "ns-other", "10.0.0.2", "node-2", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "db-3", "ns-target", "10.0.0.3", "node-1", corev1.PodRunning, labels)

	// Search only in ns-target
	targets, err := tc.GetPodsBySelector(ctx, "ns-target", "component=database")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if len(targets) != 2 {
		t.Errorf("expected 2 targets in ns-target, got %d", len(targets))
	}

	// Verify all are from the target namespace
	for _, target := range targets {
		if target.Namespace != "ns-target" {
			t.Errorf("expected namespace 'ns-target', got %q", target.Namespace)
		}
	}
}

// TestGetPodsBySelector_ExcludesPodsFromOtherNamespaces tests that other namespaces are excluded
func TestGetPodsBySelector_ExcludesPodsFromOtherNamespaces(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"app": "service"}

	createClientTestPodWithLabels(t, tc, ctx, "service-1", "included", "10.0.0.1", "node-1", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "service-2", "excluded", "10.0.0.2", "node-2", corev1.PodRunning, labels)

	targets, err := tc.GetPodsBySelector(ctx, "included", "app=service")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if len(targets) != 1 {
		t.Errorf("expected 1 target, got %d", len(targets))
	}

	if targets[0].Namespace != "included" {
		t.Errorf("expected namespace 'included', got %q", targets[0].Namespace)
	}
}

// TestGetPodsBySelector_FiltersOutPendingPods tests that pending pods are excluded
func TestGetPodsBySelector_FiltersOutPendingPods(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"app": "mixed"}

	createClientTestPodWithLabels(t, tc, ctx, "running-pod", "default", "10.0.0.1", "node-1", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "pending-pod", "default", "", "node-2", corev1.PodPending, labels)

	targets, err := tc.GetPodsBySelector(ctx, "default", "app=mixed")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if len(targets) != 1 {
		t.Errorf("expected 1 target (only running), got %d", len(targets))
	}

	if targets[0].Name != "running-pod" {
		t.Errorf("expected running-pod, got %q", targets[0].Name)
	}
}

// TestGetPodsBySelector_FiltersOutFailedPods tests that failed pods are excluded
func TestGetPodsBySelector_FiltersOutFailedPods(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"app": "batch"}

	createClientTestPodWithLabels(t, tc, ctx, "running-job", "default", "10.0.0.1", "node-1", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "failed-job", "default", "10.0.0.2", "node-2", corev1.PodFailed, labels)

	targets, err := tc.GetPodsBySelector(ctx, "default", "app=batch")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if len(targets) != 1 {
		t.Errorf("expected 1 target (only running), got %d", len(targets))
	}
}

// TestGetPodsBySelector_FiltersOutSucceededPods tests that completed pods are excluded
func TestGetPodsBySelector_FiltersOutSucceededPods(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"app": "job"}

	createClientTestPodWithLabels(t, tc, ctx, "running-job", "default", "10.0.0.1", "node-1", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "completed-job", "default", "10.0.0.2", "node-2", corev1.PodSucceeded, labels)

	targets, err := tc.GetPodsBySelector(ctx, "default", "app=job")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if len(targets) != 1 {
		t.Errorf("expected 1 target (only running), got %d", len(targets))
	}
}

// TestGetPodsBySelector_FiltersAllNonRunningPhases tests filtering of all non-running phases
func TestGetPodsBySelector_FiltersAllNonRunningPhases(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"app": "test"}

	// Create pods in various phases
	createClientTestPodWithLabels(t, tc, ctx, "running", "default", "10.0.0.1", "node-1", corev1.PodRunning, labels)
	createClientTestPodWithLabels(t, tc, ctx, "pending", "default", "", "node-2", corev1.PodPending, labels)
	createClientTestPodWithLabels(t, tc, ctx, "failed", "default", "10.0.0.3", "node-1", corev1.PodFailed, labels)
	createClientTestPodWithLabels(t, tc, ctx, "succeeded", "default", "10.0.0.4", "node-2", corev1.PodSucceeded, labels)
	createClientTestPodWithLabels(t, tc, ctx, "unknown", "default", "10.0.0.5", "node-1", corev1.PodUnknown, labels)

	targets, err := tc.GetPodsBySelector(ctx, "default", "app=test")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if len(targets) != 1 {
		t.Errorf("expected 1 target (only running), got %d", len(targets))
	}

	if targets[0].Name != "running" {
		t.Errorf("expected 'running' pod, got %q", targets[0].Name)
	}
}

// TestGetPodsBySelector_ReturnsEmptySliceWhenNoMatch tests that no match returns empty slice
func TestGetPodsBySelector_ReturnsEmptySliceWhenNoMatch(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create a pod with different labels
	labels := map[string]string{"app": "different"}
	createClientTestPodWithLabels(t, tc, ctx, "different-app", "default", "10.0.0.1", "node-1", corev1.PodRunning, labels)

	// Search for non-existent label
	targets, err := tc.GetPodsBySelector(ctx, "default", "app=nonexistent")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if targets == nil {
		// nil slice is acceptable for "no results"
		return
	}

	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

// TestGetPodsBySelector_ReturnsEmptySliceForEmptyNamespace tests empty result with no pods
func TestGetPodsBySelector_ReturnsEmptySliceForEmptyNamespace(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Don't create any pods - search should return empty
	targets, err := tc.GetPodsBySelector(ctx, "empty-namespace", "app=any")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if targets != nil && len(targets) != 0 {
		t.Errorf("expected empty slice, got %d targets", len(targets))
	}
}

// TestGetPodsBySelector_ReturnsEmptyWhenOnlyNonRunningPodsMatch tests that only non-running pods means empty result
func TestGetPodsBySelector_ReturnsEmptyWhenOnlyNonRunningPodsMatch(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	labels := map[string]string{"app": "pending-only"}

	// Create only non-running pods
	createClientTestPodWithLabels(t, tc, ctx, "pending-1", "default", "", "node-1", corev1.PodPending, labels)
	createClientTestPodWithLabels(t, tc, ctx, "pending-2", "default", "", "node-2", corev1.PodPending, labels)

	targets, err := tc.GetPodsBySelector(ctx, "default", "app=pending-only")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if targets != nil && len(targets) != 0 {
		t.Errorf("expected empty slice (no running pods), got %d targets", len(targets))
	}
}

// TestGetPodsBySelector_MultiLabelSelector tests complex label selectors
func TestGetPodsBySelector_MultiLabelSelector(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create pods with different label combinations
	createClientTestPodWithLabels(t, tc, ctx, "match-both", "default", "10.0.0.1", "node-1", corev1.PodRunning,
		map[string]string{"app": "web", "env": "prod"})
	createClientTestPodWithLabels(t, tc, ctx, "match-app-only", "default", "10.0.0.2", "node-2", corev1.PodRunning,
		map[string]string{"app": "web", "env": "dev"})
	createClientTestPodWithLabels(t, tc, ctx, "match-env-only", "default", "10.0.0.3", "node-1", corev1.PodRunning,
		map[string]string{"app": "api", "env": "prod"})

	// Search with multi-label selector
	targets, err := tc.GetPodsBySelector(ctx, "default", "app=web,env=prod")
	if err != nil {
		t.Fatalf("GetPodsBySelector failed: %v", err)
	}

	if len(targets) != 1 {
		t.Errorf("expected 1 target matching both labels, got %d", len(targets))
	}

	if len(targets) > 0 && targets[0].Name != "match-both" {
		t.Errorf("expected 'match-both' pod, got %q", targets[0].Name)
	}
}

// CleanupStaleSessions is a test version that uses the fake clientset
func (tc *testClient) CleanupStaleSessions(ctx context.Context, maxAge time.Duration) (int, error) {
	// List all namespaces with podscope label
	namespaces, err := tc.fakeClientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/managed-by=podscope-cli",
	})
	if err != nil {
		return 0, err
	}

	cleaned := 0
	now := time.Now().UTC()

	for _, ns := range namespaces.Items {
		// Check creation timestamp from annotation
		createdAtStr, ok := ns.Annotations["podscope.io/created-at"]
		if !ok {
			// Fallback to namespace creation time if annotation missing
			createdAtStr = ns.CreationTimestamp.Format(time.RFC3339)
		}

		createdAt, err := time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			continue
		}

		age := now.Sub(createdAt)
		if age > maxAge {
			err := tc.fakeClientset.CoreV1().Namespaces().Delete(ctx, ns.Name, metav1.DeleteOptions{})
			if err != nil {
				continue
			}
			cleaned++
		}
	}

	return cleaned, nil
}

// createPodscopeNamespace creates a namespace with podscope labels and annotations for testing
func createPodscopeNamespace(t *testing.T, tc *testClient, ctx context.Context, name string, createdAt time.Time) {
	t.Helper()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "podscope",
				"app.kubernetes.io/managed-by": "podscope-cli",
			},
			Annotations: map[string]string{
				"podscope.io/created-at": createdAt.Format(time.RFC3339),
			},
		},
	}

	_, err := tc.fakeClientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}
}

// createPodscopeNamespaceWithoutAnnotation creates a namespace with podscope labels but no created-at annotation
func createPodscopeNamespaceWithoutAnnotation(t *testing.T, tc *testClient, ctx context.Context, name string) {
	t.Helper()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "podscope",
				"app.kubernetes.io/managed-by": "podscope-cli",
			},
		},
	}

	_, err := tc.fakeClientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}
}

// createNonPodscopeNamespace creates a namespace without podscope labels
func createNonPodscopeNamespace(t *testing.T, tc *testClient, ctx context.Context, name string, createdAt time.Time) {
	t.Helper()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app.kubernetes.io/name": "other-app",
			},
			Annotations: map[string]string{
				"podscope.io/created-at": createdAt.Format(time.RFC3339),
			},
		},
	}

	_, err := tc.fakeClientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}
}

// TestCleanupStaleSessions_DeletesNamespacesOlderThanMaxAge tests that old namespaces are deleted
func TestCleanupStaleSessions_DeletesNamespacesOlderThanMaxAge(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create a namespace that is 2 hours old
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	createPodscopeNamespace(t, tc, ctx, "podscope-old-session", oldTime)

	// Cleanup namespaces older than 1 hour
	cleaned, err := tc.CleanupStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	if cleaned != 1 {
		t.Errorf("expected 1 namespace cleaned, got %d", cleaned)
	}

	// Verify namespace was deleted
	_, err = tc.fakeClientset.CoreV1().Namespaces().Get(ctx, "podscope-old-session", metav1.GetOptions{})
	if err == nil {
		t.Error("expected namespace to be deleted, but it still exists")
	}
}

// TestCleanupStaleSessions_KeepsNamespacesNewerThanMaxAge tests that new namespaces are kept
func TestCleanupStaleSessions_KeepsNamespacesNewerThanMaxAge(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create a namespace that is 30 minutes old
	newTime := time.Now().UTC().Add(-30 * time.Minute)
	createPodscopeNamespace(t, tc, ctx, "podscope-new-session", newTime)

	// Cleanup namespaces older than 1 hour
	cleaned, err := tc.CleanupStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	if cleaned != 0 {
		t.Errorf("expected 0 namespaces cleaned, got %d", cleaned)
	}

	// Verify namespace still exists
	ns, err := tc.fakeClientset.CoreV1().Namespaces().Get(ctx, "podscope-new-session", metav1.GetOptions{})
	if err != nil {
		t.Errorf("expected namespace to still exist, but got error: %v", err)
	}
	if ns == nil {
		t.Error("expected namespace to still exist, but it was nil")
	}
}

// TestCleanupStaleSessions_UsesCreatedAtAnnotationForAge tests that the annotation is used for age calculation
func TestCleanupStaleSessions_UsesCreatedAtAnnotationForAge(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create namespace with old annotation timestamp
	oldTime := time.Now().UTC().Add(-3 * time.Hour)
	createPodscopeNamespace(t, tc, ctx, "podscope-annotated-session", oldTime)

	// Cleanup namespaces older than 2 hours
	cleaned, err := tc.CleanupStaleSessions(ctx, 2*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	if cleaned != 1 {
		t.Errorf("expected 1 namespace cleaned based on annotation age, got %d", cleaned)
	}

	// Verify namespace was deleted
	_, err = tc.fakeClientset.CoreV1().Namespaces().Get(ctx, "podscope-annotated-session", metav1.GetOptions{})
	if err == nil {
		t.Error("expected namespace to be deleted based on annotation age")
	}
}

// TestCleanupStaleSessions_OnlyTargetsPodscopeNamespaces tests that non-podscope namespaces are ignored
func TestCleanupStaleSessions_OnlyTargetsPodscopeNamespaces(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create an old podscope namespace
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	createPodscopeNamespace(t, tc, ctx, "podscope-old", oldTime)

	// Create an old non-podscope namespace
	createNonPodscopeNamespace(t, tc, ctx, "other-app-old", oldTime)

	// Cleanup namespaces older than 1 hour
	cleaned, err := tc.CleanupStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	// Only the podscope namespace should be cleaned
	if cleaned != 1 {
		t.Errorf("expected 1 namespace cleaned (only podscope), got %d", cleaned)
	}

	// Verify podscope namespace was deleted
	_, err = tc.fakeClientset.CoreV1().Namespaces().Get(ctx, "podscope-old", metav1.GetOptions{})
	if err == nil {
		t.Error("expected podscope namespace to be deleted")
	}

	// Verify non-podscope namespace still exists
	ns, err := tc.fakeClientset.CoreV1().Namespaces().Get(ctx, "other-app-old", metav1.GetOptions{})
	if err != nil {
		t.Errorf("expected non-podscope namespace to still exist, got error: %v", err)
	}
	if ns == nil {
		t.Error("expected non-podscope namespace to still exist")
	}
}

// TestCleanupStaleSessions_ReturnsCountOfDeletedNamespaces tests that the correct count is returned
func TestCleanupStaleSessions_ReturnsCountOfDeletedNamespaces(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create multiple old namespaces
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	createPodscopeNamespace(t, tc, ctx, "podscope-session-1", oldTime)
	createPodscopeNamespace(t, tc, ctx, "podscope-session-2", oldTime)
	createPodscopeNamespace(t, tc, ctx, "podscope-session-3", oldTime)

	// Create one new namespace
	newTime := time.Now().UTC().Add(-10 * time.Minute)
	createPodscopeNamespace(t, tc, ctx, "podscope-session-new", newTime)

	// Cleanup namespaces older than 1 hour
	cleaned, err := tc.CleanupStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	if cleaned != 3 {
		t.Errorf("expected 3 namespaces cleaned, got %d", cleaned)
	}
}

// TestCleanupStaleSessions_ReturnsZeroWhenNoStaleNamespaces tests that 0 is returned when nothing to clean
func TestCleanupStaleSessions_ReturnsZeroWhenNoStaleNamespaces(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create only new namespaces
	newTime := time.Now().UTC().Add(-5 * time.Minute)
	createPodscopeNamespace(t, tc, ctx, "podscope-fresh-1", newTime)
	createPodscopeNamespace(t, tc, ctx, "podscope-fresh-2", newTime)

	// Cleanup namespaces older than 1 hour
	cleaned, err := tc.CleanupStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	if cleaned != 0 {
		t.Errorf("expected 0 namespaces cleaned, got %d", cleaned)
	}
}

// TestCleanupStaleSessions_HandlesEmptyNamespaceList tests behavior with no podscope namespaces
func TestCleanupStaleSessions_HandlesEmptyNamespaceList(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Don't create any namespaces

	cleaned, err := tc.CleanupStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	if cleaned != 0 {
		t.Errorf("expected 0 namespaces cleaned, got %d", cleaned)
	}
}

// TestCleanupStaleSessions_MixedAgesDeletesOnlyOld tests that only old namespaces are deleted in mixed scenarios
func TestCleanupStaleSessions_MixedAgesDeletesOnlyOld(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create namespaces with various ages
	veryOldTime := time.Now().UTC().Add(-24 * time.Hour)
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	newTime := time.Now().UTC().Add(-10 * time.Minute)

	createPodscopeNamespace(t, tc, ctx, "podscope-very-old", veryOldTime)
	createPodscopeNamespace(t, tc, ctx, "podscope-old", oldTime)
	createPodscopeNamespace(t, tc, ctx, "podscope-new", newTime)

	// Cleanup namespaces older than 1 hour
	cleaned, err := tc.CleanupStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	if cleaned != 2 {
		t.Errorf("expected 2 namespaces cleaned, got %d", cleaned)
	}

	// Verify the new namespace still exists
	ns, err := tc.fakeClientset.CoreV1().Namespaces().Get(ctx, "podscope-new", metav1.GetOptions{})
	if err != nil {
		t.Errorf("expected new namespace to still exist, got error: %v", err)
	}
	if ns == nil {
		t.Error("expected new namespace to still exist")
	}

	// Verify old namespaces were deleted
	_, err = tc.fakeClientset.CoreV1().Namespaces().Get(ctx, "podscope-very-old", metav1.GetOptions{})
	if err == nil {
		t.Error("expected very old namespace to be deleted")
	}
	_, err = tc.fakeClientset.CoreV1().Namespaces().Get(ctx, "podscope-old", metav1.GetOptions{})
	if err == nil {
		t.Error("expected old namespace to be deleted")
	}
}

// TestCleanupStaleSessions_UsesAppLabel tests that the correct label selector is used
func TestCleanupStaleSessions_UsesAppLabel(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create namespace with podscope label
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	createPodscopeNamespace(t, tc, ctx, "podscope-labeled", oldTime)

	// Create namespace with different managed-by label
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "different-managed-by",
			Labels: map[string]string{
				"app.kubernetes.io/name":       "podscope",
				"app.kubernetes.io/managed-by": "helm",
			},
			Annotations: map[string]string{
				"podscope.io/created-at": oldTime.Format(time.RFC3339),
			},
		},
	}
	_, err := tc.fakeClientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}

	// Cleanup should only target podscope-cli managed namespaces
	cleaned, err := tc.CleanupStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	if cleaned != 1 {
		t.Errorf("expected 1 namespace cleaned (only podscope-cli managed), got %d", cleaned)
	}

	// Verify the helm-managed namespace still exists
	helmNS, err := tc.fakeClientset.CoreV1().Namespaces().Get(ctx, "different-managed-by", metav1.GetOptions{})
	if err != nil {
		t.Errorf("expected helm-managed namespace to still exist, got error: %v", err)
	}
	if helmNS == nil {
		t.Error("expected helm-managed namespace to still exist")
	}
}

// TestCleanupStaleSessions_JustUnderMaxAgeBoundary tests namespace just under maxAge is kept
func TestCleanupStaleSessions_JustUnderMaxAgeBoundary(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create a namespace that is just under the maxAge boundary (should be kept)
	// Use 59 minutes to ensure it's clearly under the 1 hour boundary
	justUnderBoundaryTime := time.Now().UTC().Add(-59 * time.Minute)
	createPodscopeNamespace(t, tc, ctx, "podscope-just-under", justUnderBoundaryTime)

	// Cleanup namespaces older than 1 hour
	cleaned, err := tc.CleanupStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	// The namespace under 1 hour should NOT be deleted (age > maxAge check)
	if cleaned != 0 {
		t.Errorf("expected 0 namespaces cleaned just under boundary, got %d", cleaned)
	}

	// Verify namespace still exists
	ns, err := tc.fakeClientset.CoreV1().Namespaces().Get(ctx, "podscope-just-under", metav1.GetOptions{})
	if err != nil {
		t.Errorf("expected namespace to still exist, got error: %v", err)
	}
	if ns == nil {
		t.Error("expected namespace to still exist")
	}
}

// TestCleanupStaleSessions_JustOverMaxAgeBoundary tests namespace just over maxAge is deleted
func TestCleanupStaleSessions_JustOverMaxAgeBoundary(t *testing.T) {
	tc := createTestClient(t)
	ctx := context.Background()

	// Create a namespace that is just over the maxAge boundary (should be deleted)
	justOverBoundaryTime := time.Now().UTC().Add(-1*time.Hour - 1*time.Second)
	createPodscopeNamespace(t, tc, ctx, "podscope-just-over", justOverBoundaryTime)

	// Cleanup namespaces older than 1 hour
	cleaned, err := tc.CleanupStaleSessions(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	if cleaned != 1 {
		t.Errorf("expected 1 namespace cleaned just over boundary, got %d", cleaned)
	}
}

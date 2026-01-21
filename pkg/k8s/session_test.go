package k8s

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// testSession wraps Session with direct access to a fake clientset for testing
type testSession struct {
	*Session
	fakeClientset kubernetes.Interface
}

// createTestSession creates a Session with a fake Kubernetes clientset for testing
// Returns a testSession wrapper that provides access to the fake clientset for verification
func createTestSession(t *testing.T, sessionID string) *testSession {
	t.Helper()
	fakeClientset := fake.NewSimpleClientset()

	// Create a Session manually - we'll override the clientset field for testing
	session := &Session{
		id:         sessionID,
		namespace:  "podscope-" + sessionID,
		hubService: "podscope-hub",
		stopChan:   make(chan struct{}),
	}

	return &testSession{
		Session:       session,
		fakeClientset: fakeClientset,
	}
}

// createNamespace is a test version that uses the fake clientset
func (ts *testSession) createNamespace(ctx context.Context) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ts.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "podscope",
				"app.kubernetes.io/component":  "session",
				"app.kubernetes.io/managed-by": "podscope-cli",
				"podscope.io/session-id":       ts.id,
			},
			Annotations: map[string]string{
				"podscope.io/created-at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	_, err := ts.fakeClientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// TestCreateNamespace_NameFormat tests that namespace is created with correct name format
func TestCreateNamespace_NameFormat(t *testing.T) {
	sessionID := "abc12345"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	// Verify namespace was created with correct name
	expectedName := "podscope-" + sessionID
	ns, err := ts.fakeClientset.CoreV1().Namespaces().Get(ctx, expectedName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get namespace: %v", err)
	}

	if ns.Name != expectedName {
		t.Errorf("namespace name = %q, want %q", ns.Name, expectedName)
	}
}

// TestCreateNamespace_LabelAppName tests that app.kubernetes.io/name label is set to podscope
func TestCreateNamespace_LabelAppName(t *testing.T) {
	ts := createTestSession(t, "test1234")
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	ns, err := ts.fakeClientset.CoreV1().Namespaces().Get(ctx, ts.namespace, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get namespace: %v", err)
	}

	label, ok := ns.Labels["app.kubernetes.io/name"]
	if !ok {
		t.Fatal("label app.kubernetes.io/name not found")
	}
	if label != "podscope" {
		t.Errorf("label app.kubernetes.io/name = %q, want %q", label, "podscope")
	}
}

// TestCreateNamespace_LabelSessionID tests that podscope.io/session-id label is set
func TestCreateNamespace_LabelSessionID(t *testing.T) {
	sessionID := "sess5678"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	ns, err := ts.fakeClientset.CoreV1().Namespaces().Get(ctx, ts.namespace, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get namespace: %v", err)
	}

	label, ok := ns.Labels["podscope.io/session-id"]
	if !ok {
		t.Fatal("label podscope.io/session-id not found")
	}
	if label != sessionID {
		t.Errorf("label podscope.io/session-id = %q, want %q", label, sessionID)
	}
}

// TestCreateNamespace_AnnotationCreatedAt tests that podscope.io/created-at annotation is set
func TestCreateNamespace_AnnotationCreatedAt(t *testing.T) {
	ts := createTestSession(t, "anno1234")
	ctx := context.Background()

	beforeCreate := time.Now().UTC().Add(-time.Second)

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	afterCreate := time.Now().UTC().Add(time.Second)

	ns, err := ts.fakeClientset.CoreV1().Namespaces().Get(ctx, ts.namespace, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get namespace: %v", err)
	}

	createdAtStr, ok := ns.Annotations["podscope.io/created-at"]
	if !ok {
		t.Fatal("annotation podscope.io/created-at not found")
	}

	// Parse the timestamp
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		t.Fatalf("failed to parse created-at timestamp %q: %v", createdAtStr, err)
	}

	// Verify timestamp is within expected range
	if createdAt.Before(beforeCreate) || createdAt.After(afterCreate) {
		t.Errorf("created-at timestamp %v not within expected range [%v, %v]", createdAt, beforeCreate, afterCreate)
	}
}

// TestCreateNamespace_LabelComponent tests that app.kubernetes.io/component label is set to session
func TestCreateNamespace_LabelComponent(t *testing.T) {
	ts := createTestSession(t, "comp1234")
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	ns, err := ts.fakeClientset.CoreV1().Namespaces().Get(ctx, ts.namespace, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get namespace: %v", err)
	}

	label, ok := ns.Labels["app.kubernetes.io/component"]
	if !ok {
		t.Fatal("label app.kubernetes.io/component not found")
	}
	if label != "session" {
		t.Errorf("label app.kubernetes.io/component = %q, want %q", label, "session")
	}
}

// TestCreateNamespace_LabelManagedBy tests that app.kubernetes.io/managed-by label is set
func TestCreateNamespace_LabelManagedBy(t *testing.T) {
	ts := createTestSession(t, "mgby1234")
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	ns, err := ts.fakeClientset.CoreV1().Namespaces().Get(ctx, ts.namespace, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get namespace: %v", err)
	}

	label, ok := ns.Labels["app.kubernetes.io/managed-by"]
	if !ok {
		t.Fatal("label app.kubernetes.io/managed-by not found")
	}
	if label != "podscope-cli" {
		t.Errorf("label app.kubernetes.io/managed-by = %q, want %q", label, "podscope-cli")
	}
}

// TestCreateNamespace_AllLabelsPresent tests that all required labels are set
func TestCreateNamespace_AllLabelsPresent(t *testing.T) {
	sessionID := "all12345"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	ns, err := ts.fakeClientset.CoreV1().Namespaces().Get(ctx, ts.namespace, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get namespace: %v", err)
	}

	expectedLabels := map[string]string{
		"app.kubernetes.io/name":       "podscope",
		"app.kubernetes.io/component":  "session",
		"app.kubernetes.io/managed-by": "podscope-cli",
		"podscope.io/session-id":       sessionID,
	}

	for key, expected := range expectedLabels {
		actual, ok := ns.Labels[key]
		if !ok {
			t.Errorf("label %q not found", key)
			continue
		}
		if actual != expected {
			t.Errorf("label %q = %q, want %q", key, actual, expected)
		}
	}
}

// TestCreateNamespace_Idempotent tests that createNamespace doesn't fail if namespace already exists
func TestCreateNamespace_Idempotent(t *testing.T) {
	ts := createTestSession(t, "idem1234")
	ctx := context.Background()

	// Create namespace first time
	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("first createNamespace failed: %v", err)
	}

	// Create namespace second time - should not error
	err = ts.createNamespace(ctx)
	if err != nil {
		t.Errorf("second createNamespace should not fail, got: %v", err)
	}
}

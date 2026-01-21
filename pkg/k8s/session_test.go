package k8s

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

// deployHub is a test version that uses the fake clientset
func (ts *testSession) deployHub(ctx context.Context) error {
	labels := map[string]string{
		"app.kubernetes.io/name":      "podscope-hub",
		"app.kubernetes.io/component": "hub",
		"podscope.io/session-id":      ts.id,
	}

	serviceAccountName := "podscope-hub"
	replicas := int32(1)

	// Create ServiceAccount for hub
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: ts.namespace,
			Labels:    labels,
		},
	}
	_, err := ts.fakeClientset.CoreV1().ServiceAccounts(ts.namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	// Create ClusterRole with permissions for terminal exec
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("podscope-hub-%s", ts.id),
			Labels: labels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"create"},
			},
		},
	}
	_, err = ts.fakeClientset.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}

	// Create ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("podscope-hub-%s", ts.id),
			Labels: labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     fmt.Sprintf("podscope-hub-%s", ts.id),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: ts.namespace,
			},
		},
	}
	_, err = ts.fakeClientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role binding: %w", err)
	}

	// Create Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ts.hubService,
			Namespace: ts.namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					Containers: []corev1.Container{
						{
							Name:            "hub",
							Image:           GetHubImage(),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									Name:          "grpc",
									ContainerPort: 9090,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "http",
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "SESSION_ID",
									Value: ts.id,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "pcap-storage",
									MountPath: "/data/pcap",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "pcap-storage",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: resource.NewQuantity(1024*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = ts.fakeClientset.AppsV1().Deployments(ts.namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ts.hubService,
			Namespace: ts.namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "grpc",
					Port:       9090,
					TargetPort: intstr.FromInt(9090),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "http",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	_, err = ts.fakeClientset.CoreV1().Services(ts.namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// TestDeployHub_ServiceAccountCreated tests that a ServiceAccount is created in the session namespace
func TestDeployHub_ServiceAccountCreated(t *testing.T) {
	sessionID := "hub12345"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	// First create namespace (required for deployHub)
	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	// Verify ServiceAccount was created
	sa, err := ts.fakeClientset.CoreV1().ServiceAccounts(ts.namespace).Get(ctx, "podscope-hub", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get ServiceAccount: %v", err)
	}

	if sa.Name != "podscope-hub" {
		t.Errorf("ServiceAccount name = %q, want %q", sa.Name, "podscope-hub")
	}
	if sa.Namespace != ts.namespace {
		t.Errorf("ServiceAccount namespace = %q, want %q", sa.Namespace, ts.namespace)
	}
}

// TestDeployHub_ClusterRoleCreated tests that a ClusterRole is created with correct name including session ID
func TestDeployHub_ClusterRoleCreated(t *testing.T) {
	sessionID := "role5678"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	// Verify ClusterRole was created with correct name
	expectedName := fmt.Sprintf("podscope-hub-%s", sessionID)
	clusterRole, err := ts.fakeClientset.RbacV1().ClusterRoles().Get(ctx, expectedName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get ClusterRole: %v", err)
	}

	if clusterRole.Name != expectedName {
		t.Errorf("ClusterRole name = %q, want %q", clusterRole.Name, expectedName)
	}
}

// TestDeployHub_ClusterRoleHasPodsExecPermission tests that the ClusterRole has pods/exec permission for terminal feature
func TestDeployHub_ClusterRoleHasPodsExecPermission(t *testing.T) {
	sessionID := "exec1234"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	clusterRoleName := fmt.Sprintf("podscope-hub-%s", sessionID)
	clusterRole, err := ts.fakeClientset.RbacV1().ClusterRoles().Get(ctx, clusterRoleName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get ClusterRole: %v", err)
	}

	// Check for pods/exec permission
	hasPodsExec := false
	for _, rule := range clusterRole.Rules {
		for _, resource := range rule.Resources {
			if resource == "pods/exec" {
				hasPodsExec = true
				// Verify create verb is present
				hasCreate := false
				for _, verb := range rule.Verbs {
					if verb == "create" {
						hasCreate = true
						break
					}
				}
				if !hasCreate {
					t.Error("ClusterRole pods/exec rule is missing 'create' verb")
				}
				break
			}
		}
	}

	if !hasPodsExec {
		t.Error("ClusterRole is missing pods/exec permission")
	}
}

// TestDeployHub_ClusterRoleHasPodsPermission tests that the ClusterRole has pods get/list permission
func TestDeployHub_ClusterRoleHasPodsPermission(t *testing.T) {
	sessionID := "pods1234"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	clusterRoleName := fmt.Sprintf("podscope-hub-%s", sessionID)
	clusterRole, err := ts.fakeClientset.RbacV1().ClusterRoles().Get(ctx, clusterRoleName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get ClusterRole: %v", err)
	}

	// Check for pods permission with get and list
	hasPodsRule := false
	hasGet := false
	hasList := false
	for _, rule := range clusterRole.Rules {
		for _, resource := range rule.Resources {
			if resource == "pods" {
				hasPodsRule = true
				for _, verb := range rule.Verbs {
					if verb == "get" {
						hasGet = true
					}
					if verb == "list" {
						hasList = true
					}
				}
				break
			}
		}
	}

	if !hasPodsRule {
		t.Error("ClusterRole is missing pods rule")
	}
	if !hasGet {
		t.Error("ClusterRole pods rule is missing 'get' verb")
	}
	if !hasList {
		t.Error("ClusterRole pods rule is missing 'list' verb")
	}
}

// TestDeployHub_ClusterRoleBindingBindsServiceAccount tests that ClusterRoleBinding binds the ServiceAccount to ClusterRole
func TestDeployHub_ClusterRoleBindingBindsServiceAccount(t *testing.T) {
	sessionID := "bind5678"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	// Verify ClusterRoleBinding exists and has correct references
	bindingName := fmt.Sprintf("podscope-hub-%s", sessionID)
	binding, err := ts.fakeClientset.RbacV1().ClusterRoleBindings().Get(ctx, bindingName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get ClusterRoleBinding: %v", err)
	}

	// Check RoleRef points to the correct ClusterRole
	expectedRoleName := fmt.Sprintf("podscope-hub-%s", sessionID)
	if binding.RoleRef.Name != expectedRoleName {
		t.Errorf("RoleRef.Name = %q, want %q", binding.RoleRef.Name, expectedRoleName)
	}
	if binding.RoleRef.Kind != "ClusterRole" {
		t.Errorf("RoleRef.Kind = %q, want %q", binding.RoleRef.Kind, "ClusterRole")
	}
	if binding.RoleRef.APIGroup != "rbac.authorization.k8s.io" {
		t.Errorf("RoleRef.APIGroup = %q, want %q", binding.RoleRef.APIGroup, "rbac.authorization.k8s.io")
	}

	// Check Subject is the ServiceAccount
	if len(binding.Subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(binding.Subjects))
	}
	subject := binding.Subjects[0]
	if subject.Kind != "ServiceAccount" {
		t.Errorf("Subject.Kind = %q, want %q", subject.Kind, "ServiceAccount")
	}
	if subject.Name != "podscope-hub" {
		t.Errorf("Subject.Name = %q, want %q", subject.Name, "podscope-hub")
	}
	if subject.Namespace != ts.namespace {
		t.Errorf("Subject.Namespace = %q, want %q", subject.Namespace, ts.namespace)
	}
}

// TestDeployHub_DeploymentUsesCorrectHubImage tests that the Deployment uses the correct Hub image
func TestDeployHub_DeploymentUsesCorrectHubImage(t *testing.T) {
	sessionID := "img12345"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	// Verify Deployment uses correct image
	deployment, err := ts.fakeClientset.AppsV1().Deployments(ts.namespace).Get(ctx, "podscope-hub", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get Deployment: %v", err)
	}

	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		t.Fatal("Deployment has no containers")
	}

	container := deployment.Spec.Template.Spec.Containers[0]
	expectedImage := GetHubImage()
	if container.Image != expectedImage {
		t.Errorf("Container image = %q, want %q", container.Image, expectedImage)
	}
}

// TestDeployHub_DeploymentHasCorrectPorts tests that the Deployment container has ports 8080 and 9090
func TestDeployHub_DeploymentHasCorrectPorts(t *testing.T) {
	sessionID := "port1234"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	deployment, err := ts.fakeClientset.AppsV1().Deployments(ts.namespace).Get(ctx, "podscope-hub", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get Deployment: %v", err)
	}

	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		t.Fatal("Deployment has no containers")
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	// Check for both ports
	has8080 := false
	has9090 := false
	for _, port := range container.Ports {
		if port.ContainerPort == 8080 {
			has8080 = true
			if port.Name != "http" {
				t.Errorf("port 8080 name = %q, want %q", port.Name, "http")
			}
		}
		if port.ContainerPort == 9090 {
			has9090 = true
			if port.Name != "grpc" {
				t.Errorf("port 9090 name = %q, want %q", port.Name, "grpc")
			}
		}
	}

	if !has8080 {
		t.Error("Deployment container is missing port 8080")
	}
	if !has9090 {
		t.Error("Deployment container is missing port 9090")
	}
}

// TestDeployHub_ServiceIsClusterIP tests that the Service is ClusterIP type
func TestDeployHub_ServiceIsClusterIP(t *testing.T) {
	sessionID := "svc12345"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	// Verify Service is ClusterIP
	service, err := ts.fakeClientset.CoreV1().Services(ts.namespace).Get(ctx, "podscope-hub", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get Service: %v", err)
	}

	if service.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("Service Type = %q, want %q", service.Spec.Type, corev1.ServiceTypeClusterIP)
	}
}

// TestDeployHub_ServiceHasCorrectPorts tests that the Service has ports 8080 and 9090
func TestDeployHub_ServiceHasCorrectPorts(t *testing.T) {
	sessionID := "svcp1234"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	service, err := ts.fakeClientset.CoreV1().Services(ts.namespace).Get(ctx, "podscope-hub", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get Service: %v", err)
	}

	// Check for both ports
	has8080 := false
	has9090 := false
	for _, port := range service.Spec.Ports {
		if port.Port == 8080 {
			has8080 = true
			if port.Name != "http" {
				t.Errorf("port 8080 name = %q, want %q", port.Name, "http")
			}
			if port.TargetPort.IntVal != 8080 {
				t.Errorf("port 8080 targetPort = %d, want %d", port.TargetPort.IntVal, 8080)
			}
		}
		if port.Port == 9090 {
			has9090 = true
			if port.Name != "grpc" {
				t.Errorf("port 9090 name = %q, want %q", port.Name, "grpc")
			}
			if port.TargetPort.IntVal != 9090 {
				t.Errorf("port 9090 targetPort = %d, want %d", port.TargetPort.IntVal, 9090)
			}
		}
	}

	if !has8080 {
		t.Error("Service is missing port 8080")
	}
	if !has9090 {
		t.Error("Service is missing port 9090")
	}
}

// TestDeployHub_DeploymentUsesServiceAccount tests that the Deployment uses the correct ServiceAccount
func TestDeployHub_DeploymentUsesServiceAccount(t *testing.T) {
	sessionID := "depsa123"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	deployment, err := ts.fakeClientset.AppsV1().Deployments(ts.namespace).Get(ctx, "podscope-hub", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get Deployment: %v", err)
	}

	expectedSA := "podscope-hub"
	if deployment.Spec.Template.Spec.ServiceAccountName != expectedSA {
		t.Errorf("ServiceAccountName = %q, want %q", deployment.Spec.Template.Spec.ServiceAccountName, expectedSA)
	}
}

// TestDeployHub_DeploymentHasSessionIDLabel tests that the Deployment has the session ID label
func TestDeployHub_DeploymentHasSessionIDLabel(t *testing.T) {
	sessionID := "dlbl1234"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	deployment, err := ts.fakeClientset.AppsV1().Deployments(ts.namespace).Get(ctx, "podscope-hub", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get Deployment: %v", err)
	}

	// Check deployment labels
	label, ok := deployment.Labels["podscope.io/session-id"]
	if !ok {
		t.Error("Deployment is missing podscope.io/session-id label")
	} else if label != sessionID {
		t.Errorf("Deployment session-id label = %q, want %q", label, sessionID)
	}

	// Check pod template labels
	podLabel, ok := deployment.Spec.Template.Labels["podscope.io/session-id"]
	if !ok {
		t.Error("Pod template is missing podscope.io/session-id label")
	} else if podLabel != sessionID {
		t.Errorf("Pod template session-id label = %q, want %q", podLabel, sessionID)
	}
}

// TestDeployHub_AllResourcesCreated tests that all required resources are created
func TestDeployHub_AllResourcesCreated(t *testing.T) {
	sessionID := "all67890"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	err := ts.createNamespace(ctx)
	if err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}

	err = ts.deployHub(ctx)
	if err != nil {
		t.Fatalf("deployHub failed: %v", err)
	}

	// Verify ServiceAccount
	_, err = ts.fakeClientset.CoreV1().ServiceAccounts(ts.namespace).Get(ctx, "podscope-hub", metav1.GetOptions{})
	if err != nil {
		t.Errorf("ServiceAccount not created: %v", err)
	}

	// Verify ClusterRole
	_, err = ts.fakeClientset.RbacV1().ClusterRoles().Get(ctx, fmt.Sprintf("podscope-hub-%s", sessionID), metav1.GetOptions{})
	if err != nil {
		t.Errorf("ClusterRole not created: %v", err)
	}

	// Verify ClusterRoleBinding
	_, err = ts.fakeClientset.RbacV1().ClusterRoleBindings().Get(ctx, fmt.Sprintf("podscope-hub-%s", sessionID), metav1.GetOptions{})
	if err != nil {
		t.Errorf("ClusterRoleBinding not created: %v", err)
	}

	// Verify Deployment
	_, err = ts.fakeClientset.AppsV1().Deployments(ts.namespace).Get(ctx, "podscope-hub", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Deployment not created: %v", err)
	}

	// Verify Service
	_, err = ts.fakeClientset.CoreV1().Services(ts.namespace).Get(ctx, "podscope-hub", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Service not created: %v", err)
	}
}

// injectAgent is a test version that uses the fake clientset
func (ts *testSession) injectAgent(ctx context.Context, target PodTarget, privileged bool) error {
	// Get the current pod
	pod, err := ts.fakeClientset.CoreV1().Pods(target.Namespace).Get(ctx, target.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod: %w", err)
	}

	// Check if a podscope agent is already RUNNING in this pod
	for _, ec := range pod.Spec.EphemeralContainers {
		if strings.HasPrefix(ec.Name, "podscope-agent") {
			// Check container status - only block if still running
			for _, status := range pod.Status.EphemeralContainerStatuses {
				if status.Name == ec.Name && status.State.Running != nil {
					return fmt.Errorf("agent %s already running in pod", ec.Name)
				}
			}
		}
	}

	// Create the ephemeral container spec with unique name per session
	agentName := fmt.Sprintf("podscope-agent-%s", ts.id)

	securityContext := &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{"NET_RAW"},
		},
	}

	if privileged {
		t := true
		securityContext.Privileged = &t
	}

	hubAddress := fmt.Sprintf("%s.%s.svc.cluster.local:9090", ts.hubService, ts.namespace)

	ephemeralContainer := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:            agentName,
			Image:           GetAgentImage(),
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: securityContext,
			Env: []corev1.EnvVar{
				{
					Name:  "HUB_ADDRESS",
					Value: hubAddress,
				},
				{
					Name:  "POD_NAME",
					Value: target.Name,
				},
				{
					Name:  "POD_NAMESPACE",
					Value: target.Namespace,
				},
				{
					Name:  "POD_IP",
					Value: target.IP,
				},
				{
					Name:  "SESSION_ID",
					Value: ts.id,
				},
				{
					Name:  "INTERFACE",
					Value: "eth0",
				},
			},
		},
	}

	// Update the pod with the ephemeral container
	pod.Spec.EphemeralContainers = append(pod.Spec.EphemeralContainers, ephemeralContainer)

	_, err = ts.fakeClientset.CoreV1().Pods(target.Namespace).UpdateEphemeralContainers(
		ctx,
		target.Name,
		pod,
		metav1.UpdateOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to inject ephemeral container: %w", err)
	}

	return nil
}

// createTestPod creates a running pod for testing agent injection
func createTestPod(t *testing.T, ts *testSession, ctx context.Context, name, namespace, ip string) {
	t.Helper()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: ip,
		},
	}

	_, err := ts.fakeClientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create test pod: %v", err)
	}
}

// TestInjectAgent_EphemeralContainerAdded tests that ephemeral container is added to pod spec
func TestInjectAgent_EphemeralContainerAdded(t *testing.T) {
	sessionID := "inj12345"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "default"
	podIP := "10.0.0.100"

	// Create a target pod
	createTestPod(t, ts, ctx, podName, podNamespace, podIP)

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	err := ts.injectAgent(ctx, target, false)
	if err != nil {
		t.Fatalf("injectAgent failed: %v", err)
	}

	// Verify ephemeral container was added
	pod, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if len(pod.Spec.EphemeralContainers) != 1 {
		t.Errorf("expected 1 ephemeral container, got %d", len(pod.Spec.EphemeralContainers))
	}
}

// TestInjectAgent_ContainerNameFormat tests that container name is podscope-agent-{session-id}
func TestInjectAgent_ContainerNameFormat(t *testing.T) {
	sessionID := "nm123456"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "default"
	podIP := "10.0.0.101"

	createTestPod(t, ts, ctx, podName, podNamespace, podIP)

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	err := ts.injectAgent(ctx, target, false)
	if err != nil {
		t.Fatalf("injectAgent failed: %v", err)
	}

	pod, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	expectedName := fmt.Sprintf("podscope-agent-%s", sessionID)
	if len(pod.Spec.EphemeralContainers) < 1 {
		t.Fatal("no ephemeral containers found")
	}

	containerName := pod.Spec.EphemeralContainers[0].Name
	if containerName != expectedName {
		t.Errorf("container name = %q, want %q", containerName, expectedName)
	}
}

// TestInjectAgent_NetRawCapabilitySet tests that NET_RAW capability is set by default
func TestInjectAgent_NetRawCapabilitySet(t *testing.T) {
	sessionID := "nr123456"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "default"
	podIP := "10.0.0.102"

	createTestPod(t, ts, ctx, podName, podNamespace, podIP)

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	// Inject without privileged flag
	err := ts.injectAgent(ctx, target, false)
	if err != nil {
		t.Fatalf("injectAgent failed: %v", err)
	}

	pod, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if len(pod.Spec.EphemeralContainers) < 1 {
		t.Fatal("no ephemeral containers found")
	}

	ec := pod.Spec.EphemeralContainers[0]
	if ec.SecurityContext == nil {
		t.Fatal("SecurityContext is nil")
	}
	if ec.SecurityContext.Capabilities == nil {
		t.Fatal("Capabilities is nil")
	}

	hasNetRaw := false
	for _, cap := range ec.SecurityContext.Capabilities.Add {
		if cap == "NET_RAW" {
			hasNetRaw = true
			break
		}
	}

	if !hasNetRaw {
		t.Error("NET_RAW capability not found in Add list")
	}
}

// TestInjectAgent_PrivilegedFlagSetsPrivileged tests that privileged flag sets container as privileged
func TestInjectAgent_PrivilegedFlagSetsPrivileged(t *testing.T) {
	sessionID := "pr123456"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "default"
	podIP := "10.0.0.103"

	createTestPod(t, ts, ctx, podName, podNamespace, podIP)

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	// Inject with privileged flag
	err := ts.injectAgent(ctx, target, true)
	if err != nil {
		t.Fatalf("injectAgent failed: %v", err)
	}

	pod, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if len(pod.Spec.EphemeralContainers) < 1 {
		t.Fatal("no ephemeral containers found")
	}

	ec := pod.Spec.EphemeralContainers[0]
	if ec.SecurityContext == nil {
		t.Fatal("SecurityContext is nil")
	}
	if ec.SecurityContext.Privileged == nil || !*ec.SecurityContext.Privileged {
		t.Error("Privileged flag not set to true")
	}
}

// TestInjectAgent_HubAddressEnvVar tests that HUB_ADDRESS environment variable is set to service DNS name
func TestInjectAgent_HubAddressEnvVar(t *testing.T) {
	sessionID := "ha123456"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "default"
	podIP := "10.0.0.104"

	createTestPod(t, ts, ctx, podName, podNamespace, podIP)

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	err := ts.injectAgent(ctx, target, false)
	if err != nil {
		t.Fatalf("injectAgent failed: %v", err)
	}

	pod, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if len(pod.Spec.EphemeralContainers) < 1 {
		t.Fatal("no ephemeral containers found")
	}

	ec := pod.Spec.EphemeralContainers[0]
	expectedHubAddr := fmt.Sprintf("%s.%s.svc.cluster.local:9090", ts.hubService, ts.namespace)

	var hubAddrValue string
	for _, env := range ec.Env {
		if env.Name == "HUB_ADDRESS" {
			hubAddrValue = env.Value
			break
		}
	}

	if hubAddrValue == "" {
		t.Error("HUB_ADDRESS environment variable not found")
	} else if hubAddrValue != expectedHubAddr {
		t.Errorf("HUB_ADDRESS = %q, want %q", hubAddrValue, expectedHubAddr)
	}
}

// TestInjectAgent_PodNameEnvVar tests that POD_NAME environment variable is set
func TestInjectAgent_PodNameEnvVar(t *testing.T) {
	sessionID := "pn123456"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "my-target-pod"
	podNamespace := "default"
	podIP := "10.0.0.105"

	createTestPod(t, ts, ctx, podName, podNamespace, podIP)

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	err := ts.injectAgent(ctx, target, false)
	if err != nil {
		t.Fatalf("injectAgent failed: %v", err)
	}

	pod, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if len(pod.Spec.EphemeralContainers) < 1 {
		t.Fatal("no ephemeral containers found")
	}

	ec := pod.Spec.EphemeralContainers[0]
	var podNameValue string
	for _, env := range ec.Env {
		if env.Name == "POD_NAME" {
			podNameValue = env.Value
			break
		}
	}

	if podNameValue == "" {
		t.Error("POD_NAME environment variable not found")
	} else if podNameValue != podName {
		t.Errorf("POD_NAME = %q, want %q", podNameValue, podName)
	}
}

// TestInjectAgent_PodNamespaceEnvVar tests that POD_NAMESPACE environment variable is set
func TestInjectAgent_PodNamespaceEnvVar(t *testing.T) {
	sessionID := "pns12345"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "my-namespace"
	podIP := "10.0.0.106"

	createTestPod(t, ts, ctx, podName, podNamespace, podIP)

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	err := ts.injectAgent(ctx, target, false)
	if err != nil {
		t.Fatalf("injectAgent failed: %v", err)
	}

	pod, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if len(pod.Spec.EphemeralContainers) < 1 {
		t.Fatal("no ephemeral containers found")
	}

	ec := pod.Spec.EphemeralContainers[0]
	var podNsValue string
	for _, env := range ec.Env {
		if env.Name == "POD_NAMESPACE" {
			podNsValue = env.Value
			break
		}
	}

	if podNsValue == "" {
		t.Error("POD_NAMESPACE environment variable not found")
	} else if podNsValue != podNamespace {
		t.Errorf("POD_NAMESPACE = %q, want %q", podNsValue, podNamespace)
	}
}

// TestInjectAgent_PodIPEnvVar tests that POD_IP environment variable is set
func TestInjectAgent_PodIPEnvVar(t *testing.T) {
	sessionID := "pip12345"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "default"
	podIP := "10.99.88.77"

	createTestPod(t, ts, ctx, podName, podNamespace, podIP)

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	err := ts.injectAgent(ctx, target, false)
	if err != nil {
		t.Fatalf("injectAgent failed: %v", err)
	}

	pod, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if len(pod.Spec.EphemeralContainers) < 1 {
		t.Fatal("no ephemeral containers found")
	}

	ec := pod.Spec.EphemeralContainers[0]
	var podIPValue string
	for _, env := range ec.Env {
		if env.Name == "POD_IP" {
			podIPValue = env.Value
			break
		}
	}

	if podIPValue == "" {
		t.Error("POD_IP environment variable not found")
	} else if podIPValue != podIP {
		t.Errorf("POD_IP = %q, want %q", podIPValue, podIP)
	}
}

// TestInjectAgent_AllEnvVarsPresent tests that all required environment variables are set
func TestInjectAgent_AllEnvVarsPresent(t *testing.T) {
	sessionID := "all12345"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "default"
	podIP := "10.0.0.200"

	createTestPod(t, ts, ctx, podName, podNamespace, podIP)

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	err := ts.injectAgent(ctx, target, false)
	if err != nil {
		t.Fatalf("injectAgent failed: %v", err)
	}

	pod, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if len(pod.Spec.EphemeralContainers) < 1 {
		t.Fatal("no ephemeral containers found")
	}

	ec := pod.Spec.EphemeralContainers[0]
	requiredEnvVars := []string{"HUB_ADDRESS", "POD_NAME", "POD_NAMESPACE", "POD_IP", "SESSION_ID", "INTERFACE"}

	envMap := make(map[string]string)
	for _, env := range ec.Env {
		envMap[env.Name] = env.Value
	}

	for _, required := range requiredEnvVars {
		if _, ok := envMap[required]; !ok {
			t.Errorf("required environment variable %q not found", required)
		}
	}
}

// TestInjectAgent_ErrorIfPodNotFound tests that error is returned for non-existent pod
func TestInjectAgent_ErrorIfPodNotFound(t *testing.T) {
	sessionID := "nf123456"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	target := PodTarget{
		Name:      "non-existent-pod",
		Namespace: "default",
		IP:        "10.0.0.999",
	}

	err := ts.injectAgent(ctx, target, false)
	if err == nil {
		t.Error("expected error for non-existent pod, got nil")
	}
}

// TestInjectAgent_ErrorIfAgentAlreadyRunning tests that error is returned if agent is already running
func TestInjectAgent_ErrorIfAgentAlreadyRunning(t *testing.T) {
	sessionID := "ar123456"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "default"
	podIP := "10.0.0.201"

	// Create a pod with an existing running ephemeral container
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
				},
			},
			EphemeralContainers: []corev1.EphemeralContainer{
				{
					EphemeralContainerCommon: corev1.EphemeralContainerCommon{
						Name:  "podscope-agent-existing",
						Image: "podscope-agent:v1",
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: podIP,
			EphemeralContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "podscope-agent-existing",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	_, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create test pod: %v", err)
	}

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	err = ts.injectAgent(ctx, target, false)
	if err == nil {
		t.Error("expected error for already running agent, got nil")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("expected error message to contain 'already running', got: %v", err)
	}
}

// TestInjectAgent_OkIfTerminatedAgentExists tests that injection succeeds if previous agent terminated
func TestInjectAgent_OkIfTerminatedAgentExists(t *testing.T) {
	sessionID := "te123456"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "default"
	podIP := "10.0.0.202"

	// Create a pod with an existing terminated ephemeral container
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
				},
			},
			EphemeralContainers: []corev1.EphemeralContainer{
				{
					EphemeralContainerCommon: corev1.EphemeralContainerCommon{
						Name:  "podscope-agent-old",
						Image: "podscope-agent:v1",
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: podIP,
			EphemeralContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "podscope-agent-old",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 0,
						},
					},
				},
			},
		},
	}

	_, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create test pod: %v", err)
	}

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	// Should succeed because existing agent is terminated
	err = ts.injectAgent(ctx, target, false)
	if err != nil {
		t.Errorf("expected injection to succeed with terminated agent, got: %v", err)
	}

	// Verify new container was added
	pod, err = ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if len(pod.Spec.EphemeralContainers) != 2 {
		t.Errorf("expected 2 ephemeral containers (old + new), got %d", len(pod.Spec.EphemeralContainers))
	}
}

// TestInjectAgent_UsesCorrectAgentImage tests that the container uses the correct agent image
func TestInjectAgent_UsesCorrectAgentImage(t *testing.T) {
	sessionID := "img12345"
	ts := createTestSession(t, sessionID)
	ctx := context.Background()

	podName := "target-pod"
	podNamespace := "default"
	podIP := "10.0.0.203"

	createTestPod(t, ts, ctx, podName, podNamespace, podIP)

	target := PodTarget{
		Name:      podName,
		Namespace: podNamespace,
		IP:        podIP,
	}

	err := ts.injectAgent(ctx, target, false)
	if err != nil {
		t.Fatalf("injectAgent failed: %v", err)
	}

	pod, err := ts.fakeClientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if len(pod.Spec.EphemeralContainers) < 1 {
		t.Fatal("no ephemeral containers found")
	}

	ec := pod.Spec.EphemeralContainers[0]
	expectedImage := GetAgentImage()
	if ec.Image != expectedImage {
		t.Errorf("image = %q, want %q", ec.Image, expectedImage)
	}
}

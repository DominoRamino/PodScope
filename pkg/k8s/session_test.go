package k8s

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// newTestSession creates a Session with a fake clientset for testing
func newTestSession(t *testing.T, sessionID string) (*Session, *fake.Clientset) {
	t.Helper()
	fakeClientset := fake.NewSimpleClientset()
	client := &Client{
		clientset: fakeClientset,
	}
	return &Session{
		client:    client,
		id:        sessionID,
		namespace: "podscope-" + sessionID,
		stopChan:  make(chan struct{}),
	}, fakeClientset
}

// TestCleanup_DeletesClusterRoleBinding verifies that Cleanup() deletes the ClusterRoleBinding
func TestCleanup_DeletesClusterRoleBinding(t *testing.T) {
	ctx := context.Background()
	sessionID := "test1234"
	session, fakeClientset := newTestSession(t, sessionID)

	// Create the ClusterRoleBinding that should be deleted
	crbName := "podscope-hub-" + sessionID
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: crbName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     crbName,
		},
	}
	_, err := fakeClientset.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create test ClusterRoleBinding: %v", err)
	}

	// Verify it exists before cleanup
	_, err = fakeClientset.RbacV1().ClusterRoleBindings().Get(ctx, crbName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("ClusterRoleBinding should exist before cleanup: %v", err)
	}

	// Run cleanup
	err = session.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}

	// Verify ClusterRoleBinding was deleted
	_, err = fakeClientset.RbacV1().ClusterRoleBindings().Get(ctx, crbName, metav1.GetOptions{})
	if err == nil {
		t.Error("ClusterRoleBinding should have been deleted by Cleanup()")
	}
}

// TestCleanup_DeletesClusterRole verifies that Cleanup() deletes the ClusterRole
func TestCleanup_DeletesClusterRole(t *testing.T) {
	ctx := context.Background()
	sessionID := "test5678"
	session, fakeClientset := newTestSession(t, sessionID)

	// Create the ClusterRole that should be deleted
	crName := "podscope-hub-" + sessionID
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	_, err := fakeClientset.RbacV1().ClusterRoles().Create(ctx, cr, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create test ClusterRole: %v", err)
	}

	// Verify it exists before cleanup
	_, err = fakeClientset.RbacV1().ClusterRoles().Get(ctx, crName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("ClusterRole should exist before cleanup: %v", err)
	}

	// Run cleanup
	err = session.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}

	// Verify ClusterRole was deleted
	_, err = fakeClientset.RbacV1().ClusterRoles().Get(ctx, crName, metav1.GetOptions{})
	if err == nil {
		t.Error("ClusterRole should have been deleted by Cleanup()")
	}
}

// TestCleanup_ContinuesIfNotFound verifies that Cleanup() succeeds when resources are already deleted
func TestCleanup_ContinuesIfNotFound(t *testing.T) {
	ctx := context.Background()
	sessionID := "testabcd"
	session, _ := newTestSession(t, sessionID)

	// Don't create any RBAC resources - they're "already deleted"

	// Cleanup should succeed even though resources don't exist
	err := session.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup() should succeed when resources already deleted, got error: %v", err)
	}
}

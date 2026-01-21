# PRD: Go K8s - Client and Cleanup Tests

## Introduction

Add unit tests for the Kubernetes client wrapper in `pkg/k8s/client.go`. This includes pod queries by name and selector, as well as cleanup functions for stale namespaces and orphaned RBAC resources. Tests use `fake.NewSimpleClientset()` with pre-populated resources.

## Goals

- Test pod lookup by name with phase validation
- Test pod queries by label selector across namespaces
- Verify stale namespace cleanup based on age
- Verify orphaned RBAC cleanup when namespaces are deleted

## User Stories

### US-001: Test Pod Lookup by Name
**Description:** As a developer, I want tests for `GetPodByName()` to verify correct pod retrieval and running state validation.

**Acceptance Criteria:**
- [ ] Pod lookup tests pass:
  - Returns PodTarget with name, namespace, IP, node for running pod
  - Returns error for non-existent pod
  - Returns error for pod not in Running phase (Pending, Failed, etc.)
  - Correctly extracts pod IP from status
- [ ] `go test -v ./pkg/k8s/... -run TestGetPodByName` passes

---

### US-002: Test Pod Queries by Selector
**Description:** As a developer, I want tests for `GetPodsBySelector()` to verify label-based pod discovery.

**Acceptance Criteria:**
- [ ] Selector query tests pass:
  - Returns all pods matching label selector
  - Empty namespace searches across all namespaces
  - Specific namespace limits search scope
  - Filters out non-Running pods
  - Returns empty slice when no pods match
- [ ] `go test -v ./pkg/k8s/... -run TestGetPodsBySelector` passes

---

### US-003: Test Stale Namespace Cleanup
**Description:** As a developer, I want tests for `CleanupStaleNamespaces()` to verify cleanup of old PodScope namespaces.

**Acceptance Criteria:**
- [ ] Stale namespace cleanup tests pass:
  - Deletes namespaces older than maxAge
  - Keeps namespaces newer than maxAge
  - Uses `podscope.io/created-at` annotation for age
  - Falls back to metadata.creationTimestamp if annotation missing
  - Only targets namespaces with `app.kubernetes.io/name: podscope` label
- [ ] `go test -v ./pkg/k8s/... -run TestCleanupStaleNamespaces` passes

---

### US-004: Test Orphaned RBAC Cleanup
**Description:** As a developer, I want tests for `CleanupOrphanedRBAC()` to verify cleanup of RBAC resources without corresponding namespaces.

**Acceptance Criteria:**
- [ ] Orphaned RBAC cleanup tests pass:
  - Deletes ClusterRole when namespace no longer exists
  - Deletes ClusterRoleBinding when namespace no longer exists
  - Keeps ClusterRole/ClusterRoleBinding when namespace exists
  - Deletes both CR and CRB together for orphaned sessions
  - Only targets resources with `podscope.io/session-id` label
- [ ] `go test -v ./pkg/k8s/... -run TestCleanupOrphanedRBAC` passes

---

## Functional Requirements

- FR-1: Create `pkg/k8s/client_test.go` with 14 test functions
- FR-2: Use `fake.NewSimpleClientset()` pre-populated with test resources
- FR-3: Create helper to populate fake clientset with pods, namespaces, RBAC
- FR-4: Use table-driven tests for selector query variations
- FR-5: Verify cleanup via clientset List after cleanup function runs

### Test Data Setup Pattern
```go
func setupTestClient(t *testing.T) (*Client, *fake.Clientset) {
    // Create pods in various states
    runningPod := &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-pod",
            Namespace: "default",
            Labels:    map[string]string{"app": "test"},
        },
        Status: corev1.PodStatus{
            Phase: corev1.PodRunning,
            PodIP: "10.0.0.1",
        },
    }

    pendingPod := &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "pending-pod",
            Namespace: "default",
        },
        Status: corev1.PodStatus{
            Phase: corev1.PodPending,
        },
    }

    fakeClientset := fake.NewSimpleClientset(runningPod, pendingPod)
    return &Client{clientset: fakeClientset}, fakeClientset
}
```

## Non-Goals

- No testing of kubeconfig loading or in-cluster config
- No testing of actual Kubernetes API server connectivity
- No testing of rate limiting or retry logic

## Technical Considerations

- Label selectors use `metav1.LabelSelector` format
- Namespace age comparison uses `time.Since()` against creation timestamp
- RBAC resources are cluster-scoped (no namespace in lookup)

## Success Metrics

- All 14 tests pass
- `go test -cover ./pkg/k8s/...` shows >80% coverage for client.go
- Tests complete in under 3 seconds

## Open Questions

None.

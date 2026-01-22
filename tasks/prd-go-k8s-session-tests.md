# PRD: Go K8s - Session Lifecycle Tests

## Introduction

Extend the existing `pkg/k8s/session_test.go` to add comprehensive tests for the session lifecycle in `pkg/k8s/session.go`. The existing file has 3 tests for RBAC cleanup. This PRD adds tests for namespace creation, Hub deployment, and agent injection using the established `fake.NewSimpleClientset()` pattern.

## Goals

- Extend existing test file with namespace creation tests
- Add Hub deployment tests (ServiceAccount, ClusterRole, Deployment, Service)
- Add agent injection tests for ephemeral container management
- Verify correct labels, annotations, and environment variables

## User Stories

### US-001: Test Namespace Creation
**Description:** As a developer, I want tests for `createNamespace()` to verify correct namespace setup with labels and annotations.

**Acceptance Criteria:**
- [ ] Namespace creation tests pass:
  - Namespace created with name `podscope-<session-id>`
  - Labels include `app.kubernetes.io/name: podscope`
  - Labels include `podscope.io/session-id: <session-id>`
  - Annotation `podscope.io/created-at` set to timestamp
  - Idempotent: no error if namespace already exists
- [ ] `go test -v ./pkg/k8s/... -run TestCreateNamespace` passes

---

### US-002: Test Hub Deployment Resources
**Description:** As a developer, I want tests for `deployHub()` to verify all Kubernetes resources are created correctly.

**Acceptance Criteria:**
- [ ] ServiceAccount creation tests pass:
  - ServiceAccount created in session namespace
  - Name follows convention
- [ ] ClusterRole creation tests pass:
  - ClusterRole with correct name (includes session ID)
  - Has `pods/exec` permission for terminal feature
  - Has `pods` get/list permissions
- [ ] ClusterRoleBinding creation tests pass:
  - Binds ServiceAccount to ClusterRole
  - References correct namespace
- [ ] Deployment creation tests pass:
  - Uses correct Hub image (from env or default)
  - Sets resource requests/limits
  - Mounts correct volumes
  - Sets environment variables
- [ ] Service creation tests pass:
  - ClusterIP service on ports 8080, 9090
  - Selector matches Hub deployment
- [ ] `go test -v ./pkg/k8s/... -run TestDeployHub` passes

---

### US-003: Test Agent Injection
**Description:** As a developer, I want tests for `InjectAgent()` to verify ephemeral container injection into target pods.

**Acceptance Criteria:**
- [ ] Agent injection tests pass:
  - Ephemeral container added to pod spec
  - Container name is `podscope-agent-<short-id>`
  - Uses correct Agent image
  - NET_RAW capability set (default)
  - Privileged mode when `--force-privileged` flag set
- [ ] Environment variables tests pass:
  - `HUB_ADDR` set to service DNS name
  - `POD_NAME` from downward API
  - `POD_NAMESPACE` from downward API
  - `POD_IP` from downward API
- [ ] Edge cases tests pass:
  - Rejects injection if agent already running
  - Allows injection if previous agent terminated
- [ ] `go test -v ./pkg/k8s/... -run TestInjectAgent` passes

---

### US-004: Test Hub Readiness Check
**Description:** As a developer, I want tests for `waitForHub()` to verify deployment readiness polling.

**Acceptance Criteria:**
- [ ] Readiness tests pass:
  - Returns success when deployment has ready replicas
  - Returns error after timeout if not ready
  - Polls at correct interval
- [ ] `go test -v ./pkg/k8s/... -run TestWaitForHub` passes

---

### US-005: Test Agent Container Lookup
**Description:** As a developer, I want tests for `GetAgentContainer()` to verify correct container identification.

**Acceptance Criteria:**
- [ ] Container lookup tests pass:
  - Returns container name when agent exists
  - Returns error when no agent container found
  - Identifies agent by name prefix `podscope-agent-`
- [ ] `go test -v ./pkg/k8s/... -run TestGetAgentContainer` passes

---

## Functional Requirements

- FR-1: Extend `pkg/k8s/session_test.go` with 18 additional test functions
- FR-2: Reuse existing `newTestSession()` helper pattern
- FR-3: Use `fake.NewSimpleClientset()` for all Kubernetes API mocking
- FR-4: Pre-populate fake clientset with test pods for injection tests
- FR-5: Verify resources via clientset Get/List after creation

### Existing Test Pattern (to follow)
```go
func newTestSession(t *testing.T) (*Session, *fake.Clientset) {
    fakeClientset := fake.NewSimpleClientset()
    client := &Client{clientset: fakeClientset}
    session := &Session{
        ID:        "test-session",
        client:    client,
        namespace: "podscope-test-session",
    }
    return session, fakeClientset
}
```

## Non-Goals

- No testing of actual Kubernetes cluster connectivity
- No testing of port-forward (SPDY tunneling)
- No testing of pod logs streaming

## Technical Considerations

- `fake.NewSimpleClientset()` from `k8s.io/client-go/kubernetes/fake`
- Ephemeral containers require Kubernetes 1.25+
- ClusterRole/ClusterRoleBinding are cluster-scoped (not namespaced)

## Success Metrics

- All 21 tests pass (3 existing + 18 new)
- `go test -cover ./pkg/k8s/...` shows >70% coverage for session.go
- Tests complete in under 5 seconds

## Open Questions

None - follows established testing pattern in existing file.

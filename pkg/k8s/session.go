package k8s

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubectl/pkg/scheme"
)

// DefaultImageTag can be overridden at build time via -ldflags
var DefaultImageTag = "v1"

// GetAgentImage returns the agent image to use, checking env var first
func GetAgentImage() string {
	if img := os.Getenv("PODSCOPE_AGENT_IMAGE"); img != "" {
		return img
	}
	return fmt.Sprintf("podscope-agent:%s", DefaultImageTag)
}

// GetHubImage returns the hub image to use, checking env var first
func GetHubImage() string {
	if img := os.Getenv("PODSCOPE_HUB_IMAGE"); img != "" {
		return img
	}
	return fmt.Sprintf("podscope:%s", DefaultImageTag)
}

// SessionOptions contains optional configuration for a session
type SessionOptions struct {
	AnthropicAPIKey string // API key for AI features in the Hub
}

// Session manages a PodScope capture session
type Session struct {
	client          *Client
	id              string
	namespace       string
	hubService      string
	portForwarder   *portforward.PortForwarder
	stopChan        chan struct{}
	anthropicAPIKey string
}

// NewSession creates a new capture session
func NewSession(client *Client, opts SessionOptions) (*Session, error) {
	id := uuid.New().String()[:8]
	return &Session{
		client:          client,
		id:              id,
		namespace:       fmt.Sprintf("podscope-%s", id),
		hubService:      "podscope-hub",
		stopChan:        make(chan struct{}),
		anthropicAPIKey: opts.AnthropicAPIKey,
	}, nil
}

// getHubEnvVars returns the environment variables for the Hub deployment
func (s *Session) getHubEnvVars() []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name:  "SESSION_ID",
			Value: s.id,
		},
	}

	if s.anthropicAPIKey != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "ANTHROPIC_API_KEY",
			Value: s.anthropicAPIKey,
		})
	}

	return envVars
}

// Start initializes the session by creating namespace and deploying hub
func (s *Session) Start(ctx context.Context) error {
	// Check for ephemeral container support
	if err := s.client.CheckEphemeralContainerSupport(ctx); err != nil {
		return err
	}

	// Create session namespace
	if err := s.createNamespace(ctx); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Deploy the Hub
	if err := s.deployHub(ctx); err != nil {
		return fmt.Errorf("failed to deploy hub: %w", err)
	}

	// Wait for Hub to be ready
	if err := s.waitForHub(ctx); err != nil {
		return fmt.Errorf("hub failed to become ready: %w", err)
	}

	return nil
}

// createNamespace creates the session namespace
func (s *Session) createNamespace(ctx context.Context) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "podscope",
				"app.kubernetes.io/component":  "session",
				"app.kubernetes.io/managed-by": "podscope-cli",
				"podscope.io/session-id":       s.id,
			},
			Annotations: map[string]string{
				"podscope.io/created-at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	_, err := s.client.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	fmt.Printf("Created namespace: %s\n", s.namespace)
	return nil
}

// deployHub creates the Hub deployment and service
func (s *Session) deployHub(ctx context.Context) error {
	labels := map[string]string{
		"app.kubernetes.io/name":      "podscope-hub",
		"app.kubernetes.io/component": "hub",
		"podscope.io/session-id":      s.id,
	}

	serviceAccountName := "podscope-hub"
	replicas := int32(1)

	// Create ServiceAccount for hub
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: s.namespace,
			Labels:    labels,
		},
	}
	_, err := s.client.clientset.CoreV1().ServiceAccounts(s.namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	// Create ClusterRole with permissions for terminal exec
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("podscope-hub-%s", s.id),
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
	_, err = s.client.clientset.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}

	// Create ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("podscope-hub-%s", s.id),
			Labels: labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     fmt.Sprintf("podscope-hub-%s", s.id),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: s.namespace,
			},
		},
	}
	_, err = s.client.clientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role binding: %w", err)
	}

	// Create Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.hubService,
			Namespace: s.namespace,
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
							Env: s.getHubEnvVars(),
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
									SizeLimit: resource.NewQuantity(1024*1024*1024, resource.BinarySI), // 1Gi
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = s.client.clientset.AppsV1().Deployments(s.namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.hubService,
			Namespace: s.namespace,
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

	_, err = s.client.clientset.CoreV1().Services(s.namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create service: %w", err)
	}

	fmt.Printf("Deployed Hub in namespace %s\n", s.namespace)
	return nil
}

// waitForHub waits for the Hub deployment to be ready
func (s *Session) waitForHub(ctx context.Context) error {
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	fmt.Print("Waiting for Hub to be ready")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for Hub to be ready")
		case <-ticker.C:
			deployment, err := s.client.clientset.AppsV1().Deployments(s.namespace).Get(ctx, s.hubService, metav1.GetOptions{})
			if err != nil {
				continue
			}

			if deployment.Status.ReadyReplicas >= 1 {
				fmt.Println(" Ready!")
				return nil
			}
			fmt.Print(".")
		}
	}
}

// InjectAgent injects a capture agent into a target pod.
// If targetContainer is empty, it defaults to the first container in the pod.
// The agent shares the process namespace with the target container, enabling
// process debugging (ps, strace, etc.) from within the ephemeral container.
func (s *Session) InjectAgent(ctx context.Context, target PodTarget, privileged bool, targetContainer string) error {
	// Get the current pod
	pod, err := s.client.clientset.CoreV1().Pods(target.Namespace).Get(ctx, target.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod: %w", err)
	}

	// Check if a podscope agent is already RUNNING in this pod
	// (Terminated agents from previous sessions are OK - we'll create a new one)
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

	// Determine target container for process namespace sharing
	if targetContainer == "" {
		// Default to the first container in the pod
		if len(pod.Spec.Containers) > 0 {
			targetContainer = pod.Spec.Containers[0].Name
		}
	} else {
		// Validate the specified container exists
		found := false
		for _, c := range pod.Spec.Containers {
			if c.Name == targetContainer {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("container %q not found in pod %s/%s", targetContainer, target.Namespace, target.Name)
		}
	}

	// Create the ephemeral container spec with unique name per session
	agentName := fmt.Sprintf("podscope-agent-%s", s.id)

	securityContext := &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{"NET_RAW"},
		},
	}

	if privileged {
		t := true
		securityContext.Privileged = &t
	}

	hubAddress := fmt.Sprintf("%s.%s.svc.cluster.local:9090", s.hubService, s.namespace)

	ephemeralContainer := corev1.EphemeralContainer{
		TargetContainerName: targetContainer,
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:            agentName,
			Image:           GetAgentImage(),
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: securityContext,
			// Note: Resource limits cannot be set on ephemeral containers (Kubernetes limitation)
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
					Value: s.id,
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

	_, err = s.client.clientset.CoreV1().Pods(target.Namespace).UpdateEphemeralContainers(
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

// StartPortForward starts port-forwarding to the Hub
func (s *Session) StartPortForward(ctx context.Context, localPort int) (int, error) {
	// Find the Hub pod
	pods, err := s.client.clientset.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", "podscope-hub"),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to find hub pod: %w", err)
	}

	if len(pods.Items) == 0 {
		return 0, fmt.Errorf("no hub pod found")
	}

	hubPod := pods.Items[0]

	port, err := chooseAvailablePort(localPort)
	if err != nil {
		return 0, err
	}
	if port != localPort && localPort > 0 {
		fmt.Printf("Local port %d is unavailable, using %d instead.\n", localPort, port)
	}

	// Build the port-forward request
	req := s.client.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(s.namespace).
		Name(hubPod.Name).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(s.client.restConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to create round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{
		Scheme: "https",
		Path:   req.URL().Path,
		Host:   strings.TrimPrefix(s.client.restConfig.Host, "https://"),
	})

	ports := []string{fmt.Sprintf("%d:8080", port)}
	readyChan := make(chan struct{})

	fw, err := portforward.New(dialer, ports, s.stopChan, readyChan, nil, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create port forwarder: %w", err)
	}

	s.portForwarder = fw

	// Start port-forwarding in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := fw.ForwardPorts(); err != nil {
			errChan <- err
		}
	}()

	// Wait for port-forward to be ready
	select {
	case <-readyChan:
		return port, nil
	case err := <-errChan:
		return 0, fmt.Errorf("failed to start port-forward: %w", err)
	case <-time.After(10 * time.Second):
		return 0, fmt.Errorf("timeout waiting for port-forward")
	}
}

func chooseAvailablePort(preferred int) (int, error) {
	if preferred > 0 && isPortAvailable(preferred) {
		return preferred, nil
	}

	const attempts = 10
	for i := 0; i < attempts; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			continue
		}

		port := listener.Addr().(*net.TCPAddr).Port
		_ = listener.Close()

		if isPortAvailable(port) {
			return port, nil
		}
	}

	if preferred > 0 {
		return 0, fmt.Errorf("local port %d is not available", preferred)
	}

	return 0, fmt.Errorf("failed to find a free local port")
}

func isPortAvailable(port int) bool {
	if port <= 0 {
		return false
	}

	v4 := canListen(fmt.Sprintf("127.0.0.1:%d", port))
	v6 := canListen(fmt.Sprintf("[::1]:%d", port))

	return v4 || v6
}

func canListen(addr string) bool {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

// Cleanup removes all session resources
func (s *Session) Cleanup(ctx context.Context) error {
	// Stop port-forwarder
	if s.stopChan != nil {
		close(s.stopChan)
	}

	// Clean up cluster-scoped RBAC resources (not deleted by namespace cascade)
	rbacName := fmt.Sprintf("podscope-hub-%s", s.id)

	// Delete ClusterRoleBinding first (depends on ClusterRole)
	err := s.client.clientset.RbacV1().ClusterRoleBindings().Delete(ctx, rbacName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		fmt.Fprintf(os.Stderr, "Warning: failed to delete ClusterRoleBinding %s: %v\n", rbacName, err)
	}

	// Delete ClusterRole after ClusterRoleBinding
	err = s.client.clientset.RbacV1().ClusterRoles().Delete(ctx, rbacName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		fmt.Fprintf(os.Stderr, "Warning: failed to delete ClusterRole %s: %v\n", rbacName, err)
	}

	// Delete the session namespace (this cascades to namespace-scoped resources)
	err = s.client.clientset.CoreV1().Namespaces().Delete(ctx, s.namespace, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	fmt.Printf("Deleted namespace: %s\n", s.namespace)
	return nil
}

// Namespace returns the session namespace
func (s *Session) Namespace() string {
	return s.namespace
}

// ID returns the session ID
func (s *Session) ID() string {
	return s.id
}

// TerminalStreams wraps the I/O streams for terminal exec
type TerminalStreams interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

// ExecInPod executes a shell in a pod and streams I/O via the provided streams
func (s *Session) ExecInPod(ctx context.Context, namespace, podName, container string, streams TerminalStreams) error {
	req := s.client.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{"/bin/sh"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(s.client.restConfig, http.MethodPost, req.URL())
	if err != nil {
		return fmt.Errorf("failed to create SPDY executor: %w", err)
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             streams,
		Stdout:            streams,
		Stderr:            streams,
		Tty:               true,
		TerminalSizeQueue: streams,
	})
}

// GetAgentContainer finds the podscope agent ephemeral container in a pod
func (s *Session) GetAgentContainer(ctx context.Context, namespace, podName string) (string, error) {
	pod, err := s.client.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	for _, ec := range pod.Spec.EphemeralContainers {
		if strings.HasPrefix(ec.Name, "podscope-agent") {
			return ec.Name, nil
		}
	}

	return "", fmt.Errorf("no podscope agent container found in pod %s/%s", namespace, podName)
}

// Client returns the underlying Kubernetes client
func (s *Session) Client() *Client {
	return s.client
}

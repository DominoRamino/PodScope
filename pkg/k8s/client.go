package k8s

import (
	"context"
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client wraps the Kubernetes clientset
type Client struct {
	clientset  *kubernetes.Clientset
	restConfig *rest.Config
}

// PodTarget represents a pod to capture traffic from
type PodTarget struct {
	Name      string
	Namespace string
	IP        string
	Node      string
}

// NewClient creates a new Kubernetes client using the default kubeconfig
func NewClient() (*Client, error) {
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		// Try in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{
		clientset:  clientset,
		restConfig: config,
	}, nil
}

// GetPodByName returns a single pod by name
func (c *Client) GetPodByName(ctx context.Context, namespace, name string) ([]PodTarget, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
	}

	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("pod %s/%s is not running (phase: %s)", namespace, name, pod.Status.Phase)
	}

	return []PodTarget{{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		IP:        pod.Status.PodIP,
		Node:      pod.Spec.NodeName,
	}}, nil
}

// GetPodsBySelector returns pods matching a label selector
func (c *Client) GetPodsBySelector(ctx context.Context, namespace, selector string) ([]PodTarget, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}

	var pods *corev1.PodList
	var err error

	if namespace == "" {
		pods, err = c.clientset.CoreV1().Pods("").List(ctx, listOptions)
	} else {
		pods, err = c.clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
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

// Clientset returns the underlying Kubernetes clientset
func (c *Client) Clientset() *kubernetes.Clientset {
	return c.clientset
}

// RESTConfig returns the REST config
func (c *Client) RESTConfig() *rest.Config {
	return c.restConfig
}

// CheckEphemeralContainerSupport verifies that the cluster supports ephemeral containers
func (c *Client) CheckEphemeralContainerSupport(ctx context.Context) error {
	// Ephemeral containers are GA in Kubernetes 1.25+
	// We'll check the server version
	version, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get server version: %w", err)
	}

	fmt.Printf("Kubernetes server version: %s.%s\n", version.Major, version.Minor)
	return nil
}

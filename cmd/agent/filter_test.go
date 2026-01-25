package main

import (
	"strings"
	"testing"
)

// TestBuildHubExclusionFilter_WithBothIPs tests the precise filter when both pod IP and hub IP are available
func TestBuildHubExclusionFilter_WithBothIPs(t *testing.T) {
	// Use localhost which should resolve to 127.0.0.1 or ::1
	filter, hubIP := buildHubExclusionFilter("localhost:9090", "10.0.0.5")

	// Should have resolved hub IP
	if hubIP == "" {
		t.Fatal("hubIP should not be empty when localhost resolves")
	}

	// Filter should contain both pod IP and hub IP (bidirectional)
	if !strings.Contains(filter, "host 10.0.0.5") {
		t.Errorf("filter should contain 'host 10.0.0.5', got: %s", filter)
	}

	// Filter should contain hub IP
	if !strings.Contains(filter, "host "+hubIP) {
		t.Errorf("filter should contain 'host %s', got: %s", hubIP, filter)
	}

	// Filter should use tcp ports
	if !strings.Contains(filter, "tcp port 8080") || !strings.Contains(filter, "tcp port 9090") {
		t.Errorf("filter should contain 'tcp port 8080' and 'tcp port 9090', got: %s", filter)
	}

	// Filter should be a negation
	if !strings.HasPrefix(filter, "not ") {
		t.Errorf("filter should start with 'not ', got: %s", filter)
	}
}

// TestBuildHubExclusionFilter_WithPodIPOnly tests fallback when hub resolution fails
func TestBuildHubExclusionFilter_WithPodIPOnly(t *testing.T) {
	// Use an invalid hostname that won't resolve
	filter, hubIP := buildHubExclusionFilter("invalid-hostname-that-wont-resolve.local:9090", "10.0.0.5")

	// Hub IP should be empty since resolution failed
	if hubIP != "" {
		t.Errorf("hubIP should be empty when resolution fails, got: %s", hubIP)
	}

	// Filter should still work using pod IP constraint only
	if filter == "" {
		t.Fatal("filter should not be empty when pod IP is available")
	}

	// Should use host constraint (bidirectional)
	if !strings.Contains(filter, "host 10.0.0.5") {
		t.Errorf("filter should contain 'host 10.0.0.5', got: %s", filter)
	}

	// Should use tcp ports
	if !strings.Contains(filter, "tcp port 8080") || !strings.Contains(filter, "tcp port 9090") {
		t.Errorf("filter should contain 'tcp port 8080' and 'tcp port 9090', got: %s", filter)
	}
}

// TestBuildHubExclusionFilter_NoPodIP tests fallback when only hub IP is available
func TestBuildHubExclusionFilter_NoPodIP(t *testing.T) {
	// Use localhost which resolves, but no pod IP
	filter, hubIP := buildHubExclusionFilter("localhost:9090", "")

	// Should have resolved hub IP
	if hubIP == "" {
		t.Fatal("hubIP should not be empty when localhost resolves")
	}

	// Filter should fall back to host-based filter (less precise)
	if !strings.Contains(filter, "host "+hubIP) {
		t.Errorf("filter should contain 'host %s', got: %s", hubIP, filter)
	}

	// Should NOT have src/dst constraints since we don't know pod IP
	if strings.Contains(filter, "src host") || strings.Contains(filter, "dst host") {
		t.Errorf("filter should not have src/dst constraints without pod IP, got: %s", filter)
	}
}

// TestBuildHubExclusionFilter_NoIPs tests when neither IP is available
func TestBuildHubExclusionFilter_NoIPs(t *testing.T) {
	// Use invalid hostname and no pod IP
	filter, hubIP := buildHubExclusionFilter("invalid-hostname-that-wont-resolve.local:9090", "")

	// Should return empty filter (capture everything)
	if filter != "" {
		t.Errorf("filter should be empty when no IPs available, got: %s", filter)
	}

	if hubIP != "" {
		t.Errorf("hubIP should be empty when resolution fails, got: %s", hubIP)
	}
}

// TestBuildHubExclusionFilter_HostnameWithPort tests that port is correctly stripped
func TestBuildHubExclusionFilter_HostnameWithPort(t *testing.T) {
	// The function should strip the port before resolution
	filter, hubIP := buildHubExclusionFilter("localhost:9090", "10.0.0.5")

	// Should successfully resolve
	if hubIP == "" {
		t.Fatal("should resolve localhost")
	}

	// Filter should be valid
	if filter == "" {
		t.Fatal("filter should not be empty")
	}
}

// TestBuildHubExclusionFilter_HostnameWithoutPort tests hostname without port
func TestBuildHubExclusionFilter_HostnameWithoutPort(t *testing.T) {
	// Should handle hostname without port
	filter, hubIP := buildHubExclusionFilter("localhost", "10.0.0.5")

	// Should successfully resolve
	if hubIP == "" {
		t.Fatal("should resolve localhost")
	}

	// Filter should be valid
	if filter == "" {
		t.Fatal("filter should not be empty")
	}
}

// TestBuildHubExclusionFilter_IPv6HubAddress tests IPv6 hub address
func TestBuildHubExclusionFilter_IPv6PodIP(t *testing.T) {
	// Use localhost and an IPv6 pod IP
	filter, hubIP := buildHubExclusionFilter("localhost:9090", "::1")

	if hubIP == "" {
		t.Fatal("should resolve localhost")
	}

	// Filter should contain the IPv6 pod IP
	if !strings.Contains(filter, "::1") {
		t.Errorf("filter should contain IPv6 pod IP '::1', got: %s", filter)
	}
}

// TestBuildHubExclusionFilter_PreservesLegitimateTraffic verifies the filter won't block
// legitimate traffic to other services on 8080/9090
func TestBuildHubExclusionFilter_PreservesLegitimateTraffic(t *testing.T) {
	filter, hubIP := buildHubExclusionFilter("localhost:9090", "10.0.0.5")

	if hubIP == "" {
		t.Fatal("should resolve localhost")
	}

	// The filter should be specific enough that traffic to OTHER services on 8080/9090
	// is NOT filtered. This is verified by checking that BOTH pod and hub IPs are required.
	// "host A and host B" means both endpoints must match, so traffic to other services won't match.
	if !strings.Contains(filter, "host 10.0.0.5") || !strings.Contains(filter, "host "+hubIP) {
		t.Errorf("filter should require both pod and hub IPs to preserve legitimate traffic, got: %s", filter)
	}
}

// TestBuildHubExclusionFilter_KubernetesStyleHostname tests with K8s service DNS format
func TestBuildHubExclusionFilter_KubernetesStyleHostname(t *testing.T) {
	// This will fail to resolve but tests the parsing logic
	filter, hubIP := buildHubExclusionFilter("podscope-hub.podscope-abc123.svc.cluster.local:9090", "10.244.0.15")

	// Hub IP will be empty (hostname won't resolve outside cluster)
	// But pod IP fallback should work
	if hubIP == "" && filter == "" {
		t.Error("should have pod IP fallback filter even when hub resolution fails")
	}

	// If no hub IP, should fall back to pod IP constraint only (bidirectional)
	if hubIP == "" {
		if !strings.Contains(filter, "host 10.244.0.15") {
			t.Errorf("fallback filter should use pod IP, got: %s", filter)
		}
	}
}

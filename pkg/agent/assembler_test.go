package agent

import (
	"testing"
)

// Test flowKey normalization - ensures bidirectional flows produce identical keys
func TestFlowKey_SourceIPLessThanDest(t *testing.T) {
	// When source IP is lexically less than dest IP, key uses src-dst order
	key := flowKey("10.0.0.1", "10.0.0.5", 12345, 80)
	expected := "10.0.0.1:12345-10.0.0.5:80"
	if key != expected {
		t.Errorf("flowKey() = %q, want %q", key, expected)
	}
}

func TestFlowKey_SourceIPGreaterThanDest(t *testing.T) {
	// When source IP is lexically greater than dest IP, key swaps to dst-src order
	key := flowKey("10.0.0.5", "10.0.0.1", 80, 12345)
	expected := "10.0.0.1:12345-10.0.0.5:80"
	if key != expected {
		t.Errorf("flowKey() = %q, want %q", key, expected)
	}
}

func TestFlowKey_SameIPSortsBy_Port(t *testing.T) {
	// When IPs are equal, lower source port comes first
	key := flowKey("192.168.1.1", "192.168.1.1", 8080, 3000)
	expected := "192.168.1.1:3000-192.168.1.1:8080"
	if key != expected {
		t.Errorf("flowKey() = %q, want %q", key, expected)
	}
}

func TestFlowKey_SameIPHigherPortFirst(t *testing.T) {
	// When IPs are equal and source port is already lower, key is src-dst
	key := flowKey("192.168.1.1", "192.168.1.1", 3000, 8080)
	expected := "192.168.1.1:3000-192.168.1.1:8080"
	if key != expected {
		t.Errorf("flowKey() = %q, want %q", key, expected)
	}
}

func TestFlowKey_BidirectionalEquivalence(t *testing.T) {
	// A->B and B->A must produce identical keys
	keyAtoB := flowKey("192.168.1.10", "10.0.0.5", 45678, 80)
	keyBtoA := flowKey("10.0.0.5", "192.168.1.10", 80, 45678)

	if keyAtoB != keyBtoA {
		t.Errorf("flowKey bidirectional mismatch: A->B=%q, B->A=%q", keyAtoB, keyBtoA)
	}
}

func TestFlowKey_BidirectionalWithSameIP(t *testing.T) {
	// Even with same IP, direction shouldn't matter
	key1 := flowKey("127.0.0.1", "127.0.0.1", 5000, 3000)
	key2 := flowKey("127.0.0.1", "127.0.0.1", 3000, 5000)

	if key1 != key2 {
		t.Errorf("flowKey bidirectional (same IP) mismatch: %q vs %q", key1, key2)
	}
}

func TestFlowKey_IPv6Addresses(t *testing.T) {
	// Test that IPv6 addresses also normalize correctly
	keyAtoB := flowKey("::1", "2001:db8::1", 8080, 443)
	keyBtoA := flowKey("2001:db8::1", "::1", 443, 8080)

	if keyAtoB != keyBtoA {
		t.Errorf("flowKey IPv6 bidirectional mismatch: A->B=%q, B->A=%q", keyAtoB, keyBtoA)
	}
}

func TestFlowKey_ConsistentFormat(t *testing.T) {
	// Verify the key format is IP:port-IP:port
	key := flowKey("10.0.0.1", "10.0.0.2", 1234, 5678)

	// Since 10.0.0.1 < 10.0.0.2 lexically, it should be src-dst order
	expected := "10.0.0.1:1234-10.0.0.2:5678"
	if key != expected {
		t.Errorf("flowKey() = %q, want %q", key, expected)
	}
}

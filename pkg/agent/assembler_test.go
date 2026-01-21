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

// Test isHTTPMethod - verifies detection of HTTP request/response patterns

func TestIsHTTPMethod_DetectsGET(t *testing.T) {
	payload := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if !isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return true for GET request")
	}
}

func TestIsHTTPMethod_DetectsPOST(t *testing.T) {
	payload := []byte("POST /api/users HTTP/1.1\r\nHost: example.com\r\nContent-Length: 0\r\n\r\n")
	if !isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return true for POST request")
	}
}

func TestIsHTTPMethod_DetectsPUT(t *testing.T) {
	payload := []byte("PUT /api/users/1 HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if !isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return true for PUT request")
	}
}

func TestIsHTTPMethod_DetectsDELETE(t *testing.T) {
	payload := []byte("DELETE /api/users/1 HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if !isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return true for DELETE request")
	}
}

func TestIsHTTPMethod_DetectsHEAD(t *testing.T) {
	payload := []byte("HEAD /api/health HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if !isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return true for HEAD request")
	}
}

func TestIsHTTPMethod_DetectsOPTIONS(t *testing.T) {
	payload := []byte("OPTIONS /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if !isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return true for OPTIONS request")
	}
}

func TestIsHTTPMethod_DetectsPATCH(t *testing.T) {
	payload := []byte("PATCH /api/users/1 HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if !isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return true for PATCH request")
	}
}

func TestIsHTTPMethod_DetectsCONNECT(t *testing.T) {
	payload := []byte("CONNECT example.com:443 HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if !isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return true for CONNECT request")
	}
}

func TestIsHTTPMethod_DetectsHTTPResponse(t *testing.T) {
	payload := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{}")
	if !isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return true for HTTP response")
	}
}

func TestIsHTTPMethod_DetectsHTTP10Response(t *testing.T) {
	payload := []byte("HTTP/1.0 404 Not Found\r\nContent-Type: text/plain\r\n\r\n")
	if !isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return true for HTTP/1.0 response")
	}
}

func TestIsHTTPMethod_RejectsTLSClientHello(t *testing.T) {
	// TLS ClientHello starts with 0x16 0x03 (handshake record, TLS version)
	payload := []byte{0x16, 0x03, 0x01, 0x00, 0x05, 0x01, 0x00, 0x00, 0x01, 0x00}
	if isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return false for TLS ClientHello")
	}
}

func TestIsHTTPMethod_RejectsBinaryData(t *testing.T) {
	// Random binary data
	payload := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xAB, 0xCD, 0xEF}
	if isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return false for binary data")
	}
}

func TestIsHTTPMethod_RejectsEmptyPayload(t *testing.T) {
	payload := []byte{}
	if isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return false for empty payload")
	}
}

func TestIsHTTPMethod_RejectsRandomText(t *testing.T) {
	payload := []byte("Hello, this is not HTTP")
	if isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return false for random text")
	}
}

func TestIsHTTPMethod_CaseSensitive_LowercaseGET(t *testing.T) {
	// HTTP methods are case-sensitive per RFC 7230 - lowercase should NOT match
	payload := []byte("get /api/users HTTP/1.1\r\n")
	if isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return false for lowercase 'get' (HTTP methods are case-sensitive)")
	}
}

func TestIsHTTPMethod_CaseSensitive_LowercasePOST(t *testing.T) {
	payload := []byte("post /api/users HTTP/1.1\r\n")
	if isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return false for lowercase 'post'")
	}
}

func TestIsHTTPMethod_CaseSensitive_MixedCaseHttp(t *testing.T) {
	// "Http/1.1" should not match "HTTP/"
	payload := []byte("Http/1.1 200 OK\r\n")
	if isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return false for mixed case 'Http/'")
	}
}

func TestIsHTTPMethod_RequiresSpaceAfterMethod(t *testing.T) {
	// "GETDATA" should not match - method must be followed by space
	payload := []byte("GETDATA /api HTTP/1.1\r\n")
	if isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return false when method isn't followed by space")
	}
}

func TestIsHTTPMethod_RejectsPartialMethod(t *testing.T) {
	// "GE" or "POS" should not match
	payload := []byte("GE")
	if isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return false for partial method 'GE'")
	}
}

func TestIsHTTPMethod_RejectsSimilarButInvalidMethod(t *testing.T) {
	// "GETTER " looks like it could be a method but isn't
	payload := []byte("GETTER /resource HTTP/1.1\r\n")
	if isHTTPMethod(payload) {
		t.Error("isHTTPMethod() should return false for invalid method 'GETTER'")
	}
}

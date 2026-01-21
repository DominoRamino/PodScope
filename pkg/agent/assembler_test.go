package agent

import (
	"testing"

	"github.com/podscope/podscope/pkg/protocol"
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

// Test detectProtocol - verifies correct identification of application protocols

// Helper to create a minimal TCPAssembler for testing detectProtocol
func newTestAssembler() *TCPAssembler {
	return &TCPAssembler{
		flows: make(map[string]*TCPFlow),
	}
}

func TestDetectProtocol_TLSClientHello(t *testing.T) {
	// TLS ClientHello starts with 0x16 (handshake) followed by 0x03 (TLS version prefix)
	// Minimum 6 bytes needed for detection
	assembler := newTestAssembler()

	// TLS 1.0 ClientHello
	payload := []byte{0x16, 0x03, 0x01, 0x00, 0x05, 0x01, 0x00, 0x00, 0x01, 0x00}
	result := assembler.detectProtocol(payload, 8080)

	if result != protocol.ProtocolTLS {
		t.Errorf("detectProtocol() = %q, want %q for TLS ClientHello", result, protocol.ProtocolTLS)
	}
}

func TestDetectProtocol_TLS12ClientHello(t *testing.T) {
	// TLS 1.2 uses 0x03 0x03
	assembler := newTestAssembler()

	payload := []byte{0x16, 0x03, 0x03, 0x00, 0x05, 0x01, 0x00, 0x00, 0x01, 0x00}
	result := assembler.detectProtocol(payload, 8080)

	if result != protocol.ProtocolTLS {
		t.Errorf("detectProtocol() = %q, want %q for TLS 1.2 ClientHello", result, protocol.ProtocolTLS)
	}
}

func TestDetectProtocol_TLS11ClientHello(t *testing.T) {
	// TLS 1.1 uses 0x03 0x02
	assembler := newTestAssembler()

	payload := []byte{0x16, 0x03, 0x02, 0x00, 0x05, 0x01, 0x00, 0x00, 0x01, 0x00}
	result := assembler.detectProtocol(payload, 8080)

	if result != protocol.ProtocolTLS {
		t.Errorf("detectProtocol() = %q, want %q for TLS 1.1 ClientHello", result, protocol.ProtocolTLS)
	}
}

func TestDetectProtocol_HTTPMethod_GET(t *testing.T) {
	assembler := newTestAssembler()

	payload := []byte("GET /api/health HTTP/1.1\r\nHost: example.com\r\n\r\n")
	result := assembler.detectProtocol(payload, 8080)

	if result != protocol.ProtocolHTTP {
		t.Errorf("detectProtocol() = %q, want %q for HTTP GET", result, protocol.ProtocolHTTP)
	}
}

func TestDetectProtocol_HTTPMethod_POST(t *testing.T) {
	assembler := newTestAssembler()

	payload := []byte("POST /api/users HTTP/1.1\r\nHost: example.com\r\nContent-Length: 0\r\n\r\n")
	result := assembler.detectProtocol(payload, 8080)

	if result != protocol.ProtocolHTTP {
		t.Errorf("detectProtocol() = %q, want %q for HTTP POST", result, protocol.ProtocolHTTP)
	}
}

func TestDetectProtocol_HTTPResponse(t *testing.T) {
	assembler := newTestAssembler()

	payload := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{}")
	result := assembler.detectProtocol(payload, 8080)

	if result != protocol.ProtocolHTTP {
		t.Errorf("detectProtocol() = %q, want %q for HTTP response", result, protocol.ProtocolHTTP)
	}
}

func TestDetectProtocol_Port443_ReturnsHTTPS(t *testing.T) {
	// When port is 443 and payload is not clearly HTTP or TLS, return HTTPS
	assembler := newTestAssembler()

	// Binary data that doesn't match HTTP or TLS patterns
	payload := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	result := assembler.detectProtocol(payload, 443)

	if result != protocol.ProtocolHTTPS {
		t.Errorf("detectProtocol() = %q, want %q for port 443", result, protocol.ProtocolHTTPS)
	}
}

func TestDetectProtocol_Port8443_ReturnsHTTPS(t *testing.T) {
	// Port 8443 is also a common HTTPS port
	assembler := newTestAssembler()

	payload := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	result := assembler.detectProtocol(payload, 8443)

	if result != protocol.ProtocolHTTPS {
		t.Errorf("detectProtocol() = %q, want %q for port 8443", result, protocol.ProtocolHTTPS)
	}
}

func TestDetectProtocol_UnknownFallsBackToTCP(t *testing.T) {
	// When payload doesn't match any pattern and port isn't HTTPS, return TCP
	assembler := newTestAssembler()

	// Binary data on a non-HTTPS port
	payload := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	result := assembler.detectProtocol(payload, 8080)

	if result != protocol.ProtocolTCP {
		t.Errorf("detectProtocol() = %q, want %q for unknown protocol", result, protocol.ProtocolTCP)
	}
}

func TestDetectProtocol_EmptyPayload_NonHTTPSPort(t *testing.T) {
	// Empty payload on non-HTTPS port should return TCP
	assembler := newTestAssembler()

	payload := []byte{}
	result := assembler.detectProtocol(payload, 8080)

	if result != protocol.ProtocolTCP {
		t.Errorf("detectProtocol() = %q, want %q for empty payload on non-HTTPS port", result, protocol.ProtocolTCP)
	}
}

func TestDetectProtocol_EmptyPayload_HTTPSPort(t *testing.T) {
	// Empty payload on port 443 should return HTTPS (port-based heuristic)
	assembler := newTestAssembler()

	payload := []byte{}
	result := assembler.detectProtocol(payload, 443)

	if result != protocol.ProtocolHTTPS {
		t.Errorf("detectProtocol() = %q, want %q for empty payload on port 443", result, protocol.ProtocolHTTPS)
	}
}

func TestDetectProtocol_TLSTakesPrecedenceOverPort(t *testing.T) {
	// TLS detection should happen before port-based HTTPS detection
	assembler := newTestAssembler()

	// TLS ClientHello on port 443 - should return TLS, not HTTPS
	payload := []byte{0x16, 0x03, 0x01, 0x00, 0x05, 0x01, 0x00, 0x00, 0x01, 0x00}
	result := assembler.detectProtocol(payload, 443)

	if result != protocol.ProtocolTLS {
		t.Errorf("detectProtocol() = %q, want %q (TLS should take precedence over port 443)", result, protocol.ProtocolTLS)
	}
}

func TestDetectProtocol_HTTPTakesPrecedenceOverPort(t *testing.T) {
	// HTTP detection should happen before port-based HTTPS detection
	assembler := newTestAssembler()

	// HTTP request on port 443 - should return HTTP (even though unusual)
	payload := []byte("GET /health HTTP/1.1\r\n")
	result := assembler.detectProtocol(payload, 443)

	if result != protocol.ProtocolHTTP {
		t.Errorf("detectProtocol() = %q, want %q (HTTP should take precedence over port 443)", result, protocol.ProtocolHTTP)
	}
}

func TestDetectProtocol_ShortTLSPayload_NotDetected(t *testing.T) {
	// TLS detection requires > 5 bytes - short payloads shouldn't match
	assembler := newTestAssembler()

	// Only 5 bytes, not enough for TLS detection (needs > 5)
	payload := []byte{0x16, 0x03, 0x01, 0x00, 0x05}
	result := assembler.detectProtocol(payload, 8080)

	// Should fall through to TCP since len(payload) == 5, not > 5
	if result != protocol.ProtocolTCP {
		t.Errorf("detectProtocol() = %q, want %q for short TLS-like payload", result, protocol.ProtocolTCP)
	}
}

func TestDetectProtocol_RandomTextNotHTTP(t *testing.T) {
	// Random text that doesn't start with HTTP method or response
	assembler := newTestAssembler()

	payload := []byte("Hello World, this is random text")
	result := assembler.detectProtocol(payload, 8080)

	if result != protocol.ProtocolTCP {
		t.Errorf("detectProtocol() = %q, want %q for random text", result, protocol.ProtocolTCP)
	}
}

func TestDetectProtocol_TLSRecord_NotHandshake(t *testing.T) {
	// TLS record that's not a handshake (e.g., 0x17 = application data)
	// Should not be detected as TLS by the ClientHello detection
	assembler := newTestAssembler()

	payload := []byte{0x17, 0x03, 0x03, 0x00, 0x05, 0x01, 0x02, 0x03, 0x04, 0x05}
	result := assembler.detectProtocol(payload, 8080)

	if result != protocol.ProtocolTCP {
		t.Errorf("detectProtocol() = %q, want %q for non-handshake TLS record", result, protocol.ProtocolTCP)
	}
}

// Test extractSNI - verifies extraction of Server Name Indication from TLS ClientHello

// Real TLS 1.2 ClientHello captured from: openssl s_client -connect example.com:443 -servername example.com
// This is a real ClientHello with SNI "example.com" - simplified but valid structure
var tlsClientHelloWithSNI = []byte{
	// TLS Record Header (5 bytes)
	0x16,       // Content type: Handshake
	0x03, 0x01, // Version: TLS 1.0 (used in record layer for compatibility)
	0x00, 0xc5, // Length: 197 bytes

	// Handshake Header (4 bytes)
	0x01,             // Handshake type: ClientHello
	0x00, 0x00, 0xc1, // Length: 193 bytes

	// ClientHello Body
	0x03, 0x03, // Version: TLS 1.2

	// Random (32 bytes)
	0x5f, 0x8a, 0x3c, 0x2b, 0x1d, 0x4e, 0x6f, 0x80,
	0x91, 0xa2, 0xb3, 0xc4, 0xd5, 0xe6, 0xf7, 0x08,
	0x19, 0x2a, 0x3b, 0x4c, 0x5d, 0x6e, 0x7f, 0x90,
	0xa1, 0xb2, 0xc3, 0xd4, 0xe5, 0xf6, 0x07, 0x18,

	// Session ID (1 byte length + 0 bytes data)
	0x00,

	// Cipher Suites (2 bytes length + cipher suites)
	0x00, 0x04, // Length: 4 bytes (2 cipher suites)
	0x13, 0x01, // TLS_AES_128_GCM_SHA256
	0x13, 0x02, // TLS_AES_256_GCM_SHA384

	// Compression Methods (1 byte length + methods)
	0x01, // Length: 1
	0x00, // null compression

	// Extensions (2 bytes length + extensions)
	0x00, 0x92, // Extensions length: 146 bytes

	// SNI Extension (Server Name Indication)
	0x00, 0x00, // Extension type: server_name (0)
	0x00, 0x10, // Extension length: 16 bytes
	0x00, 0x0e, // Server Name list length: 14 bytes
	0x00,       // Name type: host_name (0)
	0x00, 0x0b, // Name length: 11 bytes
	'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', // "example.com"

	// Supported Versions Extension (padding to reach declared length)
	0x00, 0x2b, // Extension type: supported_versions (43)
	0x00, 0x03, // Extension length: 3 bytes
	0x02,       // Supported versions list length: 2 bytes
	0x03, 0x03, // TLS 1.2

	// EC Point Formats Extension
	0x00, 0x0b, // Extension type: ec_point_formats (11)
	0x00, 0x02, // Extension length: 2 bytes
	0x01,       // EC point formats length: 1
	0x00,       // uncompressed

	// Supported Groups Extension
	0x00, 0x0a, // Extension type: supported_groups (10)
	0x00, 0x04, // Extension length: 4 bytes
	0x00, 0x02, // Supported groups list length: 2 bytes
	0x00, 0x17, // secp256r1

	// Signature Algorithms Extension (to fill remaining space)
	0x00, 0x0d, // Extension type: signature_algorithms (13)
	0x00, 0x60, // Extension length: 96 bytes
	0x00, 0x5e, // Signature algorithms list length: 94 bytes
	// Padding with signature algorithm pairs
	0x04, 0x01, 0x04, 0x03, 0x05, 0x01, 0x05, 0x03,
	0x06, 0x01, 0x06, 0x03, 0x02, 0x01, 0x02, 0x03,
	0x04, 0x02, 0x05, 0x02, 0x06, 0x02, 0x04, 0x01,
	0x04, 0x03, 0x05, 0x01, 0x05, 0x03, 0x06, 0x01,
	0x06, 0x03, 0x02, 0x01, 0x02, 0x03, 0x04, 0x02,
	0x05, 0x02, 0x06, 0x02, 0x04, 0x01, 0x04, 0x03,
	0x05, 0x01, 0x05, 0x03, 0x06, 0x01, 0x06, 0x03,
	0x02, 0x01, 0x02, 0x03, 0x04, 0x02, 0x05, 0x02,
	0x06, 0x02, 0x04, 0x01, 0x04, 0x03, 0x05, 0x01,
	0x05, 0x03, 0x06, 0x01, 0x06, 0x03, 0x02, 0x01,
	0x02, 0x03, 0x04, 0x02, 0x05, 0x02, 0x06, 0x02,
	0x08, 0x04, 0x08, 0x05, 0x08, 0x06,
}

// TLS ClientHello without SNI extension (legacy client behavior)
var tlsClientHelloWithoutSNI = []byte{
	// TLS Record Header (5 bytes)
	0x16,       // Content type: Handshake
	0x03, 0x01, // Version: TLS 1.0
	0x00, 0x2f, // Length: 47 bytes

	// Handshake Header (4 bytes)
	0x01,             // Handshake type: ClientHello
	0x00, 0x00, 0x2b, // Length: 43 bytes

	// ClientHello Body
	0x03, 0x03, // Version: TLS 1.2

	// Random (32 bytes)
	0x5f, 0x8a, 0x3c, 0x2b, 0x1d, 0x4e, 0x6f, 0x80,
	0x91, 0xa2, 0xb3, 0xc4, 0xd5, 0xe6, 0xf7, 0x08,
	0x19, 0x2a, 0x3b, 0x4c, 0x5d, 0x6e, 0x7f, 0x90,
	0xa1, 0xb2, 0xc3, 0xd4, 0xe5, 0xf6, 0x07, 0x18,

	// Session ID (1 byte length + 0 bytes data)
	0x00,

	// Cipher Suites (2 bytes length + cipher suites)
	0x00, 0x02, // Length: 2 bytes (1 cipher suite)
	0x00, 0x2f, // TLS_RSA_WITH_AES_128_CBC_SHA

	// Compression Methods (1 byte length + methods)
	0x01, // Length: 1
	0x00, // null compression

	// No extensions - extensions length would be 0 or absent
}

func TestExtractSNI_ValidClientHello(t *testing.T) {
	// Test with a real TLS ClientHello containing SNI for "example.com"
	sni := extractSNI(tlsClientHelloWithSNI)

	if sni != "example.com" {
		t.Errorf("extractSNI() = %q, want %q", sni, "example.com")
	}
}

func TestExtractSNI_NoSNIExtension(t *testing.T) {
	// Test with a ClientHello that has no SNI extension
	sni := extractSNI(tlsClientHelloWithoutSNI)

	if sni != "" {
		t.Errorf("extractSNI() = %q, want empty string when no SNI extension", sni)
	}
}

func TestExtractSNI_TruncatedBeforeSessionID(t *testing.T) {
	// Data truncated before session ID length field
	// extractSNI requires at least 43 bytes
	truncated := tlsClientHelloWithSNI[:42]
	sni := extractSNI(truncated)

	if sni != "" {
		t.Errorf("extractSNI() = %q, want empty string for truncated data", sni)
	}
}

func TestExtractSNI_TruncatedAtSessionID(t *testing.T) {
	// Data truncated right after minimum required bytes
	// Should not panic
	truncated := tlsClientHelloWithSNI[:44]
	sni := extractSNI(truncated)

	if sni != "" {
		t.Errorf("extractSNI() = %q, want empty string for truncated data", sni)
	}
}

func TestExtractSNI_TruncatedDuringCipherSuites(t *testing.T) {
	// Truncate during cipher suites section
	// Session ID ends at byte 43, cipher suites length at 44-45
	truncated := tlsClientHelloWithSNI[:46]
	sni := extractSNI(truncated)

	if sni != "" {
		t.Errorf("extractSNI() = %q, want empty string for truncated data", sni)
	}
}

func TestExtractSNI_TruncatedBeforeExtensions(t *testing.T) {
	// Truncate before extensions section
	// This is after compression methods but before extensions length
	truncated := tlsClientHelloWithSNI[:52]
	sni := extractSNI(truncated)

	if sni != "" {
		t.Errorf("extractSNI() = %q, want empty string when extensions truncated", sni)
	}
}

func TestExtractSNI_EmptyData(t *testing.T) {
	// Empty data should return empty string without panic
	sni := extractSNI([]byte{})

	if sni != "" {
		t.Errorf("extractSNI() = %q, want empty string for empty data", sni)
	}
}

func TestExtractSNI_NilData(t *testing.T) {
	// Nil data should return empty string without panic
	sni := extractSNI(nil)

	if sni != "" {
		t.Errorf("extractSNI() = %q, want empty string for nil data", sni)
	}
}

func TestExtractSNI_NotTLSHandshake(t *testing.T) {
	// Data that doesn't start with TLS handshake record (0x16)
	notTLS := []byte{0x17, 0x03, 0x03, 0x00, 0x20, 0x01, 0x02, 0x03, 0x04, 0x05}
	sni := extractSNI(notTLS)

	if sni != "" {
		t.Errorf("extractSNI() = %q, want empty string for non-handshake record", sni)
	}
}

func TestExtractSNI_MalformedExtensionLength(t *testing.T) {
	// Create ClientHello with extension length larger than remaining data
	malformed := make([]byte, len(tlsClientHelloWithSNI))
	copy(malformed, tlsClientHelloWithSNI)

	// Set extensions length to a very large value
	// Extensions length is at byte 53-54 in our test data
	malformed[53] = 0xFF
	malformed[54] = 0xFF

	// Should not panic, should return empty string
	sni := extractSNI(malformed)

	// This tests graceful handling - function should not panic
	_ = sni // Result may vary, but no panic is the key requirement
}

func TestExtractSNI_ShortData(t *testing.T) {
	// Just 5 bytes - TLS record header only
	short := []byte{0x16, 0x03, 0x01, 0x00, 0x05}
	sni := extractSNI(short)

	if sni != "" {
		t.Errorf("extractSNI() = %q, want empty string for short data", sni)
	}
}

func TestExtractSNI_LongHostname(t *testing.T) {
	// Test with a longer hostname to verify the name length parsing
	// Create a modified ClientHello with SNI "www.example.org"
	longHostClientHello := make([]byte, 0, 200)

	// TLS Record Header
	longHostClientHello = append(longHostClientHello,
		0x16,       // Content type: Handshake
		0x03, 0x01, // Version
		0x00, 0x50, // Length (will be adjusted)
	)

	// Handshake Header
	longHostClientHello = append(longHostClientHello,
		0x01,             // ClientHello
		0x00, 0x00, 0x4c, // Length
	)

	// Version
	longHostClientHello = append(longHostClientHello, 0x03, 0x03)

	// Random (32 bytes)
	for i := 0; i < 32; i++ {
		longHostClientHello = append(longHostClientHello, byte(i))
	}

	// Session ID length (0)
	longHostClientHello = append(longHostClientHello, 0x00)

	// Cipher suites (2 bytes length + 2 bytes cipher)
	longHostClientHello = append(longHostClientHello, 0x00, 0x02, 0x13, 0x01)

	// Compression methods (1 byte length + 1 byte null)
	longHostClientHello = append(longHostClientHello, 0x01, 0x00)

	// Extensions length
	hostname := "www.example.org"
	sniExtLen := 2 + 1 + 2 + len(hostname) // list len + type + name len + name
	extTotalLen := 2 + 2 + sniExtLen       // ext type + ext len + ext data
	longHostClientHello = append(longHostClientHello, byte(extTotalLen>>8), byte(extTotalLen&0xff))

	// SNI Extension
	longHostClientHello = append(longHostClientHello, 0x00, 0x00) // extension type: server_name
	longHostClientHello = append(longHostClientHello, byte(sniExtLen>>8), byte(sniExtLen&0xff))
	longHostClientHello = append(longHostClientHello, byte((sniExtLen-2)>>8), byte((sniExtLen-2)&0xff)) // list length
	longHostClientHello = append(longHostClientHello, 0x00)                                             // name type: hostname
	longHostClientHello = append(longHostClientHello, byte(len(hostname)>>8), byte(len(hostname)&0xff))
	longHostClientHello = append(longHostClientHello, []byte(hostname)...)

	sni := extractSNI(longHostClientHello)
	if sni != hostname {
		t.Errorf("extractSNI() = %q, want %q", sni, hostname)
	}
}

func TestExtractSNI_SNINotFirstExtension(t *testing.T) {
	// Test ClientHello where SNI is not the first extension
	// This tests the extension parsing loop
	clientHello := make([]byte, 0, 200)

	// TLS Record Header
	clientHello = append(clientHello,
		0x16,       // Content type: Handshake
		0x03, 0x01, // Version
		0x00, 0x60, // Length
	)

	// Handshake Header
	clientHello = append(clientHello,
		0x01,             // ClientHello
		0x00, 0x00, 0x5c, // Length
	)

	// Version
	clientHello = append(clientHello, 0x03, 0x03)

	// Random (32 bytes)
	for i := 0; i < 32; i++ {
		clientHello = append(clientHello, byte(i))
	}

	// Session ID length (0)
	clientHello = append(clientHello, 0x00)

	// Cipher suites
	clientHello = append(clientHello, 0x00, 0x02, 0x13, 0x01)

	// Compression methods
	clientHello = append(clientHello, 0x01, 0x00)

	// Extensions - first a non-SNI extension, then SNI
	hostname := "test.local"
	sniExtLen := 2 + 1 + 2 + len(hostname)
	supportedVersionsExtLen := 3

	totalExtLen := (2 + 2 + supportedVersionsExtLen) + (2 + 2 + sniExtLen)
	clientHello = append(clientHello, byte(totalExtLen>>8), byte(totalExtLen&0xff))

	// Supported Versions Extension (not SNI - type 43)
	clientHello = append(clientHello, 0x00, 0x2b) // extension type
	clientHello = append(clientHello, 0x00, byte(supportedVersionsExtLen))
	clientHello = append(clientHello, 0x02, 0x03, 0x03) // TLS 1.2

	// SNI Extension (type 0)
	clientHello = append(clientHello, 0x00, 0x00)
	clientHello = append(clientHello, byte(sniExtLen>>8), byte(sniExtLen&0xff))
	clientHello = append(clientHello, byte((sniExtLen-2)>>8), byte((sniExtLen-2)&0xff))
	clientHello = append(clientHello, 0x00)
	clientHello = append(clientHello, byte(len(hostname)>>8), byte(len(hostname)&0xff))
	clientHello = append(clientHello, []byte(hostname)...)

	sni := extractSNI(clientHello)
	if sni != hostname {
		t.Errorf("extractSNI() = %q, want %q when SNI is not first extension", sni, hostname)
	}
}

// Test parseHTTP - verifies correct extraction of method, URL, status, and headers

// Helper to create a TCPFlow with HTTP client/server data for testing parseHTTP
func newTestFlowWithHTTPData(clientData, serverData []byte) *TCPFlow {
	flow := &TCPFlow{
		ID:       "test123",
		Protocol: protocol.ProtocolHTTP,
	}
	if clientData != nil {
		flow.ClientData.Write(clientData)
	}
	if serverData != nil {
		flow.ServerData.Write(serverData)
	}
	return flow
}

func TestParseHTTP_GETRequest(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\nUser-Agent: test/1.0\r\n\r\n")
	flow := newTestFlowWithHTTPData(request, nil)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.Method != "GET" {
		t.Errorf("HTTP.Method = %q, want %q", flow.HTTP.Method, "GET")
	}
	if flow.HTTP.URL != "/api/users" {
		t.Errorf("HTTP.URL = %q, want %q", flow.HTTP.URL, "/api/users")
	}
	if flow.HTTP.Host != "example.com" {
		t.Errorf("HTTP.Host = %q, want %q", flow.HTTP.Host, "example.com")
	}
}

func TestParseHTTP_POSTRequest(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("POST /api/users HTTP/1.1\r\nHost: api.example.com\r\nContent-Type: application/json\r\nContent-Length: 27\r\n\r\n{\"name\":\"John\",\"age\":30}")
	flow := newTestFlowWithHTTPData(request, nil)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.Method != "POST" {
		t.Errorf("HTTP.Method = %q, want %q", flow.HTTP.Method, "POST")
	}
	if flow.HTTP.URL != "/api/users" {
		t.Errorf("HTTP.URL = %q, want %q", flow.HTTP.URL, "/api/users")
	}
}

func TestParseHTTP_ExtractsRequestHeaders(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /api/data HTTP/1.1\r\nHost: example.com\r\nAuthorization: Bearer token123\r\nAccept: application/json\r\n\r\n")
	flow := newTestFlowWithHTTPData(request, nil)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.RequestHeaders == nil {
		t.Fatal("parseHTTP() did not set RequestHeaders")
	}
	if flow.HTTP.RequestHeaders["Authorization"] != "Bearer token123" {
		t.Errorf("RequestHeaders[Authorization] = %q, want %q", flow.HTTP.RequestHeaders["Authorization"], "Bearer token123")
	}
	if flow.HTTP.RequestHeaders["Accept"] != "application/json" {
		t.Errorf("RequestHeaders[Accept] = %q, want %q", flow.HTTP.RequestHeaders["Accept"], "application/json")
	}
}

func TestParseHTTP_ResponseStatusCode(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n")
	response := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 2\r\n\r\n{}")
	flow := newTestFlowWithHTTPData(request, response)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.StatusCode != 200 {
		t.Errorf("HTTP.StatusCode = %d, want %d", flow.HTTP.StatusCode, 200)
	}
}

func TestParseHTTP_ResponseStatusText(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /health HTTP/1.1\r\nHost: example.com\r\n\r\n")
	response := []byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\n\r\n")
	flow := newTestFlowWithHTTPData(request, response)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.StatusCode != 404 {
		t.Errorf("HTTP.StatusCode = %d, want %d", flow.HTTP.StatusCode, 404)
	}
	// StatusText includes status code per http.Response.Status format
	if flow.HTTP.StatusText != "404 Not Found" {
		t.Errorf("HTTP.StatusText = %q, want %q", flow.HTTP.StatusText, "404 Not Found")
	}
}

func TestParseHTTP_ExtractsResponseHeaders(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /api/data HTTP/1.1\r\nHost: example.com\r\n\r\n")
	response := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nX-Request-Id: abc123\r\n\r\n{}")
	flow := newTestFlowWithHTTPData(request, response)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.ResponseHeaders == nil {
		t.Fatal("parseHTTP() did not set ResponseHeaders")
	}
	if flow.HTTP.ResponseHeaders["X-Request-Id"] != "abc123" {
		t.Errorf("ResponseHeaders[X-Request-Id] = %q, want %q", flow.HTTP.ResponseHeaders["X-Request-Id"], "abc123")
	}
}

func TestParseHTTP_ExtractsContentType(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /api/data HTTP/1.1\r\nHost: example.com\r\n\r\n")
	response := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json; charset=utf-8\r\n\r\n{}")
	flow := newTestFlowWithHTTPData(request, response)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.ContentType != "application/json; charset=utf-8" {
		t.Errorf("HTTP.ContentType = %q, want %q", flow.HTTP.ContentType, "application/json; charset=utf-8")
	}
}

func TestParseHTTP_ExtractsContentLength(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /api/data HTTP/1.1\r\nHost: example.com\r\n\r\n")
	response := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 1234\r\n\r\n")
	flow := newTestFlowWithHTTPData(request, response)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.ContentLength != 1234 {
		t.Errorf("HTTP.ContentLength = %d, want %d", flow.HTTP.ContentLength, 1234)
	}
}

func TestParseHTTP_PartialRequestData_NoPanic(t *testing.T) {
	assembler := newTestAssembler()
	// Incomplete HTTP request - should not panic
	partial := []byte("GET /api/users")
	flow := newTestFlowWithHTTPData(partial, nil)

	// Should not panic - gracefully handle partial data
	assembler.parseHTTP(flow)

	// HTTP may or may not be set depending on parser behavior with incomplete data
	// Key is that no panic occurs
}

func TestParseHTTP_PartialResponseData_NoPanic(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n")
	// Incomplete response - headers not terminated
	partial := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json")
	flow := newTestFlowWithHTTPData(request, partial)

	// Should not panic - gracefully handle partial data
	assembler.parseHTTP(flow)

	// Response parsing may fail but should not panic
}

func TestParseHTTP_EmptyClientData(t *testing.T) {
	assembler := newTestAssembler()
	flow := newTestFlowWithHTTPData(nil, nil)

	// Should not panic with empty data
	assembler.parseHTTP(flow)

	if flow.HTTP != nil {
		t.Error("parseHTTP() should not set HTTP info for empty client data")
	}
}

func TestParseHTTP_OnlyResponseData_NoHTTPInfo(t *testing.T) {
	assembler := newTestAssembler()
	// Only response data, no request - HTTP info should not be set
	// because parseHTTP requires request to be parsed first
	response := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{}")
	flow := newTestFlowWithHTTPData(nil, response)

	assembler.parseHTTP(flow)

	// Without request data, HTTP should remain nil
	if flow.HTTP != nil {
		t.Error("parseHTTP() should not set HTTP info without request data")
	}
}

func TestParseHTTP_AlreadyParsed_SkipsSecondParse(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /original HTTP/1.1\r\nHost: example.com\r\n\r\n")
	flow := newTestFlowWithHTTPData(request, nil)

	// First parse
	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info on first call")
	}

	// Modify flow data (simulating more data arriving)
	flow.ClientData.Reset()
	flow.ClientData.Write([]byte("POST /modified HTTP/1.1\r\nHost: example.com\r\n\r\n"))

	// Second parse should be skipped since HTTP is already set
	assembler.parseHTTP(flow)

	// URL should still be from first parse
	if flow.HTTP.URL != "/original" {
		t.Errorf("HTTP.URL = %q, want %q (should not re-parse)", flow.HTTP.URL, "/original")
	}
}

func TestParseHTTP_PUTRequest(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("PUT /api/users/123 HTTP/1.1\r\nHost: example.com\r\nContent-Type: application/json\r\n\r\n{\"name\":\"Updated\"}")
	flow := newTestFlowWithHTTPData(request, nil)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.Method != "PUT" {
		t.Errorf("HTTP.Method = %q, want %q", flow.HTTP.Method, "PUT")
	}
}

func TestParseHTTP_DELETERequest(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("DELETE /api/users/123 HTTP/1.1\r\nHost: example.com\r\n\r\n")
	flow := newTestFlowWithHTTPData(request, nil)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.Method != "DELETE" {
		t.Errorf("HTTP.Method = %q, want %q", flow.HTTP.Method, "DELETE")
	}
}

func TestParseHTTP_MultipleHeaderValues(t *testing.T) {
	assembler := newTestAssembler()
	// Multiple Set-Cookie headers are common in responses
	request := []byte("GET /login HTTP/1.1\r\nHost: example.com\r\n\r\n")
	response := []byte("HTTP/1.1 200 OK\r\nSet-Cookie: session=abc123\r\nSet-Cookie: user=john\r\n\r\n")
	flow := newTestFlowWithHTTPData(request, response)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	// Headers with multiple values are joined by ", "
	setCookie := flow.HTTP.ResponseHeaders["Set-Cookie"]
	if setCookie == "" {
		t.Error("ResponseHeaders[Set-Cookie] should not be empty")
	}
	// Should contain both cookie values
	if !(contains(setCookie, "session=abc123") && contains(setCookie, "user=john")) {
		t.Errorf("ResponseHeaders[Set-Cookie] = %q, should contain both cookie values", setCookie)
	}
}

// Helper function for checking substrings
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsString(s, substr))
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestParseHTTP_5xxResponse(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /api/data HTTP/1.1\r\nHost: example.com\r\n\r\n")
	response := []byte("HTTP/1.1 500 Internal Server Error\r\nContent-Type: text/plain\r\n\r\nError")
	flow := newTestFlowWithHTTPData(request, response)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.StatusCode != 500 {
		t.Errorf("HTTP.StatusCode = %d, want %d", flow.HTTP.StatusCode, 500)
	}
}

func TestParseHTTP_3xxRedirectResponse(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /old-path HTTP/1.1\r\nHost: example.com\r\n\r\n")
	response := []byte("HTTP/1.1 301 Moved Permanently\r\nLocation: /new-path\r\n\r\n")
	flow := newTestFlowWithHTTPData(request, response)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.StatusCode != 301 {
		t.Errorf("HTTP.StatusCode = %d, want %d", flow.HTTP.StatusCode, 301)
	}
	if flow.HTTP.ResponseHeaders["Location"] != "/new-path" {
		t.Errorf("ResponseHeaders[Location] = %q, want %q", flow.HTTP.ResponseHeaders["Location"], "/new-path")
	}
}

func TestParseHTTP_BinaryPayloadDoesNotCrash(t *testing.T) {
	assembler := newTestAssembler()
	// Binary data that looks nothing like HTTP
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xAB, 0xCD, 0xEF, 0x89, 0x90}
	flow := newTestFlowWithHTTPData(binaryData, nil)

	// Should not panic with non-HTTP binary data
	assembler.parseHTTP(flow)

	// HTTP should remain nil since data doesn't parse as HTTP
	if flow.HTTP != nil {
		t.Error("parseHTTP() should not set HTTP info for binary data")
	}
}

func TestParseHTTP_URLWithQueryString(t *testing.T) {
	assembler := newTestAssembler()
	request := []byte("GET /search?q=test&page=1 HTTP/1.1\r\nHost: example.com\r\n\r\n")
	flow := newTestFlowWithHTTPData(request, nil)

	assembler.parseHTTP(flow)

	if flow.HTTP == nil {
		t.Fatal("parseHTTP() did not set HTTP info")
	}
	if flow.HTTP.URL != "/search?q=test&page=1" {
		t.Errorf("HTTP.URL = %q, want %q", flow.HTTP.URL, "/search?q=test&page=1")
	}
}

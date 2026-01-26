package agent

import "fmt"

// cipherSuiteNames maps TLS cipher suite IDs to their human-readable names.
// Covers common TLS 1.2 and TLS 1.3 cipher suites.
var cipherSuiteNames = map[uint16]string{
	// TLS 1.3 cipher suites (RFC 8446)
	0x1301: "TLS_AES_128_GCM_SHA256",
	0x1302: "TLS_AES_256_GCM_SHA384",
	0x1303: "TLS_CHACHA20_POLY1305_SHA256",
	0x1304: "TLS_AES_128_CCM_SHA256",
	0x1305: "TLS_AES_128_CCM_8_SHA256",

	// TLS 1.2 ECDHE cipher suites
	0xc02b: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	0xc02c: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	0xc02f: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	0xc030: "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	0xc023: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
	0xc024: "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384",
	0xc027: "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
	0xc028: "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384",
	0xc009: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
	0xc00a: "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	0xc013: "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
	0xc014: "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",

	// TLS 1.2 ECDHE with ChaCha20-Poly1305
	0xcca8: "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
	0xcca9: "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
	0xccaa: "TLS_DHE_RSA_WITH_CHACHA20_POLY1305_SHA256",

	// TLS 1.2 DHE cipher suites
	0x009e: "TLS_DHE_RSA_WITH_AES_128_GCM_SHA256",
	0x009f: "TLS_DHE_RSA_WITH_AES_256_GCM_SHA384",
	0x0067: "TLS_DHE_RSA_WITH_AES_128_CBC_SHA256",
	0x006b: "TLS_DHE_RSA_WITH_AES_256_CBC_SHA256",
	0x0033: "TLS_DHE_RSA_WITH_AES_128_CBC_SHA",
	0x0039: "TLS_DHE_RSA_WITH_AES_256_CBC_SHA",

	// TLS 1.2 RSA cipher suites (no forward secrecy)
	0x009c: "TLS_RSA_WITH_AES_128_GCM_SHA256",
	0x009d: "TLS_RSA_WITH_AES_256_GCM_SHA384",
	0x003c: "TLS_RSA_WITH_AES_128_CBC_SHA256",
	0x003d: "TLS_RSA_WITH_AES_256_CBC_SHA256",
	0x002f: "TLS_RSA_WITH_AES_128_CBC_SHA",
	0x0035: "TLS_RSA_WITH_AES_256_CBC_SHA",

	// Legacy cipher suites (often seen in negotiation)
	0x000a: "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	0x0016: "TLS_DHE_RSA_WITH_3DES_EDE_CBC_SHA",
	0xc008: "TLS_ECDHE_ECDSA_WITH_3DES_EDE_CBC_SHA",
	0xc012: "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",

	// GREASE values (RFC 8701) - clients send these to detect buggy servers
	0x0a0a: "GREASE",
	0x1a1a: "GREASE",
	0x2a2a: "GREASE",
	0x3a3a: "GREASE",
	0x4a4a: "GREASE",
	0x5a5a: "GREASE",
	0x6a6a: "GREASE",
	0x7a7a: "GREASE",
	0x8a8a: "GREASE",
	0x9a9a: "GREASE",
	0xaaaa: "GREASE",
	0xbaba: "GREASE",
	0xcaca: "GREASE",
	0xdada: "GREASE",
	0xeaea: "GREASE",
	0xfafa: "GREASE",
}

// CipherSuiteName returns the human-readable name for a TLS cipher suite ID.
// If the cipher suite is unknown, it returns the hex representation.
func CipherSuiteName(id uint16) string {
	if name, ok := cipherSuiteNames[id]; ok {
		return name
	}
	return fmt.Sprintf("0x%04x", id)
}

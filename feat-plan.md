     Low-Hanging Fruit: Advanced Packet Metrics Implementation

     Overview

     Implement 5 "low-hanging fruit" features that extract additional data from
     captured packets. The fields already exist in the Go structs and TypeScript
     types but are never populated.

     Key Insight

     The UI already displays these fields (cipher suite, ALPN, body content, timing)
     - they just show as empty. Populating them requires only backend changes + minor
      UI polish.

     ---
     Features to Implement

     1. TLS Cipher Suite Extraction

     Effort: Low | Value: High

     The extractSNI() function already parses through cipher suites but skips them.
      We just need to capture the list before skipping.

     Files:
     - pkg/agent/assembler.go - Extract cipher suites in extractSNI()
     - Add cipher suite name mapping (e.g., 0x1301 → TLS_AES_128_GCM_SHA256)

     Implementation:
     // In extractSNI(), around line 331-333, before skipping:
     cipherSuitesLen := int(data[offset])<<8 | int(data[offset+1])
     cipherSuites := data[offset+2 : offset+2+cipherSuitesLen]
     // Parse 2-byte pairs and map to names

     2. ALPN Protocol Extraction

     Effort: Low | Value: Medium

     Add parsing for extension type 16 (ALPN) in the existing extension loop.

     Files:
     - pkg/agent/assembler.go - Add ALPN case in extension parsing loop (around
     line 370)

     Implementation:
     // In the extension parsing loop, add:
     case 16: // ALPN
         alpnLen := int(extData[0])<<8 | int(extData[1])
         // Parse protocol list (length-prefixed strings)

     3. HTTP Body Preview

     Effort: Low | Value: High

     Read first 1KB of request/response body after header parsing.

     Files:
     - pkg/agent/assembler.go - Enhance parseHTTP() (around line 240-260)

     Implementation:
     // After http.ReadRequest():
     if req.Body != nil {
         body := make([]byte, MaxBodySize) // 1024
         n, _ := req.Body.Read(body)
         flow.HTTP.RequestBody = string(body[:n])
     }
     // Similar for response

     4. Time-to-First-Byte (TTFB)

     Effort: Medium | Value: High

     Track when first server data arrives after request sent.

     Files:
     - pkg/agent/assembler.go:
       - Add FirstServerDataTime to flowState struct
       - Track in ProcessPacket() when server sends first data
       - Calculate in completeFlow()

     Implementation:
     // In flowState struct:
     FirstServerDataTime time.Time

     // In ProcessPacket(), when server data first seen:
     if !flow.FirstServerDataTime.IsZero() && isFromServer {
         flow.FirstServerDataTime = timestamp
     }

     // In completeFlow():
     if !flow.FirstServerDataTime.IsZero() {
         f.TimeToFirstByte =
     flow.FirstServerDataTime.Sub(flow.FirstDataTime).Milliseconds()
     }

     5. TLS Handshake Time

     Effort: Medium | Value: Medium

     Track ClientHello → ServerHello timing.

     Files:
     - pkg/agent/assembler.go:
       - Add ClientHelloTime and ServerHelloTime to flowState
       - Record ClientHello timestamp in parseTLS()
       - Parse ServerHello in server data to get completion time
       - Calculate in completeFlow()

     Challenge: ServerHello is in server data buffer, needs basic TLS record
     parsing.

     ---
     UI Enhancement: Advanced Metrics Section

     Add a collapsible "Advanced Metrics" section in FlowDetail between Data Transfer
      and TLS/HTTP sections.

     File: ui/src/components/FlowDetail.tsx

     Derived Insights (new section):
     - Performance Score - Visual indicator (good/warning/poor) based on TTFB
     thresholds
     - Throughput - Calculated bytes/second with human-readable formatting
     - TLS Security Grade - A/B/C/F based on cipher strength (deprecated ciphers
     = F)
     - Protocol Version - HTTP/1.1 vs HTTP/2 badge (derived from ALPN)

     Existing Fields to Populate:
     - TLS cipher suite display (already in TLS section - just empty)
     - ALPN protocols list (already in TLS section - just empty)
     - Request/Response body preview (already in HTTP section - just empty)
     - Timing bar segments for TTFB and TLS handshake (already exists - needs data)

     Styling:
     - Advanced Metrics section uses grid of StatCards with color-coded indicators
     - Body preview uses monospace font with max-height scroll
     - Security grade uses semantic colors (green=A, yellow=B, orange=C, red=F)

     ---
     Implementation Order

     1. Phase 1: Backend - TLS (cipher suite + ALPN)
       - Modify extractSNI() to capture cipher suites
       - Add ALPN extension parsing
       - Add cipher suite name mapping helper
     2. Phase 2: Backend - HTTP (body extraction)
       - Enhance parseHTTP() to read truncated bodies
       - Handle edge cases (chunked encoding, gzip)
     3. Phase 3: Backend - Timing (TTFB + TLS handshake)
       - Add timestamp tracking fields to flowState
       - Implement directional data detection
       - Calculate metrics in completeFlow()
     4. Phase 4: UI Polish
       - Add "Advanced Metrics" section to FlowDetail
       - Add security/performance indicators
       - Style body preview with syntax highlighting (optional)

     ---
     Files to Modify
     ┌──────────────────────────────────┬─────────────────────────────────────────────┐
     │               File               │                   Changes                   │
     ├──────────────────────────────────┼─────────────────────────────────────────────┤
     │ pkg/agent/assembler.go           │ Cipher suite extraction, ALPN parsing, body │
     ├──────────────────────────────────┼─────────────────────────────────────────────┤
     │ extraction, timing calculations  │                                             │
     ├──────────────────────────────────┼─────────────────────────────────────────────┤
     │ pkg/agent/cipher_suites.go       │ NEW - Cipher suite ID → name mapping        │
     ├──────────────────────────────────┼─────────────────────────────────────────────┤
     │ ui/src/components/FlowDetail.tsx │ Advanced Metrics section, body preview      │
     ├──────────────────────────────────┼─────────────────────────────────────────────┤
     │ styling                          │                                             │
     ├──────────────────────────────────┼─────────────────────────────────────────────┤
     │ ui/src/types.ts                  │ No changes needed (fields exist)            │
     ├──────────────────────────────────┼─────────────────────────────────────────────┤
     │ pkg/protocol/flow.go             │ No changes needed (fields exist)            │
     └──────────────────────────────────┴─────────────────────────────────────────────┘
     ---
     Verification

     1. TLS Cipher/ALPN:
       - Capture HTTPS traffic to any site
       - Verify cipher suite shows (e.g., "TLS_AES_256_GCM_SHA384")
       - Verify ALPN shows (e.g., ["h2", "http/1.1"])
     2. HTTP Body:
       - Capture plaintext HTTP traffic
       - Verify request/response body preview shows (truncated at 1KB)
     3. TTFB:
       - Capture HTTP request
       - Verify TimeToFirstByte populated in flow detail
       - Verify timing bar shows TTFB segment
     4. TLS Handshake Time:
       - Capture HTTPS traffic
       - Verify TLSHandshakeMs populated
       - Verify timing bar shows TLS segment separately from TCP
     5. UI:
       - Open flow detail panel
       - Verify Advanced Metrics section appears
       - Verify all new fields display correctly

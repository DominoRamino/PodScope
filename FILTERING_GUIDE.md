# PodScope Filtering System

## Overview

PodScope now captures **everything** but filters what you see by default. This gives you flexibility to explore different traffic types without restarting capture sessions.

---

## How It Works

### 1. Capture Layer (BPF Filter)
**What it blocks:** Only hub traffic (ports 8080/9090 to prevent feedback loops)
**What it captures:** Everything else - HTTP, HTTPS, DNS, TCP, UDP, etc.

```
BPF Filter: not (host <hub-ip> and (port 8080 or port 9090))
```

### 2. View Layer (UI Filters)
**Default view:** HTTP/HTTPS traffic only (ports 80, 443, 8080, 8443, 3000, 5000, 8000, 8888, 9090)
**Configurable:** Toggle to show DNS, all ports, or custom filters

### 3. Export Layer (PCAP Downloads)
**Default:** Downloads respect current UI filters
**Customizable:** Filter parameters passed to server (implementation pending)

---

## UI Filter Controls

Located in the header below the search bar:

### **HTTP/HTTPS Only** (Default: ON)
- Shows only common HTTP/HTTPS ports
- Includes: 80, 443, 8080, 8443, 3000, 5000, 8000, 8888, 9090
- Perfect for web traffic analysis
- Hides: DNS, database ports, custom protocols

### **Show DNS** (Default: OFF)
- Toggles visibility of DNS traffic (port 53)
- Useful for: debugging DNS issues, seeing resolution delays
- Works with other filters (e.g., HTTP + DNS)

### **Show All Ports** (Default: OFF)
- Disables port filtering entirely
- Shows all captured traffic
- Overrides "HTTP/HTTPS Only" setting

---

## Common Use Cases

### Java Application Debugging
```
✓ HTTP/HTTPS Only  (captures port 8080)
□ Show DNS
□ Show All Ports
Search: "my-app"
```

### DNS Troubleshooting
```
□ HTTP/HTTPS Only
✓ Show DNS
□ Show All Ports
Search: "google.com"
```

### Database Traffic Analysis
```
□ HTTP/HTTPS Only
□ Show DNS
✓ Show All Ports
Search: "5432"  (PostgreSQL)
```

### Full Network Visibility
```
□ HTTP/HTTPS Only
✓ Show DNS
✓ Show All Ports
```

---

## Filter Logic

### Priority Rules
1. **Show All Ports** overrides **HTTP/HTTPS Only**
2. **Show DNS** is independent (works with other filters)
3. Text search applies after port/protocol filtering

### Example Filtering Flow
```
Captured packets → Port filter → DNS filter → Text search → Display
```

**Example 1:** HTTP/HTTPS Only + Show DNS = OFF
- Result: Shows ports 80, 443, 8080, 8443, etc. (NO port 53)

**Example 2:** HTTP/HTTPS Only + Show DNS = ON
- Result: Shows ports 80, 443, 8080, 8443, etc. PLUS port 53

**Example 3:** Show All Ports + Show DNS = OFF
- Result: Shows all ports EXCEPT 53

---

## PCAP Download Filtering

When you download a PCAP file, current filter settings are passed to the server:

### Query Parameters
- `onlyHTTP=true` - Filter to HTTP/HTTPS ports
- `includeDNS=true` - Include DNS traffic
- `allPorts=true` - Include all ports
- `search=text` - Text filter

### Filename Suffixes
- `podscope-session-http.pcap` - HTTP/HTTPS filtered
- `podscope-session-all.pcap` - All ports
- `podscope-session.pcap` - Default (respects current filters)

### Current Status
✅ Filter parameters passed to server
⏳ Server-side packet filtering **not yet implemented**
📝 Currently returns all captured packets

**TODO:** Implement server-side PCAP filtering in `pkg/hub/server.go`:
- Parse filter parameters
- Read PCAP buffer
- Apply filters to individual packets
- Rebuild filtered PCAP file

---

## Technical Details

### HTTP Port List
```go
HTTP_PORTS = [80, 443, 8080, 8443, 3000, 5000, 8000, 8888, 9090]
```

### DNS Port
```go
DNS_PORT = 53
```

### Filter State
```typescript
interface FilterOptions {
  searchText: string
  showOnlyHTTP: boolean  // Default: true
  showDNS: boolean       // Default: false
  showAllPorts: boolean  // Default: false
}
```

---

## Build Info

**Version:** v3
**Changes:**
- Removed DNS from BPF filter (capture everything)
- Added UI filter toggles
- Default to HTTP/HTTPS view
- PCAP download filter parameters

---

## Testing Checklist

### UI Filtering
- [x] Default shows only HTTP/HTTPS traffic
- [x] "Show DNS" toggle reveals port 53
- [x] "Show All Ports" shows everything
- [x] Text search works with filters
- [x] Filter state persists during session

### Capture Verification
```bash
# Start capture
./podscope tap -n default -l app=podinfo

# Verify BPF filter in agent logs
kubectl logs <pod> -c <agent> | grep "BPF FILTER"
# Should NOT include "not port 53"

# Test DNS capture
kubectl exec <pod> -- nslookup google.com
# DNS flows should appear if "Show DNS" is enabled
```

### PCAP Download
```bash
# Download with filters
# 1. Enable "HTTP/HTTPS Only"
# 2. Click "Download PCAP"
# 3. Check hub logs for filter parameters
kubectl logs <hub-pod> -n podscope-* | grep "PCAP download with filters"
```

---

## Future Enhancements

### Server-Side PCAP Filtering
Implement actual packet filtering in PCAP exports:
- Use `gopacket` to read/write PCAP
- Apply BPF filters to individual packets
- Rebuild PCAP with only matching packets

### Additional Filters
- **Port Range:** "Show ports 8000-9000"
- **Protocol:** "TCP only", "UDP only"
- **Flow Direction:** "Inbound", "Outbound", "Pod-to-Pod"
- **Pod Selector:** "From pod X", "To pod Y"
- **Response Status:** "HTTP 2xx", "HTTP 5xx"

### Saved Filters
- Save common filter combinations
- Quick access to "Java Debug", "DNS Only", etc.
- Export/import filter presets

---

## Troubleshooting

### "I don't see DNS traffic even with Show DNS enabled"
1. Check agent logs: `kubectl logs <pod> -c <agent> | grep "BPF FILTER"`
2. Verify it does NOT include `not port 53`
3. Generate DNS traffic: `kubectl exec <pod> -- nslookup google.com`
4. Refresh UI or wait for WebSocket update

### "HTTP/HTTPS Only shows too much/too little"
The HTTP_PORTS list may need adjustment for your environment. Edit `ui/src/App.tsx`:
```typescript
const HTTP_PORTS = new Set([80, 443, 8080, 8443, 3000, 5000, 8000, 8888, 9090])
// Add your ports here
```

### "PCAP download includes filtered packets"
Server-side PCAP filtering is not yet implemented. Downloaded PCAP files contain ALL captured packets. Use Wireshark filters after download:
```
tcp.port in {80 443 8080 8443}  # HTTP/HTTPS
not dns                          # Exclude DNS
```

---

## Migration from Previous Versions

### Before (v1-v2)
- BPF filter: `not port 53 and not (host X and (port 8080 or port 9090))`
- DNS traffic never captured
- No UI filtering

### After (v3+)
- BPF filter: `not (host X and (port 8080 or port 9090))`
- All traffic captured
- UI filters with HTTP/HTTPS default
- Same visual experience, more flexibility

**No action required** - existing sessions continue working. Start new sessions to capture DNS.

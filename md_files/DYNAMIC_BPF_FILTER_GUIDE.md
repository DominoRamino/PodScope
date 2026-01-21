# Dynamic BPF Filter Update Guide

## Overview

**Version**: v4
**Feature**: Live BPF filter updates without restarting capture sessions

PodScope now supports **dynamic BPF filter updates** - you can change the packet capture filter on running agents without stopping the session. This is perfect for:
- Drilling down into specific traffic patterns
- Troubleshooting specific protocols (DNS, HTTP, database connections)
- Reducing capture noise during investigation
- Testing different filter expressions interactively

---

## How It Works

### Architecture

```
UI Input → Hub API → Health Heartbeat → Agent → Live BPF Update
   │          │            (5s interval)     │
   │          └─ Stores filter              └─ Applies to pcap handle
   │             in memory
   └─ User enters BPF filter expression
```

**Key Components:**

1. **Hub Server** (`/api/bpf-filter`)
   - Stores current BPF filter in memory
   - Exposes GET/POST endpoints for filter management
   - Returns filter in `/api/health` response

2. **Agent Heartbeat** (every 5 seconds)
   - Checks for filter updates in health response
   - Applies new filter if changed
   - Updates running pcap handle without restart

3. **UI Controls** (Header component)
   - Text input for BPF filter expression
   - Apply/Clear buttons
   - Visual indicator of active filter

---

## UI Usage

### Location

The BPF filter controls appear in the header, below the view filter toggles:

```
┌─────────────────────────────────────────────────────────────┐
│ 🦈 PodScope      [Search: ___________]    Status | Actions  │
├─────────────────────────────────────────────────────────────┤
│ Filter: [✓ HTTP/HTTPS Only] [Show DNS] [Show All Ports]    │
│ BPF Filter: [____________________] [Apply Filter] [Clear]  │
└─────────────────────────────────────────────────────────────┘
```

### Steps

1. **Enter BPF Filter Expression**
   ```
   Input: tcp port 80 or udp port 53
   ```

2. **Apply the Filter**
   - Click "Apply Filter" button
   - Or press Enter in the input field
   - Status changes to "Applying..."

3. **Wait for Propagation**
   - Agents update on next heartbeat (within 5 seconds)
   - "Active: your-filter" indicator appears

4. **Clear the Filter** (optional)
   - Click "Clear" button to reset to default
   - Captures all traffic except hub feedback loop

---

## BPF Filter Examples

### Common Use Cases

**Java/Spring Boot Debugging** (ports 8080, 8443)
```
tcp port 8080 or tcp port 8443
```

**DNS Only** (troubleshooting DNS issues)
```
udp port 53 or tcp port 53
```

**HTTP/HTTPS Traffic**
```
tcp port 80 or tcp port 443 or tcp port 8080
```

**PostgreSQL Database**
```
tcp port 5432
```

**MySQL/MariaDB Database**
```
tcp port 3306
```

**Redis Cache**
```
tcp port 6379
```

**Specific Host Communication**
```
host 10.244.0.5
```

**Exclude Specific Traffic**
```
not tcp port 22
```

**Complex Expression**
```
(tcp port 80 or tcp port 443) and host 10.244.0.5
```

**TCP SYN Packets Only** (connection analysis)
```
tcp[tcpflags] & tcp-syn != 0
```

---

## Technical Details

### API Endpoints

#### `POST /api/bpf-filter`
Updates the BPF filter on the hub.

**Request:**
```json
{
  "filter": "tcp port 80 or udp port 53"
}
```

**Response:**
```json
{
  "success": true,
  "filter": "tcp port 80 or udp port 53",
  "message": "BPF filter will be applied on next heartbeat"
}
```

#### `GET /api/bpf-filter`
Retrieves current BPF filter.

**Response:**
```json
{
  "filter": "tcp port 80"
}
```

#### `GET /api/health`
Health check that includes current BPF filter.

**Response:**
```json
{
  "status": "healthy",
  "sessionId": "abc123",
  "timestamp": "2026-01-16T06:20:00Z",
  "bpfFilter": "tcp port 80 or udp port 53"
}
```

### Agent Implementation

**Heartbeat Check** (`pkg/agent/client.go`):
```go
// Every 5 seconds, agent checks for BPF filter updates
resp := client.Get(hubURL + "/api/health")
if healthResp.BPFFilter != lastFilter {
    capturer.UpdateBPFFilter(healthResp.BPFFilter)
}
```

**Live Update** (`pkg/agent/capture.go`):
```go
func (c *Capturer) UpdateBPFFilter(filter string) error {
    // Apply to running pcap handle
    if err := c.handle.SetBPFFilter(filter); err != nil {
        return err
    }
    c.bpfFilter = filter
    return nil
}
```

### Filter Propagation Timeline

```
0s    User clicks "Apply Filter" → Hub stores filter
↓
1-5s  Agent sends heartbeat → Receives new filter
↓
<1s   Agent applies filter → Capture updated
↓
Total: 1-6 seconds typical latency
```

---

## Differences from View Filtering

PodScope has **two types** of filtering:

### 1. View Filters (UI Only) ✓ Already in v3
- **Location**: Header "Filter" row
- **Scope**: What you *see* in the UI
- **Impact**: Client-side JavaScript filtering
- **Speed**: Instant
- **Examples**: "HTTP/HTTPS Only", "Show DNS"

### 2. BPF Capture Filters (v4) ✓ NEW
- **Location**: Header "BPF Filter" row
- **Scope**: What agents *capture*
- **Impact**: Kernel-level packet filtering
- **Speed**: 1-6 seconds (heartbeat latency)
- **Examples**: "tcp port 80", "host 10.244.0.5"

**Use Cases:**

| Scenario | Use View Filter | Use BPF Filter |
|----------|----------------|----------------|
| Quick exploration | ✓ | |
| Reduce UI clutter | ✓ | |
| Reduce capture load | | ✓ |
| Focus on specific protocol | | ✓ |
| Save bandwidth | | ✓ |
| Reduce storage | | ✓ |
| Test filter expressions | ✓ (faster) | ✓ (thorough) |

---

## Testing

### Setup Test Environment

```bash
# 1. Start capture session with v4 images
./podscope tap -n default -l app=podinfo

# 2. Open UI
open http://localhost:8899

# 3. In another terminal, generate test traffic
kubectl exec -n default <pod> -- curl https://google.com
kubectl exec -n default <pod> -- nslookup google.com
kubectl exec -n default <pod> -- curl http://podinfo:9898
```

### Test Case 1: Apply DNS Filter

1. In UI, enter BPF filter: `udp port 53`
2. Click "Apply Filter"
3. Generate DNS traffic: `kubectl exec <pod> -- nslookup google.com`
4. **Expected**: Only DNS queries appear in flow list
5. Generate HTTP traffic: `kubectl exec <pod> -- curl http://google.com`
6. **Expected**: HTTP traffic does NOT appear

### Test Case 2: HTTP Port Filter

1. Clear previous filter
2. Enter: `tcp port 80 or tcp port 443`
3. Click "Apply Filter"
4. Generate HTTP: `kubectl exec <pod> -- curl http://google.com`
5. **Expected**: HTTP flows appear
6. Generate DNS: `kubectl exec <pod> -- nslookup google.com`
7. **Expected**: DNS does NOT appear

### Test Case 3: Clear Filter

1. Click "Clear" button
2. **Expected**: "Active: ..." indicator disappears
3. Generate mixed traffic
4. **Expected**: All traffic types appear (except hub feedback)

### Verify in Agent Logs

```bash
# Get agent container name
kubectl get pod <pod> -n default -o json | \
  jq '.spec.ephemeralContainers[].name'

# Check for filter update logs
kubectl logs <pod> -n default -c <agent-container> | grep "BPF"
```

**Expected Output:**
```
====================================
UPDATING BPF FILTER:
  Old: not (host 10.X.X.X and (port 8080 or port 9090))
  New: tcp port 80 or udp port 53
====================================
SUCCESS: BPF filter updated on running capture
```

---

## Troubleshooting

### Filter Not Applied

**Symptom:** Still seeing traffic that should be filtered out

**Checks:**
1. Verify "Active: ..." indicator shows your filter
2. Wait 5-10 seconds for heartbeat propagation
3. Check agent logs for "UPDATING BPF FILTER" message
4. Check agent logs for "ERROR: Failed to update BPF filter"

**Common Causes:**
- Invalid BPF syntax → Check hub logs for errors
- Agent not receiving heartbeat → Check network connectivity
- Ephemeral container issue → Restart capture session

### Invalid BPF Filter Syntax

**Symptom:** "Apply Filter" succeeds but agent logs show error

**Example Error:**
```
ERROR: Failed to update BPF filter: syntax error in filter expression
```

**Solution:**
- Use `tcpdump` syntax validator:
  ```bash
  tcpdump -i any "your-filter" -c 1
  ```
- Check for typos (e.g., `tcp port80` vs `tcp port 80`)
- Use parentheses for complex expressions: `(tcp port 80) or (udp port 53)`

### Heartbeat Not Working

**Symptom:** Filter never propagates to agents

**Checks:**
```bash
# 1. Check if agents can reach hub
kubectl logs <pod> -c <agent> | grep "Heartbeat"

# 2. Check hub health endpoint
curl http://localhost:8899/api/health

# 3. Verify filter stored in hub
curl http://localhost:8899/api/bpf-filter
```

### Filter Clears After Session Restart

**Expected Behavior:** BPF filters are **session-scoped**, not persistent.

**Workaround:**
- Reapply filter via UI after restarting
- Or set initial filter in agent startup code (requires rebuild)

---

## Limitations

### Current Limitations

1. **No Filter Persistence**
   - Filters reset when hub restarts
   - Must reapply after stopping/starting capture

2. **No Per-Agent Filters**
   - Same filter applied to all agents in session
   - Cannot filter differently per pod

3. **Heartbeat Latency**
   - 1-6 second delay before filter activates
   - No instant feedback on filter validity

4. **No Syntax Validation**
   - Invalid filters accepted by hub
   - Agents log errors but continue with old filter

### Future Enhancements

**Filter Validation**
```go
// Validate BPF syntax before storing
func validateBPFFilter(filter string) error {
    // Use libpcap to compile and validate
    _, err := pcap.CompileBPFFilter(...)
    return err
}
```

**Per-Agent Filters**
```json
POST /api/bpf-filter
{
  "filters": {
    "agent-abc123": "tcp port 8080",
    "agent-xyz789": "tcp port 5432"
  }
}
```

**Filter Presets**
```typescript
// UI dropdown with common filters
const FILTER_PRESETS = {
  "HTTP/HTTPS": "tcp port 80 or tcp port 443",
  "DNS": "udp port 53 or tcp port 53",
  "Databases": "tcp port 5432 or tcp port 3306 or tcp port 6379",
}
```

---

## Best Practices

### Performance Considerations

**DO:**
- ✅ Use specific filters to reduce capture load
- ✅ Combine filters efficiently: `tcp port 80 or tcp port 443`
- ✅ Filter at capture time for high-traffic pods
- ✅ Test filters with `tcpdump` before applying

**DON'T:**
- ❌ Use overly complex filters (slow to evaluate)
- ❌ Forget to clear filter when investigation complete
- ❌ Apply filters without testing syntax first

### Investigation Workflow

**Step 1: Start Broad**
```
Filter: (none)
Observe: All traffic patterns
```

**Step 2: Narrow Down**
```
Filter: tcp
Observe: TCP-only flows
```

**Step 3: Focus**
```
Filter: tcp port 8080 and host 10.244.0.5
Observe: Specific service communication
```

**Step 4: Clear**
```
Filter: (cleared)
Observe: Back to full visibility
```

---

## Version History

### v4 (Current) - Dynamic BPF Filters
- ✅ Live filter updates via API
- ✅ UI controls for filter management
- ✅ Heartbeat-based propagation
- ✅ Agent-side live pcap handle updates

### v3 - View Filtering
- ✅ UI-side flow filtering
- ✅ Default HTTP/HTTPS view
- ✅ DNS toggle
- ✅ All ports toggle

### v2 - Static BPF Filters
- ✅ Initial BPF filter on startup
- ✅ Hub exclusion filter
- ❌ No runtime updates

### v1 - No Filtering
- ❌ Captured everything including DNS noise

---

## Quick Reference

### BPF Filter Cheat Sheet

```bash
# Protocols
tcp                    # TCP only
udp                    # UDP only
icmp                   # ICMP only

# Ports
port 80                # Port 80 (any protocol)
tcp port 80            # TCP port 80
src port 80            # Source port 80
dst port 80            # Destination port 80

# Hosts
host 10.244.0.5        # Specific IP
src host 10.244.0.5    # Source IP
dst host 10.244.0.5    # Destination IP

# Networks
net 10.244.0.0/24      # Subnet

# Combinations
tcp and port 80        # TCP on port 80
tcp or udp             # TCP or UDP
not tcp port 22        # Everything except SSH
(tcp port 80) or (udp port 53)  # HTTP or DNS

# Advanced
tcp[tcpflags] & tcp-syn != 0    # SYN packets
greater 100                      # Packets > 100 bytes
less 100                         # Packets < 100 bytes
```

### UI Quick Keys

- **Enter** in BPF input → Apply filter
- **Esc** in BPF input → Clear input (not the active filter)

---

## Support & Feedback

**Issues with BPF filters?**
1. Check agent logs for errors
2. Validate syntax with `tcpdump`
3. Try simpler filter expression first
4. Report issues with filter expression and error logs

**Feature requests:**
- Per-agent filters
- Filter presets
- Syntax validation
- Filter history

---

## Summary

Build **v4** adds powerful dynamic BPF filtering to PodScope:

✅ **No session restart required**
✅ **Updates within 5 seconds**
✅ **Standard tcpdump syntax**
✅ **Visual feedback in UI**
✅ **Reduces capture overhead**

Perfect for interactive troubleshooting and focused investigation!

# PodScope Testing Checklist

## Current Build
**Version**: v0.1.0-20260116-052952
**Status**: Loaded in minikube, ready for testing

---

## Test 1: Start Fresh Session

```bash
# Stop any existing session (Ctrl+C if running)

# Start new capture
./podscope tap -n default -l app=podinfo

# Expected: Should see new pod created with fresh images
```

---

## Test 2: Verify Version Info

```bash
# Find the agent pod
kubectl get pods -n default -o json | \
  jq '.items[] | select(.spec.ephemeralContainers) | .metadata.name'

# Check agent logs for version banner
kubectl logs <pod-name> -n default -c <agent-container> | head -20
```

**Expected Output**:
```
====================================
PodScope Agent starting...
  Version: 0.1.0
  Built: 2026-01-16T05:29:52Z
  Commit: <git-hash>
====================================
```

---

## Test 3: Verify DNS Filtering

```bash
# Run automated test
./test-bpf-filter.sh
```

**Expected**:
- ✓ PASS: No DNS packets captured with filter
- Filter should show: `not port 53 and not (host X and (port 8080 or port 9090))`

---

## Test 4: Terminal Functionality

1. Open UI: http://localhost:8899
2. Click terminal icon on any pod
3. Terminal should open **without infinite growth**

**Test commands in terminal**:
```bash
# Basic shell
ls -la
ps aux

# Network diagnostics (Feature 3)
tcpdump -i eth0 -c 5           # Capture 5 packets
curl -I https://google.com      # HTTP test
dig google.com                  # DNS lookup
nc -zv google.com 443           # Port check
traceroute google.com           # Route tracing
wget -O- https://ifconfig.me    # Get external IP
```

**Expected**:
- Terminal opens smoothly
- All tools work (tcpdump, curl, dig, nc, traceroute, wget)
- Resize works when changing browser window size
- Close/reopen works

---

## Test 5: Pause Button

1. Open UI: http://localhost:8899
2. Generate some traffic in target pod
3. Click **Pause** button
4. Verify flow list stops updating
5. Click **Resume** button
6. Verify flows resume updating

**Expected**:
- Pause stops UI updates but keeps WebSocket connected
- Status shows "Live" throughout
- Resume continues from current state

---

## Test 6: Pod Name Detection

In the UI flow list, verify:
- Source pod names appear (not just IPs)
- Destination pod names appear when traffic is pod-to-pod
- External destinations show IPs/domains

**Check agent logs** for pod matching:
```bash
kubectl logs <pod> -c <agent> | grep "DEBUG: completeFlow"
```

---

## Troubleshooting

### If old version appears in logs
**Problem**: Session started before rebuild
**Solution**: Stop and restart capture session

### If DNS packets still visible
1. Verify version in logs: `kubectl logs <pod> -c <agent> | grep "Version:"`
2. Check BPF filter: `kubectl logs <pod> -c <agent> | grep "BPF FILTER"`
3. Run test: `./test-bpf-filter.sh`

### If terminal grows infinitely
**Problem**: Using old hub image
**Solution**: Restart capture session (hub gets recreated)

### If networking tools missing
**Problem**: Using old agent image
**Solution**: Restart capture session (agents get re-injected)

---

## Known Issues

1. **Ephemeral containers are immutable** - Once injected, agents can't be updated. Must restart pods or target different pods.

2. **Hub deployment** - May need manual restart if not recreated:
   ```bash
   kubectl rollout restart deployment podscope-hub -n <session-namespace>
   ```

---

## Success Criteria

- ✅ Version banner shows 0.1.0 with today's build date
- ✅ BPF filter includes `not port 53`
- ✅ No DNS packets captured (test-bpf-filter.sh passes)
- ✅ Terminal opens without infinite growth
- ✅ All networking tools available in terminal
- ✅ Pause button stops/resumes UI updates
- ✅ Pod names appear in flow list

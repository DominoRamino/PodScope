# Image Refresh Complete ✅

## What Was Done

### 1. **Removed ALL Old Images**
```
✓ ghcr.io/podscope/hub:latest (OLD - from previous session)
✓ ghcr.io/podscope/agent:latest (OLD - from previous session)
✓ docker.io/library/podscope:latest (OLD - pre-versioning)
✓ docker.io/library/podscope-agent:latest (OLD - pre-versioning)
```

### 2. **Loaded Fresh Versioned Images**
```
✓ docker.io/library/podscope-agent:latest
  - Version: 0.1.0
  - Build: 2026-01-16T04:47:16Z
  - Commit: 532839e
  - Features: DNS filtering, Pod name detection, Debug logging

✓ docker.io/library/podscope:latest
  - Version: 0.1.0
  - Build: 2026-01-16T04:47:16Z
  - Commit: 532839e
```

## What's New in These Images

### Agent Improvements
1. **DNS Filtering**: BPF filter now includes `not port 53`
2. **Version Logging**: Shows version, build date, commit on startup
3. **Enhanced Pod Detection**: IP matching + port-based fallback
4. **Debug Logging**: Detailed logs for troubleshooting
5. **BPF Verification**: Logs success/failure of filter application

### Hub Improvements
1. **Version Logging**: Shows version on startup
2. **Pause API**: Fixed to actually stop PCAP capture
3. **Terminal Support**: WebSocket proxy for kubectl exec

## How to Verify

### Start a New Session
```bash
./podscope tap -n default -l app=podinfo
```

### Check Agent Version
```bash
# Find pod with agent
kubectl get pods -n default -o json | \
  jq '.items[] | select(.spec.ephemeralContainers) | .metadata.name'

# Check logs (replace <pod-name> and <agent-container>)
kubectl logs <pod-name> -n default -c <agent-container> | head -20
```

### Expected Output
```
====================================
PodScope Agent starting...
  Version: 0.1.0
  Built: 2026-01-16T04:47:16Z
  Commit: 532839e
====================================
  Agent ID: xxxxxxxx
  Session: xxxxxxxx
  Pod: default/podinfo-xxxxx
  Pod IP: "10.244.0.X"
  Interface: eth0
  Hub: podscope-hub.podscope-xxxxx.svc.cluster.local:9090
====================================
APPLYING BPF FILTER:
  not port 53 and not (host 10.X.X.X and (port 8080 or port 9090))
====================================
SUCCESS: BPF filter applied to pcap handle
Starting packet capture...
```

### Test DNS Filtering
```bash
chmod +x test-bpf-filter.sh
./test-bpf-filter.sh
```

Should show:
- ✓ PASS: No DNS packets captured with filter

## What Changed

### Before (Old Images)
```
PodScope Agent starting...
  Agent ID: xxx
  Session: xxx
  Pod: default/mypod (10.244.0.5)   ← Old format
  BPF Filter: not (host X and ...)  ← No DNS filtering
```

### After (New Images)
```
====================================
PodScope Agent starting...
  Version: 0.1.0                     ← NEW: Version info
  Built: 2026-01-16T04:47:16Z        ← NEW: Build timestamp
  Commit: 532839e                    ← NEW: Git commit
====================================
  Pod IP: "10.244.0.5"               ← NEW: Format with quotes
====================================
APPLYING BPF FILTER:                 ← NEW: Banner
  not port 53 and not (...)          ← NEW: DNS filtering
====================================
SUCCESS: BPF filter applied         ← NEW: Verification
```

## Troubleshooting

### If you still see old format logs
**Problem**: The session was started before loading new images

**Solution**: Stop and restart capture session
```bash
# Press Ctrl+C to stop
./podscope tap -n default -l app=podinfo
```

### If DNS packets still appear
**Check 1**: Verify version in logs
```bash
kubectl logs <pod> -c <agent> | grep "Version:"
```

Should show `Version: 0.1.0`

**Check 2**: Run BPF test
```bash
./test-bpf-filter.sh
```

**Check 3**: Test manually
```bash
kubectl exec <pod> -c <agent> -- tcpdump -i eth0 "not port 53" -c 10
```

Should show NO port 53 traffic.

### If pod names don't appear
**Check**: Agent logs for DEBUG messages
```bash
kubectl logs <pod> -c <agent> | grep "DEBUG: completeFlow"
```

Should show pod matching logic.

## Files Updated

Configuration:
- `VERSION` - Set to 0.1.0
- `Makefile` - Added versioning system
- `docker/Dockerfile.agent` - Build args for version
- `docker/Dockerfile.hub` - Build args for version

Code:
- `cmd/agent/main.go` - Version logging
- `cmd/agent/main.go` - BPF filter with DNS
- `pkg/agent/assembler.go` - Enhanced pod detection with debug logs
- `pkg/agent/capture.go` - BPF filter verification
- `pkg/k8s/session.go` - Environment-based image selection

Scripts:
- `test-bpf-filter.sh` - Automated BPF testing
- `debug.sh` - Debugging helper
- `VERSIONING_GUIDE.md` - Complete versioning docs

## Current Status

✅ All old images removed from minikube
✅ Fresh v0.1.0 images loaded
✅ Images include version metadata
✅ DNS filtering enabled
✅ Pod name detection improved
✅ Debug logging added
✅ Test scripts ready

🎯 **Ready for testing!**

Start a new capture session and verify the improvements work as expected.

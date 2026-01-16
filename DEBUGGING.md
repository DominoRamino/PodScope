# PodScope Debugging Guide

## Quick Start

The images now have extensive debug logging. Follow these steps to diagnose issues:

### 1. Start a new capture session

```bash
# Stop any existing session (Ctrl+C)
# Start a new one
./podscope tap -n <namespace> --pod <pod-name>

# OR use label selector
./podscope tap -n <namespace> -l app=<label>
```

### 2. Run the automated debugging script

```bash
chmod +x debug.sh
./debug.sh
```

This script will:
- Find your podscope namespace and pods
- Show agent logs with critical debug information
- Show hub logs
- Guide you through what to look for

## What to Look For

### In Agent Logs

#### 1. POD_IP Value
Look for:
```
Pod IP: "10.244.0.X"
```

**Problem**: If you see `Pod IP: ""` (empty quotes)
- The POD_IP environment variable is not being set correctly
- Check `pkg/k8s/session.go:InjectAgent()` - line 306 sets this value

#### 2. BPF Filter Application
Look for:
```
====================================
APPLYING BPF FILTER:
  not port 53 and not (host X.X.X.X and (port 8080 or port 9090))
====================================
SUCCESS: BPF filter applied to pcap handle
```

**Problem**: If you see `WARNING: No BPF filter set!`
- The filter wasn't passed to the capturer
- Check `cmd/agent/main.go:79-86`

**Problem**: If you see `ERROR: Failed to set BPF filter:`
- The BPF syntax is invalid
- Try simplifying the filter to just `not port 53` to test

#### 3. DNS Packet Warnings
If DNS is being filtered correctly, you should NOT see:
```
WARNING: DNS packet captured despite BPF filter! UDP 53->XXXX
```

**Problem**: If you see these warnings
- BPF filter is not working or has wrong syntax
- The pcap library might not support the filter syntax
- Try testing with tcpdump manually: `tcpdump -i eth0 "not port 53"`

#### 4. Pod Name Matching
Look for DEBUG logs like:
```
DEBUG: completeFlow - agentPodName="my-pod" agentPodIP="10.244.0.5" flow.SrcIP=10.244.0.5 flow.DstIP=8.8.8.8 flow.SrcPort=45678 flow.DstPort=443
DEBUG: Matched SrcIP=10.244.0.5 to pod default/my-pod
DEBUG: Final flow - SrcPod="my-pod" DstPod=""
```

**Problem**: If `Final flow` shows empty `SrcPod=""` and `DstPod=""`
- Neither IP matching nor fallback logic is working
- Check if `agentPodName` is set (shown in first DEBUG line)
- Check if the IPs match correctly
- Fallback logic triggers if ports > 1024 (outgoing) or < 1024 (incoming)

### In Hub Logs

Look for flows being received:
```
Received flow: HTTP 10.244.0.5:45678 -> 8.8.8.8:443 [CLOSED]
```

Check if flows have pod information when they arrive at the hub.

## Manual Testing Steps

### Test 1: Check if ephemeral container has POD_IP

```bash
# Find your pod with agent
kubectl get pods -n <namespace> -o json | jq '.items[] | select(.spec.ephemeralContainers != null)'

# Get the agent container name
AGENT_CONTAINER=$(kubectl get pod <pod-name> -n <namespace> -o json | jq -r '.spec.ephemeralContainers[0].name')

# Check environment variables
kubectl exec <pod-name> -n <namespace> -c $AGENT_CONTAINER -- env | grep POD
```

Expected output:
```
POD_NAME=my-pod
POD_NAMESPACE=default
POD_IP=10.244.0.5
```

### Test 2: Verify BPF filter syntax

```bash
# Exec into the agent container
kubectl exec -it <pod-name> -n <namespace> -c $AGENT_CONTAINER -- sh

# Test the BPF filter with tcpdump
tcpdump -i eth0 "not port 53" -c 10

# You should see packets but NO DNS (port 53)
```

### Test 3: Check pod IP matches traffic

```bash
# Get the pod's IP
POD_IP=$(kubectl get pod <pod-name> -n <namespace> -o jsonpath='{.status.podIP}')
echo "Pod IP: $POD_IP"

# Check agent logs to see if captured traffic uses this IP
kubectl logs <pod-name> -n <namespace> -c $AGENT_CONTAINER | grep "completeFlow" | head -5
```

The `flow.SrcIP` or `flow.DstIP` in the DEBUG logs should match the pod's IP.

## Common Issues and Fixes

### Issue 1: Pod names not appearing

**Symptom**: Flow details show only IPs, no pod names

**Debug**:
1. Check agent logs for `DEBUG: Final flow` - are SrcPod/DstPod empty?
2. Check if `POD_IP` environment variable is set correctly
3. Verify the pod's actual IP matches what's in the environment variable

**Fix**: The fallback logic should work even without IP matching. If neither works:
```go
// In pkg/agent/assembler.go, the logic assumes:
// - High src port (>1024) = outgoing from our pod (set SrcPod)
// - Low src port (<1024) = incoming to our pod (set DstPod)
```

If your app listens on a high port (e.g., 8080), the heuristic breaks. We may need to:
- Always set SrcPod for all outgoing traffic (when pod is on eth0 interface)
- Or improve the IP matching to be more reliable

### Issue 2: DNS packets still appearing

**Symptom**: Seeing lots of port 53 traffic in UI

**Debug**:
1. Check agent logs for `WARNING: DNS packet captured despite BPF filter!`
2. Verify BPF filter was applied: look for `SUCCESS: BPF filter applied`
3. Test BPF filter syntax manually with tcpdump

**Fix Options**:
- The BPF filter syntax might be wrong
- Try simplifying: change `buildHubExclusionFilter` to return just `"not port 53"`
- Some network setups might require `not port 53 and not port 5353` (mDNS)

### Issue 3: Terminal icon not appearing

**Symptom**: No terminal icon next to pod names in UI

**Debug**:
1. This only appears when pod names are populated (see Issue 1)
2. Check browser console for any errors

**Fix**: Fix pod name population first, then terminal icon will appear automatically.

## Actual CLI Commands

The correct command is `tap`, not `capture`. Here are the actual flags:

```bash
# Capture from a specific pod
./podscope tap -n <namespace> --pod <pod-name>

# Capture using label selector
./podscope tap -n default -l app=myapp

# Capture from all namespaces
./podscope tap -A -l app=myapp

# Force privileged mode
./podscope tap -n default --pod mypod --force-privileged

# Custom UI port
./podscope tap -n default --pod mypod --ui-port 9999
```

## Build Optimization

Use the new Makefile for faster builds:

```bash
# Build both images in parallel (fastest)
make build

# Build and load into minikube
make all

# Build only one image
make build-agent
make build-hub

# Quick iteration (build only, skip load)
make quick

# Clean and rebuild from scratch
make rebuild

# Show all targets
make help
```

The Makefile uses Docker BuildKit caching, so subsequent builds are much faster (5-10s instead of 40s).

## Next Steps After Debugging

Once you've run the debug script and checked the logs:

1. **Share the relevant log sections** showing:
   - The BPF filter being applied
   - The POD_IP value
   - Any DEBUG messages about pod matching
   - Any WARNING messages about DNS packets

2. **Share the output of**: `kubectl describe pod <pod-name> -n <namespace>`

3. We can then identify the exact issue and fix it.

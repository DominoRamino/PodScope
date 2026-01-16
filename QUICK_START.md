# PodScope Quick Start - CORRECTED COMMANDS

## ✅ What's Been Fixed

1. ✅ Added DNS filtering (`not port 53` in BPF filter)
2. ✅ Improved pod name detection (IP matching + fallback logic)
3. ✅ Added comprehensive debug logging
4. ✅ Built the CLI binary: `./podscope`
5. ✅ Created Makefile for faster builds

## 🚀 How to Test the Fixes

### Step 1: Stop current session (if running)
Press `Ctrl+C` in the terminal running podscope

### Step 2: Choose a testing approach

**Option A: Use curl-client pod (fastest - no agent yet)**
```bash
./podscope tap -n default --pod curl-client
```

**Option B: Restart podinfo deployment**
```bash
kubectl rollout restart deployment podinfo -n default
# Wait 30 seconds for new pods
./podscope tap -n default -l app=podinfo
```

**Option C: Use label selector (captures all matching pods)**
```bash
./podscope tap -n default -l app=podinfo
```

### Step 3: Verify new image is running

After the capture starts, check the agent logs:

```bash
# Get pod and container info
kubectl get pods -n default -o json | \
  jq '.items[] | select(.spec.ephemeralContainers) | {name: .metadata.name, container: .spec.ephemeralContainers[].name}'

# View agent logs (replace with actual names)
kubectl logs <pod-name> -n default -c <agent-container> --tail=50
```

**Look for these signs of the NEW image:**
- ✅ `Pod IP: "10.244.0.X"` (with quotes)
- ✅ `APPLYING BPF FILTER:`
- ✅ `not port 53 and not (host ...` (DNS filtering)
- ✅ `SUCCESS: BPF filter applied to pcap handle`
- ✅ `DEBUG: completeFlow` messages

**You should NOT see:**
- ❌ `Pod: default/podname (10.244.0.X)` (old format)
- ❌ `WARNING: DNS packet captured despite BPF filter!`

### Step 4: Generate traffic and check UI

```bash
# In another terminal, generate some traffic
kubectl exec -n default curl-client -- curl http://podinfo:9898

# Open UI in browser
# Default: http://localhost:8899
```

**In the UI, check:**
1. Flow details should show pod names (not just IPs)
2. Terminal icon should appear next to pod names
3. No DNS (port 53) traffic should appear

## 📋 All Available Commands

### Build Commands
```bash
make build-cli   # Build the podscope CLI binary
make build       # Build agent and hub Docker images (parallel)
make load        # Load images into minikube
make all         # Build CLI, build images, and load (default)
make clean       # Remove images and binary
make rebuild     # Clean and rebuild everything
```

### Capture Commands
```bash
# Capture from specific pod
./podscope tap -n <namespace> --pod <pod-name>

# Capture using label selector
./podscope tap -n <namespace> -l app=<label>

# Capture from all namespaces
./podscope tap -A -l app=<label>

# Force privileged mode
./podscope tap -n <namespace> --pod <pod-name> --force-privileged

# Custom UI port
./podscope tap -n <namespace> --pod <pod-name> --ui-port 9999
```

### Debug Commands
```bash
# Find active podscope namespace
kubectl get namespaces | grep podscope

# Find pods with agents
kubectl get pods --all-namespaces -o json | \
  jq -r '.items[] | select(.spec.ephemeralContainers) | "\(.metadata.namespace)/\(.metadata.name)"'

# Get agent container name
kubectl get pod <pod-name> -n <namespace> -o json | \
  jq -r '.spec.ephemeralContainers[].name'

# View agent logs
kubectl logs <pod-name> -n <namespace> -c <agent-container>

# Check for DNS filtering
kubectl logs <pod-name> -n <namespace> -c <agent-container> | grep -E "port 53|DNS|BPF"

# Check for pod name matching
kubectl logs <pod-name> -n <namespace> -c <agent-container> | grep "DEBUG: completeFlow"
```

## ⚠️ Important Notes

### Ephemeral Containers Are Immutable
Once injected, ephemeral containers **cannot be updated**. They will continue running the old image even after rebuilding. To test new images, you **must** inject into a different pod or restart the target pods.

### Why Pod Names May Not Appear
1. **POD_IP not set**: Check if `POD_IP` environment variable is populated
2. **IP mismatch**: The agent's POD_IP doesn't match the traffic's source/dest IP
3. **Fallback logic**: Uses port numbers (>1024 = outgoing, <1024 = incoming) when IP doesn't match

### Why DNS Packets May Still Appear
1. **Old image**: Agent is running the old image without DNS filtering
2. **BPF filter failed**: Check logs for `ERROR: Failed to set BPF filter`
3. **Filter syntax**: The BPF syntax might not be supported by your kernel

## 🐛 Troubleshooting

If things still don't work after injecting into a new pod:

1. **Check the agent logs** - Look for the indicators above
2. **Verify POD_IP** - `kubectl exec <pod> -c <agent-container> -- env | grep POD`
3. **Test BPF filter manually** - `kubectl exec <pod> -c <agent-container> -- tcpdump -i eth0 "not port 53" -c 10`
4. **Share the logs** - Copy relevant sections showing BPF filter and DEBUG messages

## 📝 Summary

The fixes are ready and loaded into minikube. The current agents are using the old image because ephemeral containers can't be updated. Start a new capture session with a fresh pod to test the fixes!

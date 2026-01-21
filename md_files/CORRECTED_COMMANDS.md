# Corrected PodScope Commands

## ⚠️ Important: Actual CLI Syntax

The command is **`tap`**, not `capture`!

## Basic Commands

### Capture from a specific pod
```bash
./podscope tap -n default --pod podinfo-65cc875c69-426bz
```

### Capture using label selector
```bash
./podscope tap -n default -l app=podinfo
```

### Capture from all namespaces
```bash
./podscope tap -A -l app=myapp
```

### Force privileged mode
```bash
./podscope tap -n default --pod mypod --force-privileged
```

### Custom UI port
```bash
./podscope tap -n default --pod mypod --ui-port 9999
```

## Available Flags

```
-n, --namespace string     Target namespace (default "default")
-l, --selector string      Label selector to filter pods (e.g., app=frontend)
    --pod string           Specific pod name to target
-A, --all-namespaces       Target all namespaces
    --force-privileged     Force privileged mode for the capture agent
    --hub-port int         Port for the Hub gRPC server (default 8080)
    --ui-port int          Local port for the UI (default 8899)
```

## Testing the New Debug Images

Since ephemeral containers are immutable, you need to inject into a **new** pod to test the updated images.

### Option 1: Use curl-client pod (doesn't have an agent yet)
```bash
./podscope tap -n default --pod curl-client
```

### Option 2: Restart podinfo and re-inject
```bash
# Stop current capture session (Ctrl+C)

# Restart the deployment to get fresh pods
kubectl rollout restart deployment podinfo -n default

# Wait for new pods to be ready
kubectl get pods -n default -w

# Inject into the new pod
./podscope tap -n default -l app=podinfo
```

### Option 3: Use label selector to capture all podinfo pods
```bash
# This will inject into ALL pods matching the label
./podscope tap -n default -l app=podinfo
```

## Verify New Image is Running

After starting a new tap session, check the logs:

```bash
# Get pod info
kubectl get pods -n default -o json | jq '.items[] | select(.spec.ephemeralContainers) | {name: .metadata.name, container: .spec.ephemeralContainers[].name}'

# Check agent logs (replace with actual pod/container names)
kubectl logs <pod-name> -n default -c <agent-container> --tail=50

# Look for these indicators of the NEW image:
# ✓ "Pod IP: \"10.244.0.X\"" (with quotes)
# ✓ "APPLYING BPF FILTER:"
# ✓ "not port 53 and not (host ..."
# ✓ "SUCCESS: BPF filter applied to pcap handle"
# ✓ "DEBUG: completeFlow" messages
```

## Quick Check for DNS Filtering

```bash
# View logs and filter for DNS-related messages
kubectl logs <pod-name> -n default -c <agent-container> | grep -E "port 53|DNS|BPF"

# Should see:
# ✓ BPF filter with "not port 53"
# ✓ SUCCESS message for BPF filter
# ✗ Should NOT see "WARNING: DNS packet captured"
```

## Current Session Info

To find your active session:

```bash
# Find podscope namespace
kubectl get namespaces | grep podscope

# Find pods with agents
kubectl get pods --all-namespaces -o json | \
  jq -r '.items[] | select(.spec.ephemeralContainers) | "\(.metadata.namespace)/\(.metadata.name)"'

# Get agent container name for a pod
kubectl get pod <pod-name> -n <namespace> -o json | \
  jq -r '.spec.ephemeralContainers[].name'
```

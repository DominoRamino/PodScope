# PodScope Versioning & Testing Guide

## ✅ What's Been Implemented

### 1. **Version Tagging System**
Every build now gets:
- Version number from `VERSION` file (currently: 0.1.0)
- Timestamp tag (e.g., `v0.1.0-20260116-044716`)
- Git commit hash
- Build date

### 2. **Image Labels**
All images include OCI standard labels:
```bash
docker inspect podscope-agent:latest | jq '.[0].Config.Labels'
```

### 3. **Version Logging**
Agents now log version info on startup:
```
====================================
PodScope Agent starting...
  Version: 0.1.0
  Built: 2026-01-16T04:47:16Z
  Commit: 532839e
====================================
```

### 4. **Configurable Image Tags**
Use environment variables to specify exact image versions:
```bash
export PODSCOPE_AGENT_IMAGE=podscope-agent:v0.1.0-20260116-044716
export PODSCOPE_HUB_IMAGE=podscope:v0.1.0-20260116-044716
./podscope tap -n default --pod mypod
```

## 🚀 Building with Versions

### Check Current Version
```bash
make version
```

Output:
```
Version: 0.1.0
Tag: v0.1.0-20260116-044716
Commit: 532839e
Build Date: 2026-01-16T04:47:16Z
```

### Build Images
```bash
# Build both images with version tags
make build

# This creates TWO tags per image:
#   podscope-agent:latest
#   podscope-agent:v0.1.0-20260116-044716
#   podscope:latest
#   podscope:v0.1.0-20260116-044716
```

### Load into Minikube
```bash
make load
```

### Inspect Image Versions
```bash
make inspect
```

## 📝 Using Specific Versions

### Method 1: Environment Variables
```bash
export PODSCOPE_AGENT_IMAGE=podscope-agent:v0.1.0-20260116-044716
export PODSCOPE_HUB_IMAGE=podscope:v0.1.0-20260116-044716
./podscope tap -n default --pod mypod
```

### Method 2: Inline
```bash
PODSCOPE_AGENT_IMAGE=podscope-agent:v0.1.0-20260116-044716 \
./podscope tap -n default --pod mypod
```

### Method 3: Use :latest (default)
```bash
# Just use whatever :latest points to
./podscope tap -n default --pod mypod
```

## 🔍 Verifying Which Version is Running

### Check Agent Logs
```bash
# Find the agent container
kubectl get pods -n default -o json | \
  jq '.items[] | select(.spec.ephemeralContainers) | {name: .metadata.name, container: .spec.ephemeralContainers[].name}'

# Check logs for version
kubectl logs <pod-name> -n default -c <agent-container> --tail=20 | grep -A 3 "Version:"
```

You should see:
```
====================================
PodScope Agent starting...
  Version: 0.1.0
  Built: 2026-01-16T04:47:16Z
  Commit: 532839e
====================================
```

### Check Image Labels in Kubernetes
```bash
kubectl get pod <pod-name> -n default -o json | \
  jq '.spec.ephemeralContainers[] | select(.name | startswith("podscope-agent")) | .image'
```

## 🧪 Testing BPF Filter

### Automated Test Script
```bash
chmod +x test-bpf-filter.sh
./test-bpf-filter.sh
```

This script will:
1. Find a pod with an agent
2. Test capturing WITHOUT DNS filter (shows DNS traffic)
3. Test capturing WITH DNS filter (should show NO DNS)
4. Count DNS packets captured with filter
5. Show agent logs with filter configuration

### Manual BPF Test
```bash
# Find pod with agent
kubectl get pods -n default -o json | jq '.items[] | select(.spec.ephemeralContainers)'

# Test WITHOUT filter (will show DNS on port 53)
kubectl exec -n default <pod-name> -c <agent-container> -- \
  tcpdump -i eth0 -c 10

# Test WITH filter (should NOT show DNS on port 53)
kubectl exec -n default <pod-name> -c <agent-container> -- \
  tcpdump -i eth0 "not port 53" -c 10
```

### Expected BPF Filter in Logs
Look for this in agent logs:
```
====================================
APPLYING BPF FILTER:
  not port 53 and not (host 10.103.138.235 and (port 8080 or port 9090))
====================================
SUCCESS: BPF filter applied to pcap handle
```

## 🐛 Troubleshooting

### "DNS packets still appearing"

**Check 1: Verify agent version**
```bash
kubectl logs <pod> -c <agent> --tail=50 | head -10
```
Should show version 0.1.0 with today's date.

**Check 2: Verify BPF filter**
```bash
kubectl logs <pod> -c <agent> | grep "BPF FILTER"
```
Should show `not port 53` at the start.

**Check 3: Test filter manually**
```bash
./test-bpf-filter.sh
```

**Check 4: Verify image tag**
```bash
kubectl get pod <pod> -n default -o json | \
  jq '.spec.ephemeralContainers[].image'
```

### "Still using old version"

**Problem**: Ephemeral containers are immutable.

**Solution**: Start fresh capture session
```bash
# Stop current capture (Ctrl+C)

# Option A: Use different pod
./podscope tap -n default --pod curl-client

# Option B: Restart target deployment
kubectl rollout restart deployment podinfo -n default
# Wait for new pods
./podscope tap -n default -l app=podinfo
```

## 📊 Version Management Workflow

### Development Cycle
```bash
# 1. Make changes to code
vim pkg/agent/assembler.go

# 2. Build with new timestamp
make build

# 3. Load into minikube
make load

# 4. Start NEW capture session (old agents won't update!)
./podscope tap -n default --pod <different-pod>

# 5. Verify version in logs
kubectl logs <pod> -c <agent> | head -10

# 6. Test functionality
./test-bpf-filter.sh
```

### Production Releases
```bash
# Update version
echo "0.2.0" > VERSION

# Build
make build

# Tag in git
git tag v0.2.0
git push origin v0.2.0

# Images are tagged as:
#   podscope-agent:v0.2.0-YYYYMMDD-HHMMSS
#   podscope:v0.2.0-YYYYMMDD-HHMMSS
```

## 🎯 Quick Reference

```bash
# Show version info
make version

# Build with version tags
make build

# Load into minikube
make load

# Inspect image labels
make inspect

# Test BPF filter
./test-bpf-filter.sh

# Use specific version
PODSCOPE_AGENT_IMAGE=podscope-agent:v0.1.0-20260116-044716 \
./podscope tap -n default --pod mypod

# Verify running version
kubectl logs <pod> -c <agent> | grep -A 3 "Version:"
```

## 🔑 Key Points

1. **Every build gets a unique timestamp tag** - You can always track which version is running
2. **Version info is logged on startup** - Easy to verify which version an agent is using
3. **Images use OCI labels** - Standard metadata format
4. **Environment variables control image selection** - Easy to test specific versions
5. **Ephemeral containers are immutable** - Must start fresh capture to test new versions
6. **BPF filter test script** - Automated verification that DNS filtering works

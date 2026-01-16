#!/bin/bash

# PodScope Debugging Script
# This script helps diagnose why pod names aren't showing and DNS packets are appearing

set -e

echo "===================================="
echo "PodScope Debugging Guide"
echo "===================================="
echo ""

# Step 1: Find the podscope namespace
echo "Step 1: Finding podscope namespace..."
PODSCOPE_NS=$(kubectl get namespaces | grep podscope | awk '{print $1}' | head -1)

if [ -z "$PODSCOPE_NS" ]; then
    echo "ERROR: No podscope namespace found!"
    echo "Have you started a capture session with './podscope tap ...'?"
    exit 1
fi

echo "Found namespace: $PODSCOPE_NS"
echo ""

# Step 2: Find the hub pod
echo "Step 2: Finding hub pod..."
HUB_POD=$(kubectl get pods -n "$PODSCOPE_NS" -l app.kubernetes.io/name=podscope-hub -o jsonpath='{.items[0].metadata.name}')
echo "Hub pod: $HUB_POD"
echo ""

# Step 3: Find pods with ephemeral containers
echo "Step 3: Finding pods with podscope agents..."
echo "Enter the namespace where you injected the agent (or press Enter for 'default'):"
read TARGET_NS
TARGET_NS=${TARGET_NS:-default}

echo ""
echo "Searching for pods with ephemeral containers in namespace: $TARGET_NS"
AGENT_PODS=$(kubectl get pods -n "$TARGET_NS" -o json | jq -r '.items[] | select(.spec.ephemeralContainers != null and (.spec.ephemeralContainers | any(.name | startswith("podscope-agent")))) | .metadata.name')

if [ -z "$AGENT_PODS" ]; then
    echo "ERROR: No pods found with podscope agents in namespace $TARGET_NS"
    exit 1
fi

echo "Found pods with agents:"
echo "$AGENT_PODS"
echo ""

# Step 4: Select a pod to debug
POD_NAME=$(echo "$AGENT_PODS" | head -1)
echo "Debugging pod: $POD_NAME"
echo ""

# Step 5: Get the agent container name
AGENT_CONTAINER=$(kubectl get pod "$POD_NAME" -n "$TARGET_NS" -o json | jq -r '.spec.ephemeralContainers[] | select(.name | startswith("podscope-agent")) | .name')
echo "Agent container: $AGENT_CONTAINER"
echo ""

# Step 6: Check agent logs
echo "===================================="
echo "Step 6: Checking agent logs"
echo "===================================="
echo ""
echo "Looking for:"
echo "  1. POD_IP value (should not be empty)"
echo "  2. BPF filter (should include 'not port 53')"
echo "  3. DNS warning messages"
echo "  4. DEBUG messages about pod name matching"
echo ""
echo "Press Enter to view agent logs (Ctrl+C to exit log view)..."
read

kubectl logs "$POD_NAME" -n "$TARGET_NS" -c "$AGENT_CONTAINER" --tail=100

echo ""
echo "===================================="
echo "Step 7: Check hub logs"
echo "===================================="
echo ""
echo "Press Enter to view hub logs (Ctrl+C to exit log view)..."
read

kubectl logs "$HUB_POD" -n "$PODSCOPE_NS" --tail=50

echo ""
echo "===================================="
echo "Debugging Complete"
echo "===================================="
echo ""
echo "Key things to check in the logs:"
echo ""
echo "1. Agent logs should show:"
echo "   - 'Pod IP: \"<some-ip>\"' (not empty quotes)"
echo "   - 'APPLYING BPF FILTER:' followed by 'not port 53 and ...'"
echo "   - 'SUCCESS: BPF filter applied to pcap handle'"
echo "   - 'DEBUG: completeFlow' messages showing pod matching"
echo ""
echo "2. If you see 'WARNING: DNS packet captured despite BPF filter!'"
echo "   - The BPF filter is not working correctly"
echo "   - Check if the filter syntax is correct"
echo ""
echo "3. If POD_IP is empty (shows as \"\"):"
echo "   - The environment variable is not being set"
echo "   - Check pkg/k8s/session.go:InjectAgent function"
echo ""
echo "4. If DEBUG logs show 'Final flow - SrcPod=\"\" DstPod=\"\"':"
echo "   - Neither IP matching nor fallback logic is working"
echo "   - Check if agentPodName and agentPodIP are set correctly"
echo ""

#!/bin/bash

# Script to test BPF filter directly in an agent container

set -e

echo "======================================"
echo "BPF Filter Test Script"
echo "======================================"
echo ""

# Find a pod with an agent
echo "Finding pods with podscope agents..."
POD_INFO=$(kubectl get pods --all-namespaces -o json | \
  jq -r '.items[] | select(.spec.ephemeralContainers) |
  select(.spec.ephemeralContainers | any(.name | startswith("podscope-agent"))) |
  "\(.metadata.namespace) \(.metadata.name) \(.spec.ephemeralContainers[0].name)"' | head -1)

if [ -z "$POD_INFO" ]; then
    echo "ERROR: No pods found with podscope agents"
    echo "Start a capture session first: ./podscope tap -n default --pod <pod-name>"
    exit 1
fi

read NAMESPACE POD_NAME AGENT_CONTAINER <<< "$POD_INFO"

echo "Found agent in pod:"
echo "  Namespace: $NAMESPACE"
echo "  Pod: $POD_NAME"
echo "  Container: $AGENT_CONTAINER"
echo ""

echo "======================================"
echo "Test 1: Capture WITHOUT filter"
echo "======================================"
echo "This will show ALL traffic including DNS (port 53)"
echo ""
echo "Running: tcpdump -i eth0 -c 20"
echo ""
kubectl exec -n "$NAMESPACE" "$POD_NAME" -c "$AGENT_CONTAINER" -- timeout 5 tcpdump -i eth0 -c 20 2>&1 | grep -E "^\d|DNS|port 53"
echo ""

echo "======================================"
echo "Test 2: Capture WITH DNS filter"
echo "======================================"
echo "This should show NO DNS traffic (port 53)"
echo ""
echo "Running: tcpdump -i eth0 'not port 53' -c 20"
echo ""
kubectl exec -n "$NAMESPACE" "$POD_NAME" -c "$AGENT_CONTAINER" -- timeout 5 tcpdump -i eth0 "not port 53" -c 20 2>&1 | grep -E "^\d|DNS|port 53"
echo ""

echo "======================================"
echo "Test 3: Check if DNS packets are captured"
echo "======================================"
echo "Capturing for 5 seconds with DNS filter..."
echo ""

DNS_COUNT=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -c "$AGENT_CONTAINER" -- \
  timeout 5 tcpdump -i eth0 "not port 53" -nn 2>&1 | grep -c " 53:" || echo "0")

if [ "$DNS_COUNT" -eq 0 ]; then
    echo "✓ PASS: No DNS packets captured with filter"
else
    echo "✗ FAIL: $DNS_COUNT DNS packets captured despite filter!"
    echo ""
    echo "This means the BPF filter is not working correctly."
fi

echo ""
echo "======================================"
echo "Test 4: Check agent logs for filter"
echo "======================================"
kubectl logs -n "$NAMESPACE" "$POD_NAME" -c "$AGENT_CONTAINER" --tail=100 | grep -A 5 "BPF"

echo ""
echo "======================================"
echo "Summary"
echo "======================================"
echo ""
echo "If you see DNS packets in Test 2 or Test 3, the filter is not working."
echo ""
echo "Possible causes:"
echo "1. The agent is using an old image without DNS filtering"
echo "2. The BPF filter syntax is not supported by your kernel"
echo "3. The filter is not being applied to the pcap handle"
echo ""
echo "Check the agent logs above for:"
echo "  - 'APPLYING BPF FILTER:' message"
echo "  - 'not port 53 and not (host ...' in the filter"
echo "  - 'SUCCESS: BPF filter applied' message"
echo ""

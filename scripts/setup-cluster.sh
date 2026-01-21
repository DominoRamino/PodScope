#!/bin/bash
set -e

echo "=== PodScope Cluster Setup ==="

# 1. Check/start minikube
if ! minikube status 2>/dev/null | grep -q "Running"; then
    echo "Starting minikube..."
    minikube start --driver=docker --cpus=4 --memory=8192
else
    echo "✓ Minikube already running"
fi

# 2. Set context
kubectl config use-context minikube

# 3. Deploy podinfo if not present
if ! kubectl get deployment podinfo -n default &>/dev/null; then
    echo "Deploying podinfo test workload..."
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    kubectl apply -f "$SCRIPT_DIR/test-workloads/podinfo.yaml"
else
    echo "✓ Podinfo already deployed"
fi

# 4. Wait for ready
echo "Waiting for podinfo pods to be ready..."
kubectl wait --for=condition=available deployment/podinfo -n default --timeout=60s

# 5. Status
echo ""
echo "✓ Cluster ready"
echo ""
kubectl get pods -l app=podinfo

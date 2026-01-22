.PHONY: all build build-agent build-hub build-cli build-cli-linux load clean help version inspect use-tag setup-cluster dev dev-quick dev-ui load-agent load-hub restart-test-pods test test-ui

# Default target
all: build-cli build load

# Enable BuildKit for faster builds with better caching
export DOCKER_BUILDKIT=1

# Version and build info
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
IMAGE_TAG := $(GIT_COMMIT)

help:
	@echo "PodScope Build Targets:"
	@echo "  make build-cli   - Build the podscope CLI binary"
	@echo "  make build       - Build both agent and hub images (tagged with git commit)"
	@echo "  make build-agent - Build only agent image"
	@echo "  make build-hub   - Build only hub image"
	@echo "  make load        - Load images into minikube"
	@echo "  make all         - Build CLI and images, then load (default)"
	@echo "  make clean       - Remove built images and binary"
	@echo "  make rebuild     - Clean and rebuild everything"
	@echo ""
	@echo "Development Workflow:"
	@echo "  make dev              - Full dev loop: build, load, run (one command!)"
	@echo "  make dev-quick        - Smart rebuild: only rebuild changed components"
	@echo "  make dev-ui           - UI-only development with Vite hot-reload"
	@echo "  make setup-cluster    - Ensure minikube running with podinfo test workload"
	@echo "  make restart-test-pods - Restart podinfo pods to clear ephemeral containers"
	@echo ""
	@echo "Testing:"
	@echo "  make test             - Run Go backend tests"
	@echo "  make test-ui          - Run UI tests (single run)"
	@echo ""
	@echo "Version Management:"
	@echo "  make version     - Show current version info"
	@echo "  make inspect     - Inspect image labels"

# Build the CLI binary (Windows)
build-cli:
	@echo "Building podscope CLI..."
	@echo "  Version: $(VERSION)"
	@echo "  Commit: $(GIT_COMMIT)"
	@go build -ldflags "-X github.com/podscope/podscope/pkg/cli.Version=$(VERSION) -X github.com/podscope/podscope/pkg/k8s.DefaultImageTag=$(IMAGE_TAG)" -o podscope ./cmd/podscope
	@echo "✓ CLI built successfully: ./podscope"
	@echo ""
	@echo "Usage: ./podscope tap -n <namespace> --pod <pod-name>"
	@echo "Will use images: podscope-agent:$(IMAGE_TAG) and podscope:$(IMAGE_TAG)"

# Build the CLI binary for Linux (to run in )
build-cli-linux:
	@echo "Building podscope CLI (Linux)..."
	@echo "  Version: $(VERSION)"
	@echo "  Commit: $(GIT_COMMIT)"
	@GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/podscope/podscope/pkg/cli.Version=$(VERSION) -X github.com/podscope/podscope/pkg/k8s.DefaultImageTag=$(IMAGE_TAG)" -o podscope-linux ./cmd/podscope
	@echo "✓ CLI built successfully: ./podscope-linux"

# Build both images in parallel
build:
	@echo "Building both images with version tags..."
	@echo "  Version: $(VERSION)"
	@echo "  Tag: $(IMAGE_TAG)"
	@echo "  Commit: $(GIT_COMMIT)"
	@echo ""
	@$(MAKE) -j2 build-agent build-hub

# Build agent image
build-agent:
	@echo "Building podscope-agent:$(IMAGE_TAG)..."
	@docker build -f docker/Dockerfile.agent \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t podscope-agent:$(IMAGE_TAG) \
		.
	@echo "✓ Agent image built: podscope-agent:$(IMAGE_TAG)"

# Build hub image
build-hub:
	@echo "Building podscope:$(IMAGE_TAG)..."
	@docker build -f docker/Dockerfile.hub \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t podscope:$(IMAGE_TAG) \
		.
	@echo "✓ Hub image built: podscope:$(IMAGE_TAG)"

# Load images into minikube (via )
load:
	@echo "Loading images into minikube..."
	@minikube image load podscope-agent:$(IMAGE_TAG)
	@minikube image load podscope:$(IMAGE_TAG)
	@echo "✓ Images loaded into minikube:"
	@echo "    podscope-agent:$(IMAGE_TAG)"
	@echo "    podscope:$(IMAGE_TAG)"

# Show current version info
version:
	@echo "Image Tag: $(IMAGE_TAG)"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Inspect image labels
inspect:
	@echo "Agent image labels ($(IMAGE_TAG)):"
	@docker inspect podscope-agent:$(IMAGE_TAG) | jq '.[0].Config.Labels' || true
	@echo ""
	@echo "Hub image labels ($(IMAGE_TAG)):"
	@docker inspect podscope:$(IMAGE_TAG) | jq '.[0].Config.Labels' || true

# Clean up images and binary
clean:
	@echo "Removing images and binary..."
	-docker rmi podscope-agent:$(IMAGE_TAG) 2>/dev/null || true
	-docker rmi podscope:$(IMAGE_TAG) 2>/dev/null || true
	-rm -f podscope 2>/dev/null || true
	@echo "✓ Build $(IMAGE_TAG) images and binary removed"

# Rebuild from scratch
rebuild: clean all

# Quick rebuild (for iterative development)
quick: build
	@echo "✓ Quick build complete. Run 'make load' to load into minikube."

# =============================================================================
# Development Workflow Targets
# =============================================================================

# Ensure cluster is ready with test workloads
setup-cluster:
	@bash ./scripts/setup-cluster.sh

# Full development loop: build everything, load, run
dev:
	@$(MAKE) setup-cluster build-cli-linux build load restart-test-pods
	@echo ""
	@echo "Starting PodScope session..."
	@./podscope-linux tap -n default -l app.kubernetes.io/name=podinfo --ui-port 8899

# Smart rebuild: only rebuild changed components
dev-quick: setup-cluster build-cli-linux
	@echo "Checking for changes..."
	@if ! git diff --quiet HEAD -- cmd/agent pkg/agent; then \
		echo "Agent changed, rebuilding..."; \
		$(MAKE) build-agent load-agent; \
	else \
		echo "Agent unchanged, skipping."; \
	fi
	@if ! git diff --quiet HEAD -- cmd/hub pkg/hub ui; then \
		echo "Hub changed, rebuilding..."; \
		$(MAKE) build-hub load-hub; \
	else \
		echo "Hub unchanged, skipping."; \
	fi
	@$(MAKE) restart-test-pods
	@echo ""
	@echo "Starting PodScope session..."
	@./podscope-linux tap -n default -l app.kubernetes.io/name=podinfo --ui-port 8899

# UI-only development (Vite hot-reload)
dev-ui:
	@cd ui && npm run dev

# =============================================================================
# Testing Targets
# =============================================================================

# Run Go backend tests
test:
	@echo "Running Go tests..."
	@go test -v -race ./pkg/...

# Run UI tests
test-ui:
	@echo "Running UI tests..."
	@cd ui && npm test -- --run

# Individual image loading (via )
load-agent:
	@minikube image load podscope-agent:$(IMAGE_TAG)

load-hub:
	@minikube image load podscope:$(IMAGE_TAG)

# Clean up ephemeral containers by restarting test pods (via )
restart-test-pods:
	@echo "Restarting podinfo pods to clear ephemeral containers..."
	@kubectl rollout restart deploy podinfo -n default 
	@kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=podinfo -n default --timeout=60s
	@echo "✓ Test pods restarted"

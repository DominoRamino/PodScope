.PHONY: all build build-agent build-hub build-cli load clean help version inspect use-tag

# Default target
all: build-cli build load

# Enable BuildKit for faster builds with better caching
export DOCKER_BUILDKIT=1

# Version and build info
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
IMAGE_TAG := v$(VERSION)-$(shell date -u +"%Y%m%d-%H%M%S")

help:
	@echo "PodScope Build Targets:"
	@echo "  make build-cli   - Build the podscope CLI binary"
	@echo "  make build       - Build both agent and hub images (parallel, with version tags)"
	@echo "  make build-agent - Build only agent image"
	@echo "  make build-hub   - Build only hub image"
	@echo "  make load        - Load images into minikube"
	@echo "  make all         - Build CLI and images, then load (default)"
	@echo "  make clean       - Remove built images and binary"
	@echo "  make rebuild     - Clean and rebuild everything"
	@echo ""
	@echo "Version Management:"
	@echo "  make version     - Show current version info"
	@echo "  make inspect     - Inspect image labels"
	@echo "  make use-tag     - Show how to use specific image tags"

# Build the CLI binary
build-cli:
	@echo "Building podscope CLI..."
	@echo "  Version: $(VERSION)"
	@go build -ldflags "-X github.com/podscope/podscope/pkg/cli.Version=$(VERSION)" -o podscope ./cmd/podscope
	@echo "✓ CLI built successfully: ./podscope"
	@echo ""
	@echo "Usage: ./podscope tap -n <namespace> --pod <pod-name>"

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
	@echo "Building podscope-agent..."
	@docker build -f docker/Dockerfile.agent \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t podscope-agent:latest \
		-t podscope-agent:$(IMAGE_TAG) \
		.
	@echo "✓ Agent image built:"
	@echo "    podscope-agent:latest"
	@echo "    podscope-agent:$(IMAGE_TAG)"

# Build hub image
build-hub:
	@echo "Building podscope..."
	@docker build -f docker/Dockerfile.hub \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t podscope:latest \
		-t podscope:$(IMAGE_TAG) \
		.
	@echo "✓ Hub image built:"
	@echo "    podscope:latest"
	@echo "    podscope:$(IMAGE_TAG)"

# Load images into minikube
load:
	@echo "Loading images into minikube..."
	@minikube image load podscope-agent:latest
	@minikube image load podscope:latest
	@echo "✓ Images loaded into minikube"

# Use a specific tagged version
use-tag:
	@echo "To use a specific image tag, set these env vars:"
	@echo "  export PODSCOPE_AGENT_IMAGE=podscope-agent:$(IMAGE_TAG)"
	@echo "  export PODSCOPE_HUB_IMAGE=podscope:$(IMAGE_TAG)"
	@echo ""
	@echo "Then run: ./podscope tap ..."
	@echo ""
	@echo "Or set them inline:"
	@echo "  PODSCOPE_AGENT_IMAGE=podscope-agent:$(IMAGE_TAG) ./podscope tap ..."

# Show current version info
version:
	@echo "Version: $(VERSION)"
	@echo "Tag: $(IMAGE_TAG)"
	@echo "Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Inspect image labels
inspect:
	@echo "Agent image labels:"
	@docker inspect podscope-agent:latest | jq '.[0].Config.Labels' || true
	@echo ""
	@echo "Hub image labels:"
	@docker inspect podscope:latest | jq '.[0].Config.Labels' || true

# Clean up images and binary
clean:
	@echo "Removing images and binary..."
	-docker rmi podscope-agent:latest 2>/dev/null || true
	-docker rmi podscope:latest 2>/dev/null || true
	-rm -f podscope 2>/dev/null || true
	@echo "✓ Images and binary removed"

# Rebuild from scratch
rebuild: clean all

# Quick rebuild (for iterative development)
quick: build
	@echo "✓ Quick build complete. Run 'make load' to load into minikube."

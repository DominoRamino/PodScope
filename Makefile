.PHONY: all build build-agent build-hub build-cli load clean help version inspect use-tag

# Default target
all: build-cli build load

# Enable BuildKit for faster builds with better caching
export DOCKER_BUILDKIT=1

# Version and build info
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_NUMBER := $(shell cat BUILD_NUMBER 2>/dev/null || echo "1")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
IMAGE_TAG := v$(BUILD_NUMBER)

help:
	@echo "PodScope Build Targets:"
	@echo "  make build-cli   - Build the podscope CLI binary"
	@echo "  make build       - Build both agent and hub images (with version tag v$(BUILD_NUMBER))"
	@echo "  make build-agent - Build only agent image"
	@echo "  make build-hub   - Build only hub image"
	@echo "  make load        - Load images into minikube"
	@echo "  make all         - Build CLI and images, then load (default)"
	@echo "  make clean       - Remove built images and binary"
	@echo "  make rebuild     - Clean and rebuild everything"
	@echo ""
	@echo "Version Management:"
	@echo "  make version     - Show current build number and version info"
	@echo "  make increment   - Increment build number (do this before rebuilding)"
	@echo "  make inspect     - Inspect image labels"

# Build the CLI binary
build-cli:
	@echo "Building podscope CLI..."
	@echo "  Version: $(VERSION)"
	@echo "  Build: $(BUILD_NUMBER)"
	@echo "  Default Image Tag: $(IMAGE_TAG)"
	@go build -ldflags "-X github.com/podscope/podscope/pkg/cli.Version=$(VERSION) -X github.com/podscope/podscope/pkg/k8s.DefaultImageTag=$(IMAGE_TAG)" -o podscope ./cmd/podscope
	@echo "✓ CLI built successfully: ./podscope"
	@echo ""
	@echo "Usage: ./podscope tap -n <namespace> --pod <pod-name>"
	@echo "Will use images: podscope-agent:$(IMAGE_TAG) and podscope:$(IMAGE_TAG)"

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

# Load images into minikube
load:
	@echo "Loading images into minikube..."
	@minikube image load podscope-agent:$(IMAGE_TAG)
	@minikube image load podscope:$(IMAGE_TAG)
	@echo "✓ Images loaded into minikube:"
	@echo "    podscope-agent:$(IMAGE_TAG)"
	@echo "    podscope:$(IMAGE_TAG)"

# Increment build number
increment:
	@echo "Current build: $(BUILD_NUMBER)"
	@echo $$(($(BUILD_NUMBER) + 1)) > BUILD_NUMBER
	@echo "New build number: $$(cat BUILD_NUMBER)"
	@echo ""
	@echo "Next build will be tagged as: v$$(cat BUILD_NUMBER)"

# Show current version info
version:
	@echo "Build Number: $(BUILD_NUMBER)"
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

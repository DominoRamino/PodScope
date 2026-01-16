# PodScope Makefile
.PHONY: all build build-cli build-hub build-agent build-ui clean test docker-build docker-push

# Version
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/podscope/podscope/pkg/cli.Version=$(VERSION)"

# Docker
REGISTRY ?= ghcr.io/podscope
AGENT_IMAGE := $(REGISTRY)/agent:$(VERSION)
HUB_IMAGE := $(REGISTRY)/hub:$(VERSION)

# Go settings
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

all: build

# Build all Go binaries
build: build-cli build-hub build-agent

build-cli:
	@echo "Building CLI..."
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/podscope ./cmd/podscope

build-hub:
	@echo "Building Hub..."
	CGO_ENABLED=0 go build -o bin/hub ./cmd/hub

build-agent:
	@echo "Building Agent..."
	CGO_ENABLED=1 go build -o bin/agent ./cmd/agent

# Build agent with static linking for containers
build-agent-static:
	@echo "Building Agent (static)..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
		-ldflags '-extldflags "-static"' \
		-tags netgo \
		-o bin/agent-linux-amd64 ./cmd/agent

# Build UI
build-ui:
	@echo "Building UI..."
	cd ui && npm install && npm run build

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf ui/dist/
	rm -rf ui/node_modules/

# Docker builds
docker-build: docker-build-hub docker-build-agent

docker-build-hub: build-hub build-ui
	@echo "Building Hub Docker image..."
	docker build -t $(HUB_IMAGE) -f docker/Dockerfile.hub .

docker-build-agent: build-agent-static
	@echo "Building Agent Docker image..."
	docker build -t $(AGENT_IMAGE) -f docker/Dockerfile.agent .

docker-push:
	docker push $(HUB_IMAGE)
	docker push $(AGENT_IMAGE)

# Development helpers
dev-hub:
	@echo "Starting Hub in development mode..."
	go run ./cmd/hub

dev-ui:
	@echo "Starting UI in development mode..."
	cd ui && npm run dev

# Generate protobuf (requires protoc)
proto:
	protoc --go_out=. --go-grpc_out=. api/proto/podscope.proto

# Install CLI locally
install: build-cli
	cp bin/podscope /usr/local/bin/

# Cross-compile CLI for all platforms
release:
	@echo "Building releases..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/podscope-linux-amd64 ./cmd/podscope
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/podscope-linux-arm64 ./cmd/podscope
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/podscope-darwin-amd64 ./cmd/podscope
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/podscope-darwin-arm64 ./cmd/podscope
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/podscope-windows-amd64.exe ./cmd/podscope

help:
	@echo "PodScope Build Targets:"
	@echo "  build        - Build all Go binaries"
	@echo "  build-cli    - Build CLI only"
	@echo "  build-hub    - Build Hub only"
	@echo "  build-agent  - Build Agent only"
	@echo "  build-ui     - Build UI"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker images"
	@echo "  docker-push  - Push Docker images"
	@echo "  dev-hub      - Run Hub in dev mode"
	@echo "  dev-ui       - Run UI in dev mode"
	@echo "  install      - Install CLI to /usr/local/bin"
	@echo "  release      - Build CLI for all platforms"

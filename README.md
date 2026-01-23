# PodScope

A lightweight, ephemeral network traffic capture and analysis tool for Kubernetes. PodScope attaches to pods using ephemeral containers to capture and analyze network traffic without modifying your deployments.


https://github.com/user-attachments/assets/d88ba17b-ae1d-4d20-9349-0989ac500ce8

## How It Works

1. **Session Creation**: The CLI creates a dedicated namespace (`podscope-<id>`) and deploys the Hub.

2. **Agent Injection**: For each target pod, an ephemeral container running the capture agent is injected using `kubectl debug` equivalent API.

3. **Packet Capture**: The agent uses `gopacket` with `AF_PACKET` to capture all traffic on the pod's network interface.

4. **Protocol Analysis**: TCP streams are reassembled, and HTTP/TLS protocols are parsed to extract metadata.

5. **Data Streaming**: Flow events and raw PCAP data are streamed to the Hub via gRPC.

6. **Visualization**: The Hub serves a React UI that connects via WebSocket for real-time updates.

7. **Cleanup**: On exit, the CLI deletes the session namespace, removing all resources.

## Features

- **Zero-intrusion packet capture** via Kubernetes ephemeral containers
- **HTTP/1.1 plaintext traffic analysis** - Full visibility into requests/responses
- **TLS handshake metadata extraction** - SNI, cipher suites, timing
- **Real-time traffic visualization** - Live updating web UI
- **PCAP export** - Download captures for Wireshark analysis
- **Session-based** - All resources cleaned up on exit

## Quick Start

### Prerequisites

- Kubernetes 1.25+ (ephemeral containers support)
- `NET_RAW` capability permitted in your cluster
- Go 1.22+ (for building)
- Node.js 20+ (for UI development)

### Installation

```bash
# Clone the repository
git clone https://github.com/podscope/podscope.git
cd podscope

# Build CLI and Docker images (agent + hub), then load into minikube
make all

# Or build components individually:
make build-cli    # CLI binary only
make build        # Docker images only (agent + hub)
make load         # Load images into minikube
```

### Usage

```bash
# Capture traffic from pods with a label selector
podscope tap -n default -l app=frontend

# Capture traffic from a specific pod
podscope tap -n default --pod my-pod-abc123

# Capture from all namespaces
podscope tap -A -l app=api

# Force privileged mode (if NET_RAW is blocked)
podscope tap -n default -l app=frontend --force-privileged
```

Once running, open `http://localhost:8899` in your browser to view the traffic.

Press `Ctrl+C` to stop and clean up all resources.

## Development

### Building

```bash
# Build everything (CLI + images) and load into minikube
make all

# Build just the CLI
make build-cli

# Build Docker images (agent + hub in parallel)
make build

# Load images into minikube
make load
```

### Development Workflow

```bash
# Full dev loop: build, load, restart test pods, run capture
make dev

# Smart rebuild: only rebuilds changed components
make dev-quick

# UI-only development with Vite hot-reload
make dev-ui
```

### Running Tests

#### Go Backend Tests

```bash
# Run all Go tests
make test

# Run tests with verbose output
go test -v ./pkg/...

# Run specific package tests
go test -v ./pkg/hub/...      # Hub server tests
go test -v ./pkg/agent/...    # Agent/capture tests
go test -v ./pkg/k8s/...      # Kubernetes client tests

# Run specific test by name
go test -v ./pkg/hub/... -run TestHandleFlows
go test -v ./pkg/agent/... -run TestFlowKey
go test -v ./pkg/k8s/... -run TestInjectAgent

# Run with race detection
go test -race ./pkg/...
```

#### React UI Tests

```bash
# Navigate to UI directory
cd ui

# Run tests in watch mode
npm test

# Run tests once (CI mode)
npm test -- --run

# Run specific test file
npm test -- --run App
npm test -- --run FlowList
npm test -- --run Header

# Run with coverage report
npm run test:coverage
```

#### Test Coverage

| Package | Description |
|---------|-------------|
| `pkg/hub/flowbuffer_test.go` | Ring buffer for flow storage |
| `pkg/hub/pcap_test.go` | PCAP encoding and file operations |
| `pkg/hub/server_test.go` | HTTP API endpoints |
| `pkg/agent/assembler_test.go` | TCP reassembly, protocol detection |
| `pkg/agent/capture_test.go` | PCAP packet encoding |
| `pkg/agent/client_test.go` | Hub client connection |
| `pkg/k8s/session_test.go` | Session lifecycle, agent injection |
| `pkg/k8s/client_test.go` | Pod lookup, cleanup |
| `ui/src/__tests__/` | React component tests |

### Project Structure

```
.
├── cmd/
│   ├── podscope/     # CLI entry point
│   ├── hub/          # Hub server entry point
│   └── agent/        # Capture agent entry point
├── pkg/
│   ├── cli/          # CLI commands (cobra)
│   ├── hub/          # Hub server implementation
│   ├── agent/        # Packet capture and analysis
│   ├── k8s/          # Kubernetes client utilities
│   └── protocol/     # Shared data types
├── api/
│   └── proto/        # gRPC protobuf definitions
├── ui/               # React frontend
│   └── src/
│       ├── components/
│       └── types.ts
├── docker/           # Dockerfiles
└── Makefile
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI (podscope)                       │
│  - Session Controller                                        │
│  - Creates namespace, deploys Hub                            │
│  - Injects Agents via kubectl debug                          │
│  - Port-forwards to Hub UI                                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Session Namespace                         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    Hub (Deployment)                  │    │
│  │  - Receives flows from Agents (gRPC)                │    │
│  │  - Stores PCAP data                                 │    │
│  │  - Serves WebSocket API for UI                      │    │
│  │  - Serves React UI                                  │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ gRPC
┌─────────────────────────────┼───────────────────────────────┐
│            Target Pod       │                               │
│  ┌────────────────────┐    │    ┌─────────────────────┐    │
│  │   App Container    │◄───┼───►│  Capture Agent      │    │
│  │                    │ network │  (Ephemeral)        │    │
│  │                    │   ns    │  - gopacket capture │    │
│  └────────────────────┘         │  - TCP reassembly   │    │
│                                 │  - HTTP/TLS parsing │    │
│                                 └─────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Security Considerations

- **Capabilities**: The agent requires `NET_RAW` capability for packet capture
- **Network Isolation**: Hub is ClusterIP only, accessed via port-forward
- **Data Lifecycle**: All data stored in emptyDir, deleted with namespace

## Limitations (MVP)

- **No HTTPS Decryption**: Encrypted payloads are not decrypted
- **HTTP/1.1 Only**: HTTP/2 and gRPC treated as raw TCP
- **Ephemeral Container Persistence**: Cannot remove agents until pod restart

## License

MIT License - see LICENSE file for details.

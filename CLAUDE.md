# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PodScope Shark is a Kubernetes network traffic capture and analysis tool. It uses ephemeral containers to inject capture agents into target pods, streaming traffic metadata to a Hub server for real-time visualization via a React UI.

**Key Technology Stack:**
- **Backend**: Go 1.24+, gopacket for packet capture, Kubernetes client-go
- **Frontend**: React 18, TypeScript, Vite, TailwindCSS
- **Protocols**: gRPC for agent-hub communication (HTTP fallback), WebSocket for UI real-time updates
- **Container**: Docker multi-stage builds for Hub and Agent images

## Quick Start: Development Workflow

**IMPORTANT**: This project uses minikube running in WSL. Use these one-command workflows for testing:

### One-Command Development (Recommended)

```bash
# Full development loop - builds everything, loads images, restarts test pods, runs capture
make dev
```

This single command:
1. Increments build version
2. Ensures minikube is running with podinfo test workload
3. Builds Linux CLI binary (for WSL compatibility)
4. Builds Hub and Agent Docker images
5. Loads images into minikube
6. Restarts podinfo pods (clears old ephemeral containers)
7. Starts PodScope capture session
8. Opens UI (port will be shown in output)

### Quick Iteration

```bash
# Smart rebuild - only rebuilds changed components
make dev-quick
```

Uses git diff to detect changes:
- Only rebuilds Agent image if `cmd/agent` or `pkg/agent` changed
- Only rebuilds Hub image if `cmd/hub`, `pkg/hub`, or `ui` changed
- Always rebuilds CLI (fast)
- Always restarts test pods before capture

### UI-Only Development

```bash
# Terminal 1: Run capture session
make dev

# Terminal 2: Vite hot-reload for UI changes
make dev-ui
```

### Other Useful Commands

```bash
make restart-test-pods    # Clear ephemeral containers from previous sessions
make setup-cluster        # Ensure minikube + podinfo are ready
make help                 # Show all available targets
```

### Environment Notes

- **Minikube runs in WSL** - all kubectl/minikube commands use `wsl` prefix
- **CLI is cross-compiled for Linux** - `podscope-linux` binary runs in WSL
- **Test workload**: podinfo deployment with label `app.kubernetes.io/name=podinfo`

---

## Build and Development Commands

### Go Backend

```bash
# Build all components
make build                    # Builds CLI, Hub, and Agent

# Build individual components
make build-cli                # CLI tool (bin/podscope)
make build-hub                # Hub server (bin/hub)
make build-agent              # Agent with CGO for gopacket
make build-agent-static       # Static binary for Docker (Linux amd64)

# Run tests
make test                     # go test -v ./...

# Development mode
make dev-hub                  # Run Hub locally: go run ./cmd/hub
```

### React UI

```bash
# Build UI for production
make build-ui                 # cd ui && npm install && npm run build

# Development mode
make dev-ui                   # cd ui && npm run dev (Vite dev server)

# Working directly in ui/ directory
cd ui
npm install                   # Install dependencies
npm run dev                   # Start dev server on http://localhost:5173
npm run build                 # Build production bundle to ui/dist/
npm run lint                  # ESLint with TypeScript
```

### Docker Images

```bash
make docker-build             # Build both Hub and Agent images
make docker-build-hub         # Build Hub image only
make docker-build-agent       # Build Agent image only
make docker-push              # Push to registry (REGISTRY env var)
```

### Installation and Release

```bash
make install                  # Copy bin/podscope to /usr/local/bin/
make release                  # Cross-compile for all platforms (Linux, Darwin, Windows)
```

## High-Level Architecture

### Three-Tier Design

```
CLI (podscope tap) → Hub (Deployment) ← Agent (Ephemeral Container)
                      ↓
                   UI (React)
```

### Data Flow

1. **CLI Orchestration** (`pkg/cli/tap.go`, `pkg/k8s/session.go`)
   - Generates unique session ID (8-char UUID)
   - Creates ephemeral namespace: `podscope-<session-id>`
   - Deploys Hub as Deployment with ClusterIP Service
   - Injects Agent into target pods using `UpdateEphemeralContainers` API
   - Establishes port-forward (SPDY) from localhost to Hub:8080
   - Cleanup: Single namespace deletion cascades to all resources

2. **Agent Capture** (`pkg/agent/capture.go`, `pkg/agent/assembler.go`)
   - Uses `gopacket` with `AF_PACKET` to capture all traffic on eth0
   - BPF filter excludes DNS (port 53) and Hub traffic (ports 8080/9090)
   - TCP stream reassembly with bidirectional flow tracking
   - Protocol detection: HTTP (plaintext parsing), TLS (ClientHello SNI), HTTPS (port heuristic)
   - Sends flow metadata (JSON) and PCAP chunks (binary) to Hub via HTTP POST

3. **Hub Aggregation** (`pkg/hub/server.go`, `pkg/hub/pcap.go`)
   - Receives flows via `POST /api/flows` → stores in-memory slice
   - Receives PCAP via `POST /api/pcap/upload` → writes per-agent files to `/data/pcap/agent-<id>.pcap`
   - Broadcasts new flows to all WebSocket clients (`GET /api/flows/ws`)
   - Serves React UI as static files from `/app/ui`
   - Merges per-agent PCAP files for download (skips duplicate headers)

4. **UI Real-Time Updates** (`ui/src/App.tsx`)
   - WebSocket connection receives all existing flows on connect (catch-up)
   - New flows broadcast as they arrive
   - Pause mechanism: stops PCAP storage, flows still processed
   - PCAP download triggers full session merge

### Key Implementation Patterns

#### TCP Stream Reassembly (`pkg/agent/assembler.go`)

**Flow Key Normalization:**
- Bidirectional flows use same key: smaller IP/port pair always comes first
- Format: `192.168.1.10:45678-10.0.0.5:80`

**State Machine:**
- Tracks TCP flags: SYN, SYN-ACK, FIN, RST
- Calculates handshake timing: `SYNACKTime - SYNTime`
- Flow completion triggers:
  - FIN+ACK received
  - RST packet (immediate)
  - 30-second inactivity timeout (cleanup goroutine)

**Protocol Detection:**
- **TLS**: First byte `0x16` (Handshake) + version `0x03xx`
- **HTTP**: Payload starts with `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, or `HTTP/`
- **HTTPS**: Port-based heuristic (443, 8443)

**HTTP Parsing:**
- Uses Go's `net/http` package: `http.ReadRequest` and `http.ReadResponse`
- Parses on first complete buffered payload
- Captures 1KB of body maximum

**TLS Parsing:**
- Manual ClientHello parsing (no TLS library, no decryption)
- Extracts: TLS version, SNI (Server Name extension type 0)
- Complex byte slicing: TLS record → Handshake → Session ID → Cipher Suites → Compression → Extensions

**Pod Name Attribution:**
- Agent knows its pod IP from environment variables
- Matches flow source/destination IPs against pod IP
- Heuristic: high port (>1024) = ephemeral = likely outgoing = source is pod

#### PCAP Format (`pkg/agent/capture.go:178-221`)

**Libpcap Format:**
- Global header: magic `0xa1b2c3d4`, version, snaplen, link type
- Per-packet header: timestamp (sec + µsec), capture length, original length
- Little-endian encoding

**Streaming Strategy:**
- 500ms flush interval to Hub
- Buffered in-memory before flush
- Sent via HTTP POST with `X-Agent-ID` header

#### WebSocket Broadcasting (`pkg/hub/server.go:265-302`)

**Connection Management:**
- Map of active connections: `map[*websocket.Conn]bool`
- On connect: send all existing flows (full catch-up)
- On new flow: broadcast to all clients
- Dead connection cleanup on write error

**Message Format:**
- JSON serialization of `protocol.Flow` structs
- TypeScript types in `ui/src/types.ts` mirror Go structs exactly

#### Ephemeral Container Injection (`pkg/k8s/session.go:256-335`)

**API Call:**
- Uses `CoreV1().Pods().UpdateEphemeralContainers()` (not `kubectl debug` directly)
- Container name: `podscope-agent-<short-id>`
- Image: from environment or default registry
- Shares target pod's network namespace automatically

**Capabilities:**
- `NET_RAW` for packet capture (default)
- Fallback: `Privileged: true` if `--force-privileged` flag set

**Environment Variables:**
- `HUB_ADDR`: `<service>.<namespace>.svc.cluster.local:9090`
- `POD_NAME`, `POD_NAMESPACE`, `POD_IP`: Kubernetes downward API

**Limitations:**
- Ephemeral containers cannot be removed until pod restart (Kubernetes limitation)
- Agent persists in pod spec even after session cleanup

#### BPF Filter Strategy (`cmd/agent/main.go:116-147`)

**Filter Logic:**
- Excludes DNS (port 53) to reduce noise
- Excludes traffic to/from Hub IP on ports 8080/9090 (prevents feedback loop)
- Resolves Hub hostname to IP at startup for accurate filtering
- Fallback if resolution fails: `not port 53 and not (port 8080 or port 9090)`

**Why IP Resolution:**
- BPF filters use IP addresses, not hostnames
- `net.LookupHost()` on `HUB_ADDR` environment variable
- Prevents agent from capturing its own gRPC/HTTP traffic to Hub

#### Pause Mechanism (`pkg/hub/server.go:217-263`)

**Behavior:**
- Controlled via `POST /api/pause` (toggle or set `{"paused": true}`)
- When paused: PCAP data silently dropped (`AddPCAPData` returns early)
- Flows still processed and broadcast to UI
- WebSocket connections remain active
- Allows inspecting current state without filling disk

#### Terminal Integration (`pkg/hub/terminal.go`, `pkg/hub/server.go:441-523`)

**Architecture:**
- WebSocket ↔ Kubernetes SPDY bridge
- Hub must run in-cluster with RBAC to exec into pods
- Opens shell in agent's ephemeral container (shares network namespace)

**Message Protocol:**
- `{"type": "input", "data": "ls\n"}` → stdin
- `{"type": "resize", "cols": 80, "rows": 24}` → terminal resize
- Stdout/stderr → `{"type": "output", "data": "..."}`

**Shell Command:**
- Default: `/bin/sh` (can be changed in code)
- TTY enabled for interactive shell

#### gRPC vs HTTP Hybrid (`pkg/agent/client.go`)

**Current Implementation:**
- Agent uses HTTP despite gRPC server existing on Hub
- Port translation: gRPC addr `:9090` → HTTP addr `:8080`
- Reason: Simpler MVP, no protobuf compilation needed

**Future Optimization:**
- gRPC server implemented but unused (`pkg/hub/grpc.go`)
- Can switch to streaming gRPC for lower latency
- Protobuf definitions in `api/proto/podscope.proto`

### Storage and Memory Considerations

**In-Memory Flow Storage:**
- `[]*protocol.Flow` grows unbounded (MVP limitation)
- RWMutex for concurrent access
- No flow expiration or archival

**PCAP File Storage:**
- Per-agent files: `/data/pcap/agent-<id>.pcap`
- emptyDir volume mounted at `/data/pcap` (1GB limit)
- File handles kept open, synced before read
- Merge strategy: skip each agent's 24-byte global header, concatenate packets

**Trade-offs:**
- Simple implementation vs scalability
- Suitable for short debug sessions, not long-term monitoring
- Session cleanup deletes all data automatically

### Session Lifecycle

1. **Start**: CLI creates namespace, deploys Hub, waits for ready
2. **Inject**: CLI injects agents into target pods via ephemeral containers
3. **Capture**: Agents capture traffic, send to Hub
4. **Monitor**: User views live UI at `http://localhost:<port>`
5. **Stop**: Ctrl+C triggers cleanup
6. **Cleanup**: CLI deletes namespace (cascades to Hub deployment, service, emptyDir volume)
7. **Persistence**: Agents remain in pod spec until pod restart

### Type Safety and Protocol Definitions

**Go Structs (`pkg/protocol/flow.go`):**
- `Flow`: Network 5-tuple, identity (pod/namespace), metrics, timing, protocol-specific fields
- `HTTPInfo`: Method, URL, headers, status, content-type, body
- `TLSInfo`: Version, SNI, cipher suites, handshake timing
- `AgentInfo`: ID, pod name, namespace, IP

**TypeScript Types (`ui/src/types.ts`):**
- Mirror Go structs exactly for JSON serialization
- No custom marshaling needed
- Type-safe WebSocket message handling

**Status Enum:**
- `closed`: Normal FIN handshake
- `reset`: RST packet received
- `timeout`: 30s inactivity
- `active`: Still processing (shouldn't appear in UI)

## Important Development Notes

### CGO and Static Linking

- **Agent requires CGO**: `gopacket` depends on `libpcap` C library
- **Static builds**: Use `make build-agent-static` with `-extldflags "-static"` and `netgo` tag
- **Docker**: Multi-stage build with Alpine base for minimal image size

### Kubernetes RBAC

**Hub Deployment:**
- Needs no RBAC for basic capture (receives data from agents)
- Terminal feature requires ServiceAccount with `pods/exec` permission

**CLI:**
- Needs permissions to:
  - Create namespaces
  - Create deployments and services
  - List and get pods
  - Update ephemeral containers (`pods/ephemeralcontainers` subresource)

### Port Conflicts

- Hub HTTP: 8080 (internal to cluster)
- Hub gRPC: 9090 (internal to cluster)
- UI port-forward: 8899 (default, configurable with `--ui-port`)
- If `--ui-port` unavailable, CLI auto-increments to next available port

### WebSocket Security

**CORS:**
- `CheckOrigin: func(r *http.Request) bool { return true }`
- Allows all origins for local development
- Should be restricted for production use

**No Authentication:**
- Hub has no auth (MVP)
- Security model: Hub only accessible via port-forward or in-cluster
- Not exposed via Ingress/LoadBalancer

### Testing Strategy

**Current State:**
- `make test` runs `go test -v ./...`
- Limited test coverage (MVP)

**Test Considerations:**
- Agent requires root/NET_RAW for packet capture (difficult to test in CI)
- Use mocks for Kubernetes client (`k8s.io/client-go/kubernetes/fake`)
- PCAP parsing can be tested with pre-captured files

### Common Gotchas

1. **Ephemeral containers persist**: Cannot remove agents until pod restart
2. **BPF filter**: Must resolve Hub hostname to IP, not hostname-based filtering
3. **Flow key normalization**: Bidirectional flows must use same key for reassembly
4. **PCAP header merge**: Skip first 24 bytes of each agent file when merging
5. **Pause behavior**: Flows still processed, only PCAP dropped
6. **Terminal requires in-cluster**: Hub must run in Kubernetes to exec into pods
7. **CGO for agent**: Cross-compiling requires appropriate C toolchain

### Directory Structure

```
.
├── cmd/
│   ├── podscope/       # CLI entry point (Cobra commands)
│   ├── hub/            # Hub server entry point
│   └── agent/          # Agent entry point (packet capture loop)
├── pkg/
│   ├── cli/            # CLI commands implementation
│   ├── hub/            # Hub server (HTTP, gRPC, WebSocket, PCAP storage)
│   ├── agent/          # Packet capture, TCP reassembly, protocol parsing
│   ├── k8s/            # Kubernetes client, session management, ephemeral containers
│   └── protocol/       # Shared data structures (Flow, AgentInfo, etc.)
├── api/
│   └── proto/          # gRPC protobuf definitions (currently unused by agent)
├── ui/                 # React frontend (TypeScript, Vite, TailwindCSS)
│   ├── src/
│   │   ├── components/ # React components
│   │   ├── types.ts    # TypeScript types (mirrors Go structs)
│   │   └── App.tsx     # Main app, WebSocket connection
│   └── dist/           # Build output (served by Hub at /app/ui)
├── docker/             # Dockerfiles for Hub and Agent
│   ├── Dockerfile.hub
│   └── Dockerfile.agent
└── Makefile            # Build targets
```

### Key Files to Understand

- `pkg/k8s/session.go`: Session lifecycle, namespace creation, Hub deployment, agent injection
- `pkg/agent/capture.go`: Packet capture loop, BPF filter, PCAP streaming
- `pkg/agent/assembler.go`: TCP reassembly, protocol detection, HTTP/TLS parsing
- `pkg/hub/server.go`: HTTP API, WebSocket broadcasting, flow storage
- `pkg/hub/pcap.go`: PCAP file storage, per-agent files, merge logic
- `pkg/hub/terminal.go`: WebSocket ↔ Kubernetes exec bridge
- `ui/src/App.tsx`: React UI, WebSocket connection, flow table

### Protobuf Generation

```bash
make proto    # Requires protoc, protoc-gen-go, protoc-gen-go-grpc
```

**Output:**
- `api/proto/podscope.pb.go`: Message definitions
- `api/proto/podscope_grpc.pb.go`: Service definitions

**Note:** Currently not used by agent (uses HTTP instead), but available for future optimization.

## Code Style and Conventions

- Go code follows standard `gofmt` formatting
- Error handling: wrap errors with `fmt.Errorf("context: %w", err)`
- Logging: use `log.Printf` for Hub/Agent, `fmt.Printf` for CLI output
- Struct tags: `json:"fieldName"` for JSON serialization
- Concurrency: use `sync.RWMutex` for read-heavy data structures (flows, PCAP buffer)
- Resource cleanup: always defer cleanup in CLI, use context cancellation for goroutines

## UI Design System: "Precision Observatory"

The UI follows a sophisticated dark theme inspired by scientific visualization and observatory control panels - the feeling of watching network traffic through a precision instrument.

### Design Philosophy

- **Bold but intentional**: Deep blacks with bioluminescent cyan accents
- **Data-first**: Typography and layout optimized for scanning network flows
- **Subtle depth**: Glass effects, noise textures, and layered backgrounds create atmosphere without distraction

### Color Palette (defined in `tailwind.config.js`)

```
Backgrounds (void):
  void-950: #050508  (deepest - main background)
  void-900: #080810  (primary surfaces)
  void-800: #0c0c18  (cards, elevated surfaces)
  void-700: #141426  (borders, subtle elements)
  void-600: #1a1a30  (hover states)

Primary Accent (glow):
  glow-400: #00ffd5  (primary cyan - buttons, highlights, active states)
  glow-500: #00e4c4  (hover states)
  glow-600: #00baa3  (pressed states)

Status Colors:
  status-success: #00ffa3  (HTTP 2xx, closed connections)
  status-warning: #ffd000  (HTTP 4xx, timeouts)
  status-error:   #ff4757  (HTTP 5xx, resets)
  status-info:    #00d4ff  (TCP, informational)

Protocol Colors:
  HTTP:  emerald-400 (#34d399) - green tint
  HTTPS/TLS: amber-400 (#fbbf24) - gold/secure
  TCP:   blue-400 (#60a5fa) - neutral blue
```

### Typography

```
Display/UI: Sora (Google Fonts)
  - Geometric sans-serif with distinctive character
  - Used for headings, labels, buttons
  - Weights: 400 (body), 500 (labels), 600 (headings), 700 (emphasis)

Monospace: JetBrains Mono (Google Fonts)
  - Used for ALL data: IPs, ports, timestamps, sizes, code, IDs
  - Clear distinction between similar characters (0/O, 1/l)
  - Weights: 400 (data), 500 (emphasis), 600 (headers)
```

### Component Classes (defined in `index.css`)

```css
/* Glass card - primary container style */
.glass-card - Semi-transparent background with blur and subtle glow border

/* Buttons */
.btn-primary   - Gradient cyan, dark text, glow on hover
.btn-secondary - Dark with border, glows on hover
.btn-ghost     - Transparent, icon-friendly, subtle hover
.btn-danger    - Red-tinted for destructive actions

/* Form elements */
.input-field   - Dark input with glow focus ring

/* Data display */
.status-badge  - Protocol/status pills with semantic colors
.protocol-http, .protocol-https, .protocol-tcp - Protocol-specific styling

/* Effects */
.glow-text     - Text with cyan glow shadow
.pulse-dot     - Animated live indicator
.divider       - Gradient horizontal rule
```

### Visual Effects

1. **Noise overlay** (`noise-overlay` class): Subtle grain texture over entire UI
2. **Grid pattern** (`grid-bg` class): Faint cyan grid lines in background
3. **Radial glow** (`glow-bg` class): Soft glow emanating from header area
4. **Row hover** (`row-glow` class): Sweep effect on table row hover

### Animation Patterns

- `animate-pulse-glow`: Pulsing opacity for live indicators
- `animate-fade-in`: Entrance fade (0.3s)
- `animate-slide-in`: Slide from right (detail panel)
- `animate-slide-up`: Slide from bottom (filter panel)
- Stagger classes: `.stagger-1` through `.stagger-6` for sequential reveals

### Key UI Components

**Header** (`components/Header.tsx`):
- Logo with gradient icon and pulsing live indicator
- Search bar with glow focus state
- Stats pills (flow count, PCAP size)
- Collapsible filter panel with protocol chips and BPF input

**FlowList** (`components/FlowList.tsx`):
- Virtualized for performance (1000+ flows)
- Server/Globe icons for source/destination
- Lock icon for encrypted traffic (HTTPS/TLS)
- Protocol badges with semantic colors
- Status codes with HTTP-aware coloring

**FlowDetail** (`components/FlowDetail.tsx`):
- Glass card sections for visual hierarchy
- Quick stats row at top
- Connection visualization with source/destination cards
- Timing bar with TCP/TLS/TTFB breakdown
- HTTP request/response with headers tables

**Terminal** (`components/Terminal.tsx`):
- xterm.js with matching color scheme
- Cyan cursor matching glow accent
- Gradient background matching void palette

### Demo Mode

For UI development without a running Hub:

```typescript
// In App.tsx
const DEMO_MODE = import.meta.env.DEV && !window.location.port?.includes('8')
```

- Auto-enables on Vite dev server (port 5173)
- Auto-disables when port contains '8' (Hub ports 8080, 8899)
- Loads mock flows from `lib/mockData.ts`
- Simulates new flows arriving every 3 seconds

Run `npm run dev` in `ui/` directory to preview with mock data.

### Design Considerations for Future Changes

1. **Color changes**: Update `tailwind.config.js` colors - all components use semantic names (void, glow, status)

2. **Adding new status colors**: Add to `tailwind.config.js` under `status`, then use as `text-status-{name}` or `bg-status-{name}/10`

3. **New protocol types**: Add protocol class in `index.css` following `.protocol-http` pattern

4. **Component consistency**: Always use the defined button classes (`btn-primary`, etc.) and `glass-card` for containers

5. **Animations**: Keep animations subtle and fast (<0.3s) - this is a data tool, not a marketing site

6. **Typography**: Data should ALWAYS be monospace (`font-mono` class). Never use proportional fonts for IPs, ports, sizes, or timestamps

7. **Spacing**: Use Tailwind's spacing scale. Prefer `gap-` over margin for flex/grid layouts

8. **Icons**: Use Lucide React icons exclusively. Keep icons small (w-3.5 to w-5) to not overpower data

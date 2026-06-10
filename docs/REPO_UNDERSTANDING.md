# PodScope Repository Understanding Dossier

> Created by Hermes after initial deep-read of `DominoRamino/podscope` at commit `162cf36`.
>
> Purpose: shared working memory for future development. This is not marketing docs; it is an engineering map of how the repo actually works today, including mismatches and questions.

## Executive Summary

PodScope is a Kubernetes network-traffic observability/debugging tool. A local CLI creates an ephemeral session namespace, deploys a Hub, injects packet-capture Agents into target pods via Kubernetes ephemeral containers, port-forwards the Hub UI to localhost, and cleans up session resources on exit.

The current implementation is best understood as three cooperating systems:

1. **CLI/session orchestrator**: `cmd/podscope`, `pkg/cli`, `pkg/k8s`
2. **Capture plane**: `cmd/agent`, `pkg/agent`, `pkg/protocol`
3. **Hub + UI plane**: `cmd/hub`, `pkg/hub`, `ui/`

The README/architecture text often says Agent ↔ Hub uses gRPC. In the current code, active Agent ↔ Hub transport is HTTP POST + health polling on port `8080`; gRPC server/proto code exists but appears unused by the Agent path.

## Current Repo Shape

Approximate source composition, excluding `.git`, dependency folders, and build output:

- Go: 27 files, ~16.2k lines
- React TSX: 11 files, ~5.7k lines
- TypeScript: 13 files, ~1.8k lines
- Markdown: 12 files, ~3.4k lines
- Shell/scripts: 5 files, ~790 lines
- JSON/config: 5 files, ~5.5k lines
- Dockerfiles: 2 files
- Proto: 1 file

Important top-level files/directories:

- `README.md`: product overview, quick start, architecture, limitations.
- `CLAUDE.md`: very detailed repo guidance, but currently stale in several places.
- `Makefile`: local build/dev/test workflow. Has a syntax error in `make help`.
- `VERSION`: currently `0.1.2`.
- `feat-plan.md`: plan for advanced packet metrics; many fields already exist in structs/UI.
- `cmd/`: binaries for CLI, Hub, Agent.
- `pkg/`: Go libraries for CLI, Kubernetes, Agent, Hub, shared protocol.
- `ui/`: React/Vite/Tailwind UI.
- `docker/`: Hub and Agent images.
- `scripts/`: minikube setup, stress tests, debug helpers.
- `.github/workflows/release.yml`: release-only workflow triggered by merged PR with `release` label.
- `md_files/`: older supplemental docs/guides, many stale relative to code.

## Architecture and Runtime Flow

### 1. CLI Orchestration

Primary files:

- `cmd/podscope/main.go`
- `pkg/cli/root.go`
- `pkg/cli/tap.go`
- `pkg/k8s/client.go`
- `pkg/k8s/session.go`

Primary entrypoint:

- `pkg/cli/tap.go:runTap`

High-level sequence:

1. Parse CLI flags:
   - `--namespace` / `-n`
   - `--selector` / `-l`
   - `--pod`
   - `--all-namespaces` / `-A`
   - `--force-privileged`
   - `--hub-port` (currently appears unused/misleading)
   - `--ui-port`
   - `--container`
   - `--anthropic-api-key`
2. Create Kubernetes client with current kubeconfig/context.
3. Run cleanup helpers for stale PodScope namespaces/RBAC.
4. Create a session object with random session ID.
5. Create namespace `podscope-<sessionID>`.
6. Deploy Hub resources into that namespace.
7. Resolve target pods by exact pod name or label selector.
8. Inject Agent ephemeral containers into target pods.
9. Port-forward local UI port to Hub HTTP port `8080`.
10. Wait until context cancellation, then clean up namespace and session-scoped RBAC.

### 2. Kubernetes Resources

Created programmatically in `pkg/k8s/session.go:deployHub`:

- Namespace: `podscope-<sessionID>`
- ServiceAccount for Hub
- ClusterRole for pod list/get and pod exec
- ClusterRoleBinding to Hub ServiceAccount
- Hub Deployment
- Hub ClusterIP Service
- `emptyDir` mounted to `/data/pcap`, size limit 1Gi

Agent injection happens in `pkg/k8s/session.go:InjectAgent`:

- Uses Kubernetes `UpdateEphemeralContainers` API.
- Appends an ephemeral container named `podscope-agent-<sessionID>`.
- Sets env:
  - `HUB_ADDRESS`
  - `POD_NAME`
  - `POD_NAMESPACE`
  - `POD_IP`
  - `SESSION_ID`
  - `INTERFACE=eth0`
- Adds `NET_RAW`; optionally privileged with `--force-privileged`.
- Targets first app container unless `--container` is provided.

Important Kubernetes fact: ephemeral containers cannot be removed from a pod spec; restarting the target pod is the practical reset path.

### 3. Agent Capture Plane

Primary files:

- `cmd/agent/main.go`
- `pkg/agent/capture.go`
- `pkg/agent/assembler.go`
- `pkg/agent/client.go`
- `pkg/agent/cipher_suites.go`
- `pkg/protocol/flow.go`

Agent boot sequence:

1. Read env config from injected ephemeral container.
2. Build `protocol.AgentInfo`.
3. Create `HubClient`.
4. Register with Hub.
5. Create `Capturer` for interface `eth0`.
6. Resolve Hub IP/hostname and build exclusion filter.
7. Start packet capture using `gopacket/pcap`.
8. For each packet:
   - Write raw packet to in-memory PCAP buffer.
   - If TCP, pass packet metadata and app payload to `TCPAssembler`.
9. Flush PCAP chunks to Hub every 500ms.
10. Send completed flows to Hub.
11. Poll Hub health every 5s and pick up dynamic BPF filter changes.

Capture/filtering details:

- `pkg/agent/capture.go:BuildCombinedFilter` combines Hub feedback exclusion with user BPF.
- Current default filter excludes Hub traffic, not all DNS traffic.
- `pkg/agent/capture.go:isHubDNSPacket` filters DNS packets specifically to/from Hub DNS lookups in code.
- Dynamic BPF is pulled from `GET /api/health` on heartbeat.

TCP/protocol parsing:

- `pkg/agent/assembler.go:flowKey` normalizes bidirectional flows by sorting endpoint strings.
- `ProcessPacket` tracks SYN/SYN-ACK/FIN/RST, byte counts, packets, payload buffers, timing.
- `detectProtocol` detects HTTP methods, TLS ClientHello shape, and 443/8443 HTTPS heuristic.
- `parseHTTP` uses Go `net/http` parsing for request/response metadata.
- `parseTLS` extracts ClientHello details and ServerHello cipher when available.
- `completeFlow` produces `protocol.Flow` and does pod attribution based on the Agent pod IP/env.

Limitations to keep in mind:

- This is not full TCP sequence reassembly; out-of-order packets/retransmits/midstream captures can confuse parsing.
- First observed packet can define direction incorrectly if capture begins mid-connection.
- Completion logic is simpler than full TCP lifecycle; high-fidelity stream accounting may need deeper work.

### 4. Hub Plane

Primary files:

- `cmd/hub/main.go`
- `pkg/hub/server.go`
- `pkg/hub/pcap.go`
- `pkg/hub/flowbuffer.go`
- `pkg/hub/grpc.go`
- `pkg/hub/terminal.go`

Hub responsibilities:

- Serve static UI from `/app/ui`.
- Receive Agent flows over `POST /api/flows`.
- Receive PCAP chunks over `POST /api/pcap/upload`.
- Store recent flows in a bounded ring buffer.
- Broadcast flow batches to UI over WebSocket.
- Store/merge PCAP files for download.
- Expose terminal WebSocket into injected Agent containers.
- Store dynamic BPF filters and validate them.
- Proxy Anthropic BPF-generation requests.

Important endpoints:

- `GET /api/health`: health + current BPF filter.
- `GET /api/stats`: flow count/capacity, WebSocket clients, PCAP size/full, session ID, uptime, paused state.
- `GET/POST /api/flows`: flow list / Agent flow ingestion.
- `GET /api/flows/ws`: UI WebSocket; catch-up then batched updates.
- `POST /api/pcap/upload`: Agent PCAP chunk upload.
- `GET /api/pcap`: download merged session PCAP.
- `GET /api/pcap/<streamID>`: currently not truly stream-scoped; returns session PCAP behavior.
- `POST /api/pcap/reset`: clear PCAP data.
- `GET/POST /api/pause`: pause state; code pauses PCAP storage, not necessarily all flow ingest.
- `GET/POST /api/bpf-filter`: current/apply/clear dynamic BPF.
- `GET /api/terminal/ws`: terminal into Agent container.
- `POST /api/ai/anthropic`: Anthropic proxy.

Flow/PCAP storage:

- `FlowRingBuffer` defaults to env `MAX_FLOWS` or 10,000.
- WebSocket batching defaults:
  - `WS_BATCH_INTERVAL_MS=150`
  - `WS_CATCHUP_LIMIT=200`
- PCAP buffer is hardcoded at 100MiB in `hub.NewServer`, while `emptyDir` is 1Gi.

### 5. React UI

Primary files:

- `ui/src/main.tsx`
- `ui/src/App.tsx`
- `ui/src/components/Header.tsx`
- `ui/src/components/FlowList.tsx`
- `ui/src/components/FlowDetail.tsx`
- `ui/src/components/Terminal.tsx`
- `ui/src/types.ts`
- `ui/src/utils.ts`
- `ui/src/lib/bpfPresets.ts`

UI structure:

- `App.tsx` owns global state, WebSocket connection, stats polling, filters, selected flow, terminal state.
- `Header.tsx` handles connection/status/search/filter controls, BPF UI, AI filter generation, PCAP controls, pause/resume.
- `FlowList.tsx` renders virtualized, sortable flow rows.
- `FlowDetail.tsx` renders selected flow details: endpoints, timing, transfer, advanced metrics, TLS, HTTP.
- `Terminal.tsx` wraps xterm.js and connects to `/api/terminal/ws`.

UI data flow:

- Opens `/api/flows/ws`.
- Accepts `{ type: "catchup", flows: [...] }` and `{ type: "batch", flows: [...] }`.
- Merges by `flow.id`, sorts newest first, stores max 1000 flows client-side.
- Polls `/api/stats` every 5 seconds.
- Applies client-side filtering before rendering `FlowList`.

Notable UI issues:

- `selectedFlow` stores the object, not ID; detail can become stale if a flow object is updated in `flows` later.
- Default filter hides TCP/non-HTTP-ish traffic, which may make new captures look empty.
- Pause semantics are unclear: UI ignores new WebSocket data while paused; backend mainly drops PCAP storage.
- PCAP download controls send filter query params that backend currently does not enforce.
- Errors are mostly `console.error` or `alert`, not integrated toasts/status banners.
- Azure OpenAI API key path appears browser-exposed via `VITE_*`, unlike Anthropic proxy.
- Accessibility gaps: icon-only buttons need `aria-label`, sort headers should expose sort state.

## Build, Test, Dev, Release

### Local tools available in this environment during inspection

- Node: `v20.20.2`
- npm: `10.8.2`
- Go: not installed on this host
- Docker/kubectl/minikube: not found on this host

So Go tests, Docker builds, and minikube live runs could not be verified here yet.

### Makefile targets

Key targets:

- `make build-cli`
- `make build-cli-linux`
- `make build`
- `make build-agent`
- `make build-hub`
- `make load`
- `make all`
- `make setup-cluster`
- `make dev`
- `make dev-quick`
- `make dev-ui`
- `make restart-test-pods`
- `make test`
- `make test-ui`
- stress-test targets

Resolved in the stabilization pass:

- `make help` no longer fails due to the previous unterminated quote in the `Version Management` line.
- Makefile dev/restart/stress-test selectors now use `app=podinfo`, matching `scripts/setup-cluster.sh` and `scripts/test-workloads/podinfo.yaml`.

Historical dev workflow problem:

- `scripts/test-workloads/podinfo.yaml` uses label `app: podinfo`.
- `setup-cluster.sh` checks `app=podinfo`.
- Older Makefile dev/restart paths used `app.kubernetes.io/name=podinfo`; this has been aligned to `app=podinfo`.

### Dockerfiles

`docker/Dockerfile.agent`:

- Go builder on Alpine, installs libpcap/build deps.
- Builds Agent with CGO enabled.
- Final Alpine image includes libpcap and debug tools (`tcpdump`, `curl`, DNS tools, netcat, traceroute, wget).

`docker/Dockerfile.hub`:

- Builds Go Hub binary.
- Builds React UI with Node 20.
- Final Alpine image exposes `8080` and `9090`, stores PCAP at `/data/pcap`.

### Release workflow

`.github/workflows/release.yml`:

- Trigger: merged PR to `main` with label `release`.
- Builds Linux amd64 CLI.
- Pushes Docker Hub images:
  - `dominoramino/podscope:<version>` and `latest`
  - `dominoramino/podscope-agent:<version>` and `latest`
- Creates GitHub Release with `podscope-linux-amd64`.

Historical release risk:

- CLI embeds `pkg/k8s.DefaultImageTag=<git_commit>`.
- Runtime defaults become local/unqualified `podscope:<git_commit>` and `podscope-agent:<git_commit>`.
- Release workflow pushes version/latest tags under `dominoramino/...`, not git-commit local tags.
- The stabilization pass added release-only `DefaultHubImage` / `DefaultAgentImage` ldflags while preserving local commit-tagged defaults when those values are unset.

CI gap:

- No regular PR CI workflow.
- Release workflow does not run Go tests, UI tests, lint, or build matrix before publishing.

## Data Model

Shared Go schema: `pkg/protocol/flow.go`

Core `Flow` fields:

- Identity/timing: `id`, `timestamp`, `duration`
- Source: `srcIp`, `srcPort`, `srcPod`, `srcNamespace`
- Destination: `dstIp`, `dstPort`, `dstPod`, `dstNamespace`, `dstService`
- Protocol/status: `protocol`, `status`
- Size counters: `bytesSent`, `bytesReceived`, `packetsSent`, `packetsReceived`
- Timing metrics: `tcpHandshakeMs`, `tlsHandshakeMs`, `ttfbMs`
- Nested metadata: `http`, `tls`
- Agent-noise tags: `isAgentTraffic`, `agentTrafficType`

HTTP fields already include body preview fields:

- `requestBody`
- `responseBody`

TLS fields already include advanced fields:

- `cipherSuite`
- `cipherSuites`
- `alpn`

This matches `feat-plan.md`: many desired advanced metrics have schema/UI places already present; the main remaining work is robust population and polish.

## High-Value Improvement Backlog

### Stabilization / hygiene first

1. Add PR CI for Go tests, UI tests/build, and lint.
2. Continue replacing stale docs with the current HTTP transport and local-dev/release image split.
3. Install/check toolchain expectations: Go 1.24, Node 20, Docker, kubectl, minikube.
4. Decide later whether to remove, revive, or clearly mark unused gRPC/proto code as experimental.
5. Fix remaining stale docs under `md_files/` and decide which historical docs should be archived.

### Product/UX correctness

1. Clarify pause semantics and make backend/UI behavior match.
2. Implement or remove PCAP filtered-download UI until filters are real.
3. Store `selectedFlowId`, derive current flow detail from `flows`.
4. Add hidden-by-filter empty state versus no-traffic empty state.
5. Add toast/status surfaces for errors.
6. Add BPF application state per Agent.
7. Improve terminal UX/security expectations.

### Capture/protocol depth

1. Populate TLS cipher suites and ALPN from ClientHello.
2. Populate HTTP body previews safely with truncation/content-type awareness.
3. Improve TTFB and TLS handshake timing.
4. Improve TCP stream direction/reassembly robustness.
5. Add support for HTTP/2/gRPC metadata from ALPN/headers where feasible.
6. Add service attribution for Kubernetes destinations.
7. Add tests with packet fixtures for realistic HTTP/TLS flows.

### Security / operations

1. Tighten WebSocket origin policy if Hub can be exposed beyond localhost.
2. Add optional local auth token for UI/API/terminal.
3. Avoid browser-exposed AI provider keys; proxy consistently.
4. Review cluster-scoped RBAC footprint and cleanup guarantees.
5. Make PCAP retention/storage configurable.
6. Add warnings around sensitive payload capture.

## Known Mismatches / Stale Docs

- README now says Go 1.24+, matching `go.mod`, Dockerfiles, and Actions.
- README/CLAUDE now describe HTTP as the active Agent transport; some older docs may still mention gRPC/proto historically.
- `CLAUDE.md` mentions many Makefile targets that do not exist:
  - `make build-ui`
  - `make dev-hub`
  - `make build-agent-static`
  - `make docker-build*`
  - `make docker-push`
  - `make install`
  - `make release`
  - `make proto`
- `md_files/VERSIONING_GUIDE.md` describes timestamp/latest local tagging; Makefile uses git commit tags.
- Some script docs use `./debug.sh`; actual path is `scripts/debug.sh`.
- Scripts are generally not executable; invoke with `bash scripts/...` or chmod.
- Dynamic BPF docs say no syntax validation, but Hub has `validateBPFFilter` now.
- Older filtering docs say DNS is globally excluded; current code only excludes Hub feedback traffic / Hub DNS specifics.

## Questions for Ramy

1. **Primary target environment:** Is PodScope mainly for local minikube/WSL dogfooding, or should we make it installable for arbitrary Kubernetes clusters first?
2. **Transport direction:** Do you want to keep HTTP Agent ↔ Hub because it is simple/debuggable, or should we revive the gRPC/proto path as the canonical transport?
3. **Release model:** Should released CLI default images be `dominoramino/podscope:<version>` and `dominoramino/podscope-agent:<version>`, or do you want local/minikube commit tags as the primary workflow?
4. **Near-term priority:** Should the next work focus on stabilization/dev-loop fixes, advanced packet metrics from `feat-plan.md`, UI polish, or production-readiness/security?
5. **Capture scope:** Is plaintext HTTP body capture a desired default, or should it be opt-in/redacted because payloads can be sensitive?
6. **Terminal feature:** Is browser terminal into the Agent meant as a core feature, a debug-only feature, or something we should gate behind a flag/auth?
7. **AI BPF generation:** Which provider path should be canonical: Anthropic proxy, Azure OpenAI, local model, or provider-agnostic backend proxy?
8. **Testing environment:** Do you have a preferred minikube/WSL setup on this machine that I should use for live verification, or should I provision missing tools here?

## Suggested First PR

A good first improvement PR would be a stabilization sweep:

1. Add a minimal PR CI workflow for:
   - `go test ./pkg/...` once Go is available in CI
   - `cd ui && npm ci && npm run build && npm test -- --run`
2. Fix remaining stale docs under `md_files/` and decide which historical docs should be archived.
3. Decide later whether backend should enforce PCAP filtered download query params or UI should stop presenting unsupported filters.

This would make future feature work much safer and make the developer loop trustworthy.


## Ramy's Product Direction Answers

Recorded after owner feedback. These supersede open questions above where applicable.

- **Primary target environment:** PodScope should be Kubernetes-flavor agnostic. AKS is an important real target, but the app should not bake in AKS-specific assumptions. Minikube is only for local/dev testing.
- **Agent ↔ Hub transport:** Hermes should make the engineering recommendation. Default recommendation for stabilization: keep HTTP as canonical for now because it is already implemented, simpler to debug, and sufficient for dev-loop stabilization; revisit gRPC only when throughput/backpressure or API shape demands it.
- **Image tagging:** Keep local git-commit image tags for minikube development. Release/install paths should be handled separately so non-minikube users get registry-qualified version/latest images without breaking local dev.
- **Immediate priority:** Stabilization/dev-loop fixes before deeper feature work.
- **Plaintext body capture:** Acceptable and useful by default for debugging. The security model is ephemeral sessions rather than avoiding plaintext capture.
- **Terminal feature:** Core feature. Treat it as a netshoot-like shell for testing connectivity from inside the cluster, not a temporary debug hack.
- **AI/BPF integration:** Low priority for now. Long term, accepting multiple connector/provider mutations would be ideal; current AI integration is acceptable as-is for the moment.

## Transport Recommendation

For the stabilization phase, keep **HTTP Agent ↔ Hub** as canonical and update docs/code naming around that reality. Reasons:

1. The implementation already works around HTTP endpoints: flows, PCAP upload, health/BPF polling.
2. It is easier to inspect with curl, browser/devtools, and Kubernetes port-forwarding while stabilizing minikube and real-cluster loops.
3. Current gRPC code is not active in the Agent path, so reviving it would add risk before the basics are stable.
4. Once dev-loop, release images, tests, and Kubernetes-flavor-agnostic assumptions are solid, gRPC can be reconsidered specifically for streaming/backpressure/performance.

Immediate implication: docs should stop claiming gRPC as the active transport, `--hub-port` naming should be fixed or removed, and gRPC code should be marked experimental/dead-code-candidate until intentionally revived.

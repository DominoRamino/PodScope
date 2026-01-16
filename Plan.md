\# Project Plan: PodScope Shark (MVP)



This plan assumes \*\*`NET\_RAW` capability is permitted\*\* within Ephemeral Containers and targets \*\*HTTP/1.1\*\* for the MVP. The architecture relies on packet sniffing (PCAP) rather than proxying, ensuring zero intrusion into the application process.



---



\## 0) Reality Constraints \& Design Guardrails



\*   \*\*Targeting:\*\* Strict dependency on \*\*Ephemeral Containers\*\* (`kubectl debug`). If the cluster API version is too old or the feature is disabled/blocked by policy, the CLI will \*\*hard fail\*\*. No sidecar injection or rolling restart fallbacks.

\*   \*\*Privileges:\*\* The capture agent runs with `CAP\_NET\_RAW` (or `privileged: true` if permitted) but \*\*strictly scoped to the Pod Network Namespace\*\*. No access to Node network or host filesystem.

\*   \*\*Visibility Scope:\*\*

&nbsp;   \*   \*\*HTTPS:\*\* L4 TCP timing, TLS Handshake analysis (SNI, ALPN, Cipher), Encrypted Data size.

&nbsp;   \*   \*\*HTTP (Plaintext):\*\* Full L7 visibility (Method, URL, Headers, Body, Status Code).

&nbsp;   \*   \*\*Traffic:\*\* Inbound and Outbound visibility (via interface sniffing).



---



\## 1) High-Level Architecture



\### A) CLI: Session Controller (`podscope`)

\*   \*\*Role:\*\* The user entry point and orchestrator.

\*   \*\*Logic:\*\*

&nbsp;   1.  Uses user's current kubeconfig.

&nbsp;   2.  Creates a session namespace (e.g., `podscope-<id>`).

&nbsp;   3.  Deploys the \*\*Hub\*\* (Deployment + Service).

&nbsp;   4.  Locates target pods via selectors.

&nbsp;   5.  Injects the \*\*Capture Agent\*\* via `EphemeralContainer`.

&nbsp;   6.  Establishes `port-forward` to the Hub UI.

&nbsp;   7.  \*\*Watchdog:\*\* Monitors connection. On `SIGINT` (Ctrl+C), it deletes the session namespace (cleaning up Hub and triggering Agent termination).



\### B) Capture Agent (The "Tap")

\*   \*\*Format:\*\* A lightweight, statically compiled Go binary (wrapping `gopacket` or `libpcap`).

\*   \*\*Deployment:\*\* Runs inside the Ephemeral Container attached to the target Pod.

\*   \*\*Function:\*\*

&nbsp;   \*   Binds to `eth0` (or `any`) using `AF\_PACKET`.

&nbsp;   \*   \*\*Fast Path:\*\* Buffers raw packets and streams them to the Hub (for PCAP download).

&nbsp;   \*   \*\*Analysis Path:\*\*

&nbsp;       \*   Reassembles TCP streams in memory.

&nbsp;       \*   Parses TLS ClientHello/ServerHello for metadata (SNI, latency).

&nbsp;       \*   Parses HTTP/1.1 plaintext payloads for Headers/Body.

&nbsp;   \*   \*\*Transport:\*\* Sends structured events (Protobuf/gRPC) to the Hub over mTLS.



\### C) Hub (Session Aggregator)

\*   \*\*Location:\*\* Running in the session namespace.

\*   \*\*Storage:\*\* In-memory Ring Buffer + `emptyDir` for PCAP chunks.

\*   \*\*Role:\*\*

&nbsp;   \*   Receives streams from multiple Agents.

&nbsp;   \*   Correlates request/response pairs.

&nbsp;   \*   Serves the WebSocket API for the UI.

&nbsp;   \*   Generates downloadable `.pcap` files on demand by concatenating chunks.



\### D) Web UI

\*   \*\*Stack:\*\* React (Single Page App).

\*   \*\*Mode:\*\* "Live Tail" view similar to Wireshark/Kubeshark.

\*   \*\*Data:\*\* Connects to Hub via WebSocket.



---



\## 2) MVP Feature Set



\### Stream List (Left Panel)

\*   \*\*Rows:\*\* Live updating list of TCP flows.

\*   \*\*Columns:\*\*

&nbsp;   \*   `Timestamp`

&nbsp;   \*   `Source` (Pod Name / IP)

&nbsp;   \*   `Destination` (Service Name / External IP)

&nbsp;   \*   `Protocol` (HTTP / HTTPS / TCP)

&nbsp;   \*   `Status` (HTTP Code for plaintext; "Connected/Reset" for TCP/TLS)

&nbsp;   \*   `Latency` (TTFB or Handshake duration)

&nbsp;   \*   `Size`



\### Detail View (Right Panel)

\*   \*\*Summary:\*\*

&nbsp;   \*   Total duration, Client IP, Server IP.

\*   \*\*Timing Waterfall:\*\*

&nbsp;   \*   TCP Handshake (SYN → ACK).

&nbsp;   \*   TLS Handshake (ClientHello → Finished).

&nbsp;   \*   Processing Time (Request Last Byte → Response First Byte).

\*   \*\*L7 Attributes:\*\*

&nbsp;   \*   \*\*TLS:\*\* SNI, TLS Version, Cipher Suite.

&nbsp;   \*   \*\*HTTP (Plaintext):\*\* Request Headers, Response Headers, Body (truncated to 1KB for MVP).

&nbsp;   \*   \*\*HTTP (Encrypted):\*\* "Payload Encrypted" placeholder.



\### Actions

\*   \*\*Download PCAP:\*\* Button to download the raw pcap file for a specific stream or the whole session.

\*   \*\*Filter:\*\* Simple text filter (regex) against Namespace, Pod, or Host.



---



\## 3) Security Model



1\.  \*\*Capabilities:\*\*

&nbsp;   \*   Agent Container requests `securityContext: { capabilities: { add: \["NET\_RAW"] } }`.

&nbsp;   \*   If the cluster enforces `restricted-pod-security-standards`, the CLI will fail fast unless the user explicitly acknowledges a `--force-privileged` flag that attempts to elevate the Ephemeral Container.



2\.  \*\*Network Isolation:\*\*

&nbsp;   \*   Hub Service is `ClusterIP` only. Access is solely via `kubectl port-forward` managed by the CLI.

&nbsp;   \*   Agent-to-Hub communication uses mutual TLS (certs generated by CLI at startup and mounted to pods).



3\.  \*\*Data Lifecycle:\*\*

&nbsp;   \*   All data stored in `emptyDir` (RAM or node disk).

&nbsp;   \*   Namespace deletion (on exit) wipes all data instantly.



---



\## 4) Implementation Plan



\### Phase 1: The "Wire" (Weeks 1-2)

\*\*Goal:\*\* Successfully tap a pod and get a PCAP out.

1\.  \*\*CLI Scaffold:\*\* Implement `podscope start` that creates a namespace and a dummy Hub.

2\.  \*\*Injector:\*\* Implement `kubectl debug` wrapper logic in Go.

&nbsp;   \*   Must handle container image selection (use a scratch image with the static binary).

3\.  \*\*Agent V1 (Sniffer):\*\*

&nbsp;   \*   Use `gopacket` to open `eth0`.

&nbsp;   \*   Write packets to a local file.

&nbsp;   \*   Stream file chunks to Hub.

4\.  \*\*Verification:\*\* CLI creates session -> PCAP accumulates in Hub -> User can `curl` the Hub to download it.



\### Phase 2: The "Analyzer" (Weeks 3-4)

\*\*Goal:\*\* Turn packets into structured metadata.

1\.  \*\*Stream Assembly:\*\* Implement TCP reassembly in the Agent.

2\.  \*\*Parsers:\*\*

&nbsp;   \*   \*\*TLS Parser:\*\* Extract SNI and handshake timing.

&nbsp;   \*   \*\*HTTP/1.1 Parser:\*\* Extract Method, URL, Status, Headers.

3\.  \*\*Event Pipeline:\*\* Send parsed `Flow` objects to Hub via gRPC.

4\.  \*\*UI V1:\*\* Basic table showing the `Flow` objects coming from the socket.



\### Phase 3: The "Experience" (Weeks 5-6)

\*\*Goal:\*\* Kubeshark-like look and feel.

1\.  \*\*Frontend Polish:\*\* Split view (List/Details).

2\.  \*\*Filtering:\*\* Implement client-side filtering in the UI.

3\.  \*\*Correlations:\*\* Map IP addresses to Kubernetes Service names (Hub queries K8s API to resolve IPs).

4\.  \*\*Cleanup:\*\* Ensure `SIGINT` handler in CLI reliably destroys the ephemeral containers and namespace.



---



\## 5) Tech Stack



\*   \*\*CLI:\*\* Go (using `client-go`).

\*   \*\*Agent:\*\* Go (using `google/gopacket`). Statically linked.

\*   \*\*Hub:\*\* Go (HTTP/Websocket server).

\*   \*\*UI:\*\* React + Tailwind + Vite.

\*   \*\*Protocol:\*\* gRPC (Agent -> Hub), JSON/WebSocket (Hub -> UI).



---



\## 6) Acceptance Criteria



1\.  \*\*Installation-Free:\*\* User downloads 1 binary (`podscope`) and runs it. No Helm charts, no DaemonSets.

2\.  \*\*Targeting:\*\* `podscope tap -n default -l app=frontend` attaches to all matching pods.

3\.  \*\*Visibility:\*\*

&nbsp;   \*   User curls the target pod.

&nbsp;   \*   UI shows the request immediately.

&nbsp;   \*   If HTTP: Headers are visible.

&nbsp;   \*   If HTTPS: SNI and Timing are visible.

4\.  \*\*Artifacts:\*\* Clicking "Download PCAP" provides a file readable by Wireshark.

5\.  \*\*Teardown:\*\* Closing the CLI leaves \*\*zero\*\* resources in the cluster (Namespace deleted, Ephemeral Containers terminate when Pods are eventually rolled or via GC).



---



\## 7) Known Limitations (MVP)



\*   \*\*Encrypted Traffic:\*\* We will \*not\* perform MITM/Decryption for HTTPS in MVP. We provide metadata only.

\*   \*\*HTTP/2 \& gRPC:\*\* Will be treated as raw TCP/TLS streams (no header parsing).

\*   \*\*Container Restart:\*\* Ephemeral containers persist until the Pod is restarted. We cannot "remove" the container, only stop the process inside it. (This is a K8s limitation).


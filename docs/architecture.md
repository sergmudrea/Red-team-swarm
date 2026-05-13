# Hive 2.0 Red Swarm — Architecture Document

**Version:** 1.0  
**Status:** Approved  
**Audience:** Developers, Red Team Operators, Security Engineers  
**Last Updated:** 2026-05-13

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Design Principles](#design-principles)
3. [Component Architecture](#component-architecture)
   - 3.1 [Agent (Bee)](#agent-bee)
   - 3.2 [Command Centre (Hive)](#command-centre-hive)
   - 3.3 [Fronting Proxy](#fronting-proxy)
4. [Communication Protocol](#communication-protocol)
   - 4.1 [WebSocket Channel](#websocket-channel)
   - 4.2 [Message Format](#message-format)
   - 4.3 [Message Types](#message-types)
5. [Deployment Topology](#deployment-topology)
   - 5.1 [Single Proxy, Single Hive](#single-proxy-single-hive)
   - 5.2 [Multiple Proxies, Multiple Hives (Swarm)](#multiple-proxies-multiple-hives-swarm)
6. [Security Model](#security-model)
   - 6.1 [Transport Security](#transport-security)
   - 6.2 [Authentication & Authorization](#authentication-authorization)
   - 6.3 [Data-at-Rest Encryption](#data-at-rest-encryption)
7. [Operational Modes](#operational-modes)
   - 7.1 [Red Mode](#red-mode)
   - 7.2 [Black Mode](#black-mode)
8. [Scalability & Resilience](#scalability-resilience)
9. [Future Evolution (Black Swarm)](#future-evolution-black-swarm)
10. [Glossary](#glossary)

---

## 1. System Overview

Hive 2.0 is a **decentralised command-and-control (C2) framework** designed for **Red Team operations**. It consists of two primary node types:

- **Bee (Agent):** A lightweight implant deployed on target systems. Bees execute commands, run SOCKS5 proxies, and can self-destruct on command.
- **Hive (Server):** The operator’s command centre. It manages connected bees via a WebSocket-based protocol and serves a web dashboard for situational awareness.

Communication between bees and hives is routed through **fronting Nginx proxies** deployed on inexpensive VPS instances. These proxies mask the real C2 traffic by serving a legitimate‑looking static website alongside the WebSocket endpoint. This architecture provides **traffic masking**, **IP rotation**, and **cheap infrastructure** while maintaining full Red Team capabilities.

The entire system is implemented in **Go** (single binary for both roles), with a React‑based dashboard embedded into the server.

---

## 2. Design Principles

1. **Simplicity & Realisability:** The initial release favours standard libraries and well‑known dependencies over esoteric stealth techniques. This ensures rapid development, easy auditing, and a solid foundation for future hardening.

2. **Mode‑Agnostic Binary:** The same `hive` executable can run as an agent (`-mode=agent`) or as a server (`-mode=server`). Configuration is loaded from an encrypted JSON file.

3. **Traffic Masking:** Agent‑server WebSocket connections are proxied through Nginx nodes that also serve a static cover site. To an external observer, the TLS‑encrypted traffic to `proxy1.example.com` is indistinguishable from a normal website visit.

4. **Defence in Depth:** Encryption is applied at multiple layers – TLS 1.3 for transport, AES‑256‑GCM for configuration files, and optional mutual TLS for agent authentication.

5. **Resilience:** Agents automatically reconnect with exponential backoff. Proxies can be rotated or destroyed without affecting the swarm’s core logic.

6. **Operator Empowerment:** A web dashboard provides real‑time agent listing, command sending, and result viewing. Reports are generated in Markdown for integration with existing documentation workflows.

---

## 3. Component Architecture

### 3.1 Agent (Bee)

**Responsibility:** Execute operator commands, forward SOCKS5 traffic, and maintain a persistent WebSocket connection to the hive (through one or more proxies).

**Key Modules:**

| Module | File | Description |
|--------|------|-------------|
| WebSocket Client | `internal/agent/agent.go` | Manages connection lifecycle, reconnection with exponential backoff, and message dispatching. |
| Command Executor | `internal/agent/executor.go` | Runs shell commands via `os/exec` with timeout and context support. |
| SOCKS5 Proxy | `internal/agent/proxy.go` | Starts/stops a SOCKS5 server using `go-socks5`; allows the operator to pivot through the agent. |
| Self-Destruct | `internal/agent/selfdestruct.go` | Removes the agent binary and terminates the process. On Windows, uses `MoveFileEx` with delayed deletion. |
| System Info | `internal/utils/sysinfo.go` | Gathers hostname, OS, and internal IP for registration. |

**Startup Sequence:**

1. Load and decrypt configuration file (AES‑256‑GCM, key from environment).
2. Read server URL from configuration (first proxy domain).
3. Optionally load client TLS certificate for mutual authentication.
4. Enter connection loop: dial WebSocket (`wss://<proxy>/ws`), send `RegisterMsg`, wait for tasks.
5. On `TaskMsg`, fork a goroutine to execute the command and return `ResultMsg`.
6. On `DestroyMsg`, trigger `SelfDestruct`.

### 3.2 Command Centre (Hive)

**Responsibility:** Accept WebSocket connections from agents (via proxy), provide a dashboard for the operator, dispatch commands, and collect results.

**Key Modules:**

| Module | File | Description |
|--------|------|-------------|
| Agent Manager | `internal/server/manager.go` | Thread‑safe storage of connected agents using `sync.Map`. |
| WebSocket Handler | `internal/server/handler.go` | Upgrades HTTP to WebSocket, handles registration, read/write pumps, and task queuing. |
| Report Generator | `internal/server/reporting.go` | Builds a Markdown report containing agent inventory and task/result logs. |
| HTTP Server | `internal/server/server.go` | Serves the React dashboard and the `/ws` endpoint. Supports TLS. |

**Startup Sequence:**

1. Load configuration similarly to the agent.
2. Create `AgentManager` and `AgentHandler`.
3. Start HTTP server on configured port (default `:8443`).
4. Serve embedded React app from `/`; WebSocket upgrade at `/ws`.
5. When an agent connects, read `RegisterMsg`, add to manager, create a task queue, and start the writer goroutine.
6. The operator can send tasks via the dashboard (or API), which are enqueued to the respective agent’s channel and delivered asynchronously.

### 3.3 Fronting Proxy

**Responsibility:** Mask the true C2 server behind a legitimate‑looking HTTPS website, relaying WebSocket traffic to the backend hive.

**Implementation:** Nginx with a cover site (static HTML) and a reverse proxy configuration for `/ws`.

**Configuration Highlights:**

```nginx
server {
    listen 443 ssl http2;
    server_name proxy1.example.com;

    ssl_certificate     /etc/nginx/certs/proxy.crt;
    ssl_certificate_key /etc/nginx/certs/proxy.key;

    root /var/www/hive-cover;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    location /ws {
        proxy_pass https://<hive-backend>:8443;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        # ... additional headers
        proxy_read_timeout 86400s;
    }
}

The proxy’s TLS certificate can be a self‑signed or Let’s Encrypt certificate. Since the agent uses InsecureSkipVerify by default, self‑signed certificates are acceptable for initial deployments.

Deployment Automation: An Ansible playbook (deploy_proxy.yml) installs Nginx, generates a certificate, deploys the cover site, and enables the configuration. Helper scripts (rotate_certs.sh, destroy_proxy.sh) manage certificate rotation and proxy tear‑down.
4. Communication Protocol
4.1 WebSocket Channel

All agent‑hive communication is conducted over a single full‑duplex WebSocket connection. The connection is initiated by the agent to wss://<proxy>/ws. The proxy forwards the upgrade to the backend hive. Once established, the connection remains open for the lifetime of the agent.
4.2 Message Format

Every message is a JSON object with the following structure:
json

{
  "type": "<message type>",
  "payload": { ... }
}

The payload field is a JSON‑encoded object whose schema depends on the type.
4.3 Message Types
Type	Direction	Payload (Go struct)	Purpose
register	Agent → Server	RegisterMsg	Initial registration after connection. Contains agent identity and system info.
task	Server → Agent	TaskMsg	A command to execute. Includes a unique task_id, the command string, and an optional timeout.
result	Agent → Server	ResultMsg	The result of a previously received task. Contains task_id, stdout, stderr, and an error field if execution failed.
destroy	Server → Agent	(none)	Instructs the agent to self‑destruct immediately.
proxy_on	Server → Agent	(none)	Starts the SOCKS5 proxy on the agent.
proxy_off	Server → Agent	(none)	Stops the SOCKS5 proxy.

Note: The protocol is intentionally simple. Future versions may add heartbeat messages, streaming results, and chunked file transfers.
5. Deployment Topology
5.1 Single Proxy, Single Hive

The simplest production‑ready setup:
text

[Operator Browser] ----> [Hive Server] <---(WebSocket via proxy)--- [Agent]
                              |
                              +-- serves dashboard

In this configuration, the agent connects to proxy1.example.com, which forwards to the hive on the internal network. The operator accesses the hive dashboard directly (or through a separate secure channel).
5.2 Multiple Proxies, Multiple Hives (Swarm)

For larger operations, multiple fronting proxies can be deployed, each pointing to the same hive or to different hives. Agents can be pre‑configured with a list of proxies and will try them in order. This allows:

    Load Distribution: Different agents connect to different proxies.

    Geographic Diversity: Proxies in multiple countries.

    Failover: If one proxy is taken down, agents reconnect through the next in the list.

In a true swarm, hives themselves may communicate through a gossip protocol (planned for Black Swarm).
6. Security Model
6.1 Transport Security

    TLS 1.3 is mandatory for all WebSocket connections. The proxy terminates TLS and forwards to the backend over an internal network (ideally also encrypted).

    Agent certificates are optional but recommended for mutual TLS. When provided, the proxy/hive can verify the agent’s identity cryptographically.

    Perfect Forward Secrecy is provided by TLS 1.3’s ephemeral key exchange. Session resumption is disabled to prevent long‑term key compromise.

6.2 Authentication & Authorization

    Agent Authentication: At the application layer, the agent sends a pre‑configured AgentID during registration. The server can maintain a whitelist of allowed IDs. Future enhancements will include HMAC‑signed registration tokens.

    Operator Authentication: The dashboard currently has no built‑in authentication; it is expected to be protected by network controls (VPN, firewall). Production deployments should place the hive behind a reverse proxy with OAuth or mutual TLS.

6.3 Data-at-Rest Encryption

    Configuration files are encrypted with AES‑256‑GCM using a 32‑byte key provided via the HIVE_CONFIG_KEY environment variable. Without this key, the JSON configuration is unreadable.

    The encryption nonce is random and prepended to the ciphertext, ensuring that identical plaintexts produce different ciphertexts.

7. Operational Modes

The system supports two distinct modes of operation, selectable via configuration or task metadata (future).
7.1 Red Mode

    Description: Overt penetration testing with active scanning, exploitation, and detailed reporting.

    Characteristics: Agent traffic may be slightly more aggressive; full command execution and SOCKS5 proxy are enabled. Results are logged and compiled into a comprehensive Markdown report.

    Use Case: Authorised penetration tests where the customer expects noise and a formal deliverable.

7.2 Black Mode

    Description: Covert operations with maximum stealth. (Full implementation targeted for Black Swarm release.)

    Characteristics: Passive reconnaissance only; no active scanning. Communication is limited to low‑entropy channels (e.g., DNS tunnelling, TLS steganography). Agents avoid disk writes and remain dormant for long periods.

    Use Case: Long‑term persistent access, espionage simulations, or operations where detection must be avoided at all costs.

8. Scalability & Resilience

    Agent Reconnection: Exponential backoff from 2s to 60s ensures that brief proxy or hive outages do not lose agents.

    Concurrent Connections: The server uses Go’s lightweight goroutines and sync.Map to handle hundreds of simultaneous agents.

    Proxy Rotation: Agents can be configured with multiple proxy addresses. A script (rotate_certs.sh) can renew TLS certificates without downtime.

    Stateless Hive (Planned): The current hive stores agent state in memory. For swarm resilience, a future version will allow agents to re‑register with any hive node, with state synchronised via a distributed store or gossip protocol.

9. Future Evolution (Black Swarm)

Hive 2.0 is the foundation for the Black Swarm concept. Key enhancements on the roadmap include:

    L0 Assembly Loader: Tiny position‑independent shellcode for initial compromise.

    L1 Core in C: Native agent with direct syscalls, polymorphic code, and memory encryption.

    Gossip Protocol: Decentralised command dissemination without a single C2 server.

    On‑board LLM: Quantised Mistral 7B running in‑memory for automated phishing, document analysis, and tactical decision‑making.

    Multi‑protocol Tunnels: DNS, ICMP, and TLS steganography with adaptive traffic shaping.

    Self‑Learning Swarm: Reinforcement learning across nodes to evade EDR detection over time.

The architecture described in this document is designed to be extended without fundamental redesign.
10. Glossary
Term	Definition
Agent (Bee)	A lightweight implant deployed on a target machine.
Hive	The operator’s command centre; a server managing one or more agents.
Fronting Proxy	An Nginx server that masks C2 traffic behind a benign website.
WebSocket	Full‑duplex communication protocol over TCP, used for agent‑hive signalling.
SOCKS5	A proxy protocol that allows tunnelling of arbitrary TCP/UDP traffic.
TLS	Transport Layer Security, used to encrypt all network communications.
AES‑256‑GCM	Symmetric encryption algorithm providing confidentiality and integrity.
Red Mode	Active penetration testing with full tooling and reporting.
Black Mode	Covert, stealth‑oriented operations with minimal footprint.

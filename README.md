# Red-team-swarm
# Hive 2.0 — Red Swarm

Hive 2.0 is a **decentralised, self‑organising command and control framework** for Red Team operations.  
It provides a flexible platform to manage multiple agents (bees) via a lightweight WebSocket protocol,  
masked behind cheap fronting proxies serving a benign cover website.

**Status:** Production‑ready foundation. Advanced stealth features (Black Swarm) are under development.

## Key Features

- **Single binary for both roles:** Run as agent (`-mode=agent`) or server (`-mode=server`).
- **WebSocket C2 channel** with automatic reconnection and exponential backoff.
- **Fronting Nginx proxies** that hide real C2 traffic behind a static site.
- **Encrypted configuration** (AES‑256‑GCM) — secrets never touch disk in plaintext.
- **Embedded React dashboard** for real‑time agent monitoring and tasking.
- **SOCKS5 proxy on agent** for internal network pivoting.
- **Self‑destruct** command that removes the agent from the target.
- **Cross‑platform agent** (Linux, Windows).
- **Operational report generation** in Markdown.

## Architecture Overview

[Operator Dashboard] [Fronting Proxy] [Hive Server]
| | |
+------- HTTPS --------+------ WebSocket -----+
|
[Agent (Bee)]
text


- **Agent** connects to `wss://proxy.example.com/ws`, which is forwarded to the **Hive**.
- The **Proxy** serves a static website on port 443, making the traffic look legitimate.
- The **Hive** manages all agents and provides a web UI for the operator.

Full architecture description: [docs/architecture.md](docs/architecture.md)

## Quick Start

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- OpenSSL (for key generation)
- Ansible 2.14+ (for proxy deployment)
- A VPS for the fronting proxy
- A domain/subdomain pointing to the proxy

### 1. Build from Source

```bash
git clone https://github.com/blackswarm/hive.git
cd hive
make build

Binaries are placed in bin/:

    bin/hive-server — the command centre.

    bin/hive-agent — the implant.

2. Generate Configuration

Create a 32‑byte encryption key:
bash

export HIVE_CONFIG_KEY=$(openssl rand -base64 32)

Prepare plain JSON configs (see docs/operations.md for detailed steps), then encrypt them with the provided tooling (not yet included — manual steps in docs).
3. Deploy a Fronting Proxy

Use the Ansible playbook on a fresh Debian VPS:
bash

ansible-playbook -i "proxy.example.com," scripts/deploy_proxy.yml

This installs Nginx, generates a TLS certificate, and configures traffic forwarding.
4. Start the Hive Server
bash

./bin/hive-server -mode=server -config=./server_config.json

If server.crt and server.key exist in the working directory, the server will use TLS 1.3 automatically.
5. Start an Agent (via Proxy)
bash

./bin/hive-agent -mode=agent -config=./agent_config.json

The agent will connect to the proxy and register with the hive. You should see a log entry on the server.
6. Send Commands

From the operator machine, you can send a JSON task over the WebSocket (e.g., using websocat):
json

{"type":"task","payload":{"task_id":"1","command":"whoami","timeout":10}}

The agent will reply with a result message.
7. Generate Report

Use the built‑in reporting function to create a Markdown summary of agents, tasks, and results.
Documentation

    Architecture Document — deep dive into design, protocols, and security.

    Operations Manual — step‑by‑step instructions for deployment and usage.

Project Structure
text

hive/
├── cmd/hive/main.go           # Entry point
├── internal/
│   ├── agent/                 # Agent logic (executor, proxy, selfdestruct, agent loop)
│   ├── server/                # Hive server (manager, handler, reporting, server)
│   ├── config/                # Configuration loading/saving (encrypted)
│   ├── crypto/                # AES‑256‑GCM, TLS certificate helpers
│   └── protocol/              # WebSocket message definitions
├── configs/                   # Example configuration templates
├── scripts/                   # Deployment and maintenance scripts
├── web/                       # React dashboard source
├── Makefile                   # Build automation
├── Dockerfile.agent           # Containerised agent
├── Dockerfile.server          # Containerised server
├── docs/                      # Documentation
└── README.md

Operational Modes
Red Mode (active)

    Full command execution, SOCKS5 proxy, detailed reporting.

    Designed for authorised penetration tests.

Black Mode (covert — future)

    Passive reconnaissance only, minimal footprint, advanced traffic obfuscation.

    Coming in Black Swarm release.

Security

    TLS 1.3 for all network connections.

    AES‑256‑GCM encrypts configuration files; key never stored on disk.

    Optional mutual TLS for agent authentication.

    Fronting proxy masks C2 traffic as normal web browsing.

Full security model: architecture.md#security-model
Roadmap (Black Swarm)

Hive 2.0 is the stepping stone toward the Black Swarm concept:

    L0/L1 native implants (assembly + C, no‑disk footprint).

    Gossip protocol for decentralised command propagation.

    On‑board LLM (Mistral 7B) for automated phishing and decision‑making.

    Multi‑protocol covert channels (DNS, ICMP, TLS steganography).

    Polymorphic code generation to evade signature‑based detection.

Contributing

This project is currently developed by a small team. External contributions are welcome but must follow the strict code quality and security standards outlined in the architecture document.
License

Proprietary. All rights reserved.
Contact

For questions or engagement inquiries, reach out to the project maintainers.
https://github.com/sergmudrea/Red-team-swarm

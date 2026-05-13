package agent

import (
	"errors"
	"fmt"
	"net"
	"sync"

	socks5 "github.com/armon/go-socks5"
)

// proxyServer wraps a SOCKS5 server and its listener.
type proxyServer struct {
	mu       sync.Mutex
	server   *socks5.Server
	listener net.Listener
	running  bool
}

var currentProxy = &proxyServer{}

// StartSOCKS5 starts a SOCKS5 proxy on the given port.
// It is safe to call concurrently; if a proxy is already running, it returns an error.
func StartSOCKS5(port int) error {
	currentProxy.mu.Lock()
	defer currentProxy.mu.Unlock()

	if currentProxy.running {
		return errors.New("agent: SOCKS5 proxy is already running")
	}

	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	currentProxy.server = server
	currentProxy.listener = listener
	currentProxy.running = true

	go server.Serve(listener)
	return nil
}

// StopSOCKS5 stops the currently running SOCKS5 proxy, if any.
func StopSOCKS5() error {
	currentProxy.mu.Lock()
	defer currentProxy.mu.Unlock()

	if !currentProxy.running {
		return nil
	}

	err := currentProxy.listener.Close()
	currentProxy.running = false
	currentProxy.server = nil
	currentProxy.listener = nil
	return err
}

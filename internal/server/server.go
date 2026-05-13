package server

import (
	"context"
	"crypto/tls"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"
)

//go:embed web/build
var webAssets embed.FS

// Server is the main hive server that handles WebSocket agents and serves the operator UI.
type Server struct {
	addr    string
	handler *AgentHandler
	logger  *slog.Logger
	httpSrv *http.Server
}

// NewServer creates a new Server instance.
func NewServer(addr string, handler *AgentHandler, logger *slog.Logger) *Server {
	return &Server{
		addr:    addr,
		handler: handler,
		logger:  logger,
	}
}

// Start begins listening for TLS connections. If server.crt and server.key exist,
// TLS 1.3 is used; otherwise the server falls back to plain HTTP (not recommended for production).
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handler.ServeHTTP)

	// Try to serve embedded operator UI; if missing, show a simple status page.
	staticFS, err := fs.Sub(webAssets, "web/build")
	if err != nil {
		s.logger.Warn("embedded web assets not found, serving fallback page", "error", err)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hive Server Active"))
		})
	} else {
		fileServer := http.FileServer(http.FS(staticFS))
		mux.Handle("/", fileServer)
	}

	s.httpSrv = &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// Load TLS if certificates are available; otherwise warn and use plain HTTP.
	certFile := "server.crt"
	keyFile := "server.key"
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				return err
			}
			s.httpSrv.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS13,
			}
			s.logger.Info("starting TLS server", "addr", s.addr)
			return s.httpSrv.ListenAndServeTLS("", "")
		}
	}
	s.logger.Warn("no TLS credentials found, starting plain HTTP server (not secure)", "addr", s.addr)
	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

package server

import (
	"context"
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/blackswarm/hive/internal/protocol"
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

// Start begins listening for TLS connections.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handler.ServeHTTP)

	// REST API for the React dashboard
	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agents := s.handler.manager.ListAgents()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agents)
	})

	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			AgentID string `json:"agent_id"`
			Command string `json:"command"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.AgentID == "" || req.Command == "" {
			http.Error(w, "agent_id and command are required", http.StatusBadRequest)
			return
		}
		taskID := time.Now().Format("20060102150405.000") // simple unique id
		task := protocol.TaskMsg{
			TaskID:  taskID,
			Command: req.Command,
			Timeout: 0, // use default
		}
		if err := s.handler.SendTask(req.AgentID, task); err != nil {
			http.Error(w, fmt.Sprintf("failed to send task: %v", err), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"task_id": taskID})
	})

	// Serve static React app
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

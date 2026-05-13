package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blackswarm/hive/internal/agent"
	"github.com/blackswarm/hive/internal/config"
	"github.com/blackswarm/hive/internal/crypto"
	"github.com/blackswarm/hive/internal/server"
	"github.com/blackswarm/hive/internal/utils"
)

func main() {
	mode := flag.String("mode", "", "run mode: agent or server")
	configPath := flag.String("config", "", "path to configuration file")
	encryptIn := flag.String("encrypt-in", "", "path to plaintext JSON config to encrypt")
	encryptOut := flag.String("encrypt-out", "", "path to write encrypted config")
	flag.Parse()

	// Encrypt config mode
	if *encryptIn != "" || *encryptOut != "" {
		if *encryptIn == "" || *encryptOut == "" {
			fmt.Fprintf(os.Stderr, "both -encrypt-in and -encrypt-out are required for encryption\n")
			os.Exit(1)
		}
		keyEnv := os.Getenv("HIVE_CONFIG_KEY")
		if keyEnv == "" {
			fmt.Fprintf(os.Stderr, "HIVE_CONFIG_KEY environment variable not set\n")
			os.Exit(1)
		}
		key := []byte(keyEnv)
		if len(key) != 32 {
			fmt.Fprintf(os.Stderr, "HIVE_CONFIG_KEY must be 32 bytes\n")
			os.Exit(1)
		}
		// Load plain config (unencrypted JSON)
		plain, err := os.ReadFile(*encryptIn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read input file: %v\n", err)
			os.Exit(1)
		}
		// Encrypt and write
		ciphertext, err := crypto.Encrypt(plain, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "encryption failed: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*encryptOut, ciphertext, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write encrypted config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Config encrypted successfully.")
		return
	}

	// Normal run modes
	if *mode != "agent" && *mode != "server" {
		fmt.Fprintf(os.Stderr, "usage: hive -mode=agent|server -config=<path> [or use -encrypt-in/-encrypt-out]\n")
		os.Exit(1)
	}
	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "config path is required\n")
		os.Exit(1)
	}

	// Logger
	logger := utils.NewLogger(slog.LevelInfo, os.Stdout)

	// Load encryption key
	keyEnv := os.Getenv("HIVE_CONFIG_KEY")
	if keyEnv == "" {
		logger.Error("HIVE_CONFIG_KEY environment variable not set")
		os.Exit(1)
	}
	key := []byte(keyEnv)
	if len(key) != 32 {
		logger.Error("HIVE_CONFIG_KEY must be 32 bytes")
		os.Exit(1)
	}

	// Load config
	cfg, err := config.LoadConfig(*configPath, key)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("shutting down")
		cancel()
	}()

	switch *mode {
	case "agent":
		runAgent(ctx, cfg, logger)
	case "server":
		runServer(ctx, cfg, logger)
	}
}

func runAgent(ctx context.Context, cfg *config.Config, logger *slog.Logger) {
	if len(cfg.Proxies) == 0 {
		logger.Error("at least one proxy domain must be configured")
		return
	}
	serverURL := fmt.Sprintf("wss://%s/ws", cfg.Proxies[0])

	var clientCert *tls.Certificate
	if certFile := os.Getenv("AGENT_CERT_FILE"); certFile != "" {
		keyFile := os.Getenv("AGENT_KEY_FILE")
		if keyFile == "" {
			logger.Error("AGENT_CERT_FILE set but AGENT_KEY_FILE is missing")
			return
		}
		cert, err := crypto.LoadCert(certFile, keyFile)
		if err != nil {
			logger.Error("failed to load agent certificate", "error", err)
			return
		}
		clientCert = &cert
	}

	a := agent.NewAgent(serverURL, clientCert, cfg.AgentID, logger)
	go a.Run()

	<-ctx.Done()
	logger.Info("agent exiting")
}

func runServer(ctx context.Context, cfg *config.Config, logger *slog.Logger) {
	manager := &server.AgentManager{}
	handler := server.NewAgentHandler(manager, logger)

	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	srv := server.NewServer(addr, handler, logger)

	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("server error", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}
	logger.Info("server stopped")
}

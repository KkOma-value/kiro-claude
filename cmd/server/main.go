package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/yourusername/kiro-claude/internal/api"
	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/logger"
	"github.com/yourusername/kiro-claude/internal/middleware"
	"github.com/yourusername/kiro-claude/internal/model"
	"github.com/yourusername/kiro-claude/internal/runtime"
)

var (
	configPath = flag.String("config", "~/.config/kiro-claude/config.yaml", "Path to config file")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		// Use defaults if config file doesn't exist
		cfg = config.DefaultConfig()
	}

	// Initialize logger
	log := logger.NewSimpleLogger(logger.ParseLevel(cfg.Logging.Level))

	// Print startup banner
	printStartupBanner(cfg, log)

	buildResult, err := runtime.BuildClient(cfg, log)
	if err != nil {
		log.Errorf("Failed to initialize backend: %v", err)
		os.Exit(1)
	}
	defer buildResult.Client.Close()

	// Create API handler
	apiHandler := api.NewHandler(buildResult.Client, log)

	// Create model resolver and admin handler
	resolver := model.NewResolver()
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	adminHandler := api.NewAdminHandler(cfg, resolver, buildResult.TokenMgr, log, addr)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/messages", apiHandler.HandleMessages)
	mux.HandleFunc("GET /v1/models", apiHandler.HandleModels)
	mux.HandleFunc("GET /health", apiHandler.HandleHealth)
	mux.HandleFunc("GET /", apiHandler.HandleHealth)

	// Management API routes (for web frontend)
	mux.HandleFunc("GET /api/status", adminHandler.HandleStatus)
	mux.HandleFunc("POST /api/resolve-model", adminHandler.HandleResolveModel)
	mux.HandleFunc("GET /api/endpoints", adminHandler.HandleEndpoints)

	// Apply middleware layers (outer → inner)
	var handler http.Handler = mux

	// Debug logger middleware (innermost)
	handler = middleware.NewDebugLogger(cfg.Logging.DebugDump, log, handler)

	// Auth guard middleware (outermost — checked first)
	handler = middleware.NewAuthGuard(cfg.Security.ProxyAPIKey, handler)

	// Create HTTP server
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Infof("Shutting down...")
		server.Close()
	}()

	// Start server
	log.Infof("Listening on http://%s", addr)
	log.Infof("Claude Code setup: export ANTHROPIC_BASE_URL=http://%s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Errorf("Server error: %v", err)
		os.Exit(1)
	}

	log.Infof("Gateway stopped")
}

// printStartupBanner logs a structured startup summary.
func printStartupBanner(cfg *config.Config, log logger.Logger) {
	log.Infof("Starting Kiro → Claude Code gateway")
	log.Infof("Runtime mode: %s", cfg.Runtime.Mode)
	log.Infof("Active backend: %s", cfg.Runtime.UpstreamEndpoint)

	// Auth guard status
	if cfg.Security.ProxyAPIKey != "" {
		maskedKey := api.MaskAPIKey(cfg.Security.ProxyAPIKey)
		log.Infof("Auth guard: enabled (key: %s)", maskedKey)
	} else {
		log.Infof("Auth guard: disabled (no proxy_api_key configured)")
	}

	// Debug dump
	if cfg.Logging.DebugDump {
		log.Infof("Debug dump: enabled")
	}
}

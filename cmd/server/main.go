package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourusername/kiro-claude/internal/api"
	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/logger"
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
	log.Infof("Starting Kiro → Claude Code gateway")
	log.Infof("Config: %+v", cfg)

	client, err := runtime.BuildClient(cfg, log)
	if err != nil {
		log.Errorf("Failed to initialize backend: %v", err)
		os.Exit(1)
	}
	defer client.Close()

	// Create API handler
	apiHandler := api.NewHandler(client, log)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/messages", apiHandler.HandleMessages)
	mux.HandleFunc("GET /v1/models", apiHandler.HandleModels)
	mux.HandleFunc("GET /health", apiHandler.HandleHealth)
	mux.HandleFunc("GET /", apiHandler.HandleHealth)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
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

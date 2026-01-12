// Package main is the entry point for the mesnada MCP orchestrator.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sevir/mesnada/internal/config"
	"github.com/sevir/mesnada/internal/orchestrator"
	"github.com/sevir/mesnada/internal/server"
)

var (
	version = "1.0.0"
	commit  = "dev"
)

func main() {
	// Parse flags
	var (
		configPath  = flag.String("config", "", "Path to config file")
		host        = flag.String("host", "", "Server host (default: 127.0.0.1)")
		port        = flag.Int("port", 0, "Server port (default: 8765)")
		storePath   = flag.String("store", "", "Path to task store file")
		logDir      = flag.String("log-dir", "", "Directory for agent logs")
		maxParallel = flag.Int("max-parallel", 0, "Maximum parallel agents")
		showVersion = flag.Bool("version", false, "Show version and exit")
		initConfig  = flag.Bool("init", false, "Initialize default config and exit")
		useStdio    = flag.Bool("stdio", false, "Use stdio transport instead of HTTP")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("mesnada %s (%s)\n", version, commit)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override with flags
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *storePath != "" {
		cfg.Orchestrator.StorePath = *storePath
	}
	if *logDir != "" {
		cfg.Orchestrator.LogDir = *logDir
	}
	if *maxParallel != 0 {
		cfg.Orchestrator.MaxParallel = *maxParallel
	}

	if *initConfig {
		if err := cfg.Save(*configPath); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		fmt.Println("Configuration initialized")
		os.Exit(0)
	}

	// Create orchestrator
	orch, err := orchestrator.New(orchestrator.Config{
		StorePath:        cfg.Orchestrator.StorePath,
		LogDir:           cfg.Orchestrator.LogDir,
		MaxParallel:      cfg.Orchestrator.MaxParallel,
		DefaultMCPConfig: cfg.Orchestrator.DefaultMCPConfig,
		DefaultEngine:    cfg.Orchestrator.DefaultEngine,
	})
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Create server
	srv := server.New(server.Config{
		Addr:         cfg.Address(),
		Orchestrator: orch,
		Version:      version,
		Commit:       commit,
		UseStdio:     *useStdio,
		AppConfig:    cfg,
	})

	// Handle shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		if err := orch.Shutdown(); err != nil {
			log.Printf("Orchestrator shutdown error: %v", err)
		}
	}()

	// Print startup info
	if *useStdio {
		log.Printf("mesnada %s starting in stdio mode", version)
	} else {
		log.Printf("mesnada %s starting", version)
		log.Printf("UI endpoint:  http://%s/ui", cfg.Address())
		log.Printf("MCP endpoint: http://%s/mcp", cfg.Address())
		log.Printf("SSE endpoint: http://%s/mcp/sse", cfg.Address())
		log.Printf("Health check: http://%s/health", cfg.Address())
	}

	// Start server
	if err := srv.Start(); err != nil {
		select {
		case <-ctx.Done():
			// Expected shutdown
		default:
			log.Fatalf("Server error: %v", err)
		}
	}
}

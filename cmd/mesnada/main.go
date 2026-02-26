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

	"github.com/mattn/go-isatty"
	"github.com/sevir/mesnada/internal/config"
	"github.com/sevir/mesnada/internal/orchestrator"
	"github.com/sevir/mesnada/internal/server"
	"github.com/sevir/mesnada/internal/tui"
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
		initConfig  = flag.Bool("init", false, "Initialize config file and exit")
		initPath    = flag.String("init-path", "", "Path where to create the config file (used with -init)")
		useStdio    = flag.Bool("stdio", false, "Use stdio transport instead of HTTP")
		enableACP   = flag.Bool("enable-acp", false, "Enable ACP agent support (overrides config)")
		noTUI       = flag.Bool("no-tui", false, "Disable TUI dashboard (use legacy log output)")
		noWebUI     = flag.Bool("no-webui", false, "Disable Web UI and HTTP endpoints")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("mesnada %s (%s)\n", version, commit)
		os.Exit(0)
	}

	// Handle init config flag
	if *initConfig {
		if err := config.InitConfig(*initPath); err != nil {
			log.Fatalf("Failed to initialize config: %v", err)
		}

		outputPath := *initPath
		if outputPath == "" {
			home, _ := os.UserHomeDir()
			outputPath = home + "/.mesnada/config.yaml"
		}
		fmt.Printf("Configuration initialized at: %s\n", outputPath)
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
	if *enableACP {
		cfg.ACP.Enabled = true
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation error: %v", err)
	}

	// Create orchestrator
	orch, err := orchestrator.New(orchestrator.Config{
		StorePath:        cfg.Orchestrator.StorePath,
		LogDir:           cfg.Orchestrator.LogDir,
		MaxParallel:      cfg.Orchestrator.MaxParallel,
		DefaultMCPConfig: cfg.Orchestrator.DefaultMCPConfig,
		DefaultEngine:    cfg.Orchestrator.DefaultEngine,
		PersonaPath:      cfg.Orchestrator.PersonaPath,
		AppConfig:        cfg,
	})
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	useTUI, startWebUI := decideInterfaceMode(interfaceModeOptions{
		UseStdio:           *useStdio,
		NoTUI:              *noTUI,
		NoWebUI:            *noWebUI,
		TUIEnabled:         cfg.TUI.Enabled,
		WebUIEnabled:       cfg.TUI.WebUI,
		AutoDetectTerminal: cfg.TUI.AutoDetectTerminal,
		IsTerminal:         isTerminalSession(),
	})
	if !*useStdio && !useTUI && !startWebUI {
		log.Fatal("No interface enabled. Use default settings, remove --no-webui, or run with --stdio.")
	}

	var srv *server.Server
	if *useStdio || startWebUI {
		srv = server.New(server.Config{
			Addr:         cfg.Address(),
			Orchestrator: orch,
			Version:      version,
			Commit:       commit,
			UseStdio:     *useStdio,
			AppConfig:    cfg,
		})
	}

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

		if srv != nil {
			if err := srv.Shutdown(shutdownCtx); err != nil {
				log.Printf("Server shutdown error: %v", err)
			}
		}

		if err := orch.Shutdown(); err != nil {
			log.Printf("Orchestrator shutdown error: %v", err)
		}
	}()

	// Print startup info (only in non-TUI mode, stdio logs are always shown)
	if *useStdio {
		log.Printf("mesnada %s starting in stdio mode", version)
	} else if !useTUI {
		log.Printf("mesnada %s starting", version)
		if startWebUI {
			log.Printf("UI endpoint:  http://%s/ui", cfg.Address())
			log.Printf("MCP endpoint: http://%s/mcp", cfg.Address())
			log.Printf("SSE endpoint: http://%s/mcp/sse", cfg.Address())
			log.Printf("Health check: http://%s/health", cfg.Address())
		} else {
			log.Printf("HTTP disabled (--no-webui or tui.webui=false): running TUI-only mode")
		}
	}

	if useTUI {
		if srv != nil {
			// TUI mode: run HTTP server in a background goroutine, then launch TUI.
			serverErr := make(chan error, 1)
			go func() {
				if err := srv.Start(); err != nil {
					select {
					case <-ctx.Done():
						// Expected shutdown — ignore.
					default:
						serverErr <- err
					}
				}
			}()

			// Small delay to let the server bind before TUI starts.
			time.Sleep(50 * time.Millisecond)

			// Check for immediate server startup failure.
			select {
			case err := <-serverErr:
				log.Fatalf("Server error: %v", err)
			default:
			}
		}

		if err := tui.Run(orch, cfg); err != nil {
			log.Fatalf("TUI error: %v", err)
		}
	} else {
		// Legacy mode: blocking server start (original behaviour).
		if srv == nil {
			log.Fatal("Legacy mode requires HTTP transport; remove --no-webui or use TUI/default mode")
		}
		if err := srv.Start(); err != nil {
			select {
			case <-ctx.Done():
				// Expected shutdown
			default:
				log.Fatalf("Server error: %v", err)
			}
		}
	}
}

type interfaceModeOptions struct {
	UseStdio           bool
	NoTUI              bool
	NoWebUI            bool
	TUIEnabled         bool
	WebUIEnabled       bool
	AutoDetectTerminal bool
	IsTerminal         bool
}

func decideInterfaceMode(opts interfaceModeOptions) (useTUI bool, startWebUI bool) {
	if opts.UseStdio {
		return false, false
	}

	useTUI = opts.TUIEnabled && !opts.NoTUI
	if opts.AutoDetectTerminal && !opts.IsTerminal {
		useTUI = false
	}

	startWebUI = opts.WebUIEnabled && !opts.NoWebUI
	return useTUI, startWebUI
}

func isTerminalSession() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) ||
		isatty.IsTerminal(os.Stdout.Fd()) ||
		isatty.IsTerminal(os.Stderr.Fd()) ||
		isatty.IsCygwinTerminal(os.Stdin.Fd()) ||
		isatty.IsCygwinTerminal(os.Stdout.Fd()) ||
		isatty.IsCygwinTerminal(os.Stderr.Fd())
}

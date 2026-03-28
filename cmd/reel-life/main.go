package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/patflynn/reel-life/internal/agent"
	"github.com/patflynn/reel-life/internal/chat"
	"github.com/patflynn/reel-life/internal/config"
	"github.com/patflynn/reel-life/internal/monitor"
	"github.com/patflynn/reel-life/internal/sonarr"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: cfg.LogLevel(),
	}))
	if cfg.Log.Format == "json" {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: cfg.LogLevel(),
		}))
	}

	sonarrClient := sonarr.NewClient(cfg.Sonarr.BaseURL, cfg.Sonarr.APIKey)
	notifier := chat.NewGoogleChat(cfg.Chat.WebhookURL, logger)

	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		fmt.Fprintf(os.Stderr, "error: ANTHROPIC_API_KEY environment variable is required\n")
		os.Exit(1)
	}

	// Build rate limiter from config (or defaults).
	var limiter *agent.RateLimiter
	rl := cfg.Agent.RateLimits
	maxPerMin, maxPerReq, maxDestructive := 30, 10, 5
	if rl != nil {
		if rl.MaxCallsPerMinute > 0 {
			maxPerMin = rl.MaxCallsPerMinute
		}
		if rl.MaxCallsPerRequest > 0 {
			maxPerReq = rl.MaxCallsPerRequest
		}
		if rl.MaxDestructive > 0 {
			maxDestructive = rl.MaxDestructive
		}
	}
	limiter = agent.NewRateLimiter(maxPerMin, maxPerReq, maxDestructive)

	agentInstance := agent.New(anthropicKey, sonarrClient, cfg.Agent.Model, cfg.Agent.MaxTokens, logger, limiter)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start monitor loop
	if cfg.Monitor.Enabled {
		mon := monitor.New(sonarrClient, notifier, cfg.Monitor.Interval, logger)
		go func() {
			if err := mon.Run(ctx); err != nil && ctx.Err() == nil {
				logger.Error("monitor error", "error", err)
			}
		}()
		logger.Info("monitor started", "interval", cfg.Monitor.Interval)
	}

	// Health endpoint for container probes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	// Webhook endpoint for incoming chat messages
	mux.HandleFunc("POST /webhook", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Parse Google Chat event payload and extract message text.
		// For now, this is a placeholder for the bidirectional chat flow.
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		logger.Info("shutting down HTTP server")
		server.Close()
	}()

	logger.Info("reel-life started", "addr", server.Addr)

	// Reference agentInstance to show it's wired up (used by webhook handler in future).
	_ = agentInstance

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("HTTP server error", "error", err)
		os.Exit(1)
	}
}

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
	"github.com/patflynn/reel-life/internal/radarr"
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

	var radarrClient radarr.Client
	if cfg.Radarr.BaseURL != "" {
		radarrClient = radarr.NewClient(cfg.Radarr.BaseURL, cfg.Radarr.APIKey)
		logger.Info("radarr client configured", "url", cfg.Radarr.BaseURL)
	}

	// Select notifier based on backend configuration.
	var notifier chat.Notifier
	var telegramBot *chat.Telegram
	switch cfg.Chat.Backend {
	case "telegram":
		tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
		if tgToken == "" {
			fmt.Fprintf(os.Stderr, "error: TELEGRAM_BOT_TOKEN environment variable is required for telegram backend\n")
			os.Exit(1)
		}
		tg, err := chat.NewTelegram(tgToken, cfg.Chat.TelegramChatID, cfg.Chat.TelegramAllowedUsers, logger)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating Telegram client: %v\n", err)
			os.Exit(1)
		}
		notifier = tg
		telegramBot = tg
		logger.Info("using Telegram notifier")
	case "googlechat", "":
		// Google Chat: Chat API (app mode) or webhook (legacy).
		if cfg.UseAppMode() {
			saKey, err := os.ReadFile(cfg.Chat.ServiceAccountFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading service account file: %v\n", err)
				os.Exit(1)
			}
			notifier, err = chat.NewGoogleChatApp(saKey, cfg.Chat.Space, logger)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error creating Chat API client: %v\n", err)
				os.Exit(1)
			}
			logger.Info("using Google Chat App (API) notifier", "space", cfg.Chat.Space)
		} else {
			notifier = chat.NewGoogleChat(cfg.Chat.WebhookURL, logger)
			logger.Info("using Google Chat webhook notifier")
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unsupported chat backend: %q\n", cfg.Chat.Backend)
		os.Exit(1)
	}

	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		fmt.Fprintf(os.Stderr, "error: ANTHROPIC_API_KEY environment variable is required\n")
		os.Exit(1)
	}

	// Build rate limiter from config (or defaults).
	var limiter *agent.RateLimiter
	rl := cfg.Agent.RateLimits
	maxPerMin, maxPerReq, maxDestructive := agent.DefaultMaxCallsPerMinute, agent.DefaultMaxCallsPerRequest, agent.DefaultMaxDestructive
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

	agentInstance := agent.New(anthropicKey, sonarrClient, radarrClient, cfg.Agent.Model, cfg.Agent.MaxTokens, logger, limiter)

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

	// Start Telegram listener if using Telegram backend.
	if telegramBot != nil {
		go func() {
			if err := telegramBot.Listen(ctx, agentInstance); err != nil && ctx.Err() == nil {
				logger.Error("telegram listener error", "error", err)
			}
		}()
		logger.Info("telegram listener started")
	}

	// Health endpoint for container probes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	// Webhook endpoint for incoming Google Chat events
	webhookHandler := chat.NewWebhookHandler(agentInstance, cfg.Chat.ProjectNumber, logger)
	mux.Handle("POST /webhook", webhookHandler)

	server := &http.Server{
		Addr:    cfg.ListenAddr(),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		logger.Info("shutting down HTTP server")
		server.Close()
	}()

	logger.Info("reel-life started", "addr", server.Addr)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("HTTP server error", "error", err)
		os.Exit(1)
	}
}

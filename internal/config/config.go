package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sonarr  SonarrConfig  `yaml:"sonarr"`
	Chat    ChatConfig    `yaml:"chat"`
	Agent   AgentConfig   `yaml:"agent"`
	Monitor MonitorConfig `yaml:"monitor"`
	Log     LogConfig     `yaml:"log"`
	Server  ServerConfig  `yaml:"server"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type SonarrConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

type ChatConfig struct {
	Backend            string  `yaml:"backend"`
	WebhookURL         string  `yaml:"webhook_url"`
	ServiceAccountFile string  `yaml:"service_account_file"`
	Space              string  `yaml:"space"`
	ProjectNumber      string  `yaml:"project_number"`
	TelegramChatID     int64   `yaml:"telegram_chat_id"`
	TelegramAllowedUsers []int64 `yaml:"telegram_allowed_users"`
}

type AgentConfig struct {
	Model      string           `yaml:"model"`
	MaxTokens  int              `yaml:"max_tokens"`
	RateLimits *RateLimitConfig `yaml:"rate_limits,omitempty"`
}

// RateLimitConfig controls how many tool calls the agent can make.
type RateLimitConfig struct {
	MaxCallsPerMinute  int `yaml:"max_calls_per_minute"`
	MaxCallsPerRequest int `yaml:"max_calls_per_request"`
	MaxDestructive     int `yaml:"max_destructive"`
}

type MonitorConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Agent: AgentConfig{
			Model:     "claude-sonnet-4-5-20250929",
			MaxTokens: 4096,
		},
		Monitor: MonitorConfig{
			Enabled:  true,
			Interval: 5 * time.Minute,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Chat: ChatConfig{
			Backend: "googlechat",
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	applyEnvOverrides(cfg)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SONARR_API_KEY"); v != "" {
		cfg.Sonarr.APIKey = v
	}
	if v := os.Getenv("SONARR_URL"); v != "" {
		cfg.Sonarr.BaseURL = v
	}
	if v := os.Getenv("GOOGLE_CHAT_WEBHOOK_URL"); v != "" {
		cfg.Chat.WebhookURL = v
	}
	if v := os.Getenv("GOOGLE_SERVICE_ACCOUNT_FILE"); v != "" {
		cfg.Chat.ServiceAccountFile = v
	}
}

// UseAppMode returns true when the Chat API (service account) notifier should be used
// instead of the legacy incoming webhook.
func (cfg *Config) UseAppMode() bool {
	return cfg.Chat.ServiceAccountFile != "" && cfg.Chat.Space != ""
}

func validate(cfg *Config) error {
	if cfg.Sonarr.BaseURL == "" {
		return fmt.Errorf("sonarr.base_url is required")
	}
	if cfg.Sonarr.APIKey == "" {
		return fmt.Errorf("sonarr.api_key is required (set SONARR_API_KEY env var)")
	}
	// Telegram backend only needs the bot token (checked at startup via env var).
	if cfg.Chat.Backend == "telegram" {
		return nil
	}
	// Either webhook URL or service account + space must be configured.
	if cfg.Chat.WebhookURL == "" && (cfg.Chat.ServiceAccountFile == "" || cfg.Chat.Space == "") {
		return fmt.Errorf("chat.webhook_url or chat.service_account_file + chat.space is required")
	}
	return nil
}

func (c *Config) ListenAddr() string {
	port := c.Server.Port
	if port == 0 {
		port = 9090
	}
	return fmt.Sprintf(":%d", port)
}

func (cfg *Config) LogLevel() slog.Level {
	switch cfg.Log.Level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

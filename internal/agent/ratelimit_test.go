package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/patflynn/reel-life/internal/sonarr"
)

func TestRateLimiterAllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(5, 3, 2)
	if err := rl.Allow("search_series", false); err != nil {
		t.Fatalf("expected allow, got %v", err)
	}
	if err := rl.Allow("add_series", true); err != nil {
		t.Fatalf("expected allow, got %v", err)
	}
}

func TestRateLimiterBlocksPerRequest(t *testing.T) {
	rl := NewRateLimiter(100, 2, 100)
	rl.Allow("search_series", false)
	rl.Allow("get_queue", false)
	err := rl.Allow("check_health", false)
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if !strings.Contains(err.Error(), "per request") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRateLimiterBlocksPerMinute(t *testing.T) {
	rl := NewRateLimiter(2, 100, 100)
	rl.Allow("search_series", false)
	rl.Allow("get_queue", false)
	err := rl.Allow("check_health", false)
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if !strings.Contains(err.Error(), "per minute") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRateLimiterBlocksDestructive(t *testing.T) {
	rl := NewRateLimiter(100, 100, 1)
	rl.Allow("add_series", true)
	err := rl.Allow("remove_failed", true)
	if err == nil {
		t.Fatal("expected rate limit error for destructive action")
	}
	if !strings.Contains(err.Error(), "destructive") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRateLimiterReset(t *testing.T) {
	rl := NewRateLimiter(100, 1, 1)
	rl.Allow("search_series", false)
	if err := rl.Allow("search_series", false); err == nil {
		t.Fatal("expected rate limit before reset")
	}
	rl.Reset()
	if err := rl.Allow("search_series", false); err != nil {
		t.Fatalf("expected allow after reset, got %v", err)
	}
}

func TestIsDestructive(t *testing.T) {
	if !IsDestructive("add_series") {
		t.Error("add_series should be destructive")
	}
	if !IsDestructive("remove_failed") {
		t.Error("remove_failed should be destructive")
	}
	if IsDestructive("search_series") {
		t.Error("search_series should not be destructive")
	}
	if IsDestructive("get_queue") {
		t.Error("get_queue should not be destructive")
	}
	if IsDestructive("check_health") {
		t.Error("check_health should not be destructive")
	}
	if IsDestructive("get_history") {
		t.Error("get_history should not be destructive")
	}
}

func TestRateLimiterDenialReturnedAsToolError(t *testing.T) {
	rl := NewRateLimiter(100, 100, 0) // zero destructive allowed
	mock := &mockSonarr{}
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	a := &Agent{sonarr: mock, logger: logger, limiter: rl}

	input, _ := json.Marshal(addSeriesInput{
		Title: "Test", TVDBID: 1, QualityProfileID: 1, RootFolderPath: "/tv",
	})
	result, isErr := a.executeToolWithAudit(context.Background(), "add_series", input, 0, "req-1")
	if !isErr {
		t.Fatal("expected error from rate-limited tool")
	}
	if !strings.Contains(result, "destructive") {
		t.Errorf("expected destructive limit message, got %s", result)
	}
}

func TestAuditLogging(t *testing.T) {
	mock := &mockSonarr{
		searchResult: []sonarr.Series{{ID: 1, Title: "Test", TVDBID: 123}},
	}
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	a := &Agent{sonarr: mock, logger: logger}

	input, _ := json.Marshal(searchSeriesInput{Term: "test"})
	a.executeToolWithAudit(context.Background(), "search_series", input, 0, "req-42")

	logs := buf.String()
	if !strings.Contains(logs, "tool invocation") {
		t.Error("missing 'tool invocation' log entry")
	}
	if !strings.Contains(logs, "tool result") {
		t.Error("missing 'tool result' log entry")
	}
	if !strings.Contains(logs, "req-42") {
		t.Error("missing request_id in log")
	}
	if !strings.Contains(logs, "search_series") {
		t.Error("missing tool name in log")
	}
}

func TestSanitizeInput(t *testing.T) {
	raw := json.RawMessage(`{"term":"test","api_key":"secret123"}`)
	result := sanitizeInput(raw)
	if strings.Contains(result, "secret123") {
		t.Error("api_key was not redacted")
	}
	if !strings.Contains(result, "REDACTED") {
		t.Error("expected REDACTED placeholder")
	}
	if !strings.Contains(result, "test") {
		t.Error("non-secret field should be preserved")
	}
}

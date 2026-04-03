package chat

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestSplitMessageShort(t *testing.T) {
	parts := splitMessage("hello", 4096)
	if len(parts) != 1 || parts[0] != "hello" {
		t.Errorf("expected single part %q, got %v", "hello", parts)
	}
}

func TestSplitMessageExactBoundary(t *testing.T) {
	msg := strings.Repeat("a", 4096)
	parts := splitMessage(msg, 4096)
	if len(parts) != 1 || parts[0] != msg {
		t.Errorf("expected single part of len 4096, got %d parts", len(parts))
	}
}

func TestSplitMessageNewlineSplit(t *testing.T) {
	// Build a message that exceeds maxLen, with a newline near the boundary.
	line1 := strings.Repeat("a", 3000) + "\n"
	line2 := strings.Repeat("b", 2000)
	msg := line1 + line2

	parts := splitMessage(msg, 4096)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if parts[0] != line1 {
		t.Errorf("part[0] len = %d, want %d (should split at newline)", len(parts[0]), len(line1))
	}
	if parts[1] != line2 {
		t.Errorf("part[1] = %q, want line2", parts[1])
	}
}

func TestSplitMessageNoNewlineFallback(t *testing.T) {
	// No newlines at all — must hard-split at maxLen.
	msg := strings.Repeat("x", 5000)
	parts := splitMessage(msg, 4096)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if len(parts[0]) != 4096 {
		t.Errorf("part[0] len = %d, want 4096", len(parts[0]))
	}
	if len(parts[1]) != 904 {
		t.Errorf("part[1] len = %d, want 904", len(parts[1]))
	}
}

func TestSplitMessageEmpty(t *testing.T) {
	parts := splitMessage("", 4096)
	if parts != nil {
		t.Errorf("expected nil for empty string, got %v", parts)
	}
}

func TestIsAllowedUserPasses(t *testing.T) {
	if !isAllowed(12345, []int64{12345, 67890}) {
		t.Error("expected user 12345 to be allowed")
	}
}

func TestIsAllowedUserRejected(t *testing.T) {
	if isAllowed(99999, []int64{12345, 67890}) {
		t.Error("expected user 99999 to be rejected")
	}
}

func TestIsAllowedEmptyListRejectsAll(t *testing.T) {
	if isAllowed(12345, nil) {
		t.Error("expected empty allowlist to reject all users")
	}
	if isAllowed(12345, []int64{}) {
		t.Error("expected empty allowlist to reject all users")
	}
}

// fakeBotAPI is a minimal BotAPI implementation for testing Send routing.
type fakeBotAPI struct {
	sentChatIDs []int64
}

func (f *fakeBotAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if msg, ok := c.(tgbotapi.MessageConfig); ok {
		f.sentChatIDs = append(f.sentChatIDs, msg.ChatID)
	}
	return tgbotapi.Message{MessageID: 1}, nil
}

func (f *fakeBotAPI) Request(_ tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return &tgbotapi.APIResponse{Ok: true}, nil
}

func (f *fakeBotAPI) GetUpdatesChan(_ tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return make(chan tgbotapi.Update)
}

func (f *fakeBotAPI) StopReceivingUpdates() {}

func TestSendAdminUsesAdminChatID(t *testing.T) {
	bot := &fakeBotAPI{}
	tg := NewTelegramWithBot(bot, 100, 200, nil, slog.Default(), nil)

	if err := tg.SendAdmin(context.Background(), "health alert", "sonarr-health"); err != nil {
		t.Fatal(err)
	}

	if len(bot.sentChatIDs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(bot.sentChatIDs))
	}
	if bot.sentChatIDs[0] != 200 {
		t.Errorf("expected message to admin chat 200, got %d", bot.sentChatIDs[0])
	}
}

func TestSendAdminFallsBackToMainChat(t *testing.T) {
	bot := &fakeBotAPI{}
	// adminChatID=0 means fall back to main chat
	tg := NewTelegramWithBot(bot, 100, 0, nil, slog.Default(), nil)

	if err := tg.SendAdmin(context.Background(), "health alert", "sonarr-health"); err != nil {
		t.Fatal(err)
	}

	if len(bot.sentChatIDs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(bot.sentChatIDs))
	}
	if bot.sentChatIDs[0] != 100 {
		t.Errorf("expected message to main chat 100, got %d", bot.sentChatIDs[0])
	}
}

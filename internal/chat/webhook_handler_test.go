package chat

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/patflynn/reel-life/internal/agent"
)

// fakeProcessor implements MessageProcessor for testing.
type fakeProcessor struct {
	lastMessage string
	lastHistory []agent.Turn
	reply       string
	err         error
}

func (f *fakeProcessor) Process(_ context.Context, msg string, history []agent.Turn) (string, error) {
	f.lastMessage = msg
	f.lastHistory = history
	return f.reply, f.err
}

func TestWebhookHandlerMessage(t *testing.T) {
	proc := &fakeProcessor{reply: "Found 3 results for Breaking Bad"}
	handler := NewWebhookHandler(proc, "", slog.Default(), nil)

	event := `{
		"type": "MESSAGE",
		"message": {"text": "search for Breaking Bad", "argumentText": "search for Breaking Bad"},
		"user": {"displayName": "Test User"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(event))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Text != "Found 3 results for Breaking Bad" {
		t.Errorf("response text = %q, want agent reply", resp.Text)
	}
	if proc.lastMessage != "search for Breaking Bad" {
		t.Errorf("processor received = %q, want %q", proc.lastMessage, "search for Breaking Bad")
	}
}

func TestWebhookHandlerArgumentTextPreferred(t *testing.T) {
	proc := &fakeProcessor{reply: "ok"}
	handler := NewWebhookHandler(proc, "", slog.Default(), nil)

	event := `{
		"type": "MESSAGE",
		"message": {"text": "@ReelLife check health", "argumentText": "check health"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(event))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if proc.lastMessage != "check health" {
		t.Errorf("processor received = %q, want %q (argumentText preferred)", proc.lastMessage, "check health")
	}
}

func TestWebhookHandlerAddedToSpace(t *testing.T) {
	proc := &fakeProcessor{}
	handler := NewWebhookHandler(proc, "", slog.Default(), nil)

	event := `{
		"type": "ADDED_TO_SPACE",
		"space": {"name": "spaces/ABC123", "type": "ROOM"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(event))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Text == "" {
		t.Error("expected welcome message for ADDED_TO_SPACE")
	}
	if proc.lastMessage != "" {
		t.Error("processor should not be called for ADDED_TO_SPACE")
	}
}

func TestWebhookHandlerRemovedFromSpace(t *testing.T) {
	proc := &fakeProcessor{}
	handler := NewWebhookHandler(proc, "", slog.Default(), nil)

	event := `{
		"type": "REMOVED_FROM_SPACE",
		"space": {"name": "spaces/ABC123"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(event))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// Should return empty body (no JSON response).
	body, _ := io.ReadAll(w.Body)
	if len(body) > 0 {
		t.Errorf("expected empty body for REMOVED_FROM_SPACE, got %q", string(body))
	}
}

func TestWebhookHandlerEmptyMessage(t *testing.T) {
	proc := &fakeProcessor{}
	handler := NewWebhookHandler(proc, "", slog.Default(), nil)

	event := `{"type": "MESSAGE", "message": {"text": ""}}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(event))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Text, "didn't get a message") {
		t.Errorf("expected help text for empty message, got %q", resp.Text)
	}
}

func TestWebhookHandlerProcessError(t *testing.T) {
	proc := &fakeProcessor{err: context.DeadlineExceeded}
	handler := NewWebhookHandler(proc, "", slog.Default(), nil)

	event := `{"type": "MESSAGE", "message": {"text": "hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(event))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Text, "error") {
		t.Errorf("expected error message, got %q", resp.Text)
	}
}

func TestWebhookHandlerInvalidJSON(t *testing.T) {
	handler := NewWebhookHandler(&fakeProcessor{}, "", slog.Default(), nil)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestWebhookHandlerJWTRequired(t *testing.T) {
	handler := NewWebhookHandler(&fakeProcessor{}, "12345", slog.Default(), nil)

	// No Authorization header — should be rejected.
	event := `{"type": "MESSAGE", "message": {"text": "hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(event))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 when JWT required but missing", w.Code)
	}
}

func TestWebhookHandlerJWTSkippedWhenNoProjectNumber(t *testing.T) {
	proc := &fakeProcessor{reply: "ok"}
	handler := NewWebhookHandler(proc, "", slog.Default(), nil)

	event := `{"type": "MESSAGE", "message": {"text": "hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(event))
	// No Authorization header, but projectNumber is empty so validation is skipped.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 when no project number configured", w.Code)
	}
}

func TestWebhookHandlerUnknownEventType(t *testing.T) {
	handler := NewWebhookHandler(&fakeProcessor{}, "", slog.Default(), nil)

	event := `{"type": "CARD_CLICKED"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(event))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for unknown event", w.Code)
	}
}

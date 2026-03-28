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
)

func TestGoogleChatAppSend(t *testing.T) {
	var received apiMessage
	var requestPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"spaces/SPACE/messages/MSG"}`))
	}))
	defer srv.Close()

	// Override the API base for testing.
	origBase := chatAPIBase
	defer func() { restoreChatAPIBase(origBase) }()
	setChatAPIBase(srv.URL)

	gc := NewGoogleChatAppWithClient(srv.Client(), "spaces/SPACE", slog.Default())

	err := gc.Send(context.Background(), "hello from app")
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if received.Text != "hello from app" {
		t.Errorf("text = %q, want %q", received.Text, "hello from app")
	}
	if !strings.HasSuffix(requestPath, "/spaces/SPACE/messages") {
		t.Errorf("path = %q, want suffix /spaces/SPACE/messages", requestPath)
	}
	if received.Thread != nil {
		t.Error("expected no thread for Send()")
	}
}

func TestGoogleChatAppSendThread(t *testing.T) {
	var received apiMessage
	var requestURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL = r.URL.String()
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"spaces/SPACE/messages/MSG"}`))
	}))
	defer srv.Close()

	origBase := chatAPIBase
	defer func() { restoreChatAPIBase(origBase) }()
	setChatAPIBase(srv.URL)

	gc := NewGoogleChatAppWithClient(srv.Client(), "spaces/SPACE", slog.Default())

	err := gc.SendThread(context.Background(), "alert!", "health-thread")
	if err != nil {
		t.Fatalf("SendThread() error: %v", err)
	}
	if received.Text != "alert!" {
		t.Errorf("text = %q, want %q", received.Text, "alert!")
	}
	if received.Thread == nil || received.Thread.ThreadKey != "health-thread" {
		t.Error("expected thread with key 'health-thread'")
	}
	if !strings.Contains(requestURL, "messageReplyOption=REPLY_MESSAGE_FALLBACK_TO_NEW_THREAD") {
		t.Errorf("URL = %q, expected reply option param", requestURL)
	}
}

func TestGoogleChatAppHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	origBase := chatAPIBase
	defer func() { restoreChatAPIBase(origBase) }()
	setChatAPIBase(srv.URL)

	gc := NewGoogleChatAppWithClient(srv.Client(), "spaces/SPACE", slog.Default())
	err := gc.Send(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

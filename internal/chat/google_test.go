package chat

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSend(t *testing.T) {
	var received chatMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json; charset=UTF-8" {
			t.Errorf("Content-Type = %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gc := NewGoogleChat(srv.URL, slog.Default())
	err := gc.Send(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if received.Text != "hello world" {
		t.Errorf("received text = %q, want %q", received.Text, "hello world")
	}
	if received.Thread != nil {
		t.Error("expected no thread for Send()")
	}
}

func TestSendThread(t *testing.T) {
	var received chatMessage
	var requestURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL = r.URL.String()
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gc := NewGoogleChat(srv.URL+"?key=k", slog.Default())
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
	if requestURL != "/?key=k&messageReplyOption=REPLY_MESSAGE_FALLBACK_TO_NEW_THREAD" {
		t.Errorf("URL = %q, expected reply option param", requestURL)
	}
}

func TestSendHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	gc := NewGoogleChat(srv.URL, slog.Default())
	err := gc.Send(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

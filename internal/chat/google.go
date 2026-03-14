package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// GoogleChat sends messages to a Google Chat space via webhook.
type GoogleChat struct {
	webhookURL string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewGoogleChat(webhookURL string, logger *slog.Logger) *GoogleChat {
	return &GoogleChat{
		webhookURL: webhookURL,
		httpClient: &http.Client{},
		logger:     logger,
	}
}

type chatMessage struct {
	Text   string      `json:"text"`
	Thread *chatThread `json:"thread,omitempty"`
}

type chatThread struct {
	ThreadKey string `json:"threadKey"`
}

func (g *GoogleChat) Send(ctx context.Context, message string) error {
	return g.post(ctx, g.webhookURL, chatMessage{Text: message})
}

func (g *GoogleChat) SendThread(ctx context.Context, message string, threadKey string) error {
	url := g.webhookURL + "&messageReplyOption=REPLY_MESSAGE_FALLBACK_TO_NEW_THREAD"
	msg := chatMessage{
		Text:   message,
		Thread: &chatThread{ThreadKey: threadKey},
	}
	return g.post(ctx, url, msg)
}

func (g *GoogleChat) post(ctx context.Context, url string, msg chatMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chat API error %d: %s", resp.StatusCode, string(respBody))
	}

	g.logger.Debug("message sent to Google Chat")
	return nil
}

package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2/google"
)

var chatAPIBase = "https://chat.googleapis.com/v1"

// setChatAPIBase overrides the API base URL (for testing).
func setChatAPIBase(url string) { chatAPIBase = url }

// restoreChatAPIBase restores the API base URL after a test.
func restoreChatAPIBase(url string) { chatAPIBase = url }

// GoogleChatApp sends messages via the Google Chat REST API using service account credentials.
// This is the Chat App identity approach, as opposed to the incoming webhook approach in GoogleChat.
type GoogleChatApp struct {
	space      string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewGoogleChatApp creates a notifier that uses the Google Chat API with service account auth.
// saKeyJSON is the raw service account key file contents.
func NewGoogleChatApp(saKeyJSON []byte, space string, logger *slog.Logger) (*GoogleChatApp, error) {
	cfg, err := google.JWTConfigFromJSON(saKeyJSON, "https://www.googleapis.com/auth/chat.bot")
	if err != nil {
		return nil, fmt.Errorf("parse service account key: %w", err)
	}
	return &GoogleChatApp{
		space:      space,
		httpClient: cfg.Client(context.Background()),
		logger:     logger,
	}, nil
}

// NewGoogleChatAppWithClient creates a GoogleChatApp with a caller-provided HTTP client (for testing).
func NewGoogleChatAppWithClient(httpClient *http.Client, space string, logger *slog.Logger) *GoogleChatApp {
	return &GoogleChatApp{
		space:      space,
		httpClient: httpClient,
		logger:     logger,
	}
}

// apiMessage is the Chat API message resource.
type apiMessage struct {
	Text   string     `json:"text"`
	Thread *apiThread `json:"thread,omitempty"`
}

type apiThread struct {
	Name      string `json:"name,omitempty"`
	ThreadKey string `json:"threadKey,omitempty"`
}

func (g *GoogleChatApp) Send(ctx context.Context, message string) error {
	url := fmt.Sprintf("%s/%s/messages", chatAPIBase, g.space)
	return g.postAPI(ctx, url, apiMessage{Text: message})
}

func (g *GoogleChatApp) SendAdmin(ctx context.Context, message string, threadKey string) error {
	return g.SendThread(ctx, message, threadKey)
}

func (g *GoogleChatApp) SendThread(ctx context.Context, message string, threadKey string) error {
	url := fmt.Sprintf("%s/%s/messages?messageReplyOption=REPLY_MESSAGE_FALLBACK_TO_NEW_THREAD", chatAPIBase, g.space)
	msg := apiMessage{
		Text:   message,
		Thread: &apiThread{ThreadKey: threadKey},
	}
	return g.postAPI(ctx, url, msg)
}

func (g *GoogleChatApp) postAPI(ctx context.Context, url string, msg apiMessage) error {
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
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("chat API error %d (and failed to read response body: %w)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("chat API error %d: %s", resp.StatusCode, string(respBody))
	}

	g.logger.Debug("message sent via Chat API", "space", g.space)
	return nil
}

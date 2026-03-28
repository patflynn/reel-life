package chat

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MessageProcessor handles an incoming user message and returns a text response.
type MessageProcessor interface {
	Process(ctx context.Context, userMessage string) (string, error)
}

// Event represents a Google Chat event payload.
type Event struct {
	Type    string        `json:"type"`
	Message *EventMessage `json:"message,omitempty"`
	Space   *EventSpace   `json:"space,omitempty"`
	User    *EventUser    `json:"user,omitempty"`
}

// EventMessage is the message portion of a Google Chat event.
type EventMessage struct {
	Text         string `json:"text"`
	ArgumentText string `json:"argumentText"`
}

// EventSpace is the space portion of a Google Chat event.
type EventSpace struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// EventUser is the user portion of a Google Chat event.
type EventUser struct {
	DisplayName string `json:"displayName"`
}

// Response is the synchronous JSON reply to Google Chat.
type Response struct {
	Text string `json:"text"`
}

// WebhookHandler handles incoming Google Chat HTTP events.
type WebhookHandler struct {
	processor     MessageProcessor
	projectNumber string
	logger        *slog.Logger
	keyCache      *jwkCache
}

// NewWebhookHandler creates a handler that routes Google Chat events to the given processor.
// projectNumber is used as the expected JWT audience; pass empty string to skip JWT validation.
func NewWebhookHandler(processor MessageProcessor, projectNumber string, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		processor:     processor,
		projectNumber: projectNumber,
		logger:        logger,
		keyCache:      &jwkCache{},
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validate JWT if project number is configured.
	if h.projectNumber != "" {
		if err := h.validateJWT(r); err != nil {
			h.logger.Warn("JWT validation failed", "error", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("failed to parse event", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var resp Response
	switch event.Type {
	case "MESSAGE":
		resp = h.handleMessage(r.Context(), &event)
	case "ADDED_TO_SPACE":
		resp = Response{Text: "Hello! I'm your media curation assistant. Ask me about your TV series library — I can search, add series, check downloads, and monitor system health."}
	case "REMOVED_FROM_SPACE":
		h.logger.Info("removed from space", "space", spaceName(event.Space))
		w.WriteHeader(http.StatusOK)
		return
	default:
		h.logger.Debug("unhandled event type", "type", event.Type)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *WebhookHandler) handleMessage(ctx context.Context, event *Event) Response {
	text := ""
	if event.Message != nil {
		// ArgumentText has the text with the @mention stripped; fall back to full Text.
		text = event.Message.ArgumentText
		if text == "" {
			text = event.Message.Text
		}
	}
	text = strings.TrimSpace(text)

	if text == "" {
		return Response{Text: "I didn't get a message. Try asking me something like \"search for Breaking Bad\"."}
	}

	userName := ""
	if event.User != nil {
		userName = event.User.DisplayName
	}
	h.logger.Info("processing chat message", "user", userName, "text_length", len(text))

	reply, err := h.processor.Process(ctx, text)
	if err != nil {
		h.logger.Error("agent processing failed", "error", err)
		return Response{Text: "Sorry, I encountered an error processing your request. Please try again."}
	}

	return Response{Text: reply}
}

func spaceName(s *EventSpace) string {
	if s == nil {
		return ""
	}
	return s.Name
}

// --- JWT validation ---

const googleChatJWKURL = "https://www.googleapis.com/service_accounts/v1/jwk/chat@system.gserviceaccount.com"

// jwkCache caches Google's public keys.
type jwkCache struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	fetchURL  string // override for testing
}

func (h *WebhookHandler) validateJWT(r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return fmt.Errorf("missing Bearer token")
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	keyFunc := func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}
		return h.keyCache.getKey(kid)
	}

	token, err := jwt.Parse(tokenStr, keyFunc,
		jwt.WithAudience(h.projectNumber),
		jwt.WithIssuer("chat@system.gserviceaccount.com"),
	)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return fmt.Errorf("token not valid")
	}
	return nil
}

func (c *jwkCache) getKey(kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	if key, ok := c.keys[kid]; ok && time.Since(c.fetchedAt) < time.Hour {
		c.mu.RUnlock()
		return key, nil
	}
	c.mu.RUnlock()

	return c.refresh(kid)
}

func (c *jwkCache) refresh(kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if key, ok := c.keys[kid]; ok && time.Since(c.fetchedAt) < time.Hour {
		return key, nil
	}

	url := googleChatJWKURL
	if c.fetchURL != "" {
		url = c.fetchURL
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch JWK: %w", err)
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWK response: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, raw := range jwks.Keys {
		var fields struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		}
		if err := json.Unmarshal(raw, &fields); err != nil {
			continue
		}
		key, err := jwkToRSAPublicKey(fields.N, fields.E)
		if err != nil {
			continue
		}
		keys[fields.Kid] = key
	}

	c.keys = keys
	c.fetchedAt = time.Now()

	key, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("kid %q not found in Google JWK set", kid)
	}
	return key, nil
}

// jwkToRSAPublicKey converts base64url-encoded JWK modulus and exponent to an RSA public key.
func jwkToRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}

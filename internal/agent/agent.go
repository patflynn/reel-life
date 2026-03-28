package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/patflynn/reel-life/internal/sonarr"
)

const systemPrompt = `You are a media curation assistant for a home media server. You help users manage their TV series library through Sonarr.

Your capabilities:
- Search for TV series and provide concise summaries of results
- Add series to the library for monitoring and automatic downloading
- Check the download queue for active and pending downloads
- Review download history for recent activity
- Monitor system health and report any issues
- Remove failed downloads and manage the blocklist

Guidelines:
- When searching, present results concisely with title, year, and a brief description
- Always confirm with the user before adding a new series
- When reporting health issues, clearly explain what each issue means and suggest fixes
- Be direct and helpful — avoid unnecessary pleasantries
- Only use the tools provided — do not make up information`

const maxToolRounds = 10

// Agent handles natural language interactions using Claude with Sonarr tools.
type Agent struct {
	client  *anthropic.Client
	sonarr  sonarr.Client
	model   string
	maxTok  int64
	logger  *slog.Logger
	limiter *RateLimiter
}

func New(apiKey string, sonarrClient sonarr.Client, model string, maxTokens int, logger *slog.Logger, limiter *RateLimiter) *Agent {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Agent{
		client:  &client,
		sonarr:  sonarrClient,
		model:   model,
		maxTok:  int64(maxTokens),
		logger:  logger,
		limiter: limiter,
	}
}

// NewWithClient creates an Agent with a pre-configured Anthropic client (for testing).
func NewWithClient(client *anthropic.Client, sonarrClient sonarr.Client, model string, maxTokens int, logger *slog.Logger, limiter *RateLimiter) *Agent {
	return &Agent{
		client:  client,
		sonarr:  sonarrClient,
		model:   model,
		maxTok:  int64(maxTokens),
		logger:  logger,
		limiter: limiter,
	}
}

// requestIDKey is the context key for the per-request identifier used in audit logs.
type requestIDKey struct{}

// WithRequestID returns a child context carrying the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func requestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

// Process runs the agentic tool-use loop for a user message and returns the final text response.
func (a *Agent) Process(ctx context.Context, userMessage string) (string, error) {
	tools := toolDefinitions()
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
	}

	if a.limiter != nil {
		a.limiter.Reset()
	}

	reqID := requestID(ctx)

	for round := range maxToolRounds {
		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(a.model),
			MaxTokens: a.maxTok,
			System: []anthropic.TextBlockParam{
				{Text: systemPrompt},
			},
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			return "", fmt.Errorf("claude API call: %w", err)
		}

		// Collect tool uses from the response
		var toolResults []anthropic.ContentBlockParamUnion
		var textResponse string

		for _, block := range resp.Content {
			switch v := block.AsAny().(type) {
			case anthropic.TextBlock:
				textResponse += v.Text
			case anthropic.ToolUseBlock:
				result, isErr := a.executeToolWithAudit(ctx, v.Name, v.Input, round, reqID)
				toolResults = append(toolResults, anthropic.NewToolResultBlock(v.ID, result, isErr))
			}
		}

		// If no tool calls, we're done
		if len(toolResults) == 0 {
			return textResponse, nil
		}

		// Add assistant response and tool results to conversation
		messages = append(messages, resp.ToParam())
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}

	return "", fmt.Errorf("exceeded maximum tool rounds (%d)", maxToolRounds)
}

// executeToolWithAudit wraps tool dispatch with rate limiting and audit logging.
func (a *Agent) executeToolWithAudit(ctx context.Context, name string, rawInput json.RawMessage, round int, reqID string) (string, bool) {
	// Audit log: invocation
	a.logger.Info("tool invocation",
		"tool", name,
		"input", sanitizeInput(rawInput),
		"round", round,
		"request_id", reqID,
	)

	// Rate limit check
	if a.limiter != nil {
		if err := a.limiter.Allow(name, IsDestructive(name)); err != nil {
			a.logger.Warn("tool rate limited",
				"tool", name,
				"error", err,
				"round", round,
				"request_id", reqID,
			)
			return jsonError(err.Error()), true
		}
	}

	start := time.Now()
	result, isErr := a.dispatchTool(ctx, name, rawInput)
	duration := time.Since(start)

	// Audit log: result
	a.logger.Info("tool result",
		"tool", name,
		"success", !isErr,
		"duration_ms", duration.Milliseconds(),
		"round", round,
		"request_id", reqID,
	)

	return result, isErr
}

// sanitizeInput returns a string summary of tool input suitable for logging.
// It strips any field whose key contains sensitive substrings to avoid leaking credentials.
func sanitizeInput(raw json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "<invalid json>"
	}
	for k := range m {
		lowerK := strings.ToLower(k)
		if strings.Contains(lowerK, "key") || strings.Contains(lowerK, "secret") || strings.Contains(lowerK, "password") || strings.Contains(lowerK, "token") {
			m[k] = "REDACTED"
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return "<failed to sanitize json>"
	}
	return string(out)
}

// dispatchTool executes a tool call and returns the JSON result string and whether it's an error.
func (a *Agent) dispatchTool(ctx context.Context, name string, rawInput json.RawMessage) (string, bool) {
	var result any
	var err error

	switch name {
	case "search_series":
		var input searchSeriesInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		result, err = a.sonarr.Search(ctx, input.Term)

	case "add_series":
		var input addSeriesInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		result, err = a.sonarr.Add(ctx, sonarr.AddSeriesRequest{
			Title:            input.Title,
			TVDBID:           input.TVDBID,
			QualityProfileID: input.QualityProfileID,
			RootFolderPath:   input.RootFolderPath,
			Monitored:        true,
			SeasonFolder:     true,
		})

	case "get_queue":
		result, err = a.sonarr.Queue(ctx)

	case "get_history":
		var input getHistoryInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		pageSize := input.PageSize
		if pageSize == 0 {
			pageSize = 20
		}
		result, err = a.sonarr.History(ctx, pageSize)

	case "check_health":
		result, err = a.sonarr.Health(ctx)

	case "remove_failed":
		var input removeFailedInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		err = a.sonarr.RemoveFailed(ctx, input.ID, input.Blocklist)
		if err == nil {
			result = map[string]string{"status": "removed"}
		}

	default:
		return jsonError("unknown tool: " + name), true
	}

	if err != nil {
		a.logger.Warn("tool error", "tool", name, "error", err)
		return jsonError(err.Error()), true
	}

	data, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return jsonError("failed to marshal result: " + marshalErr.Error()), true
	}
	return string(data), false
}

func jsonError(msg string) string {
	data, _ := json.Marshal(map[string]string{"error": msg})
	return string(data)
}

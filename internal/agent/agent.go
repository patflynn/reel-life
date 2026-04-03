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
	"github.com/patflynn/reel-life/internal/notebook"
	"github.com/patflynn/reel-life/internal/overseerr"
	"github.com/patflynn/reel-life/internal/prowlarr"
	"github.com/patflynn/reel-life/internal/radarr"
	"github.com/patflynn/reel-life/internal/sonarr"
	"github.com/patflynn/reel-life/internal/weather"
)

const systemPrompt = `You are a media curation assistant for a home media server. You help users manage their TV series library through Sonarr, their movie library through Radarr, and handle media requests through Overseerr.

Your capabilities:
- Search for TV series and movies, and provide concise summaries of results
- Add series or movies to the library for monitoring and automatic downloading
- Check the download queue for active and pending downloads (both TV and movies)
- Review download history for recent activity
- Monitor system health and report any issues
- Remove failed downloads and manage the blocklist
- Get detailed series info and episode status
- Update season monitoring settings (enable/disable monitoring per season)
- Search for available releases and see why they were accepted/rejected
- View Sonarr logs for debugging
- Check quality profiles, blocklist, root folders, and download client status
- Manage indexers via Prowlarr: list, test, enable/disable, update priority, delete, check stats and health, search across indexers
- List, approve, and decline media requests from Overseerr
- Search for movies and TV shows in Overseerr's media database
- Get request statistics (pending, approved, declined counts)

Guidelines:
- When searching, present results concisely with title, year, and a brief description
- Always confirm with the user before adding a new series or movie, or approving/declining requests
- When reporting health issues, clearly explain what each issue means and suggest fixes
- Be direct and helpful — avoid unnecessary pleasantries
- Only use the tools provided — do not make up information
- You have a persistent notebook for memory across conversations. Save useful observations: user preferences, recurring issues, operational patterns
- Pinned notes are always visible to you; reference notes need to be looked up with notebook_read
- Keep pinned notes concise and high-signal; use reference type for detailed information
- Before creating a new note, check existing notes to avoid duplicates — update instead if a similar note exists`

const maxToolRounds = 10

// Agent handles natural language interactions using Claude with Sonarr, Radarr, Prowlarr, and Overseerr tools.
type Agent struct {
	client    *anthropic.Client
	sonarr    sonarr.Client
	radarr    radarr.Client
	prowlarr  prowlarr.Client
	overseerr overseerr.Client
	notebook  notebook.Notebook
	weather   *weather.Client
	model     string
	maxTok    int64
	logger    *slog.Logger
	limiter   *RateLimiter
}

func New(apiKey string, sonarrClient sonarr.Client, radarrClient radarr.Client, prowlarrClient prowlarr.Client, overseerrClient overseerr.Client, nb notebook.Notebook, weatherClient *weather.Client, model string, maxTokens int, logger *slog.Logger, limiter *RateLimiter) *Agent {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Agent{
		client:    &client,
		sonarr:    sonarrClient,
		radarr:    radarrClient,
		prowlarr:  prowlarrClient,
		overseerr: overseerrClient,
		notebook:  nb,
		weather:   weatherClient,
		model:     model,
		maxTok:    int64(maxTokens),
		logger:    logger,
		limiter:   limiter,
	}
}

// NewWithClient creates an Agent with a pre-configured Anthropic client (for testing).
func NewWithClient(client *anthropic.Client, sonarrClient sonarr.Client, radarrClient radarr.Client, prowlarrClient prowlarr.Client, overseerrClient overseerr.Client, nb notebook.Notebook, weatherClient *weather.Client, model string, maxTokens int, logger *slog.Logger, limiter *RateLimiter) *Agent {
	return &Agent{
		client:    client,
		sonarr:    sonarrClient,
		radarr:    radarrClient,
		prowlarr:  prowlarrClient,
		overseerr: overseerrClient,
		notebook:  nb,
		weather:   weatherClient,
		model:     model,
		maxTok:    int64(maxTokens),
		logger:    logger,
		limiter:   limiter,
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

// buildSystemPrompt returns the system prompt with pinned notebook notes appended.
func (a *Agent) buildSystemPrompt(ctx context.Context) string {
	dateLine := fmt.Sprintf("Today's date is %s.", time.Now().Format("2006-01-02"))

	var locationLine string
	if a.weather != nil {
		if cond := a.weather.Current(ctx); cond != nil {
			locationLine = fmt.Sprintf("Current location: %s. Weather: %.0f°C, %s.", a.weather.Location(), cond.Temperature, cond.Description)
		} else {
			locationLine = fmt.Sprintf("Current location: %s.", a.weather.Location())
		}
	}

	var prompt string
	if locationLine != "" {
		prompt = fmt.Sprintf("%s\n%s\n\n%s", dateLine, locationLine, systemPrompt)
	} else {
		prompt = fmt.Sprintf("%s\n\n%s", dateLine, systemPrompt)
	}

	if a.notebook == nil {
		return prompt
	}

	pinned, err := a.notebook.Pinned(ctx)
	if err != nil {
		a.logger.Warn("failed to load pinned notes", "error", err)
		return prompt
	}
	if len(pinned) == 0 {
		return prompt
	}

	var sb strings.Builder
	sb.WriteString(prompt)
	sb.WriteString("\n\n## Notebook (always loaded)\n")
	for _, n := range pinned {
		sb.WriteString("### ")
		sb.WriteString(n.Title)
		sb.WriteString("\n")
		sb.WriteString(n.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}

// Process runs the agentic tool-use loop for a user message and returns the final text response.
// History turns are prepended to provide conversational context.
func (a *Agent) Process(ctx context.Context, userMessage string, history []Turn) (string, error) {
	tools := toolDefinitions()

	var messages []anthropic.MessageParam
	for _, t := range history {
		switch t.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(t.Content)))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(t.Content)))
		}
	}
	messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)))

	if a.limiter != nil {
		a.limiter.Reset()
	}

	reqID := requestID(ctx)
	sysPrompt := a.buildSystemPrompt(ctx)

	for round := range maxToolRounds {
		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(a.model),
			MaxTokens: a.maxTok,
			System: []anthropic.TextBlockParam{
				{Text: sysPrompt},
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

	case "get_series_detail":
		var input getSeriesDetailInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		result, err = a.sonarr.GetSeries(ctx, input.SeriesID)

	case "get_episodes":
		var input getEpisodesInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		result, err = a.sonarr.GetEpisodes(ctx, input.SeriesID)

	case "get_logs":
		var input getLogsInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		result, err = a.sonarr.GetLogs(ctx, input.PageSize, input.Level)

	case "manual_search":
		var input manualSearchInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		result, err = a.sonarr.ManualSearch(ctx, input.EpisodeID)

	case "get_quality_profiles":
		result, err = a.sonarr.GetQualityProfiles(ctx)

	case "get_blocklist":
		var input getBlocklistInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		result, err = a.sonarr.GetBlocklist(ctx, input.PageSize)

	case "get_root_folders":
		result, err = a.sonarr.GetRootFolders(ctx)

	case "get_download_clients":
		result, err = a.sonarr.GetDownloadClients(ctx)

	case "update_series_monitoring":
		var input updateSeriesMonitoringInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		series, getErr := a.sonarr.GetSeries(ctx, input.SeriesID)
		if getErr != nil {
			return jsonError(getErr.Error()), true
		}
		found := false
		for i, s := range series.Seasons {
			if s.SeasonNumber == input.SeasonNumber {
				series.Seasons[i].Monitored = input.Monitored
				found = true
				break
			}
		}
		if !found {
			return jsonError(fmt.Sprintf("season %d not found in series %q", input.SeasonNumber, series.Title)), true
		}
		result, err = a.sonarr.UpdateSeries(ctx, series)

	case "trigger_series_search":
		var input triggerSeriesSearchInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		cmd := sonarr.CommandRequest{
			Name:     "SeriesSearch",
			SeriesID: input.SeriesID,
		}
		if input.SeasonNumber != nil {
			cmd.Name = "SeasonSearch"
			cmd.SeasonNumber = input.SeasonNumber
		}
		result, err = a.sonarr.Command(ctx, cmd)

	case "delete_series":
		var input deleteSeriesInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		err = a.sonarr.DeleteSeries(ctx, input.SeriesID, input.DeleteFiles)
		if err == nil {
			result = map[string]string{"status": "deleted"}
		}

	case "remove_blocklist_item":
		var input removeBlocklistItemInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		err = a.sonarr.DeleteBlocklistItem(ctx, input.ID)
		if err == nil {
			result = map[string]string{"status": "removed"}
		}

	case "grab_release":
		var input grabReleaseInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true
		}
		result, err = a.sonarr.GrabRelease(ctx, input.GUID, input.IndexerID)

	case "search_movies", "add_movie", "get_movie_queue", "get_movie_history", "check_movie_health", "remove_failed_movie":
		if a.radarr == nil {
			return jsonError("Radarr integration is not configured"), true
		}
		switch name {
		case "search_movies":
			var input searchMoviesInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			result, err = a.radarr.Search(ctx, input.Term)
		case "add_movie":
			var input addMovieInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			minAvail := input.MinimumAvailability
			if minAvail == "" {
				minAvail = "released"
			}
			result, err = a.radarr.Add(ctx, radarr.AddMovieRequest{
				Title:               input.Title,
				TMDBID:              input.TMDBID,
				QualityProfileID:    input.QualityProfileID,
				RootFolderPath:      input.RootFolderPath,
				Monitored:           true,
				MinimumAvailability: minAvail,
			})
		case "get_movie_queue":
			result, err = a.radarr.Queue(ctx)
		case "get_movie_history":
			var input getMovieHistoryInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			pageSize := input.PageSize
			if pageSize == 0 {
				pageSize = 20
			}
			result, err = a.radarr.History(ctx, pageSize)
		case "check_movie_health":
			result, err = a.radarr.Health(ctx)
		case "remove_failed_movie":
			var input removeFailedMovieInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			err = a.radarr.RemoveFailed(ctx, input.ID, input.Blocklist)
			if err == nil {
				result = map[string]string{"status": "removed"}
			}
		}

	case "list_indexers", "test_indexer", "test_all_indexers", "get_indexer_stats", "check_indexer_health", "search_indexers", "enable_indexer", "update_indexer_priority", "delete_indexer":
		if a.prowlarr == nil {
			return jsonError("Prowlarr integration is not configured"), true
		}
		switch name {
		case "list_indexers":
			result, err = a.prowlarr.ListIndexers(ctx)
		case "test_indexer":
			var input testIndexerInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			err = a.prowlarr.TestIndexer(ctx, input.ID)
			if err == nil {
				result = map[string]string{"status": "ok"}
			}
		case "test_all_indexers":
			result, err = a.prowlarr.TestAllIndexers(ctx)
		case "get_indexer_stats":
			result, err = a.prowlarr.GetIndexerStats(ctx)
		case "check_indexer_health":
			result, err = a.prowlarr.CheckHealth(ctx)
		case "search_indexers":
			var input searchIndexersInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			result, err = a.prowlarr.Search(ctx, input.Query)
		case "enable_indexer":
			var input enableIndexerInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			indexers, listErr := a.prowlarr.ListIndexers(ctx)
			if listErr != nil {
				return jsonError(listErr.Error()), true
			}
			var found *prowlarr.Indexer
			for i := range indexers {
				if indexers[i].ID == input.IndexerID {
					found = &indexers[i]
					break
				}
			}
			if found == nil {
				return jsonError(fmt.Sprintf("indexer %d not found", input.IndexerID)), true
			}
			found.Enable = input.Enabled
			result, err = a.prowlarr.UpdateIndexer(ctx, found)
		case "update_indexer_priority":
			var input updateIndexerPriorityInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			indexers, listErr := a.prowlarr.ListIndexers(ctx)
			if listErr != nil {
				return jsonError(listErr.Error()), true
			}
			var found *prowlarr.Indexer
			for i := range indexers {
				if indexers[i].ID == input.IndexerID {
					found = &indexers[i]
					break
				}
			}
			if found == nil {
				return jsonError(fmt.Sprintf("indexer %d not found", input.IndexerID)), true
			}
			found.Priority = input.Priority
			result, err = a.prowlarr.UpdateIndexer(ctx, found)
		case "delete_indexer":
			var input deleteIndexerInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			err = a.prowlarr.DeleteIndexer(ctx, input.IndexerID)
			if err == nil {
				result = map[string]string{"status": "deleted"}
			}
		}

	case "list_requests", "approve_request", "decline_request", "get_request_count", "search_media":
		if a.overseerr == nil {
			return jsonError("Overseerr integration is not configured"), true
		}
		switch name {
		case "list_requests":
			var input listRequestsInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			take := input.Take
			if take == 0 {
				take = 20
			}
			result, err = a.overseerr.ListRequests(ctx, input.Filter, take, input.Skip)
		case "approve_request":
			var input approveRequestInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			err = a.overseerr.ApproveRequest(ctx, input.ID)
			if err == nil {
				result = map[string]string{"status": "approved"}
			}
		case "decline_request":
			var input declineRequestInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			err = a.overseerr.DeclineRequest(ctx, input.ID)
			if err == nil {
				result = map[string]string{"status": "declined"}
			}
		case "get_request_count":
			result, err = a.overseerr.GetRequestCount(ctx)
		case "search_media":
			var input searchMediaInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			page := input.Page
			if page == 0 {
				page = 1
			}
			result, err = a.overseerr.SearchMedia(ctx, input.Query, page)
		}

	case "notebook_write", "notebook_read", "notebook_list", "notebook_delete":
		if a.notebook == nil {
			return jsonError("Notebook integration is not configured"), true
		}
		switch name {
		case "notebook_write":
			var input notebookWriteInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			noteType := notebook.NoteType(input.Type)
			if noteType != notebook.Pinned && noteType != notebook.Reference {
				return jsonError("type must be 'pinned' or 'reference'"), true
			}
			// Check for duplicate titles to avoid creating redundant notes.
			if input.ID == "" {
				existing, searchErr := a.notebook.Search(ctx, input.Title)
				if searchErr == nil {
					for _, n := range existing {
						if strings.EqualFold(n.Title, input.Title) {
							data, _ := json.Marshal(map[string]string{
								"warning":     "a note with a similar title already exists",
								"existing_id": n.ID,
								"title":       n.Title,
							})
							return string(data), false
						}
					}
				}
			}
			err = a.notebook.Write(ctx, notebook.Note{
				ID:      input.ID,
				Type:    noteType,
				Title:   input.Title,
				Content: input.Content,
			})
			if err == nil {
				result = map[string]string{"status": "saved"}
			}
		case "notebook_read":
			var input notebookReadInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			result, err = a.notebook.Read(ctx, input.ID)
		case "notebook_list":
			var input notebookListInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			summaries, listErr := a.notebook.List(ctx)
			if listErr != nil {
				err = listErr
			} else if input.Type != "" {
				filtered := make([]notebook.NoteSummary, 0)
				for _, s := range summaries {
					if string(s.Type) == input.Type {
						filtered = append(filtered, s)
					}
				}
				result = filtered
			} else {
				result = summaries
			}
		case "notebook_delete":
			var input notebookDeleteInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return jsonError("invalid input: " + err.Error()), true
			}
			err = a.notebook.Delete(ctx, input.ID)
			if err == nil {
				result = map[string]string{"status": "deleted"}
			}
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

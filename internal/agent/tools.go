package agent

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/invopop/jsonschema"
)

type searchSeriesInput struct {
	Term string `json:"term" jsonschema_description:"The search term to look up TV series"`
}

type addSeriesInput struct {
	Title            string `json:"title" jsonschema_description:"Title of the series to add"`
	TVDBID           int    `json:"tvdb_id" jsonschema_description:"TVDB ID of the series"`
	QualityProfileID int    `json:"quality_profile_id" jsonschema_description:"Quality profile ID to use"`
	RootFolderPath   string `json:"root_folder_path" jsonschema_description:"Root folder path for the series"`
}

type removeFailedInput struct {
	ID        int  `json:"id" jsonschema_description:"Queue item ID to remove"`
	Blocklist bool `json:"blocklist" jsonschema_description:"Whether to add the release to the blocklist"`
}

type getHistoryInput struct {
	PageSize int `json:"page_size,omitempty" jsonschema_description:"Number of history records to return (default 20)"`
}

type getSeriesDetailInput struct {
	SeriesID int `json:"series_id" jsonschema_description:"Sonarr series ID"`
}

type getEpisodesInput struct {
	SeriesID int `json:"series_id" jsonschema_description:"Sonarr series ID"`
}

type getLogsInput struct {
	PageSize int    `json:"page_size,omitempty" jsonschema_description:"Number of log records to return"`
	Level    string `json:"level,omitempty" jsonschema_description:"Filter by log level: info, warn, or error"`
}

type manualSearchInput struct {
	EpisodeID int `json:"episode_id" jsonschema_description:"Episode ID to search releases for"`
}

type getBlocklistInput struct {
	PageSize int `json:"page_size,omitempty" jsonschema_description:"Number of blocklist records to return"`
}

type updateSeriesMonitoringInput struct {
	SeriesID     int  `json:"series_id" jsonschema_description:"Sonarr series ID"`
	SeasonNumber int  `json:"season_number" jsonschema_description:"Season number to update monitoring for"`
	Monitored    bool `json:"monitored" jsonschema_description:"Whether to enable (true) or disable (false) monitoring"`
}

type searchMoviesInput struct {
	Term string `json:"term" jsonschema_description:"The search term to look up movies"`
}

type addMovieInput struct {
	Title               string `json:"title" jsonschema_description:"Title of the movie to add"`
	TMDBID              int    `json:"tmdb_id" jsonschema_description:"TMDB ID of the movie"`
	QualityProfileID    int    `json:"quality_profile_id" jsonschema_description:"Quality profile ID to use"`
	RootFolderPath      string `json:"root_folder_path" jsonschema_description:"Root folder path for the movie"`
	MinimumAvailability string `json:"minimum_availability,omitempty" jsonschema_description:"When the movie is considered available: announced, inCinemas, or released (default released)"`
}

type removeFailedMovieInput struct {
	ID        int  `json:"id" jsonschema_description:"Queue item ID to remove"`
	Blocklist bool `json:"blocklist" jsonschema_description:"Whether to add the release to the blocklist"`
}

type getMovieHistoryInput struct {
	PageSize int `json:"page_size,omitempty" jsonschema_description:"Number of history records to return (default 20)"`
}

type testIndexerInput struct {
	ID int `json:"id" jsonschema_description:"Indexer ID to test connectivity for"`
}

type searchIndexersInput struct {
	Query string `json:"query" jsonschema_description:"Search query to run across all indexers"`
}

type listRequestsInput struct {
	Filter string `json:"filter,omitempty" jsonschema_description:"Filter requests by status: pending, approved, all (default all)"`
	Take   int    `json:"take,omitempty" jsonschema_description:"Number of requests to return (default 20)"`
	Skip   int    `json:"skip,omitempty" jsonschema_description:"Number of requests to skip for pagination"`
}

type approveRequestInput struct {
	ID int `json:"id" jsonschema_description:"Request ID to approve"`
}

type declineRequestInput struct {
	ID int `json:"id" jsonschema_description:"Request ID to decline"`
}

type searchMediaInput struct {
	Query string `json:"query" jsonschema_description:"Search query for movies or TV shows"`
	Page  int    `json:"page,omitempty" jsonschema_description:"Page number for results (default 1)"`
}

type notebookWriteInput struct {
	ID      string `json:"id,omitempty" jsonschema_description:"Note ID for updates. Omit to create a new note."`
	Type    string `json:"type" jsonschema_description:"Note type: pinned (always visible in prompt) or reference (looked up on demand)"`
	Title   string `json:"title" jsonschema_description:"Short descriptive title for the note"`
	Content string `json:"content" jsonschema_description:"Note content text"`
}

type notebookReadInput struct {
	ID string `json:"id" jsonschema_description:"ID of the note to read"`
}

type notebookListInput struct {
	Type string `json:"type,omitempty" jsonschema_description:"Filter by note type: pinned, reference, or omit for all"`
}

type notebookDeleteInput struct {
	ID string `json:"id" jsonschema_description:"ID of the note to delete"`
}

func generateSchema[T any]() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties,
	}
}

// toolDef pairs an Anthropic tool definition with a destructive flag for rate limiting.
type toolDef struct {
	Param       anthropic.ToolParam
	Destructive bool
}

// destructiveTools is the set of tools that modify state.
var destructiveTools = map[string]bool{
	"add_series":                 true,
	"remove_failed":              true,
	"update_series_monitoring":   true,
	"add_movie":           true,
	"remove_failed_movie": true,
	"approve_request":     true,
	"decline_request":     true,
	"notebook_write":      true,
	"notebook_delete":     true,
}

// IsDestructive reports whether the named tool modifies state.
func IsDestructive(name string) bool {
	return destructiveTools[name]
}

func allToolDefs() []toolDef {
	defs := sonarrToolDefs()
	defs = append(defs, radarrToolDefs()...)
	defs = append(defs, prowlarrToolDefs()...)
	defs = append(defs, overseerrToolDefs()...)
	defs = append(defs, notebookToolDefs()...)
	return defs
}

func sonarrToolDefs() []toolDef {
	return []toolDef{
		{
			Param: anthropic.ToolParam{
				Name:        "search_series",
				Description: anthropic.String("Search for TV series by name. Returns matching series with details like title, year, overview, and TVDB ID."),
				InputSchema: generateSchema[searchSeriesInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "add_series",
				Description: anthropic.String("Add a TV series to Sonarr for monitoring and automatic downloading. Requires the TVDB ID from a search result."),
				InputSchema: generateSchema[addSeriesInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_queue",
				Description: anthropic.String("Get the current download queue showing active and pending downloads with their status."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_history",
				Description: anthropic.String("Get recent download history showing completed, failed, and imported episodes."),
				InputSchema: generateSchema[getHistoryInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "check_health",
				Description: anthropic.String("Check Sonarr system health for warnings and errors like connectivity issues, disk space, or indexer problems."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "remove_failed",
				Description: anthropic.String("Remove a failed download from the Sonarr queue. Optionally blocklist the release to prevent re-downloading."),
				InputSchema: generateSchema[removeFailedInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_series_detail",
				Description: anthropic.String("Get detailed information about a specific series including episode counts, size on disk, and quality profile."),
				InputSchema: generateSchema[getSeriesDetailInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_episodes",
				Description: anthropic.String("Get all episodes for a series with season/episode numbers, titles, file status, and air dates."),
				InputSchema: generateSchema[getEpisodesInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_logs",
				Description: anthropic.String("Get recent Sonarr log entries for debugging. Optionally filter by level (info, warn, error)."),
				InputSchema: generateSchema[getLogsInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "manual_search",
				Description: anthropic.String("Search for available releases for a specific episode. Shows indexer, quality, size, and rejection reasons."),
				InputSchema: generateSchema[manualSearchInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_quality_profiles",
				Description: anthropic.String("List all quality profiles configured in Sonarr with their cutoff settings."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_blocklist",
				Description: anthropic.String("Get blocklisted releases that Sonarr will not re-download."),
				InputSchema: generateSchema[getBlocklistInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_root_folders",
				Description: anthropic.String("List root folders with free and total disk space."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_download_clients",
				Description: anthropic.String("List configured download clients with their status, protocol, and priority."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "update_series_monitoring",
				Description: anthropic.String("Enable or disable monitoring for a specific season of a series. Use get_series_detail first to find the series ID."),
				InputSchema: generateSchema[updateSeriesMonitoringInput](),
			},
			Destructive: true,
		},
	}
}

func radarrToolDefs() []toolDef {
	return []toolDef{
		{
			Param: anthropic.ToolParam{
				Name:        "search_movies",
				Description: anthropic.String("Search for movies by name. Returns matching movies with details like title, year, overview, and TMDB ID."),
				InputSchema: generateSchema[searchMoviesInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "add_movie",
				Description: anthropic.String("Add a movie to Radarr for monitoring and automatic downloading. Requires the TMDB ID from a search result."),
				InputSchema: generateSchema[addMovieInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_movie_queue",
				Description: anthropic.String("Get the current Radarr download queue showing active and pending movie downloads with their status."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_movie_history",
				Description: anthropic.String("Get recent Radarr download history showing completed, failed, and imported movies."),
				InputSchema: generateSchema[getMovieHistoryInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "check_movie_health",
				Description: anthropic.String("Check Radarr system health for warnings and errors like connectivity issues, disk space, or indexer problems."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "remove_failed_movie",
				Description: anthropic.String("Remove a failed download from the Radarr queue. Optionally blocklist the release to prevent re-downloading."),
				InputSchema: generateSchema[removeFailedMovieInput](),
			},
			Destructive: true,
		},
	}
}

func prowlarrToolDefs() []toolDef {
	return []toolDef{
		{
			Param: anthropic.ToolParam{
				Name:        "list_indexers",
				Description: anthropic.String("List all configured indexers in Prowlarr with their status, protocol, and priority."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "test_indexer",
				Description: anthropic.String("Test an indexer's connectivity to verify it is reachable and responding."),
				InputSchema: generateSchema[testIndexerInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_indexer_stats",
				Description: anthropic.String("Get indexer statistics including query counts, grab counts, and response times."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "check_indexer_health",
				Description: anthropic.String("Check Prowlarr system health for warnings and errors with indexers or connectivity."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "search_indexers",
				Description: anthropic.String("Search across all configured indexers for releases matching a query."),
				InputSchema: generateSchema[searchIndexersInput](),
			},
		},
	}
}

func overseerrToolDefs() []toolDef {
	return []toolDef{
		{
			Param: anthropic.ToolParam{
				Name:        "list_requests",
				Description: anthropic.String("List media requests from Overseerr. Filter by status: pending, approved, or all."),
				InputSchema: generateSchema[listRequestsInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "approve_request",
				Description: anthropic.String("Approve a pending media request in Overseerr. This sends the media to Sonarr/Radarr for downloading."),
				InputSchema: generateSchema[approveRequestInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "decline_request",
				Description: anthropic.String("Decline a pending media request in Overseerr."),
				InputSchema: generateSchema[declineRequestInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_request_count",
				Description: anthropic.String("Get counts of media requests by status (pending, approved, declined, total) from Overseerr."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "search_media",
				Description: anthropic.String("Search for movies and TV shows in Overseerr's media database."),
				InputSchema: generateSchema[searchMediaInput](),
			},
		},
	}
}

func notebookToolDefs() []toolDef {
	return []toolDef{
		{
			Param: anthropic.ToolParam{
				Name:        "notebook_write",
				Description: anthropic.String("Create or update a note in the persistent notebook. Use pinned type for high-signal info that should always be visible; use reference for detailed info looked up on demand. If updating, provide the existing note's ID."),
				InputSchema: generateSchema[notebookWriteInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "notebook_read",
				Description: anthropic.String("Read a note from the notebook by its ID. Use this to retrieve reference notes or see full details."),
				InputSchema: generateSchema[notebookReadInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "notebook_list",
				Description: anthropic.String("List all notes in the notebook. Returns ID, type, title, and last updated time. Optionally filter by type (pinned or reference)."),
				InputSchema: generateSchema[notebookListInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "notebook_delete",
				Description: anthropic.String("Delete a note from the notebook by its ID."),
				InputSchema: generateSchema[notebookDeleteInput](),
			},
			Destructive: true,
		},
	}
}

func toolDefinitions() []anthropic.ToolUnionParam {
	defs := allToolDefs()
	result := make([]anthropic.ToolUnionParam, len(defs))
	for i, d := range defs {
		p := d.Param
		result[i] = anthropic.ToolUnionParam{OfTool: &p}
	}
	return result
}

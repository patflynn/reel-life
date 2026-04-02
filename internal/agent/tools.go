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
	"add_series":          true,
	"remove_failed":       true,
	"add_movie":           true,
	"remove_failed_movie": true,
}

// IsDestructive reports whether the named tool modifies state.
func IsDestructive(name string) bool {
	return destructiveTools[name]
}

func allToolDefs() []toolDef {
	defs := sonarrToolDefs()
	defs = append(defs, radarrToolDefs()...)
	defs = append(defs, prowlarrToolDefs()...)
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

func toolDefinitions() []anthropic.ToolUnionParam {
	defs := allToolDefs()
	result := make([]anthropic.ToolUnionParam, len(defs))
	for i, d := range defs {
		p := d.Param
		result[i] = anthropic.ToolUnionParam{OfTool: &p}
	}
	return result
}

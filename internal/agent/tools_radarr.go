package agent

import "github.com/anthropics/anthropic-sdk-go"

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

type getMovieDetailInput struct {
	MovieID int `json:"movie_id" jsonschema_description:"Radarr movie ID"`
}

type getMovieBlocklistInput struct {
	PageSize int `json:"page_size,omitempty" jsonschema_description:"Number of blocklist records to return (default 20)"`
}

type manualMovieSearchInput struct {
	MovieID int `json:"movie_id" jsonschema_description:"Movie ID to search releases for"`
}

type updateMovieMonitoringInput struct {
	MovieID   int  `json:"movie_id" jsonschema_description:"Radarr movie ID"`
	Monitored bool `json:"monitored" jsonschema_description:"Whether the movie should be monitored"`
}

type deleteMovieInput struct {
	MovieID     int  `json:"movie_id" jsonschema_description:"Radarr movie ID to delete"`
	DeleteFiles bool `json:"delete_files,omitempty" jsonschema_description:"Whether to delete movie files from disk (default false)"`
}

type triggerMovieSearchInput struct {
	MovieID int `json:"movie_id" jsonschema_description:"Movie ID to trigger a search for"`
}

type grabMovieReleaseInput struct {
	GUID      string `json:"guid" jsonschema_description:"Release GUID from manual search results"`
	IndexerID int    `json:"indexer_id" jsonschema_description:"Indexer ID from manual search results"`
}

type updateMovieProfileInput struct {
	MovieID          int `json:"movie_id" jsonschema_description:"Radarr movie ID"`
	QualityProfileID int `json:"quality_profile_id" jsonschema_description:"Quality profile ID to assign. Use get_movie_quality_profiles to find available IDs."`
}

type removeMovieBlocklistItemInput struct {
	ID int `json:"id" jsonschema_description:"Blocklist item ID to remove"`
}

type updateMovieLanguageProfileInput struct {
	MovieID           int `json:"movie_id" jsonschema_description:"Radarr movie ID"`
	LanguageProfileID int `json:"language_profile_id" jsonschema_description:"Language profile ID to assign"`
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
		{
			Param: anthropic.ToolParam{
				Name:        "get_movie_detail",
				Description: anthropic.String("Get detailed information about a specific movie including file status, quality profile, and root folder path."),
				InputSchema: generateSchema[getMovieDetailInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_movie_quality_profiles",
				Description: anthropic.String("List all quality profiles configured in Radarr with their cutoff settings."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_movie_root_folders",
				Description: anthropic.String("List Radarr root folders with free and total disk space."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_movie_download_clients",
				Description: anthropic.String("List download clients configured in Radarr with their status, protocol, and priority."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_movie_blocklist",
				Description: anthropic.String("Get blocklisted releases that Radarr will not re-download."),
				InputSchema: generateSchema[getMovieBlocklistInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "manual_movie_search",
				Description: anthropic.String("Search for available releases for a specific movie. Shows indexer, quality, size, and rejection reasons."),
				InputSchema: generateSchema[manualMovieSearchInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "update_movie_monitoring",
				Description: anthropic.String("Update the monitoring status of a movie in Radarr. Fetches the movie, sets the monitored flag, and saves it back."),
				InputSchema: generateSchema[updateMovieMonitoringInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "delete_movie",
				Description: anthropic.String("Delete a movie from Radarr. Optionally delete files from disk."),
				InputSchema: generateSchema[deleteMovieInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "trigger_movie_search",
				Description: anthropic.String("Trigger an automatic search for a movie in Radarr. Sends a MoviesSearch command."),
				InputSchema: generateSchema[triggerMovieSearchInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "grab_movie_release",
				Description: anthropic.String("Grab a specific release for a movie from manual search results."),
				InputSchema: generateSchema[grabMovieReleaseInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "remove_movie_blocklist_item",
				Description: anthropic.String("Remove an item from the Radarr blocklist, allowing it to be downloaded again."),
				InputSchema: generateSchema[removeMovieBlocklistItemInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "update_movie_profile",
				Description: anthropic.String("Update the quality profile for a movie. Use get_movie_quality_profiles to find available profile IDs and get_movie_detail to see the current profile."),
				InputSchema: generateSchema[updateMovieProfileInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_movie_language_profiles",
				Description: anthropic.String("List all language profiles configured in Radarr."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_movie_custom_formats",
				Description: anthropic.String("List all custom formats configured in Radarr with their specifications."),
				InputSchema: generateSchema[struct{}](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "update_movie_language_profile",
				Description: anthropic.String("Update the language profile assigned to a movie. Fetches the movie, sets the language profile, and saves it back."),
				InputSchema: generateSchema[updateMovieLanguageProfileInput](),
			},
			Destructive: true,
		},
	}
}

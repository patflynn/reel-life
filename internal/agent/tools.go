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
	"add_series":    true,
	"remove_failed": true,
}

// IsDestructive reports whether the named tool modifies state.
func IsDestructive(name string) bool {
	return destructiveTools[name]
}

func allToolDefs() []toolDef {
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
				Description: anthropic.String("Remove a failed download from the queue. Optionally blocklist the release to prevent re-downloading."),
				InputSchema: generateSchema[removeFailedInput](),
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

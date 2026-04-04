package agent

import "github.com/anthropics/anthropic-sdk-go"

type testIndexerInput struct {
	ID int `json:"id" jsonschema_description:"Indexer ID to test connectivity for"`
}

type searchIndexersInput struct {
	Query string `json:"query" jsonschema_description:"Search query to run across all indexers"`
}

type enableIndexerInput struct {
	IndexerID int  `json:"indexer_id" jsonschema_description:"Indexer ID to enable or disable"`
	Enabled   bool `json:"enabled" jsonschema_description:"Set to true to enable, false to disable"`
}

type updateIndexerPriorityInput struct {
	IndexerID int `json:"indexer_id" jsonschema_description:"Indexer ID to update"`
	Priority  int `json:"priority" jsonschema_description:"New priority value for the indexer"`
}

type deleteIndexerInput struct {
	IndexerID int `json:"indexer_id" jsonschema_description:"Indexer ID to remove"`
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
		{
			Param: anthropic.ToolParam{
				Name:        "enable_indexer",
				Description: anthropic.String("Enable or disable a Prowlarr indexer."),
				InputSchema: generateSchema[enableIndexerInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "update_indexer_priority",
				Description: anthropic.String("Change the priority of a Prowlarr indexer."),
				InputSchema: generateSchema[updateIndexerPriorityInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "delete_indexer",
				Description: anthropic.String("Remove an indexer from Prowlarr."),
				InputSchema: generateSchema[deleteIndexerInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "test_all_indexers",
				Description: anthropic.String("Test all configured indexers at once."),
				InputSchema: generateSchema[struct{}](),
			},
		},
	}
}

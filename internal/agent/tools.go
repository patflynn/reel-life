package agent

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/invopop/jsonschema"
)

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
	"add_series":                  true,
	"remove_failed":               true,
	"update_series_monitoring":    true,
	"trigger_series_search":       true,
	"delete_series":               true,
	"remove_blocklist_item":       true,
	"grab_release":                true,
	"update_episode_monitoring":   true,
	"monitor_season_episodes":     true,
	"add_movie":                   true,
	"remove_failed_movie":         true,
	"update_movie_monitoring":     true,
	"delete_movie":                true,
	"trigger_movie_search":        true,
	"grab_movie_release":          true,
	"remove_movie_blocklist_item":    true,
	"update_series_profile":          true,
	"update_movie_profile":           true,
	"update_movie_language_profile":  true,
	"update_series_language_profile": true,
	"approve_request":             true,
	"decline_request":             true,
	"delete_request":              true,
	"retry_request":               true,
	"notebook_write":              true,
	"notebook_delete":             true,
	"enable_indexer":              true,
	"update_indexer_priority":     true,
	"delete_indexer":              true,
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

func toolDefinitions() []anthropic.ToolUnionParam {
	defs := allToolDefs()
	result := make([]anthropic.ToolUnionParam, len(defs))
	for i, d := range defs {
		p := d.Param
		result[i] = anthropic.ToolUnionParam{OfTool: &p}
	}
	return result
}

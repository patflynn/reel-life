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

// toolDef pairs an Anthropic tool definition with rate-limiting flags.
type toolDef struct {
	Param       anthropic.ToolParam
	Mutative    bool // state-changing but safe/additive
	Destructive bool // removes or deletes data (implicitly also mutative)
}

// mutativeTools is the set of tools that change state but are additive/safe.
var mutativeTools = map[string]bool{
	"add_series":                     true,
	"add_movie":                      true,
	"approve_request":                true,
	"decline_request":                true,
	"retry_request":                  true,
	"notebook_write":                 true,
	"notebook_delete":                true,
	"update_series_monitoring":       true,
	"update_episode_monitoring":      true,
	"monitor_season_episodes":        true,
	"trigger_series_search":          true,
	"trigger_movie_search":           true,
	"update_movie_monitoring":        true,
	"grab_release":                   true,
	"grab_movie_release":             true,
	"update_series_profile":          true,
	"update_movie_profile":           true,
	"update_movie_language_profile":  true,
	"update_series_language_profile": true,
	"enable_indexer":                 true,
	"update_indexer_priority":        true,
}

// destructiveTools is the set of tools that remove or delete data.
// Destructive tools are implicitly also mutative.
var destructiveTools = map[string]bool{
	"remove_failed":              true,
	"remove_failed_movie":        true,
	"delete_series":              true,
	"delete_movie":               true,
	"remove_blocklist_item":      true,
	"remove_movie_blocklist_item": true,
	"delete_request":             true,
	"delete_indexer":             true,
}

// IsMutative reports whether the named tool changes state.
// Destructive tools are implicitly mutative.
func IsMutative(name string) bool {
	return mutativeTools[name] || destructiveTools[name]
}

// IsDestructive reports whether the named tool removes or deletes data.
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

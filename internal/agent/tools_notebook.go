package agent

import "github.com/anthropics/anthropic-sdk-go"

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

func notebookToolDefs() []toolDef {
	return []toolDef{
		{
			Param: anthropic.ToolParam{
				Name:        "notebook_write",
				Description: anthropic.String("Create or update a note in the persistent notebook. Use pinned type for high-signal info that should always be visible; use reference for detailed info looked up on demand. If updating, provide the existing note's ID."),
				InputSchema: generateSchema[notebookWriteInput](),
			},
			Mutative: true,
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
			Mutative: true,
		},
	}
}

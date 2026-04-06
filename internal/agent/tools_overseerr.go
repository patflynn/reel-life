package agent

import "github.com/anthropics/anthropic-sdk-go"

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

type getRequestDetailInput struct {
	ID int `json:"id" jsonschema_description:"Request ID to get details for"`
}

type deleteRequestInput struct {
	ID int `json:"id" jsonschema_description:"Request ID to delete"`
}

type retryRequestInput struct {
	ID int `json:"id" jsonschema_description:"Request ID to retry"`
}

type searchMediaInput struct {
	Query string `json:"query" jsonschema_description:"Search query for movies or TV shows"`
	Page  int    `json:"page,omitempty" jsonschema_description:"Page number for results (default 1)"`
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
			Mutative: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "decline_request",
				Description: anthropic.String("Decline a pending media request in Overseerr."),
				InputSchema: generateSchema[declineRequestInput](),
			},
			Mutative: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "get_request_detail",
				Description: anthropic.String("Get detailed information about a specific Overseerr media request."),
				InputSchema: generateSchema[getRequestDetailInput](),
			},
		},
		{
			Param: anthropic.ToolParam{
				Name:        "delete_request",
				Description: anthropic.String("Delete a media request from Overseerr."),
				InputSchema: generateSchema[deleteRequestInput](),
			},
			Destructive: true,
		},
		{
			Param: anthropic.ToolParam{
				Name:        "retry_request",
				Description: anthropic.String("Retry a failed Overseerr media request."),
				InputSchema: generateSchema[retryRequestInput](),
			},
			Mutative: true,
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

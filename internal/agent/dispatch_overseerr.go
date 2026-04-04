package agent

import (
	"context"
	"encoding/json"
)

func (a *Agent) dispatchOverseerr(ctx context.Context, name string, rawInput json.RawMessage) (string, bool, bool) {
	if a.overseerr == nil {
		switch name {
		case "list_requests", "approve_request", "decline_request", "get_request_detail",
			"delete_request", "retry_request", "get_request_count", "search_media":
			return jsonError("Overseerr integration is not configured"), true, true
		}
	}

	var result any
	var err error

	switch name {
	case "list_requests":
		var input listRequestsInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		take := input.Take
		if take == 0 {
			take = 20
		}
		result, err = a.overseerr.ListRequests(ctx, input.Filter, take, input.Skip)
	case "approve_request":
		var input approveRequestInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		err = a.overseerr.ApproveRequest(ctx, input.ID)
		if err == nil {
			result = map[string]string{"status": "approved"}
		}
	case "decline_request":
		var input declineRequestInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		err = a.overseerr.DeclineRequest(ctx, input.ID)
		if err == nil {
			result = map[string]string{"status": "declined"}
		}
	case "get_request_detail":
		var input getRequestDetailInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.overseerr.GetRequest(ctx, input.ID)
	case "delete_request":
		var input deleteRequestInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		err = a.overseerr.DeleteRequest(ctx, input.ID)
		if err == nil {
			result = map[string]string{"status": "deleted"}
		}
	case "retry_request":
		var input retryRequestInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.overseerr.RetryRequest(ctx, input.ID)
	case "get_request_count":
		result, err = a.overseerr.GetRequestCount(ctx)
	case "search_media":
		var input searchMediaInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		page := input.Page
		if page == 0 {
			page = 1
		}
		result, err = a.overseerr.SearchMedia(ctx, input.Query, page)
	default:
		return "", false, false
	}

	if err != nil {
		a.logger.Warn("tool error", "tool", name, "error", err)
		return jsonError(err.Error()), true, true
	}

	data, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return jsonError("failed to marshal result: " + marshalErr.Error()), true, true
	}
	return string(data), false, true
}

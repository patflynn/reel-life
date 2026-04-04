package agent

import (
	"context"
	"encoding/json"
)

func (a *Agent) dispatchProwlarr(ctx context.Context, name string, rawInput json.RawMessage) (string, bool, bool) {
	if a.prowlarr == nil {
		switch name {
		case "list_indexers", "test_indexer", "test_all_indexers", "get_indexer_stats", "check_indexer_health",
			"search_indexers", "enable_indexer", "update_indexer_priority", "delete_indexer":
			return jsonError("Prowlarr integration is not configured"), true, true
		}
	}

	var result any
	var err error

	switch name {
	case "list_indexers":
		result, err = a.prowlarr.ListIndexers(ctx)
	case "test_indexer":
		var input testIndexerInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		err = a.prowlarr.TestIndexer(ctx, input.ID)
		if err == nil {
			result = map[string]string{"status": "ok"}
		}
	case "test_all_indexers":
		result, err = a.prowlarr.TestAllIndexers(ctx)
	case "get_indexer_stats":
		result, err = a.prowlarr.GetIndexerStats(ctx)
	case "check_indexer_health":
		result, err = a.prowlarr.CheckHealth(ctx)
	case "search_indexers":
		var input searchIndexersInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.prowlarr.Search(ctx, input.Query)
	case "enable_indexer":
		var input enableIndexerInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		found, errStr := a.findIndexer(ctx, input.IndexerID)
		if errStr != "" {
			return errStr, true, true
		}
		found.Enable = input.Enabled
		result, err = a.prowlarr.UpdateIndexer(ctx, found)
	case "update_indexer_priority":
		var input updateIndexerPriorityInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		found, errStr := a.findIndexer(ctx, input.IndexerID)
		if errStr != "" {
			return errStr, true, true
		}
		found.Priority = input.Priority
		result, err = a.prowlarr.UpdateIndexer(ctx, found)
	case "delete_indexer":
		var input deleteIndexerInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		err = a.prowlarr.DeleteIndexer(ctx, input.IndexerID)
		if err == nil {
			result = map[string]string{"status": "deleted"}
		}
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

package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/patflynn/reel-life/internal/sonarr"
)

func (a *Agent) dispatchSonarr(ctx context.Context, name string, rawInput json.RawMessage) (string, bool, bool) {
	if a.sonarr == nil {
		switch name {
		case "search_series", "add_series", "get_queue", "get_history", "check_health", "remove_failed",
			"get_series_detail", "get_episodes", "get_logs", "manual_search", "get_quality_profiles",
			"get_blocklist", "get_root_folders", "get_download_clients", "update_series_monitoring",
			"trigger_series_search", "delete_series", "remove_blocklist_item", "grab_release":
			return jsonError("Sonarr integration is not configured"), true, true
		}
	}

	var result any
	var err error

	switch name {
	case "search_series":
		var input searchSeriesInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.sonarr.Search(ctx, input.Term)

	case "add_series":
		var input addSeriesInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.sonarr.Add(ctx, sonarr.AddSeriesRequest{
			Title:            input.Title,
			TVDBID:           input.TVDBID,
			QualityProfileID: input.QualityProfileID,
			RootFolderPath:   input.RootFolderPath,
			Monitored:        true,
			SeasonFolder:     true,
		})

	case "get_queue":
		result, err = a.sonarr.Queue(ctx)

	case "get_history":
		var input getHistoryInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		pageSize := input.PageSize
		if pageSize == 0 {
			pageSize = 20
		}
		result, err = a.sonarr.History(ctx, pageSize)

	case "check_health":
		result, err = a.sonarr.Health(ctx)

	case "remove_failed":
		var input removeFailedInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		err = a.sonarr.RemoveFailed(ctx, input.ID, input.Blocklist)
		if err == nil {
			result = map[string]string{"status": "removed"}
		}

	case "get_series_detail":
		var input getSeriesDetailInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.sonarr.GetSeries(ctx, input.SeriesID)

	case "get_episodes":
		var input getEpisodesInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.sonarr.GetEpisodes(ctx, input.SeriesID)

	case "get_logs":
		var input getLogsInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.sonarr.GetLogs(ctx, input.PageSize, input.Level)

	case "manual_search":
		var input manualSearchInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.sonarr.ManualSearch(ctx, input.EpisodeID)

	case "get_quality_profiles":
		result, err = a.sonarr.GetQualityProfiles(ctx)

	case "get_blocklist":
		var input getBlocklistInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.sonarr.GetBlocklist(ctx, input.PageSize)

	case "get_root_folders":
		result, err = a.sonarr.GetRootFolders(ctx)

	case "get_download_clients":
		result, err = a.sonarr.GetDownloadClients(ctx)

	case "update_series_monitoring":
		var input updateSeriesMonitoringInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		series, getErr := a.sonarr.GetSeries(ctx, input.SeriesID)
		if getErr != nil {
			return jsonError(getErr.Error()), true, true
		}
		found := false
		for i, s := range series.Seasons {
			if s.SeasonNumber == input.SeasonNumber {
				series.Seasons[i].Monitored = input.Monitored
				found = true
				break
			}
		}
		if !found {
			return jsonError(fmt.Sprintf("season %d not found in series %q", input.SeasonNumber, series.Title)), true, true
		}
		result, err = a.sonarr.UpdateSeries(ctx, series)

	case "trigger_series_search":
		var input triggerSeriesSearchInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		cmd := sonarr.CommandRequest{
			Name:     "SeriesSearch",
			SeriesID: input.SeriesID,
		}
		if input.SeasonNumber != nil {
			cmd.Name = "SeasonSearch"
			cmd.SeasonNumber = input.SeasonNumber
		}
		result, err = a.sonarr.Command(ctx, cmd)

	case "delete_series":
		var input deleteSeriesInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		err = a.sonarr.DeleteSeries(ctx, input.SeriesID, input.DeleteFiles)
		if err == nil {
			result = map[string]string{"status": "deleted"}
		}

	case "remove_blocklist_item":
		var input removeBlocklistItemInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		err = a.sonarr.DeleteBlocklistItem(ctx, input.ID)
		if err == nil {
			result = map[string]string{"status": "removed"}
		}

	case "grab_release":
		var input grabReleaseInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return jsonError("invalid input: " + err.Error()), true, true
		}
		result, err = a.sonarr.GrabRelease(ctx, input.GUID, input.IndexerID)

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

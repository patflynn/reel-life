package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/patflynn/reel-life/internal/sonarr"
)

// mockSonarr implements sonarr.Client for agent testing.
type mockSonarr struct {
	searchResult  []sonarr.Series
	healthResult  []sonarr.HealthCheck
	queueResult   *sonarr.QueuePage
	historyResult *sonarr.HistoryPage
}

func (m *mockSonarr) Search(_ context.Context, _ string) ([]sonarr.Series, error) {
	return m.searchResult, nil
}
func (m *mockSonarr) Add(_ context.Context, req sonarr.AddSeriesRequest) (*sonarr.Series, error) {
	return &sonarr.Series{ID: 1, Title: req.Title, TVDBID: req.TVDBID}, nil
}
func (m *mockSonarr) Queue(_ context.Context) (*sonarr.QueuePage, error) {
	return m.queueResult, nil
}
func (m *mockSonarr) History(_ context.Context, _ int) (*sonarr.HistoryPage, error) {
	return m.historyResult, nil
}
func (m *mockSonarr) Health(_ context.Context) ([]sonarr.HealthCheck, error) {
	return m.healthResult, nil
}
func (m *mockSonarr) RemoveFailed(_ context.Context, _ int, _ bool) error {
	return nil
}

func TestDispatchSearchSeries(t *testing.T) {
	mock := &mockSonarr{
		searchResult: []sonarr.Series{
			{ID: 1, Title: "Breaking Bad", Year: 2008, TVDBID: 81189},
		},
	}
	a := &Agent{sonarr: mock, logger: slog.Default()}

	input, _ := json.Marshal(searchSeriesInput{Term: "breaking bad"})
	result, isErr := a.dispatchTool(context.Background(), "search_series", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var series []sonarr.Series
	if err := json.Unmarshal([]byte(result), &series); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(series) != 1 || series[0].Title != "Breaking Bad" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestDispatchCheckHealth(t *testing.T) {
	mock := &mockSonarr{
		healthResult: []sonarr.HealthCheck{
			{Source: "IndexerCheck", Type: "warning", Message: "test warning"},
		},
	}
	a := &Agent{sonarr: mock, logger: slog.Default()}

	input, _ := json.Marshal(struct{}{})
	result, isErr := a.dispatchTool(context.Background(), "check_health", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var checks []sonarr.HealthCheck
	if err := json.Unmarshal([]byte(result), &checks); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(checks))
	}
}

func TestDispatchRemoveFailed(t *testing.T) {
	mock := &mockSonarr{}
	a := &Agent{sonarr: mock, logger: slog.Default()}

	input, _ := json.Marshal(removeFailedInput{ID: 42, Blocklist: true})
	result, isErr := a.dispatchTool(context.Background(), "remove_failed", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var status map[string]string
	json.Unmarshal([]byte(result), &status)
	if status["status"] != "removed" {
		t.Errorf("expected status=removed, got %s", result)
	}
}

func TestDispatchUnknownTool(t *testing.T) {
	a := &Agent{sonarr: &mockSonarr{}}

	result, isErr := a.dispatchTool(context.Background(), "nonexistent", json.RawMessage("{}"))
	if !isErr {
		t.Fatal("expected error for unknown tool")
	}
	if result == "" {
		t.Fatal("expected error message")
	}
}

func TestDispatchGetQueue(t *testing.T) {
	mock := &mockSonarr{
		queueResult: &sonarr.QueuePage{
			TotalRecords: 2,
			Records: []sonarr.QueueItem{
				{ID: 1, Title: "S01E01", Status: "downloading"},
				{ID: 2, Title: "S01E02", Status: "queued"},
			},
		},
	}
	a := &Agent{sonarr: mock, logger: slog.Default()}

	input, _ := json.Marshal(struct{}{})
	result, isErr := a.dispatchTool(context.Background(), "get_queue", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var queue sonarr.QueuePage
	if err := json.Unmarshal([]byte(result), &queue); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if queue.TotalRecords != 2 {
		t.Errorf("expected 2 records, got %d", queue.TotalRecords)
	}
}

func TestDispatchAddSeries(t *testing.T) {
	mock := &mockSonarr{}
	a := &Agent{sonarr: mock, logger: slog.Default()}

	input, _ := json.Marshal(addSeriesInput{
		Title:            "Breaking Bad",
		TVDBID:           81189,
		QualityProfileID: 1,
		RootFolderPath:   "/tv",
	})
	result, isErr := a.dispatchTool(context.Background(), "add_series", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var series sonarr.Series
	if err := json.Unmarshal([]byte(result), &series); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if series.Title != "Breaking Bad" {
		t.Errorf("Title = %q, want %q", series.Title, "Breaking Bad")
	}
}

func TestDispatchGetHistory(t *testing.T) {
	mock := &mockSonarr{
		historyResult: &sonarr.HistoryPage{
			TotalRecords: 1,
			Records: []sonarr.HistoryRecord{
				{ID: 1, SourceTitle: "Test.S01E01", EventType: "grabbed"},
			},
		},
	}
	a := &Agent{sonarr: mock, logger: slog.Default()}

	input, _ := json.Marshal(getHistoryInput{PageSize: 10})
	result, isErr := a.dispatchTool(context.Background(), "get_history", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var history sonarr.HistoryPage
	if err := json.Unmarshal([]byte(result), &history); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if history.TotalRecords != 1 {
		t.Errorf("expected 1 record, got %d", history.TotalRecords)
	}
}

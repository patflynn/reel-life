package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/patflynn/reel-life/internal/prowlarr"
	"github.com/patflynn/reel-life/internal/radarr"
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

// mockRadarr implements radarr.Client for agent testing.
type mockRadarr struct {
	searchResult  []radarr.Movie
	healthResult  []radarr.HealthCheck
	queueResult   *radarr.QueuePage
	historyResult *radarr.HistoryPage
}

func (m *mockRadarr) Search(_ context.Context, _ string) ([]radarr.Movie, error) {
	return m.searchResult, nil
}
func (m *mockRadarr) Add(_ context.Context, req radarr.AddMovieRequest) (*radarr.Movie, error) {
	return &radarr.Movie{ID: 1, Title: req.Title, TMDBID: req.TMDBID}, nil
}
func (m *mockRadarr) Queue(_ context.Context) (*radarr.QueuePage, error) {
	return m.queueResult, nil
}
func (m *mockRadarr) History(_ context.Context, _ int) (*radarr.HistoryPage, error) {
	return m.historyResult, nil
}
func (m *mockRadarr) Health(_ context.Context) ([]radarr.HealthCheck, error) {
	return m.healthResult, nil
}
func (m *mockRadarr) RemoveFailed(_ context.Context, _ int, _ bool) error {
	return nil
}

// mockProwlarr implements prowlarr.Client for agent testing.
type mockProwlarr struct {
	indexers     []prowlarr.Indexer
	stats        *prowlarr.IndexerStats
	health       []prowlarr.HealthCheck
	searchResult []prowlarr.SearchResult
}

func (m *mockProwlarr) ListIndexers(_ context.Context) ([]prowlarr.Indexer, error) {
	return m.indexers, nil
}
func (m *mockProwlarr) TestIndexer(_ context.Context, _ int) error {
	return nil
}
func (m *mockProwlarr) GetIndexerStats(_ context.Context) (*prowlarr.IndexerStats, error) {
	return m.stats, nil
}
func (m *mockProwlarr) CheckHealth(_ context.Context) ([]prowlarr.HealthCheck, error) {
	return m.health, nil
}
func (m *mockProwlarr) Search(_ context.Context, _ string) ([]prowlarr.SearchResult, error) {
	return m.searchResult, nil
}

func newTestAgent() *Agent {
	return &Agent{sonarr: &mockSonarr{}, radarr: &mockRadarr{}, prowlarr: &mockProwlarr{}, logger: slog.Default()}
}

func TestDispatchSearchSeries(t *testing.T) {
	mock := &mockSonarr{
		searchResult: []sonarr.Series{
			{ID: 1, Title: "Breaking Bad", Year: 2008, TVDBID: 81189},
		},
	}
	a := newTestAgent()
	a.sonarr = mock

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
	a := newTestAgent()
	a.sonarr = mock

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
	a := newTestAgent()

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
	a := &Agent{sonarr: &mockSonarr{}, radarr: &mockRadarr{}, prowlarr: &mockProwlarr{}}

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
	a := newTestAgent()
	a.sonarr = mock

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
	a := newTestAgent()

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
	a := newTestAgent()
	a.sonarr = mock

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

func TestDispatchSearchMovies(t *testing.T) {
	mock := &mockRadarr{
		searchResult: []radarr.Movie{
			{ID: 1, Title: "Inception", Year: 2010, TMDBID: 27205},
		},
	}
	a := newTestAgent()
	a.radarr = mock

	input, _ := json.Marshal(searchMoviesInput{Term: "inception"})
	result, isErr := a.dispatchTool(context.Background(), "search_movies", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var movies []radarr.Movie
	if err := json.Unmarshal([]byte(result), &movies); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(movies) != 1 || movies[0].Title != "Inception" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestDispatchAddMovie(t *testing.T) {
	a := newTestAgent()

	input, _ := json.Marshal(addMovieInput{
		Title:            "Inception",
		TMDBID:           27205,
		QualityProfileID: 1,
		RootFolderPath:   "/movies",
	})
	result, isErr := a.dispatchTool(context.Background(), "add_movie", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var movie radarr.Movie
	if err := json.Unmarshal([]byte(result), &movie); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if movie.Title != "Inception" {
		t.Errorf("Title = %q, want %q", movie.Title, "Inception")
	}
}

func TestDispatchGetMovieQueue(t *testing.T) {
	mock := &mockRadarr{
		queueResult: &radarr.QueuePage{
			TotalRecords: 1,
			Records: []radarr.QueueItem{
				{ID: 1, Title: "Inception.2010.1080p", Status: "downloading"},
			},
		},
	}
	a := newTestAgent()
	a.radarr = mock

	input, _ := json.Marshal(struct{}{})
	result, isErr := a.dispatchTool(context.Background(), "get_movie_queue", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var queue radarr.QueuePage
	if err := json.Unmarshal([]byte(result), &queue); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if queue.TotalRecords != 1 {
		t.Errorf("expected 1 record, got %d", queue.TotalRecords)
	}
}

func TestDispatchCheckMovieHealth(t *testing.T) {
	mock := &mockRadarr{
		healthResult: []radarr.HealthCheck{
			{Source: "IndexerCheck", Type: "warning", Message: "test warning"},
		},
	}
	a := newTestAgent()
	a.radarr = mock

	input, _ := json.Marshal(struct{}{})
	result, isErr := a.dispatchTool(context.Background(), "check_movie_health", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var checks []radarr.HealthCheck
	if err := json.Unmarshal([]byte(result), &checks); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(checks))
	}
}

func TestDispatchRemoveFailedMovie(t *testing.T) {
	a := newTestAgent()

	input, _ := json.Marshal(removeFailedMovieInput{ID: 42, Blocklist: true})
	result, isErr := a.dispatchTool(context.Background(), "remove_failed_movie", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var status map[string]string
	json.Unmarshal([]byte(result), &status)
	if status["status"] != "removed" {
		t.Errorf("expected status=removed, got %s", result)
	}
}

func TestDispatchGetMovieHistory(t *testing.T) {
	mock := &mockRadarr{
		historyResult: &radarr.HistoryPage{
			TotalRecords: 1,
			Records: []radarr.HistoryRecord{
				{ID: 1, SourceTitle: "Inception.2010.1080p", EventType: "grabbed"},
			},
		},
	}
	a := newTestAgent()
	a.radarr = mock

	input, _ := json.Marshal(getMovieHistoryInput{PageSize: 10})
	result, isErr := a.dispatchTool(context.Background(), "get_movie_history", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var history radarr.HistoryPage
	if err := json.Unmarshal([]byte(result), &history); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if history.TotalRecords != 1 {
		t.Errorf("expected 1 record, got %d", history.TotalRecords)
	}
}

func TestDispatchListIndexers(t *testing.T) {
	mock := &mockProwlarr{
		indexers: []prowlarr.Indexer{
			{ID: 1, Name: "NZBgeek", Enable: true, Protocol: "usenet", Priority: 25},
		},
	}
	a := newTestAgent()
	a.prowlarr = mock

	input, _ := json.Marshal(struct{}{})
	result, isErr := a.dispatchTool(context.Background(), "list_indexers", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var indexers []prowlarr.Indexer
	if err := json.Unmarshal([]byte(result), &indexers); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(indexers) != 1 || indexers[0].Name != "NZBgeek" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestDispatchTestIndexer(t *testing.T) {
	a := newTestAgent()

	input, _ := json.Marshal(testIndexerInput{ID: 5})
	result, isErr := a.dispatchTool(context.Background(), "test_indexer", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var status map[string]string
	json.Unmarshal([]byte(result), &status)
	if status["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", result)
	}
}

func TestDispatchGetIndexerStats(t *testing.T) {
	mock := &mockProwlarr{
		stats: &prowlarr.IndexerStats{
			Indexers: []prowlarr.IndexerStatEntry{
				{IndexerID: 1, IndexerName: "NZBgeek", NumberOfQueries: 100},
			},
		},
	}
	a := newTestAgent()
	a.prowlarr = mock

	input, _ := json.Marshal(struct{}{})
	result, isErr := a.dispatchTool(context.Background(), "get_indexer_stats", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var stats prowlarr.IndexerStats
	if err := json.Unmarshal([]byte(result), &stats); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(stats.Indexers) != 1 {
		t.Errorf("expected 1 indexer stat, got %d", len(stats.Indexers))
	}
}

func TestDispatchCheckIndexerHealth(t *testing.T) {
	mock := &mockProwlarr{
		health: []prowlarr.HealthCheck{
			{Source: "IndexerStatusCheck", Type: "warning", Message: "test"},
		},
	}
	a := newTestAgent()
	a.prowlarr = mock

	input, _ := json.Marshal(struct{}{})
	result, isErr := a.dispatchTool(context.Background(), "check_indexer_health", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var checks []prowlarr.HealthCheck
	if err := json.Unmarshal([]byte(result), &checks); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(checks))
	}
}

func TestDispatchSearchIndexers(t *testing.T) {
	mock := &mockProwlarr{
		searchResult: []prowlarr.SearchResult{
			{GUID: "abc123", IndexerID: 1, Title: "Breaking.Bad.S01E01", Size: 1400000000},
		},
	}
	a := newTestAgent()
	a.prowlarr = mock

	input, _ := json.Marshal(searchIndexersInput{Query: "breaking bad"})
	result, isErr := a.dispatchTool(context.Background(), "search_indexers", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	var results []prowlarr.SearchResult
	if err := json.Unmarshal([]byte(result), &results); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Breaking.Bad.S01E01" {
		t.Errorf("unexpected result: %s", result)
	}
}

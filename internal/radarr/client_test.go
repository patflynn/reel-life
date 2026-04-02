package radarr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/movie/lookup" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("term") != "inception" {
			t.Errorf("unexpected term: %s", r.URL.Query().Get("term"))
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("missing or wrong API key header")
		}
		json.NewEncoder(w).Encode([]Movie{
			{ID: 1, Title: "Inception", Year: 2010, TMDBID: 27205},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	results, err := client.Search(context.Background(), "inception")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Inception" {
		t.Errorf("Title = %q, want %q", results[0].Title, "Inception")
	}
}

func TestAdd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v3/movie" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req AddMovieRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.TMDBID != 27205 {
			t.Errorf("TMDBID = %d, want 27205", req.TMDBID)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Movie{ID: 42, Title: req.Title, TMDBID: req.TMDBID})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	movie, err := client.Add(context.Background(), AddMovieRequest{
		Title:  "Inception",
		TMDBID: 27205,
	})
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if movie.ID != 42 {
		t.Errorf("ID = %d, want 42", movie.ID)
	}
}

func TestQueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/queue" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(QueuePage{
			TotalRecords: 1,
			Records: []QueueItem{
				{ID: 1, Title: "Inception.2010.1080p", Status: "downloading"},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	queue, err := client.Queue(context.Background())
	if err != nil {
		t.Fatalf("Queue() error: %v", err)
	}
	if queue.TotalRecords != 1 {
		t.Errorf("TotalRecords = %d, want 1", queue.TotalRecords)
	}
}

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]HealthCheck{
			{Source: "IndexerStatusCheck", Type: "warning", Message: "Indexer is unavailable"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	checks, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(checks))
	}
	if checks[0].Type != "warning" {
		t.Errorf("Type = %q, want %q", checks[0].Type, "warning")
	}
}

func TestRemoveFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v3/queue/123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("blocklist") != "true" {
			t.Errorf("blocklist = %s, want true", r.URL.Query().Get("blocklist"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	err := client.RemoveFailed(context.Background(), 123, true)
	if err != nil {
		t.Fatalf("RemoveFailed() error: %v", err)
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/history" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pageSize") != "5" {
			t.Errorf("pageSize = %s, want 5", r.URL.Query().Get("pageSize"))
		}
		json.NewEncoder(w).Encode(HistoryPage{
			TotalRecords: 1,
			Records: []HistoryRecord{
				{ID: 1, SourceTitle: "Inception.2010.1080p", EventType: "grabbed"},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	history, err := client.History(context.Background(), 5)
	if err != nil {
		t.Fatalf("History() error: %v", err)
	}
	if history.TotalRecords != 1 {
		t.Errorf("TotalRecords = %d, want 1", history.TotalRecords)
	}
}

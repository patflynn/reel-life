package prowlarr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListIndexers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/indexer" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("missing or wrong API key header")
		}
		json.NewEncoder(w).Encode([]Indexer{
			{ID: 1, Name: "NZBgeek", Enable: true, Protocol: "usenet", Priority: 25},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	results, err := client.ListIndexers(context.Background())
	if err != nil {
		t.Fatalf("ListIndexers() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "NZBgeek" {
		t.Errorf("Name = %q, want %q", results[0].Name, "NZBgeek")
	}
	if results[0].Protocol != "usenet" {
		t.Errorf("Protocol = %q, want %q", results[0].Protocol, "usenet")
	}
}

func TestTestIndexer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/indexer/test" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]int
		json.NewDecoder(r.Body).Decode(&body)
		if body["id"] != 5 {
			t.Errorf("id = %d, want 5", body["id"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	err := client.TestIndexer(context.Background(), 5)
	if err != nil {
		t.Fatalf("TestIndexer() error: %v", err)
	}
}

func TestGetIndexerStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/indexerstats" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(IndexerStats{
			Indexers: []IndexerStatEntry{
				{IndexerID: 1, IndexerName: "NZBgeek", NumberOfQueries: 100, NumberOfGrabs: 50},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	stats, err := client.GetIndexerStats(context.Background())
	if err != nil {
		t.Fatalf("GetIndexerStats() error: %v", err)
	}
	if len(stats.Indexers) != 1 {
		t.Fatalf("expected 1 indexer stat, got %d", len(stats.Indexers))
	}
	if stats.Indexers[0].NumberOfQueries != 100 {
		t.Errorf("NumberOfQueries = %d, want 100", stats.Indexers[0].NumberOfQueries)
	}
}

func TestCheckHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]HealthCheck{
			{Source: "IndexerStatusCheck", Type: "warning", Message: "Indexer is unavailable"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	checks, err := client.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth() error: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(checks))
	}
	if checks[0].Type != "warning" {
		t.Errorf("Type = %q, want %q", checks[0].Type, "warning")
	}
}

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "breaking bad" {
			t.Errorf("unexpected query: %s", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("type") != "search" {
			t.Errorf("unexpected type: %s", r.URL.Query().Get("type"))
		}
		json.NewEncoder(w).Encode([]SearchResult{
			{GUID: "abc123", IndexerID: 1, Title: "Breaking.Bad.S01E01", Size: 1400000000},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	results, err := client.Search(context.Background(), "breaking bad")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Breaking.Bad.S01E01" {
		t.Errorf("Title = %q, want %q", results[0].Title, "Breaking.Bad.S01E01")
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	_, err := client.CheckHealth(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

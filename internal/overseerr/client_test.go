package overseerr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/request" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("filter") != "pending" {
			t.Errorf("filter = %s, want pending", r.URL.Query().Get("filter"))
		}
		if r.URL.Query().Get("take") != "10" {
			t.Errorf("take = %s, want 10", r.URL.Query().Get("take"))
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("missing or wrong API key header")
		}
		json.NewEncoder(w).Encode(RequestPage{
			PageInfo: PageInfo{Pages: 1, Page: 1, Results: 1},
			Results: []Request{
				{ID: 1, Status: 1, Type: "tv", RequestedBy: UserInfo{DisplayName: "alice"}},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	page, err := client.ListRequests(context.Background(), "pending", 10, 0)
	if err != nil {
		t.Fatalf("ListRequests() error: %v", err)
	}
	if len(page.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(page.Results))
	}
	if page.Results[0].RequestedBy.DisplayName != "alice" {
		t.Errorf("DisplayName = %q, want %q", page.Results[0].RequestedBy.DisplayName, "alice")
	}
}

func TestApproveRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/request/42/approve" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("missing or wrong API key header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	err := client.ApproveRequest(context.Background(), 42)
	if err != nil {
		t.Fatalf("ApproveRequest() error: %v", err)
	}
}

func TestDeclineRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/request/42/decline" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	err := client.DeclineRequest(context.Background(), 42)
	if err != nil {
		t.Fatalf("DeclineRequest() error: %v", err)
	}
}

func TestGetRequestCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/request/count" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(RequestCount{
			Pending: 5, Approved: 10, Declined: 2, Total: 17,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	count, err := client.GetRequestCount(context.Background())
	if err != nil {
		t.Fatalf("GetRequestCount() error: %v", err)
	}
	if count.Pending != 5 {
		t.Errorf("Pending = %d, want 5", count.Pending)
	}
	if count.Total != 17 {
		t.Errorf("Total = %d, want 17", count.Total)
	}
}

func TestSearchMedia(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "the matrix" {
			t.Errorf("query = %s, want 'the matrix'", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("page = %s, want 1", r.URL.Query().Get("page"))
		}
		json.NewEncoder(w).Encode(SearchResults{
			Page:         1,
			TotalPages:   1,
			TotalResults: 1,
			Results: []SearchResult{
				{ID: 603, MediaType: "movie", Title: "The Matrix", Overview: "A computer hacker learns..."},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	results, err := client.SearchMedia(context.Background(), "the matrix", 1)
	if err != nil {
		t.Fatalf("SearchMedia() error: %v", err)
	}
	if results.TotalResults != 1 {
		t.Fatalf("TotalResults = %d, want 1", results.TotalResults)
	}
	if results.Results[0].Title != "The Matrix" {
		t.Errorf("Title = %q, want %q", results.Results[0].Title, "The Matrix")
	}
}

func TestOverseerrAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	_, err := client.GetRequestCount(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

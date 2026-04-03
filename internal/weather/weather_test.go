package weather

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWMODescription(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "clear sky"},
		{1, "partly cloudy"},
		{2, "partly cloudy"},
		{3, "partly cloudy"},
		{45, "fog"},
		{48, "fog"},
		{51, "drizzle"},
		{55, "drizzle"},
		{61, "rain"},
		{65, "rain"},
		{71, "snow"},
		{75, "snow"},
		{80, "showers"},
		{82, "showers"},
		{95, "thunderstorm"},
		{99, "thunderstorm"},
		{100, "unknown"},
	}
	for _, tt := range tests {
		got := WMODescription(tt.code)
		if got != tt.want {
			t.Errorf("WMODescription(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestCurrentCaching(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]any{
			"current": map[string]any{
				"temperature_2m": 22.5,
				"weather_code":   1,
			},
		})
	}))
	defer srv.Close()

	c := NewClient(45.5, -73.5, "Montreal, QC")
	c.httpClient = srv.Client()
	// Override the fetch URL by replacing the HTTP client's transport.
	c.httpClient.Transport = &rewriteTransport{base: srv.URL}

	ctx := context.Background()

	cond := c.Current(ctx)
	if cond == nil {
		t.Fatal("expected conditions, got nil")
	}
	if cond.Temperature != 22.5 {
		t.Errorf("Temperature = %f, want 22.5", cond.Temperature)
	}
	if cond.Description != "partly cloudy" {
		t.Errorf("Description = %q, want %q", cond.Description, "partly cloudy")
	}

	// Second call should use cache (no additional HTTP request).
	cond2 := c.Current(ctx)
	if cond2 == nil {
		t.Fatal("expected cached conditions, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}
}

func TestCurrentGracefulDegradation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(45.5, -73.5, "Montreal, QC")
	c.httpClient = srv.Client()
	c.httpClient.Transport = &rewriteTransport{base: srv.URL}

	ctx := context.Background()
	cond := c.Current(ctx)
	if cond != nil {
		t.Errorf("expected nil on failure with no cache, got %+v", cond)
	}
}

func TestCurrentReturnsStaleOnFailure(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode(map[string]any{
				"current": map[string]any{
					"temperature_2m": 15.0,
					"weather_code":   0,
				},
			})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	c := NewClient(45.5, -73.5, "Montreal, QC")
	c.httpClient = srv.Client()
	c.httpClient.Transport = &rewriteTransport{base: srv.URL}
	c.cacheTTL = 0 // Force cache expiry immediately.

	ctx := context.Background()

	// First call succeeds.
	cond := c.Current(ctx)
	if cond == nil || cond.Temperature != 15.0 {
		t.Fatalf("expected 15.0, got %+v", cond)
	}

	// Second call fails but should return stale cache.
	cond2 := c.Current(ctx)
	if cond2 == nil {
		t.Fatal("expected stale cached conditions, got nil")
	}
	if cond2.Temperature != 15.0 {
		t.Errorf("expected stale temp 15.0, got %f", cond2.Temperature)
	}
}

func TestLocation(t *testing.T) {
	c := NewClient(45.5, -73.5, "Montreal, QC")
	if c.Location() != "Montreal, QC" {
		t.Errorf("Location() = %q, want %q", c.Location(), "Montreal, QC")
	}
}

// rewriteTransport redirects all requests to the test server.
type rewriteTransport struct {
	base string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = t.base[len("http://"):]
	// Preserve path and query from original request.
	return http.DefaultTransport.RoundTrip(req)
}

func TestCacheTTLRespected(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]any{
			"current": map[string]any{
				"temperature_2m": float64(callCount) * 10,
				"weather_code":   0,
			},
		})
	}))
	defer srv.Close()

	c := NewClient(45.5, -73.5, "Montreal, QC")
	c.httpClient = srv.Client()
	c.httpClient.Transport = &rewriteTransport{base: srv.URL}

	ctx := context.Background()
	c.Current(ctx)

	// Expire the cache manually.
	c.mu.Lock()
	c.cached.FetchedAt = time.Now().Add(-31 * time.Minute)
	c.mu.Unlock()

	c.Current(ctx)
	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls after cache expiry, got %d", callCount)
	}
}

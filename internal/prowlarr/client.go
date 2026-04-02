package prowlarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client defines the operations available against a Prowlarr instance.
type Client interface {
	ListIndexers(ctx context.Context) ([]Indexer, error)
	TestIndexer(ctx context.Context, id int) error
	GetIndexerStats(ctx context.Context) (*IndexerStats, error)
	CheckHealth(ctx context.Context) ([]HealthCheck, error)
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

// HTTPClient implements Client using Prowlarr's v1 REST API.
type HTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *HTTPClient {
	return &HTTPClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *HTTPClient) ListIndexers(ctx context.Context) ([]Indexer, error) {
	u := c.url("/api/v1/indexer")

	var result []Indexer
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("list indexers: %w", err)
	}
	return result, nil
}

func (c *HTTPClient) TestIndexer(ctx context.Context, id int) error {
	u := c.url("/api/v1/indexer/test")

	body, err := json.Marshal(map[string]int{"id": id})
	if err != nil {
		return fmt.Errorf("marshal test request: %w", err)
	}

	if err := c.post(ctx, u.String(), body, nil); err != nil {
		return fmt.Errorf("test indexer: %w", err)
	}
	return nil
}

func (c *HTTPClient) GetIndexerStats(ctx context.Context) (*IndexerStats, error) {
	u := c.url("/api/v1/indexerstats")

	var result IndexerStats
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get indexer stats: %w", err)
	}
	return &result, nil
}

func (c *HTTPClient) CheckHealth(ctx context.Context) ([]HealthCheck, error) {
	u := c.url("/api/v1/health")

	var result []HealthCheck
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("health check: %w", err)
	}
	return result, nil
}

func (c *HTTPClient) Search(ctx context.Context, query string) ([]SearchResult, error) {
	u := c.url("/api/v1/search")
	q := u.Query()
	q.Set("query", query)
	q.Set("type", "search")
	u.RawQuery = q.Encode()

	var result []SearchResult
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("search indexers: %w", err)
	}
	return result, nil
}

func (c *HTTPClient) url(path string) *url.URL {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil
	}
	return u.JoinPath(path)
}

func (c *HTTPClient) get(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *HTTPClient) post(ctx context.Context, rawURL string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *HTTPClient) do(req *http.Request, out any) error {
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

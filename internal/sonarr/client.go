package sonarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Client defines the operations available against a Sonarr instance.
type Client interface {
	Search(ctx context.Context, term string) ([]Series, error)
	Add(ctx context.Context, req AddSeriesRequest) (*Series, error)
	Queue(ctx context.Context) (*QueuePage, error)
	History(ctx context.Context, pageSize int) (*HistoryPage, error)
	Health(ctx context.Context) ([]HealthCheck, error)
	RemoveFailed(ctx context.Context, id int, blocklist bool) error
	GetSeries(ctx context.Context, seriesID int) (*Series, error)
	GetEpisodes(ctx context.Context, seriesID int) ([]Episode, error)
	GetLogs(ctx context.Context, pageSize int, level string) ([]LogRecord, error)
	ManualSearch(ctx context.Context, episodeID int) ([]Release, error)
	GetQualityProfiles(ctx context.Context) ([]QualityProfile, error)
	GetBlocklist(ctx context.Context, pageSize int) (*BlocklistPage, error)
	GetRootFolders(ctx context.Context) ([]RootFolder, error)
	GetDownloadClients(ctx context.Context) ([]DownloadClient, error)
}

// HTTPClient implements Client using Sonarr's v3 REST API.
type HTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *HTTPClient {
	return &HTTPClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (c *HTTPClient) Search(ctx context.Context, term string) ([]Series, error) {
	u := c.url("/api/v3/series/lookup")
	q := u.Query()
	q.Set("term", term)
	u.RawQuery = q.Encode()

	var result []Series
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("search series: %w", err)
	}
	return result, nil
}

func (c *HTTPClient) Add(ctx context.Context, req AddSeriesRequest) (*Series, error) {
	u := c.url("/api/v3/series")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal add request: %w", err)
	}

	var result Series
	if err := c.post(ctx, u.String(), body, &result); err != nil {
		return nil, fmt.Errorf("add series: %w", err)
	}
	return &result, nil
}

func (c *HTTPClient) Queue(ctx context.Context) (*QueuePage, error) {
	u := c.url("/api/v3/queue")

	var result QueuePage
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get queue: %w", err)
	}
	return &result, nil
}

func (c *HTTPClient) History(ctx context.Context, pageSize int) (*HistoryPage, error) {
	u := c.url("/api/v3/history")
	// 0 means "not specified" (JSON omitempty zero value); let the server use its default.
	if pageSize > 0 {
		q := u.Query()
		q.Set("pageSize", strconv.Itoa(pageSize))
		u.RawQuery = q.Encode()
	}

	var result HistoryPage
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}
	return &result, nil
}

func (c *HTTPClient) Health(ctx context.Context) ([]HealthCheck, error) {
	u := c.url("/api/v3/health")

	var result []HealthCheck
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("health check: %w", err)
	}
	return result, nil
}

func (c *HTTPClient) RemoveFailed(ctx context.Context, id int, blocklist bool) error {
	u := c.url(fmt.Sprintf("/api/v3/queue/%d", id))
	q := u.Query()
	q.Set("removeFromClient", "true")
	q.Set("blocklist", strconv.FormatBool(blocklist))
	u.RawQuery = q.Encode()

	if err := c.delete(ctx, u.String()); err != nil {
		return fmt.Errorf("remove failed: %w", err)
	}
	return nil
}

func (c *HTTPClient) GetSeries(ctx context.Context, seriesID int) (*Series, error) {
	u := c.url(fmt.Sprintf("/api/v3/series/%d", seriesID))

	var result Series
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get series: %w", err)
	}
	return &result, nil
}

func (c *HTTPClient) GetEpisodes(ctx context.Context, seriesID int) ([]Episode, error) {
	u := c.url("/api/v3/episode")
	q := u.Query()
	q.Set("seriesId", strconv.Itoa(seriesID))
	u.RawQuery = q.Encode()

	var result []Episode
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get episodes: %w", err)
	}
	return result, nil
}

func (c *HTTPClient) GetLogs(ctx context.Context, pageSize int, level string) ([]LogRecord, error) {
	u := c.url("/api/v3/log")
	q := u.Query()
	q.Set("sortDirection", "descending")
	// 0 means "not specified" (JSON omitempty zero value); let the server use its default.
	if pageSize > 0 {
		q.Set("pageSize", strconv.Itoa(pageSize))
	}
	if level != "" {
		q.Set("filterKey", "level")
		q.Set("filterValue", level)
	}
	u.RawQuery = q.Encode()

	var result LogPage
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get logs: %w", err)
	}
	return result.Records, nil
}

func (c *HTTPClient) ManualSearch(ctx context.Context, episodeID int) ([]Release, error) {
	u := c.url("/api/v3/release")
	q := u.Query()
	q.Set("episodeId", strconv.Itoa(episodeID))
	u.RawQuery = q.Encode()

	var result []Release
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("manual search: %w", err)
	}
	return result, nil
}

func (c *HTTPClient) GetQualityProfiles(ctx context.Context) ([]QualityProfile, error) {
	u := c.url("/api/v3/qualityprofile")

	var result []QualityProfile
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get quality profiles: %w", err)
	}
	return result, nil
}

func (c *HTTPClient) GetBlocklist(ctx context.Context, pageSize int) (*BlocklistPage, error) {
	u := c.url("/api/v3/blocklist")
	// 0 means "not specified" (JSON omitempty zero value); let the server use its default.
	if pageSize > 0 {
		q := u.Query()
		q.Set("pageSize", strconv.Itoa(pageSize))
		u.RawQuery = q.Encode()
	}

	var result BlocklistPage
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get blocklist: %w", err)
	}
	return &result, nil
}

func (c *HTTPClient) GetRootFolders(ctx context.Context) ([]RootFolder, error) {
	u := c.url("/api/v3/rootfolder")

	var result []RootFolder
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get root folders: %w", err)
	}
	return result, nil
}

func (c *HTTPClient) GetDownloadClients(ctx context.Context) ([]DownloadClient, error) {
	u := c.url("/api/v3/downloadclient")

	var result []DownloadClient
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get download clients: %w", err)
	}
	return result, nil
}

func (c *HTTPClient) url(path string) *url.URL {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		panic(fmt.Sprintf("sonarr: failed to parse URL: %v", err))
	}
	return u
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

func (c *HTTPClient) delete(ctx context.Context, rawURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, rawURL, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *HTTPClient) do(req *http.Request, out any) error {
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("API error %d: failed to read response body: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

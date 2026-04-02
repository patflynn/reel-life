package overseerr

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

// Client defines the operations available against an Overseerr instance.
type Client interface {
	ListRequests(ctx context.Context, filter string, take, skip int) (*RequestPage, error)
	ApproveRequest(ctx context.Context, id int) error
	DeclineRequest(ctx context.Context, id int) error
	GetRequestCount(ctx context.Context) (*RequestCount, error)
	SearchMedia(ctx context.Context, query string, page int) (*SearchResults, error)
}

// HTTPClient implements Client using Overseerr's v1 REST API.
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

func (c *HTTPClient) ListRequests(ctx context.Context, filter string, take, skip int) (*RequestPage, error) {
	u := c.url("/api/v1/request")
	q := u.Query()
	if take > 0 {
		q.Set("take", strconv.Itoa(take))
	}
	if skip > 0 {
		q.Set("skip", strconv.Itoa(skip))
	}
	if filter != "" {
		q.Set("filter", filter)
	}
	u.RawQuery = q.Encode()

	var result RequestPage
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("list requests: %w", err)
	}
	return &result, nil
}

func (c *HTTPClient) ApproveRequest(ctx context.Context, id int) error {
	u := c.url(fmt.Sprintf("/api/v1/request/%d/approve", id))

	if err := c.post(ctx, u.String(), nil, nil); err != nil {
		return fmt.Errorf("approve request: %w", err)
	}
	return nil
}

func (c *HTTPClient) DeclineRequest(ctx context.Context, id int) error {
	u := c.url(fmt.Sprintf("/api/v1/request/%d/decline", id))

	if err := c.post(ctx, u.String(), nil, nil); err != nil {
		return fmt.Errorf("decline request: %w", err)
	}
	return nil
}

func (c *HTTPClient) GetRequestCount(ctx context.Context) (*RequestCount, error) {
	u := c.url("/api/v1/request/count")

	var result RequestCount
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("get request count: %w", err)
	}
	return &result, nil
}

func (c *HTTPClient) SearchMedia(ctx context.Context, query string, page int) (*SearchResults, error) {
	u := c.url("/api/v1/search")
	q := u.Query()
	q.Set("query", query)
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	u.RawQuery = q.Encode()

	var result SearchResults
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, fmt.Errorf("search media: %w", err)
	}
	return &result, nil
}

func (c *HTTPClient) url(path string) *url.URL {
	u, _ := url.Parse(c.baseURL + path)
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
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
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
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

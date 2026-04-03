package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Conditions represents the current weather at a location.
type Conditions struct {
	Temperature float64
	Description string
	FetchedAt   time.Time
}

// Client fetches and caches weather data from Open-Meteo.
type Client struct {
	latitude   float64
	longitude  float64
	location   string
	mu         sync.RWMutex
	cached     *Conditions
	cacheTTL   time.Duration
	httpClient *http.Client
}

// NewClient creates a weather client for the given coordinates.
func NewClient(latitude, longitude float64, locationName string) *Client {
	return &Client{
		latitude:  latitude,
		longitude: longitude,
		location:  locationName,
		cacheTTL:  30 * time.Minute,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetHTTPClient replaces the default HTTP client (useful for testing).
func (c *Client) SetHTTPClient(hc *http.Client) {
	c.httpClient = hc
}

// Location returns the display name for this client's location.
func (c *Client) Location() string {
	return c.location
}

// Current returns the current weather conditions, using cache if fresh.
// It never returns an error — on failure it returns cached data or nil.
func (c *Client) Current(ctx context.Context) *Conditions {
	c.mu.RLock()
	if c.cached != nil && time.Since(c.cached.FetchedAt) < c.cacheTTL {
		cached := c.cached
		c.mu.RUnlock()
		return cached
	}
	c.mu.RUnlock()

	conditions := c.fetch(ctx)
	if conditions != nil {
		c.mu.Lock()
		c.cached = conditions
		c.mu.Unlock()
		return conditions
	}

	// Return stale cache on fetch failure.
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cached
}

type openMeteoResponse struct {
	Current struct {
		Temperature float64 `json:"temperature_2m"`
		WeatherCode int     `json:"weather_code"`
	} `json:"current"`
}

func (c *Client) fetch(ctx context.Context) *Conditions {
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current=temperature_2m,weather_code&timezone=auto",
		c.latitude, c.longitude,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var data openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}

	return &Conditions{
		Temperature: data.Current.Temperature,
		Description: WMODescription(data.Current.WeatherCode),
		FetchedAt:   time.Now(),
	}
}

// WMODescription maps a WMO weather code to a human-readable description.
func WMODescription(code int) string {
	switch {
	case code == 0:
		return "clear sky"
	case code >= 1 && code <= 3:
		return "partly cloudy"
	case code >= 45 && code <= 48:
		return "fog"
	case code >= 51 && code <= 55:
		return "drizzle"
	case code >= 56 && code <= 57:
		return "freezing drizzle"
	case code >= 61 && code <= 65:
		return "rain"
	case code >= 66 && code <= 67:
		return "freezing rain"
	case code >= 71 && code <= 75:
		return "snow"
	case code == 77:
		return "snow grains"
	case code >= 80 && code <= 82:
		return "showers"
	case code >= 85 && code <= 86:
		return "snow showers"
	case code >= 95 && code <= 99:
		return "thunderstorm"
	default:
		return "unknown"
	}
}

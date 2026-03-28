package agent

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter enforces per-minute and per-request limits on tool calls,
// with a separate limit for destructive (state-modifying) tools.
type RateLimiter struct {
	maxCallsPerMinute  int
	maxCallsPerRequest int
	destructiveLimit   int

	minuteCount      int
	requestCount     int
	destructiveCount int
	windowStart      time.Time
	mu               sync.Mutex
}

// NewRateLimiter creates a RateLimiter with the given limits.
func NewRateLimiter(maxPerMin, maxPerReq, destructiveMax int) *RateLimiter {
	return &RateLimiter{
		maxCallsPerMinute:  maxPerMin,
		maxCallsPerRequest: maxPerReq,
		destructiveLimit:   destructiveMax,
		windowStart:        time.Now(),
	}
}

// Allow checks whether a tool call is permitted. It returns an error describing
// which limit was exceeded, or nil if the call is allowed.
func (r *RateLimiter) Allow(toolName string, isDestructive bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Slide the per-minute window.
	now := time.Now()
	if now.Sub(r.windowStart) >= time.Minute {
		r.minuteCount = 0
		r.windowStart = now
	}

	if r.minuteCount >= r.maxCallsPerMinute {
		return fmt.Errorf("rate limit exceeded: max %d tool calls per minute. Refusing to execute %s", r.maxCallsPerMinute, toolName)
	}

	if r.requestCount >= r.maxCallsPerRequest {
		return fmt.Errorf("rate limit exceeded: max %d tool calls per request. Refusing to execute %s", r.maxCallsPerRequest, toolName)
	}

	if isDestructive && r.destructiveCount >= r.destructiveLimit {
		return fmt.Errorf("rate limit exceeded: max %d destructive actions per request. Refusing to execute %s", r.destructiveLimit, toolName)
	}

	r.minuteCount++
	r.requestCount++
	if isDestructive {
		r.destructiveCount++
	}
	return nil
}

// Reset clears per-request counters. Call this at the start of each new request.
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requestCount = 0
	r.destructiveCount = 0
}

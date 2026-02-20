// Package client wraps the generated Strava API client with auth injection and retry logic.
package client

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/Brainsoft-Raxat/strava-cli/internal/auth"
	"github.com/Brainsoft-Raxat/strava-cli/internal/config"
)

const maxRetries = 3

// baseBackoff is a variable so tests can override it to avoid slow sleeps.
var baseBackoff = 500 * time.Millisecond //nolint:gochecknoglobals

// SetBaseBackoff overrides the base backoff duration and returns the previous value.
// Intended for use in tests only.
func SetBaseBackoff(d time.Duration) time.Duration {
	prev := baseBackoff
	baseBackoff = d
	return prev
}

// retryTransport injects the Bearer token and retries on 429/5xx with exponential backoff.
type retryTransport struct {
	cfg  *config.Config
	base http.RoundTripper
}

// NewHTTPClient returns an *http.Client that:
//   - refreshes the token if expired before each request
//   - injects Authorization: Bearer <token>
//   - retries on HTTP 429 and 5xx with exponential backoff
func NewHTTPClient(cfg *config.Config) *http.Client {
	return &http.Client{
		Transport: &retryTransport{
			cfg:  cfg,
			base: http.DefaultTransport,
		},
		Timeout: 30 * time.Second,
	}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Ensure token is fresh before the first attempt.
	if err := auth.RefreshIfExpired(t.cfg); err != nil {
		return nil, err
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(math.Pow(2, float64(attempt-1))) * baseBackoff
			time.Sleep(wait)
			// Re-check token freshness on retry (it may have expired mid-flow).
			if rerr := auth.RefreshIfExpired(t.cfg); rerr != nil {
				return nil, rerr
			}
		}

		// Clone request so we can add headers safely across retries.
		cloned := req.Clone(req.Context())
		cloned.Header.Set("Authorization", "Bearer "+t.cfg.Tokens.AccessToken)

		// Reset body for retries (POST/PUT bodies are consumed on the first attempt).
		if attempt > 0 && req.GetBody != nil {
			newBody, gbErr := req.GetBody()
			if gbErr != nil {
				return nil, fmt.Errorf("reset request body for retry: %w", gbErr)
			}
			cloned.Body = newBody
		}

		resp, err = t.base.RoundTrip(cloned)
		if err != nil {
			// Network errors are not retried.
			return nil, fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			_ = resp.Body.Close()
			if attempt == maxRetries {
				return nil, fmt.Errorf("HTTP %d after %d retries — Strava API may be temporarily unavailable", resp.StatusCode, maxRetries)
			}
			continue
		}

		return resp, nil
	}

	// Unreachable, but satisfies compiler.
	return resp, err
}

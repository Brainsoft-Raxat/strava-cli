package client_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	genclient "github.com/Brainsoft-Raxat/strava-cli/internal/client"
	"github.com/Brainsoft-Raxat/strava-cli/internal/config"
)

func freshConfig() *config.Config {
	return &config.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Tokens: config.Tokens{
			AccessToken:  "test-access",
			RefreshToken: "test-refresh",
			ExpiresAt:    time.Now().Add(1 * time.Hour).Unix(),
		},
	}
}

func TestRetryTransport_SuccessOnFirstAttempt(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		// Verify Authorization header is set
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := genclient.NewHTTPClient(freshConfig())
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRetryTransport_RetriesOn429(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Override backoff to zero for speed.
	orig := genclient.SetBaseBackoff(0)
	defer genclient.SetBaseBackoff(orig)

	c := genclient.NewHTTPClient(freshConfig())
	_, err := c.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("expected 3 calls (2 retries), got %d", calls)
	}
}

func TestRetryTransport_RetriesOn500(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	orig := genclient.SetBaseBackoff(0)
	defer genclient.SetBaseBackoff(orig)

	c := genclient.NewHTTPClient(freshConfig())
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("expected 2 calls (1 retry), got %d", calls)
	}
}

func TestRetryTransport_ExhaustsRetries(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	orig := genclient.SetBaseBackoff(0)
	defer genclient.SetBaseBackoff(orig)

	c := genclient.NewHTTPClient(freshConfig())
	_, err := c.Get(srv.URL)
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	// maxRetries=3, so 4 total attempts (0..3)
	if atomic.LoadInt32(&calls) != 4 {
		t.Errorf("expected 4 calls, got %d", calls)
	}
}

func TestRetryTransport_BearerTokenInjected(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := freshConfig()
	cfg.Tokens.AccessToken = "my-secret-token"
	c := genclient.NewHTTPClient(cfg)
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if gotHeader != "Bearer my-secret-token" {
		t.Errorf("Authorization header = %q, want %q", gotHeader, "Bearer my-secret-token")
	}
}

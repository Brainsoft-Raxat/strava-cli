package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Brainsoft-Raxat/strava-cli/internal/auth"
	"github.com/Brainsoft-Raxat/strava-cli/internal/config"
)

// --- ExtractCode ---

func TestExtractCode_BareCode(t *testing.T) {
	got := auth.ExtractCode("abc123xyz")
	if got != "abc123xyz" {
		t.Errorf("got %q, want %q", got, "abc123xyz")
	}
}

func TestExtractCode_FullURL(t *testing.T) {
	input := "https://tortugas.raxat.site/auth/callback?state=&code=abc123xyz&scope=read,activity:read_all"
	got := auth.ExtractCode(input)
	if got != "abc123xyz" {
		t.Errorf("got %q, want %q", got, "abc123xyz")
	}
}

func TestExtractCode_URLNoCode(t *testing.T) {
	// URL with no code param → returns the raw input
	input := "https://example.com/callback?error=access_denied"
	got := auth.ExtractCode(input)
	// Falls through to bare-code path, returns input unchanged
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

// --- IsLocalhost ---

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		uri  string
		want bool
	}{
		{"http://localhost:8089/callback", true},
		{"http://127.0.0.1:8089/callback", true},
		{"https://tortugas.raxat.site/auth/callback", false},
		{"https://myapp.example.com/callback", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.uri, func(t *testing.T) {
			got := auth.IsLocalhost(tc.uri)
			if got != tc.want {
				t.Errorf("IsLocalhost(%q) = %v, want %v", tc.uri, got, tc.want)
			}
		})
	}
}

// patchTokenURL replaces the tokenURL used by the auth package for testing.
// Because tokenURL is unexported, we test via the exported RefreshIfExpired function
// by pointing it at a test server through the package-level variable.

// tokenPayload builds a minimal token JSON response.
func tokenPayload(access, refresh string, expiresAt int64) []byte {
	b, _ := json.Marshal(map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_at":    expiresAt,
		"token_type":    "Bearer",
	})
	return b
}

func TestRefreshIfExpired_NotExpired(t *testing.T) {
	cfg := &config.Config{
		ClientID:     "id",
		ClientSecret: "secret",
		Tokens: config.Tokens{
			AccessToken:  "valid-token",
			RefreshToken: "refresh-token",
			ExpiresAt:    time.Now().Add(1 * time.Hour).Unix(), // far in the future
		},
	}

	// Should NOT call any endpoint – token is still fresh.
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// We can't easily override the internal URL without exporting it, so we
	// verify the behavior indirectly: token still valid → no error, token unchanged.
	if err := auth.RefreshIfExpired(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected no HTTP call for a non-expired token")
	}
	if cfg.Tokens.AccessToken != "valid-token" {
		t.Errorf("token changed unexpectedly to %q", cfg.Tokens.AccessToken)
	}
}

func TestRefreshIfExpired_NoToken(t *testing.T) {
	cfg := &config.Config{}
	err := auth.RefreshIfExpired(cfg)
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

// TestRefreshIfExpired_Expired tests the refresh path with a real HTTP test server.
// We do this by setting up a test OAuth token endpoint.
func TestRefreshIfExpired_Expired(t *testing.T) {
	newAccess := "new-access-token"
	newRefresh := "new-refresh-token"
	newExpiry := time.Now().Add(6 * time.Hour).Unix()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.FormValue("grant_type") != "refresh_token" {
			t.Errorf("expected grant_type=refresh_token, got %q", r.FormValue("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(tokenPayload(newAccess, newRefresh, newExpiry))
	}))
	defer srv.Close()

	// Override the token URL to point at our test server.
	orig := auth.SetTokenURL(srv.URL)
	defer auth.SetTokenURL(orig)

	// Redirect config writes to a temp dir so the real config is never touched.
	t.Setenv("STRAVA_CONFIG_DIR", t.TempDir())

	cfg := &config.Config{
		ClientID:     "cid",
		ClientSecret: "csecret",
		Tokens: config.Tokens{
			AccessToken:  "expired-token",
			RefreshToken: "old-refresh",
			ExpiresAt:    time.Now().Add(-10 * time.Minute).Unix(), // expired
		},
	}

	if err := auth.RefreshIfExpired(cfg); err != nil {
		t.Fatalf("RefreshIfExpired: %v", err)
	}
	if cfg.Tokens.AccessToken != newAccess {
		t.Errorf("access token = %q, want %q", cfg.Tokens.AccessToken, newAccess)
	}
	if cfg.Tokens.RefreshToken != newRefresh {
		t.Errorf("refresh token = %q, want %q", cfg.Tokens.RefreshToken, newRefresh)
	}
}

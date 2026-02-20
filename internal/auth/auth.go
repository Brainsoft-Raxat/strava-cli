// Package auth implements Strava OAuth2 login and automatic token refresh.
package auth

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Brainsoft-Raxat/strava-cli/internal/config"
)

const (
	authURL      = "https://www.strava.com/oauth/authorize"
	redirectHost = "localhost"
	redirectPort = "8089"
	scopes       = "activity:read_all,activity:write"
)

// tokenURL is a variable so tests can override it with httptest servers.
var tokenURL = "https://www.strava.com/oauth/token" //nolint:gochecknoglobals

// SetTokenURL overrides the token endpoint URL and returns the previous value.
// Intended for use in tests only.
func SetTokenURL(u string) string {
	prev := tokenURL
	tokenURL = u
	return prev
}

// Login initiates the OAuth2 authorization code flow.
//
// If redirectURI is empty or a localhost URI, a local callback server is started
// on port 8089 to capture the code automatically.
//
// If redirectURI is an external HTTPS URI, the auth URL is displayed and the user
// is prompted to paste either the full callback URL or just the code.
func Login(clientID, clientSecret, redirectURI string) (*config.Tokens, error) {
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("http://%s:%s/callback", redirectHost, redirectPort)
	}

	if isLocalhost(redirectURI) {
		return loginLocal(clientID, clientSecret, redirectURI)
	}
	return loginManual(clientID, clientSecret, redirectURI)
}

// IsLocalhost reports whether u is a localhost URI. Exported for testing.
func IsLocalhost(u string) bool { return isLocalhost(u) }

// isLocalhost reports whether u is a localhost/127.0.0.1 URI.
func isLocalhost(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	return host == "localhost" || host == "127.0.0.1"
}

// loginLocal starts a local HTTP server to capture the OAuth callback automatically.
func loginLocal(clientID, clientSecret, redirectURI string) (*config.Tokens, error) {
	authLink := buildAuthURL(clientID, redirectURI)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	parsed, _ := url.Parse(redirectURI)
	path := parsed.Path
	if path == "" {
		path = "/callback"
	}

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if e := r.URL.Query().Get("error"); e != "" {
			errCh <- fmt.Errorf("authorization denied: %s", e)
			fmt.Fprintf(w, "<html><body><h2>Authorization failed: %s</h2><p>You may close this tab.</p></body></html>", e)
			return
		}
		codeCh <- r.URL.Query().Get("code")
		fmt.Fprintf(w, "<html><body><h2>Authorization successful!</h2><p>You may close this tab.</p></body></html>")
	})

	port := parsed.Port()
	if port == "" {
		port = redirectPort
	}
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("start callback server on :%s: %w\n  Hint: ensure port %s is free", port, err, port)
	}
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	fmt.Println("Open the following URL in your browser to authorize strava-cli:")
	fmt.Println()
	fmt.Println(" ", authLink)
	fmt.Println()
	fmt.Println("Waiting for callback...")

	var code string
	select {
	case code = <-codeCh:
	case err = <-errCh:
		_ = srv.Shutdown(context.Background())
		return nil, err
	case <-time.After(5 * time.Minute):
		_ = srv.Shutdown(context.Background())
		return nil, fmt.Errorf("authorization timed out after 5 minutes")
	}
	_ = srv.Shutdown(context.Background())

	return exchangeCode(clientID, clientSecret, code, redirectURI)
}

// loginManual displays the auth URL and asks the user to paste back the code.
// The user may paste either the full callback URL (containing ?code=...) or
// just the bare authorization code.
func loginManual(clientID, clientSecret, redirectURI string) (*config.Tokens, error) {
	authLink := buildAuthURL(clientID, redirectURI)

	fmt.Println("Open the following URL in your browser to authorize strava-cli:")
	fmt.Println()
	fmt.Println(" ", authLink)
	fmt.Println()
	fmt.Println("After authorizing, Strava will redirect you to:")
	fmt.Printf("  %s?code=<code>&...\n", redirectURI)
	fmt.Println()
	fmt.Print("Paste the full redirect URL (or just the code): ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	pasted := strings.TrimSpace(scanner.Text())

	code := extractCode(pasted)
	if code == "" {
		return nil, fmt.Errorf("no authorization code found in %q\n  Hint: paste the full redirect URL or just the code value", pasted)
	}

	return exchangeCode(clientID, clientSecret, code, redirectURI)
}

// ExtractCode parses an authorization code from either a full URL or a bare code string.
// Exported for testing.
func ExtractCode(input string) string { return extractCode(input) }

// extractCode parses an authorization code from either a full URL or a bare code string.
func extractCode(input string) string {
	if strings.Contains(input, "?") || strings.Contains(input, "&") {
		parsed, err := url.Parse(input)
		if err == nil {
			if c := parsed.Query().Get("code"); c != "" {
				return c
			}
		}
	}
	// Treat the whole input as the code.
	return input
}

func buildAuthURL(clientID, redirectURI string) string {
	params := url.Values{
		"client_id":       {clientID},
		"redirect_uri":    {redirectURI},
		"response_type":   {"code"},
		"approval_prompt": {"auto"},
		"scope":           {scopes},
	}
	return authURL + "?" + params.Encode()
}

// RefreshIfExpired checks whether the access token is expired (with a 30s buffer)
// and refreshes it if necessary, updating cfg in-place and saving it.
func RefreshIfExpired(cfg *config.Config) error {
	if cfg.Tokens.AccessToken == "" {
		return fmt.Errorf("not authenticated — run: stravacli auth login")
	}
	// Refresh 30 seconds before expiry
	if time.Now().Unix()+30 < cfg.Tokens.ExpiresAt {
		return nil
	}
	tokens, err := refreshTokens(cfg.ClientID, cfg.ClientSecret, cfg.Tokens.RefreshToken)
	if err != nil {
		return fmt.Errorf("refresh token: %w\n  Hint: your session may have been revoked; run: stravacli auth login", err)
	}
	cfg.Tokens = *tokens
	return config.Save(cfg)
}

// tokenResponse is the Strava token endpoint JSON payload.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
	Errors       []struct {
		Field    string `json:"field"`
		Code     string `json:"code"`
		Resource string `json:"resource"`
	} `json:"errors"`
	Message string `json:"message"`
}

func exchangeCode(clientID, clientSecret, code, redirectURI string) (*config.Tokens, error) {
	tokens, err := postToken(url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	})
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	return tokens, nil
}

func refreshTokens(clientID, clientSecret, refreshToken string) (*config.Tokens, error) {
	return postToken(url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	})
}

func postToken(vals url.Values) (*config.Tokens, error) {
	resp, err := http.PostForm(tokenURL, vals)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", tokenURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if tr.Message != "" {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, tr.Message)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("no access_token in response")
	}
	return &config.Tokens{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    tr.ExpiresAt,
		TokenType:    tr.TokenType,
	}, nil
}

// ── Two-step remote / VPS login ───────────────────────────────────────────────

// GenerateState returns a 16-byte cryptographically random hex string suitable
// for use as an OAuth2 CSRF state token.
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// RemoteAuthURL builds a Strava authorization URL that includes a state token.
// Used for step 1 of the two-step remote login flow.
func RemoteAuthURL(clientID, redirectURI, state string) string {
	params := url.Values{
		"client_id":       {clientID},
		"redirect_uri":    {redirectURI},
		"response_type":   {"code"},
		"approval_prompt": {"auto"},
		"scope":           {scopes},
		"state":           {state},
	}
	return authURL + "?" + params.Encode()
}

// CompleteRemoteLogin extracts the authorization code from a pasted redirect URL,
// validates the state token against expectedState, and exchanges the code for tokens.
// Used for step 2 of the two-step remote login flow.
func CompleteRemoteLogin(clientID, clientSecret, redirectURI, expectedState, pastedInput string) (*config.Tokens, error) {
	code, gotState := extractCodeAndState(pastedInput)
	if expectedState != "" {
		if gotState == "" {
			return nil, fmt.Errorf("no state parameter found in the pasted URL\n  Hint: paste the full redirect URL, e.g. http://localhost:8089/callback?code=...&state=...")
		}
		if gotState != expectedState {
			return nil, fmt.Errorf("state mismatch — possible CSRF attack or stale URL\n  Run 'stravacli auth login --remote' again to get a fresh URL")
		}
	}
	if code == "" {
		return nil, fmt.Errorf("no authorization code found in %q\n  Hint: paste the full redirect URL, e.g. http://localhost:8089/callback?code=...&state=...", pastedInput)
	}
	return exchangeCode(clientID, clientSecret, code, redirectURI)
}

// extractCodeAndState parses both the code and state query params from a URL string.
// If the input looks like a URL it is parsed; otherwise it is treated as a bare code.
// Backslash escapes (e.g. \? \& from shell quoting) are stripped before parsing.
func extractCodeAndState(input string) (code, state string) {
	input = strings.TrimSpace(input)
	// Remove shell backslash escapes — they have no place in a valid URL and
	// appear when users paste a URL without single-quoting it correctly.
	input = strings.ReplaceAll(input, `\`, "")
	if strings.Contains(input, "?") || strings.Contains(input, "&") {
		if parsed, err := url.Parse(input); err == nil {
			return parsed.Query().Get("code"), parsed.Query().Get("state")
		}
	}
	// Treat the whole input as a bare authorization code (no state).
	return input, ""
}

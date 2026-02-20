package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/Brainsoft-Raxat/strava-cli/internal/auth"
	"github.com/Brainsoft-Raxat/strava-cli/internal/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Strava via OAuth2",
	Long: `Opens a browser authorization flow and stores tokens locally.

Credentials are read from environment variables if set:
  STRAVA_CLIENT_ID
  STRAVA_CLIENT_SECRET
  STRAVA_REDIRECT_URI   (optional; defaults to http://localhost:8089/callback)

You need a Strava API application. Create one at:
  https://www.strava.com/settings/api

Remote / VPS login (two-step, no browser or open port required on the server):

  # Step 1: generate the auth URL and save CSRF state
  strava auth login --remote

  # Step 2: after authorizing in your local browser, paste the redirect URL back
  strava auth login --auth-url 'http://localhost:8089/callback?code=...&state=...'`,
	RunE: runAuthLogin,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE:  runAuthStatus,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials and tokens",
	RunE:  runAuthLogout,
}

var (
	authRemote  bool
	authPasteURL string
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)

	authLoginCmd.Flags().BoolVar(&authRemote, "remote", false,
		"Two-step remote login: prints auth URL (step 1) or use with --auth-url to complete (step 2)")
	authLoginCmd.Flags().StringVar(&authPasteURL, "auth-url", "",
		"Redirect URL to complete remote login (step 2), e.g. 'http://localhost:8089/callback?code=...&state=...'")
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// --auth-url alone implies step 2 (no need to also pass --remote).
	if authPasteURL != "" {
		return completeRemoteLogin(cfg)
	}

	// Env vars take precedence over stored config.
	if v := os.Getenv("STRAVA_CLIENT_ID"); v != "" {
		cfg.ClientID = v
	}
	if v := os.Getenv("STRAVA_CLIENT_SECRET"); v != "" {
		cfg.ClientSecret = v
	}
	if v := os.Getenv("STRAVA_REDIRECT_URI"); v != "" {
		cfg.RedirectURI = v
	}

	// Fall back to interactive prompts for anything still missing.
	reader := bufio.NewReader(os.Stdin)
	if cfg.ClientID == "" {
		fmt.Print("Strava Client ID: ")
		id, _ := reader.ReadString('\n')
		cfg.ClientID = strings.TrimSpace(id)
	}
	if cfg.ClientSecret == "" {
		fmt.Print("Strava Client Secret: ")
		secret, _ := reader.ReadString('\n')
		cfg.ClientSecret = strings.TrimSpace(secret)
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return fmt.Errorf("client ID and secret are required\n  Set STRAVA_CLIENT_ID / STRAVA_CLIENT_SECRET or use the interactive prompt")
	}

	if authRemote {
		return startRemoteLogin(cfg)
	}

	// Default: local callback server or manual paste (existing behaviour).
	tokens, err := auth.Login(cfg.ClientID, cfg.ClientSecret, cfg.RedirectURI)
	if err != nil {
		return err
	}
	cfg.Tokens = *tokens
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Println("Successfully authenticated! Tokens stored in ~/.config/strava-cli/config.json")
	return nil
}

// startRemoteLogin is step 1: generate CSRF state, print auth URL, persist state.
func startRemoteLogin(cfg *config.Config) error {
	state, err := auth.GenerateState()
	if err != nil {
		return err
	}

	redirectURI := cfg.RedirectURI
	if redirectURI == "" {
		redirectURI = "http://localhost:8089/callback"
	}

	authURL := auth.RemoteAuthURL(cfg.ClientID, redirectURI, state)

	// Persist state, redirect URI, and a 10-minute expiry so step 2 can validate them.
	cfg.PendingAuth = &config.PendingAuth{
		State:       state,
		RedirectURI: redirectURI,
		ExpiresAt:   time.Now().Add(10 * time.Minute).Unix(),
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save pending state: %w", err)
	}

	fmt.Println("Open the following URL in your local browser:")
	fmt.Println()
	fmt.Println(" ", authURL)
	fmt.Println()
	fmt.Printf("Strava will redirect to:\n  %s?code=<code>&state=<state>\n", redirectURI)
	fmt.Println()
	fmt.Println("Copy that URL (even if the page shows 'connection refused') and run:")
	fmt.Println("  strava auth login --auth-url '<paste full URL here>'")
	return nil
}

// completeRemoteLogin is step 2: validate state, exchange code, store tokens.
func completeRemoteLogin(cfg *config.Config) error {
	if cfg.PendingAuth == nil {
		return fmt.Errorf("no pending login found — run 'strava auth login --remote' first")
	}
	pending := cfg.PendingAuth
	if time.Now().Unix() > pending.ExpiresAt {
		cfg.PendingAuth = nil
		_ = config.Save(cfg)
		return fmt.Errorf("auth URL expired — run 'strava auth login --remote' again to get a fresh one")
	}

	tokens, err := auth.CompleteRemoteLogin(
		cfg.ClientID, cfg.ClientSecret,
		pending.RedirectURI, pending.State,
		authPasteURL,
	)
	if err != nil {
		return err
	}

	cfg.Tokens = *tokens
	cfg.PendingAuth = nil // clear one-time state
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Println("Successfully authenticated! Tokens stored in ~/.config/strava-cli/config.json")
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.ClientID == "" {
		fmt.Println("Not authenticated — run: strava auth login")
		return nil
	}

	fmt.Printf("Client ID:    %s\n", cfg.ClientID)
	if cfg.RedirectURI != "" {
		fmt.Printf("Redirect URI: %s\n", cfg.RedirectURI)
	}
	if cfg.PendingAuth != nil {
		fmt.Println("Pending:      remote login in progress (run 'strava auth login --auth-url ...' to complete)")
	}

	if cfg.Tokens.AccessToken == "" {
		fmt.Println("Token:        not set")
		return nil
	}

	expiry := time.Unix(cfg.Tokens.ExpiresAt, 0)
	now := time.Now()
	if now.Before(expiry) {
		remaining := expiry.Sub(now).Truncate(time.Second)
		fmt.Printf("Token:        valid (expires in %s, at %s)\n",
			remaining, expiry.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Token:        expired at %s (will auto-refresh on next command)\n",
			expiry.Format("2006-01-02 15:04:05"))
	}
	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	dir, err := config.Dir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "config.json")
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		fmt.Println("Not authenticated.")
		return nil
	}

	fmt.Printf("This will delete %s and revoke local credentials.\nProceed? [y/N] ", path)
	var ans string
	fmt.Fscanln(os.Stdin, &ans)
	if strings.ToLower(strings.TrimSpace(ans)) != "y" {
		fmt.Println("Aborted.")
		return nil
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove config: %w", err)
	}
	fmt.Println("Logged out. Run 'strava auth login' to re-authenticate.")
	return nil
}

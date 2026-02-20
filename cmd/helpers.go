package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	genclient "github.com/Brainsoft-Raxat/strava-cli/internal/client"
	"github.com/Brainsoft-Raxat/strava-cli/internal/config"
)

// apiClient loads config, refreshes the token, and returns a ready API client.
func apiClient(cmd *cobra.Command) (*genclient.ClientWithResponses, *config.Config, error) {
	cfg, err := loadAndRefresh()
	if err != nil {
		return nil, nil, err
	}
	httpClient := genclient.NewHTTPClient(cfg)
	api, err := genclient.NewClientWithResponses("https://www.strava.com/api/v3",
		genclient.WithHTTPClient(httpClient))
	if err != nil {
		return nil, nil, fmt.Errorf("create API client: %w", err)
	}
	return api, cfg, nil
}

// rawClient returns an *http.Client for raw (non-generated) API calls.
// The client injects the Bearer token and retries on 429/5xx identically to apiClient.
func rawClient(cmd *cobra.Command) (*http.Client, *config.Config, error) {
	cfg, err := loadAndRefresh()
	if err != nil {
		return nil, nil, err
	}
	return genclient.NewHTTPClient(cfg), cfg, nil
}

// confirmMutation handles the --dry-run / --yes / interactive-prompt safety gate for
// write commands. It returns (proceed, err). When proceed is false and err is nil
// the caller should return nil (dry-run preview or user declined).
func confirmMutation(cmd *cobra.Command, description string) (bool, error) {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")

	if dryRun {
		fmt.Fprintf(os.Stderr, "DRY RUN: would %s\n", description)
		return false, nil
	}
	if yes {
		fmt.Fprintf(os.Stderr, "AUDIT: %s\n", description)
		return true, nil
	}
	fmt.Fprintf(os.Stderr, "About to %s\nProceed? [y/N] ", description)
	var ans string
	fmt.Fscanln(os.Stdin, &ans)
	if strings.ToLower(strings.TrimSpace(ans)) != "y" {
		fmt.Fprintln(os.Stderr, "Aborted.")
		return false, nil
	}
	fmt.Fprintf(os.Stderr, "AUDIT: %s\n", description)
	return true, nil
}

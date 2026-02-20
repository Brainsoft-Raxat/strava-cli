package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Brainsoft-Raxat/strava-cli/internal/config"
	"github.com/Brainsoft-Raxat/strava-cli/internal/output"
)

var athleteCmd = &cobra.Command{
	Use:   "athlete",
	Short: "Athlete commands",
}

var athleteMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Display the authenticated athlete's profile",
	RunE:  runAthleteMe,
}

var athleteStatsCmd = &cobra.Command{
	Use:   "stats [athlete-id]",
	Short: "Display athlete stats (defaults to the authenticated athlete)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAthleteStats,
}

var athleteZonesCmd = &cobra.Command{
	Use:   "zones",
	Short: "Display the authenticated athlete's heart rate and power zones",
	RunE:  runAthleteZones,
}

func init() {
	rootCmd.AddCommand(athleteCmd)
	athleteCmd.AddCommand(athleteMeCmd)
	athleteCmd.AddCommand(athleteStatsCmd)
	athleteCmd.AddCommand(athleteZonesCmd)
}

func runAthleteMe(cmd *cobra.Command, args []string) error {
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetLoggedInAthleteWithResponse(cmd.Context())
	if err != nil {
		return fmt.Errorf("fetch athlete: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Athlete(resp)
}

func runAthleteStats(cmd *cobra.Command, args []string) error {
	api, cfg, err := apiClient(cmd)
	if err != nil {
		return err
	}

	var athleteID int64
	if len(args) == 1 {
		if _, err := fmt.Sscan(args[0], &athleteID); err != nil {
			return fmt.Errorf("invalid athlete ID %q", args[0])
		}
	} else {
		// Fetch own ID first.
		me, err := api.GetLoggedInAthleteWithResponse(cmd.Context())
		if err != nil {
			return fmt.Errorf("fetch athlete: %w", err)
		}
		if me.HTTPResponse.StatusCode != 200 {
			return apiError(me.HTTPResponse.StatusCode, me.Body)
		}
		if me.JSON200 != nil && me.JSON200.Id != nil {
			athleteID = *me.JSON200.Id
		}
	}
	_ = cfg

	resp, err := api.GetStatsWithResponse(cmd.Context(), athleteID)
	if err != nil {
		return fmt.Errorf("fetch stats: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Stats(resp)
}

func runAthleteZones(cmd *cobra.Command, args []string) error {
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetLoggedInAthleteZonesWithResponse(cmd.Context())
	if err != nil {
		return fmt.Errorf("fetch zones: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).AthleteZones(resp)
}

// loadAndRefresh loads config and ensures the token is valid.
func loadAndRefresh() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("not configured — run: strava auth login")
	}
	return cfg, nil
}

// apiError converts an HTTP status code into an actionable error message.
func apiError(status int, body []byte) error {
	hint := ""
	switch status {
	case 401:
		hint = " — run: strava auth login"
	case 403:
		hint = " — check OAuth scopes at https://www.strava.com/settings/api"
	case 404:
		hint = " — resource does not exist or is private"
	case 429:
		hint = " — you've exceeded Strava's API limits; try again later"
	}
	if len(body) > 0 && len(body) < 400 {
		return fmt.Errorf("HTTP %d%s: %s", status, hint, stripHTML(string(body)))
	}
	return fmt.Errorf("HTTP %d%s", status, hint)
}

var htmlTagRE = regexp.MustCompile(`<[^>]+>`)

// stripHTML removes HTML tags from Strava API error messages so they render
// cleanly in the terminal.
func stripHTML(s string) string {
	return strings.TrimSpace(htmlTagRE.ReplaceAllString(s, ""))
}

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	genclient "github.com/Brainsoft-Raxat/strava-cli/internal/client"
	"github.com/Brainsoft-Raxat/strava-cli/internal/output"
)

var clubsCmd = &cobra.Command{
	Use:   "clubs",
	Short: "Club commands",
}

var (
	clubsPage    int
	clubsPerPage int
)

var clubsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List clubs the authenticated athlete belongs to",
	RunE:  runClubsList,
}

var clubsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a club by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runClubsGet,
}

var clubsMembersCmd = &cobra.Command{
	Use:   "members <id>",
	Short: "List members of a club",
	Args:  cobra.ExactArgs(1),
	RunE:  runClubsMembers,
}

var clubsActivitiesCmd = &cobra.Command{
	Use:   "activities <id>",
	Short: "List recent activities from a club",
	Args:  cobra.ExactArgs(1),
	RunE:  runClubsActivities,
}

func init() {
	rootCmd.AddCommand(clubsCmd)
	clubsCmd.AddCommand(clubsListCmd)
	clubsCmd.AddCommand(clubsGetCmd)
	clubsCmd.AddCommand(clubsMembersCmd)
	clubsCmd.AddCommand(clubsActivitiesCmd)

	for _, c := range []*cobra.Command{clubsListCmd, clubsMembersCmd, clubsActivitiesCmd} {
		c.Flags().IntVar(&clubsPage, "page", 1, "Page number")
		c.Flags().IntVar(&clubsPerPage, "per-page", 30, "Items per page")
	}
}

func runClubsList(cmd *cobra.Command, args []string) error {
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetLoggedInAthleteClubsWithResponse(cmd.Context(),
		&genclient.GetLoggedInAthleteClubsParams{Page: intPtr(clubsPage), PerPage: intPtr(clubsPerPage)})
	if err != nil {
		return fmt.Errorf("fetch clubs: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Clubs(resp)
}

func runClubsGet(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetClubByIdWithResponse(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("fetch club: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Club(resp)
}

func runClubsMembers(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetClubMembersByIdWithResponse(cmd.Context(), id,
		&genclient.GetClubMembersByIdParams{Page: intPtr(clubsPage), PerPage: intPtr(clubsPerPage)})
	if err != nil {
		return fmt.Errorf("fetch members: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).ClubMembers(resp)
}

func runClubsActivities(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetClubActivitiesByIdWithResponse(cmd.Context(), id,
		&genclient.GetClubActivitiesByIdParams{Page: intPtr(clubsPage), PerPage: intPtr(clubsPerPage)})
	if err != nil {
		return fmt.Errorf("fetch club activities: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).ClubActivities(resp)
}

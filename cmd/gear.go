package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/Brainsoft-Raxat/strava-cli/internal/output"
)

var gearCmd = &cobra.Command{
	Use:   "gear",
	Short: "Gear commands",
}

var gearGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get gear by ID (e.g. b12345 for a bike, g12345 for shoes)",
	Args:  cobra.ExactArgs(1),
	RunE:  runGearGet,
}

func init() {
	rootCmd.AddCommand(gearCmd)
	gearCmd.AddCommand(gearGetCmd)
}

func runGearGet(cmd *cobra.Command, args []string) error {
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	// Gear IDs are strings like "b12345678" or "g12345678"
	resp, err := api.GetGearByIdWithResponse(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("fetch gear: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Gear(resp)
}

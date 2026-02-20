package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "stravacli",
	Short: "A Strava CLI powered by the official API",
	Long: `stravacli is a command-line interface for the Strava API.

Configuration is stored in ~/.config/strava-cli/config.json.

To get started:
  stravacli auth login
`,
	SilenceUsage: true,
}

// SetVersion stamps the build version into the root command (called from main).
func SetVersion(v string) {
	rootCmd.Version = v
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
}

package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	genclient "github.com/Brainsoft-Raxat/strava-cli/internal/client"
	"github.com/Brainsoft-Raxat/strava-cli/internal/output"
)

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Route commands",
}

var (
	routesPage    int
	routesPerPage int
)

var routesListCmd = &cobra.Command{
	Use:   "list [athlete-id]",
	Short: "List routes (defaults to the authenticated athlete)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRoutesList,
}

var routesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a route by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoutesGet,
}

var (
	exportFormat string
	exportOut    string
)

var routesExportCmd = &cobra.Command{
	Use:   "export <id>",
	Short: "Export a route as GPX or TCX",
	Long: `Download a route as a GPX or TCX file.

The file is written to --out (defaults to route-<id>.<format>).

Examples:
  strava routes export 12345 --format gpx
  strava routes export 12345 --format tcx --out /tmp/my-route.tcx`,
	Args: cobra.ExactArgs(1),
	RunE: runRoutesExport,
}

func init() {
	rootCmd.AddCommand(routesCmd)
	routesCmd.AddCommand(routesListCmd)
	routesCmd.AddCommand(routesGetCmd)
	routesCmd.AddCommand(routesExportCmd)

	routesListCmd.Flags().IntVar(&routesPage, "page", 1, "Page number")
	routesListCmd.Flags().IntVar(&routesPerPage, "per-page", 30, "Items per page")

	routesExportCmd.Flags().StringVar(&exportFormat, "format", "gpx", "Export format: gpx or tcx")
	routesExportCmd.Flags().StringVar(&exportOut, "out", "", "Output file path (default: route-<id>.<format>)")
}

func runRoutesList(cmd *cobra.Command, args []string) error {
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}

	var athleteID int64
	if len(args) == 1 {
		if _, err := fmt.Sscan(args[0], &athleteID); err != nil {
			return fmt.Errorf("invalid athlete ID %q", args[0])
		}
	} else {
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

	resp, err := api.GetRoutesByAthleteIdWithResponse(cmd.Context(), athleteID,
		&genclient.GetRoutesByAthleteIdParams{Page: intPtr(routesPage), PerPage: intPtr(routesPerPage)})
	if err != nil {
		return fmt.Errorf("fetch routes: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Routes(resp)
}

func runRoutesGet(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetRouteByIdWithResponse(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("fetch route: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Route(resp)
}

func runRoutesExport(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}

	format := strings.ToLower(exportFormat)
	if format != "gpx" && format != "tcx" {
		return fmt.Errorf("--format must be gpx or tcx, got %q", format)
	}

	outPath := exportOut
	if outPath == "" {
		outPath = fmt.Sprintf("route-%d.%s", id, format)
	}

	httpClient, _, err := rawClient(cmd)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://www.strava.com/api/v3/routes/%d/export_%s", id, format)
	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("export route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return apiError(resp.StatusCode, body)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Saved %d bytes → %s\n", n, outPath)
	return nil
}

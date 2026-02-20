package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	genclient "github.com/Brainsoft-Raxat/strava-cli/internal/client"
	"github.com/Brainsoft-Raxat/strava-cli/internal/output"
)

var segmentsCmd = &cobra.Command{
	Use:   "segments",
	Short: "Segment commands",
}

var (
	segPage    int
	segPerPage int
)

var segmentsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a segment by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runSegmentsGet,
}

var segmentsStarredCmd = &cobra.Command{
	Use:   "starred",
	Short: "List the authenticated athlete's starred segments",
	RunE:  runSegmentsStarred,
}

var (
	exploreBounds       string
	exploreActivityType string
	exploreMinCat       int
	exploreMaxCat       int
)

var segmentsExploreCmd = &cobra.Command{
	Use:   "explore",
	Short: "Find popular segments in a bounding box",
	Long: `Find popular segments within a geographic bounding box.

--bounds format: sw_lat,sw_lng,ne_lat,ne_lng
Example: strava segments explore --bounds 51.5,-0.2,51.6,-0.1 --activity-type running`,
	RunE: runSegmentsExplore,
}

var segmentEffortsCmd = &cobra.Command{
	Use:   "efforts",
	Short: "Segment effort commands",
}

var (
	effortsSegmentID  int64
	effortsStartDate  string
	effortsEndDate    string
	effortsPerPage    int
)

var segmentEffortsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List efforts on a segment",
	RunE:  runSegmentEffortsList,
}

var segmentEffortsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a segment effort by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runSegmentEffortsGet,
}

func init() {
	rootCmd.AddCommand(segmentsCmd)
	segmentsCmd.AddCommand(segmentsGetCmd)
	segmentsCmd.AddCommand(segmentsStarredCmd)
	segmentsCmd.AddCommand(segmentsExploreCmd)
	segmentsCmd.AddCommand(segmentEffortsCmd)
	segmentEffortsCmd.AddCommand(segmentEffortsListCmd)
	segmentEffortsCmd.AddCommand(segmentEffortsGetCmd)

	segmentsStarredCmd.Flags().IntVar(&segPage, "page", 1, "Page number")
	segmentsStarredCmd.Flags().IntVar(&segPerPage, "per-page", 30, "Items per page")

	segmentsExploreCmd.Flags().StringVar(&exploreBounds, "bounds", "",
		"Bounding box: sw_lat,sw_lng,ne_lat,ne_lng (required)")
	segmentsExploreCmd.Flags().StringVar(&exploreActivityType, "activity-type", "",
		"Filter by activity type: running or riding")
	segmentsExploreCmd.Flags().IntVar(&exploreMinCat, "min-cat", 0, "Minimum climb category (0-5)")
	segmentsExploreCmd.Flags().IntVar(&exploreMaxCat, "max-cat", 0, "Maximum climb category (0-5)")
	_ = segmentsExploreCmd.MarkFlagRequired("bounds")

	segmentEffortsListCmd.Flags().Int64Var(&effortsSegmentID, "segment-id", 0, "Segment ID (required)")
	segmentEffortsListCmd.Flags().StringVar(&effortsStartDate, "start-date", "",
		"ISO 8601 start date, e.g. 2024-01-01T00:00:00Z")
	segmentEffortsListCmd.Flags().StringVar(&effortsEndDate, "end-date", "",
		"ISO 8601 end date")
	segmentEffortsListCmd.Flags().IntVar(&effortsPerPage, "per-page", 30, "Items per page")
	_ = segmentEffortsListCmd.MarkFlagRequired("segment-id")
}

func runSegmentsGet(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetSegmentByIdWithResponse(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("fetch segment: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Segment(resp)
}

func runSegmentsStarred(cmd *cobra.Command, args []string) error {
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetLoggedInAthleteStarredSegmentsWithResponse(cmd.Context(),
		&genclient.GetLoggedInAthleteStarredSegmentsParams{Page: intPtr(segPage), PerPage: intPtr(segPerPage)})
	if err != nil {
		return fmt.Errorf("fetch starred segments: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).StarredSegments(resp)
}

func runSegmentsExplore(cmd *cobra.Command, args []string) error {
	bounds, err := parseBounds(exploreBounds)
	if err != nil {
		return err
	}

	params := &genclient.ExploreSegmentsParams{Bounds: bounds}
	if exploreActivityType != "" {
		at := genclient.ExploreSegmentsParamsActivityType(exploreActivityType)
		params.ActivityType = &at
	}
	if exploreMinCat > 0 {
		params.MinCat = intPtr(exploreMinCat)
	}
	if exploreMaxCat > 0 {
		params.MaxCat = intPtr(exploreMaxCat)
	}

	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.ExploreSegmentsWithResponse(cmd.Context(), params)
	if err != nil {
		return fmt.Errorf("explore segments: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).ExploreSegments(resp)
}

func runSegmentEffortsList(cmd *cobra.Command, args []string) error {
	params := &genclient.GetEffortsBySegmentIdParams{
		SegmentId: int(effortsSegmentID),
		PerPage:   intPtr(effortsPerPage),
	}
	if effortsStartDate != "" {
		t, err := time.Parse(time.RFC3339, effortsStartDate)
		if err != nil {
			return fmt.Errorf("invalid --start-date %q: use RFC3339 format e.g. 2024-01-01T00:00:00Z", effortsStartDate)
		}
		params.StartDateLocal = &t
	}
	if effortsEndDate != "" {
		t, err := time.Parse(time.RFC3339, effortsEndDate)
		if err != nil {
			return fmt.Errorf("invalid --end-date %q: use RFC3339 format e.g. 2024-12-31T23:59:59Z", effortsEndDate)
		}
		params.EndDateLocal = &t
	}

	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetEffortsBySegmentIdWithResponse(cmd.Context(), params)
	if err != nil {
		return fmt.Errorf("fetch efforts: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).SegmentEfforts(resp)
}

func runSegmentEffortsGet(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetSegmentEffortByIdWithResponse(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("fetch effort: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).SegmentEffort(resp)
}

// parseBounds parses "sw_lat,sw_lng,ne_lat,ne_lng" into []float32.
func parseBounds(s string) ([]float32, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("--bounds must be 4 comma-separated values: sw_lat,sw_lng,ne_lat,ne_lng")
	}
	result := make([]float32, 4)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, fmt.Errorf("invalid bounds value %q: %w", p, err)
		}
		result[i] = float32(v)
	}
	return result, nil
}

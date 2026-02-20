package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	genclient "github.com/Brainsoft-Raxat/strava-cli/internal/client"
	"github.com/Brainsoft-Raxat/strava-cli/internal/output"
)

var activitiesCmd = &cobra.Command{
	Use:   "activities",
	Short: "Activity commands",
}

var (
	listBefore  int
	listAfter   int
	listPage    int
	listPerPage int
)

var activitiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your recent activities",
	Long: `List the authenticated athlete's activities.

--before and --after accept Unix timestamps.
Example: --after $(date -d '7 days ago' +%s)`,
	RunE: runActivitiesList,
}

var activitiesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a specific activity by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runActivitiesGet,
}

var activitiesLapsCmd = &cobra.Command{
	Use:   "laps <id>",
	Short: "List laps for an activity",
	Args:  cobra.ExactArgs(1),
	RunE:  runActivitiesLaps,
}

var activitiesZonesCmd = &cobra.Command{
	Use:   "zones <id>",
	Short: "Get heart rate and power zones for an activity",
	Args:  cobra.ExactArgs(1),
	RunE:  runActivitiesZones,
}

var activitiesCommentsCmd = &cobra.Command{
	Use:   "comments <id>",
	Short: "List comments on an activity",
	Args:  cobra.ExactArgs(1),
	RunE:  runActivitiesComments,
}

var activitiesKudosCmd = &cobra.Command{
	Use:   "kudos <id>",
	Short: "List athletes who kudoed an activity",
	Args:  cobra.ExactArgs(1),
	RunE:  runActivitiesKudos,
}

var (
	streamsKeys string
)

var activitiesStreamsCmd = &cobra.Command{
	Use:   "streams <id>",
	Short: "Get data streams for an activity",
	Long: `Fetch time-series data streams for an activity.

Available stream keys (comma-separated):
  time, distance, latlng, altitude, velocity_smooth, heartrate,
  cadence, watts, temp, moving, grade_smooth

Example: strava activities streams 12345 --keys time,heartrate,watts`,
	Args: cobra.ExactArgs(1),
	RunE: runActivitiesStreams,
}

// ── update ────────────────────────────────────────────────────────────────────

var (
	updateName        string
	updateDescription string
	updateType        string
	updateGearID      string
	updateCommute     bool
	updateHide        bool
)

var activitiesUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an activity's metadata",
	Long: `Update metadata for one of your activities.

Only fields you explicitly pass are changed. Requires --yes to skip the
interactive confirmation prompt, or use --dry-run to preview the change.

Examples:
  strava activities update 12345 --name "Evening Run" --yes
  strava activities update 12345 --commute --hide --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runActivitiesUpdate,
}

// ── upload ────────────────────────────────────────────────────────────────────

var (
	uploadFile        string
	uploadDataType    string
	uploadName        string
	uploadDescription string
	uploadTrainer     bool
	uploadCommute     bool
	uploadWait        bool
)

var activitiesUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload an activity file",
	Long: `Upload an activity file (FIT, GPX, TCX) to Strava.

Supported --data-type values: fit, fit.gz, tcx, tcx.gz, gpx, gpx.gz
If --data-type is omitted it is inferred from the file extension.

Use --wait to poll until Strava finishes processing and prints the new
activity ID. Requires --yes to skip the interactive confirmation prompt.

Examples:
  strava activities upload --file morning.gpx --name "Morning Run" --yes --wait
  strava activities upload --file workout.fit --trainer --yes`,
	RunE: runActivitiesUpload,
}

func init() {
	rootCmd.AddCommand(activitiesCmd)
	activitiesCmd.AddCommand(activitiesListCmd)
	activitiesCmd.AddCommand(activitiesGetCmd)
	activitiesCmd.AddCommand(activitiesLapsCmd)
	activitiesCmd.AddCommand(activitiesZonesCmd)
	activitiesCmd.AddCommand(activitiesCommentsCmd)
	activitiesCmd.AddCommand(activitiesKudosCmd)
	activitiesCmd.AddCommand(activitiesStreamsCmd)
	activitiesCmd.AddCommand(activitiesUpdateCmd)
	activitiesCmd.AddCommand(activitiesUploadCmd)

	activitiesListCmd.Flags().IntVar(&listBefore, "before", 0, "Unix timestamp: only activities before this time")
	activitiesListCmd.Flags().IntVar(&listAfter, "after", 0, "Unix timestamp: only activities after this time")
	activitiesListCmd.Flags().IntVar(&listPage, "page", 1, "Page number")
	activitiesListCmd.Flags().IntVar(&listPerPage, "per-page", 30, "Activities per page (max 200)")

	activitiesStreamsCmd.Flags().StringVar(&streamsKeys, "keys",
		"time,distance,altitude,heartrate,cadence,watts,velocity_smooth",
		"Comma-separated stream keys to fetch")

	// update flags
	activitiesUpdateCmd.Flags().StringVar(&updateName, "name", "", "New activity name")
	activitiesUpdateCmd.Flags().StringVar(&updateDescription, "description", "", "New description")
	activitiesUpdateCmd.Flags().StringVar(&updateType, "type", "", "Sport type (e.g. Run, Ride, Walk)")
	activitiesUpdateCmd.Flags().StringVar(&updateGearID, "gear-id", "", "Gear ID (e.g. b12345678 or none)")
	activitiesUpdateCmd.Flags().BoolVar(&updateCommute, "commute", false, "Mark/unmark as commute (e.g. --commute or --commute=false)")
	activitiesUpdateCmd.Flags().BoolVar(&updateHide, "hide", false, "Hide/unhide from home feed")
	activitiesUpdateCmd.Flags().Bool("yes", false, "Skip interactive confirmation")
	activitiesUpdateCmd.Flags().Bool("dry-run", false, "Print what would change without calling the API")

	// upload flags
	activitiesUploadCmd.Flags().StringVar(&uploadFile, "file", "", "Path to activity file (required)")
	activitiesUploadCmd.Flags().StringVar(&uploadDataType, "data-type", "",
		"File type: fit, fit.gz, tcx, tcx.gz, gpx, gpx.gz (inferred from extension if omitted)")
	activitiesUploadCmd.Flags().StringVar(&uploadName, "name", "", "Activity name")
	activitiesUploadCmd.Flags().StringVar(&uploadDescription, "description", "", "Activity description")
	activitiesUploadCmd.Flags().BoolVar(&uploadTrainer, "trainer", false, "Mark as indoor trainer activity")
	activitiesUploadCmd.Flags().BoolVar(&uploadCommute, "commute", false, "Mark as commute")
	activitiesUploadCmd.Flags().BoolVar(&uploadWait, "wait", false, "Poll until Strava finishes processing")
	activitiesUploadCmd.Flags().Bool("yes", false, "Skip interactive confirmation")
	activitiesUploadCmd.Flags().Bool("dry-run", false, "Print what would be uploaded without calling the API")
	_ = activitiesUploadCmd.MarkFlagRequired("file")
}

// ── read handlers ─────────────────────────────────────────────────────────────

func runActivitiesList(cmd *cobra.Command, args []string) error {
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	params := &genclient.GetLoggedInAthleteActivitiesParams{
		Page:    intPtr(listPage),
		PerPage: intPtr(listPerPage),
	}
	if listBefore > 0 {
		params.Before = intPtr(listBefore)
	}
	if listAfter > 0 {
		params.After = intPtr(listAfter)
	}
	resp, err := api.GetLoggedInAthleteActivitiesWithResponse(cmd.Context(), params)
	if err != nil {
		return fmt.Errorf("fetch activities: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Activities(resp)
}

func runActivitiesGet(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetActivityByIdWithResponse(cmd.Context(), id,
		&genclient.GetActivityByIdParams{IncludeAllEfforts: boolPtr(false)})
	if err != nil {
		return fmt.Errorf("fetch activity: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Activity(resp)
}

func runActivitiesLaps(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetLapsByActivityIdWithResponse(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("fetch laps: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Laps(resp)
}

func runActivitiesZones(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetZonesByActivityIdWithResponse(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("fetch zones: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).ActivityZones(resp)
}

func runActivitiesComments(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetCommentsByActivityIdWithResponse(cmd.Context(), id,
		&genclient.GetCommentsByActivityIdParams{PerPage: intPtr(100)})
	if err != nil {
		return fmt.Errorf("fetch comments: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Comments(resp)
}

func runActivitiesKudos(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetKudoersByActivityIdWithResponse(cmd.Context(), id,
		&genclient.GetKudoersByActivityIdParams{PerPage: intPtr(100)})
	if err != nil {
		return fmt.Errorf("fetch kudos: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Kudos(resp)
}

func runActivitiesStreams(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}

	keys := []genclient.GetActivityStreamsParamsKeys{}
	for _, k := range strings.Split(streamsKeys, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, genclient.GetActivityStreamsParamsKeys(k))
		}
	}

	api, _, err := apiClient(cmd)
	if err != nil {
		return err
	}
	resp, err := api.GetActivityStreamsWithResponse(cmd.Context(), id,
		&genclient.GetActivityStreamsParams{Keys: keys, KeyByType: true})
	if err != nil {
		return fmt.Errorf("fetch streams: %w", err)
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return apiError(resp.HTTPResponse.StatusCode, resp.Body)
	}
	return output.New(os.Stdout, jsonOutput).Streams(resp)
}

// ── write handlers ────────────────────────────────────────────────────────────

func runActivitiesUpdate(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}

	// Collect only the fields the user explicitly passed.
	body := map[string]interface{}{}
	if cmd.Flags().Changed("name") {
		body["name"] = updateName
	}
	if cmd.Flags().Changed("description") {
		body["description"] = updateDescription
	}
	if cmd.Flags().Changed("type") {
		body["sport_type"] = updateType
		body["type"] = updateType
	}
	if cmd.Flags().Changed("gear-id") {
		body["gear_id"] = updateGearID
	}
	if cmd.Flags().Changed("commute") {
		body["commute"] = updateCommute
	}
	if cmd.Flags().Changed("hide") {
		body["hide_from_home"] = updateHide
	}
	if len(body) == 0 {
		return fmt.Errorf("no fields to update; provide at least one of: --name, --description, --type, --gear-id, --commute, --hide")
	}

	// Build a human-readable description for the audit / dry-run log.
	parts := make([]string, 0, len(body))
	for k, v := range body {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	desc := fmt.Sprintf("update activity %d (%s)", id, strings.Join(parts, ", "))

	proceed, err := confirmMutation(cmd, desc)
	if err != nil || !proceed {
		return err
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	httpClient, _, err := rawClient(cmd)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://www.strava.com/api/v3/activities/%d", id)
	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodPut, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("update activity: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return apiError(resp.StatusCode, respBody)
	}

	if jsonOutput {
		fmt.Fprintln(os.Stdout, string(respBody))
		return nil
	}

	// Print a brief confirmation with the returned name.
	var result struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.Name != "" {
		fmt.Fprintf(os.Stdout, "Updated activity %d: %q\n", result.ID, result.Name)
	} else {
		fmt.Fprintln(os.Stdout, "Activity updated.")
	}
	return nil
}

func runActivitiesUpload(cmd *cobra.Command, args []string) error {
	// Infer data_type from file extension if not specified.
	dt := uploadDataType
	if dt == "" {
		base := strings.ToLower(uploadFile)
		switch {
		case strings.HasSuffix(base, ".fit.gz"):
			dt = "fit.gz"
		case strings.HasSuffix(base, ".tcx.gz"):
			dt = "tcx.gz"
		case strings.HasSuffix(base, ".gpx.gz"):
			dt = "gpx.gz"
		case strings.HasSuffix(base, ".fit"):
			dt = "fit"
		case strings.HasSuffix(base, ".tcx"):
			dt = "tcx"
		case strings.HasSuffix(base, ".gpx"):
			dt = "gpx"
		default:
			return fmt.Errorf("cannot infer --data-type from %q; specify it explicitly", uploadFile)
		}
	}

	desc := fmt.Sprintf("upload %s (data_type=%s", filepath.Base(uploadFile), dt)
	if uploadName != "" {
		desc += ", name=" + uploadName
	}
	desc += ")"

	proceed, err := confirmMutation(cmd, desc)
	if err != nil || !proceed {
		return err
	}

	// Open file only after confirmation so dry-run doesn't need a real file.
	f, err := os.Open(uploadFile)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// Build multipart form.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	part, err := mw.CreateFormFile("file", filepath.Base(uploadFile))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	_ = mw.WriteField("data_type", dt)
	if uploadName != "" {
		_ = mw.WriteField("name", uploadName)
	}
	if uploadDescription != "" {
		_ = mw.WriteField("description", uploadDescription)
	}
	if uploadTrainer {
		_ = mw.WriteField("trainer", "1")
	}
	if uploadCommute {
		_ = mw.WriteField("commute", "1")
	}
	if err := mw.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	httpClient, _, err := rawClient(cmd)
	if err != nil {
		return err
	}

	// Use *bytes.Buffer so http.NewRequestWithContext sets GetBody for safe retries.
	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodPost,
		"https://www.strava.com/api/v3/uploads", &buf)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return apiError(resp.StatusCode, respBody)
	}

	var u uploadStatus
	if err := json.Unmarshal(respBody, &u); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if jsonOutput {
		fmt.Fprintln(os.Stdout, string(respBody))
	} else {
		printUploadStatus(os.Stdout, u)
	}

	if !uploadWait {
		if !jsonOutput {
			fmt.Fprintf(os.Stderr, "To check status: strava uploads get %d\n", u.ID)
		}
		return nil
	}

	return pollUpload(cmd, httpClient, u.ID)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ID %q: must be a number", s)
	}
	return id, nil
}

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }

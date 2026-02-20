package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// uploadStatus mirrors the Strava Upload object returned by POST /uploads and
// GET /uploads/{uploadId}.
type uploadStatus struct {
	ID         int64   `json:"id"`
	IDStr      string  `json:"id_str"`
	ExternalID *string `json:"external_id"`
	Error      *string `json:"error"`
	Status     string  `json:"status"`
	ActivityID *int64  `json:"activity_id"`
}

var uploadsCmd = &cobra.Command{
	Use:   "uploads",
	Short: "Upload status commands",
}

var uploadsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get the status of an upload by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runUploadsGet,
}

func init() {
	rootCmd.AddCommand(uploadsCmd)
	uploadsCmd.AddCommand(uploadsGetCmd)
}

func runUploadsGet(cmd *cobra.Command, args []string) error {
	id, err := parseID(args[0])
	if err != nil {
		return err
	}
	httpClient, _, err := rawClient(cmd)
	if err != nil {
		return err
	}
	u, raw, err := fetchUploadStatus(cmd.Context(), httpClient, id)
	if err != nil {
		return err
	}
	if jsonOutput {
		fmt.Fprintln(os.Stdout, string(raw))
		return nil
	}
	printUploadStatus(os.Stdout, u)
	return nil
}

// fetchUploadStatus calls GET /uploads/{id} and returns the parsed status plus the
// raw response body (so callers can pass it through in --json mode).
func fetchUploadStatus(ctx context.Context, httpClient *http.Client, id int64) (uploadStatus, []byte, error) {
	url := fmt.Sprintf("https://www.strava.com/api/v3/uploads/%d", id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return uploadStatus{}, nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return uploadStatus{}, nil, fmt.Errorf("fetch upload: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return uploadStatus{}, nil, apiError(resp.StatusCode, raw)
	}
	var u uploadStatus
	if err := json.Unmarshal(raw, &u); err != nil {
		return uploadStatus{}, nil, fmt.Errorf("parse response: %w", err)
	}
	return u, raw, nil
}

// printUploadStatus writes a human-readable upload status summary to w.
func printUploadStatus(w io.Writer, u uploadStatus) {
	fmt.Fprintf(w, "Upload ID:   %d\n", u.ID)
	fmt.Fprintf(w, "Status:      %s\n", u.Status)
	if u.ActivityID != nil {
		fmt.Fprintf(w, "Activity ID: %d\n", *u.ActivityID)
	}
	if u.Error != nil {
		fmt.Fprintf(w, "Error:       %s\n", stripHTML(*u.Error))
	}
}

// pollUpload polls GET /uploads/{id} every 3 seconds until processing completes,
// an error is reported by Strava, or a 5-minute timeout is reached.
func pollUpload(cmd *cobra.Command, httpClient *http.Client, id int64) error {
	const (
		pollInterval = 3 * time.Second
		timeout      = 5 * time.Minute
	)
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	fmt.Fprintf(os.Stderr, "Polling upload %d (Ctrl-C to cancel, check later with: strava uploads get %d)\n", id, id)

	for {
		select {
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		case <-ticker.C:
			u, raw, err := fetchUploadStatus(cmd.Context(), httpClient, id)
			if err != nil {
				return err
			}
			if u.Error != nil {
				if jsonOutput {
					fmt.Fprintln(os.Stdout, string(raw))
				} else {
					printUploadStatus(os.Stdout, u)
				}
				return fmt.Errorf("upload failed: %s", stripHTML(*u.Error))
			}
			if u.ActivityID != nil {
				if jsonOutput {
					fmt.Fprintln(os.Stdout, string(raw))
				} else {
					printUploadStatus(os.Stdout, u)
					fmt.Fprintln(os.Stderr, "Done.")
				}
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("upload timed out after %v; check status with: strava uploads get %d", timeout, id)
			}
			fmt.Fprintf(os.Stderr, "  still processing: %s\n", u.Status)
		}
	}
}

// Package output renders Strava data as human-readable tables or JSON.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/Brainsoft-Raxat/strava-cli/internal/client"
)

// Printer writes formatted output to a writer.
type Printer struct {
	w    io.Writer
	JSON bool
}

// New creates a Printer that writes to w.
func New(w io.Writer, jsonMode bool) *Printer {
	return &Printer{w: w, JSON: jsonMode}
}

// Athlete prints the authenticated athlete.
func (p *Printer) Athlete(a *client.GetLoggedInAthleteResponse) error {
	if a.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, a.JSON200)
	}
	d := a.JSON200
	fmt.Fprintf(p.w, "Name:      %s %s\n", strVal(d.Firstname), strVal(d.Lastname))
	fmt.Fprintf(p.w, "ID:        %d\n", int64Val(d.Id))
	fmt.Fprintf(p.w, "City:      %s, %s, %s\n", strVal(d.City), strVal(d.State), strVal(d.Country))
	fmt.Fprintf(p.w, "Followers: %d  Following: %d\n", intVal(d.FollowerCount), intVal(d.FriendCount))
	fmt.Fprintf(p.w, "Summit:    %v\n", boolVal(d.Summit))
	if d.CreatedAt != nil {
		fmt.Fprintf(p.w, "Member since: %s\n", d.CreatedAt.Format("Jan 2006"))
	}
	return nil
}

// Activities prints a list of summary activities.
func (p *Printer) Activities(acts *client.GetLoggedInAthleteActivitiesResponse) error {
	if acts.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, acts.JSON200)
	}
	list := *acts.JSON200
	if len(list) == 0 {
		fmt.Fprintln(p.w, "No activities found.")
		return nil
	}
	fmt.Fprintf(p.w, "%-12s  %-30s  %-18s  %-9s  %-10s  %s\n",
		"ID", "Name", "Sport", "Distance", "Time", "Date")
	fmt.Fprintln(p.w, strings.Repeat("─", 105))
	for _, a := range list {
		sport := ""
		if a.SportType != nil {
			sport = string(*a.SportType)
		}
		fmt.Fprintf(p.w, "%-12d  %-30s  %-18s  %-9s  %-10s  %s\n",
			int64Val(a.Id),
			truncate(strVal(a.Name), 30),
			truncate(sport, 18),
			formatDistance(float32Val(a.Distance)),
			formatDuration(intVal(a.MovingTime)),
			formatTime(a.StartDateLocal),
		)
	}
	return nil
}

// Activity prints a single detailed activity.
func (p *Printer) Activity(a *client.GetActivityByIdResponse) error {
	if a.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, a.JSON200)
	}
	d := a.JSON200
	sport := ""
	if d.SportType != nil {
		sport = string(*d.SportType)
	}
	fmt.Fprintf(p.w, "ID:           %d\n", int64Val(d.Id))
	fmt.Fprintf(p.w, "Name:         %s\n", strVal(d.Name))
	fmt.Fprintf(p.w, "Sport:        %s\n", sport)
	fmt.Fprintf(p.w, "Date:         %s\n", formatTime(d.StartDateLocal))
	fmt.Fprintf(p.w, "Distance:     %s\n", formatDistance(float32Val(d.Distance)))
	fmt.Fprintf(p.w, "Moving time:  %s\n", formatDuration(intVal(d.MovingTime)))
	fmt.Fprintf(p.w, "Elapsed time: %s\n", formatDuration(intVal(d.ElapsedTime)))
	fmt.Fprintf(p.w, "Elevation:    %.0f m\n", float32Val(d.TotalElevationGain))
	fmt.Fprintf(p.w, "Avg speed:    %.1f km/h\n", msToKmh(float32Val(d.AverageSpeed)))
	if d.AverageWatts != nil {
		fmt.Fprintf(p.w, "Avg power:    %.0f W\n", float32Val(d.AverageWatts))
	}
	fmt.Fprintf(p.w, "Kudos:        %d\n", intVal(d.KudosCount))
	if d.Description != nil && *d.Description != "" {
		fmt.Fprintf(p.w, "Description:\n  %s\n", *d.Description)
	}
	return nil
}

// --- helpers ---

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int64Val(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func intVal(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func float32Val(v *float32) float32 {
	if v == nil {
		return 0
	}
	return *v
}

func boolVal(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// FormatDistance converts meters to a human-readable string (exported for tests).
func FormatDistance(meters float32) string {
	return formatDistance(meters)
}

// FormatDuration converts seconds to a human-readable string (exported for tests).
func FormatDuration(seconds int) string {
	return formatDuration(seconds)
}

func formatDistance(meters float32) string {
	if meters >= 1000 {
		return fmt.Sprintf("%.2f km", meters/1000)
	}
	return fmt.Sprintf("%.0f m", meters)
}

func formatDuration(seconds int) string {
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	m := int(math.Mod(d.Minutes(), 60))
	s := int(math.Mod(d.Seconds(), 60))
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	return fmt.Sprintf("%dm%02ds", m, s)
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func msToKmh(ms float32) float32 {
	return ms * 3.6
}

package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Brainsoft-Raxat/strava-cli/internal/client"
	"github.com/Brainsoft-Raxat/strava-cli/internal/output"
)

// --- FormatDistance ---

func TestFormatDistance(t *testing.T) {
	tests := []struct {
		name   string
		meters float32
		want   string
	}{
		{"zero", 0, "0 m"},
		{"sub-km", 500, "500 m"},
		{"exactly-1km", 1000, "1.00 km"},
		{"half-marathon", 21097.5, "21.10 km"},
		{"marathon", 42195, "42.19 km"},
		{"sprint", 100, "100 m"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := output.FormatDistance(tc.meters)
			if got != tc.want {
				t.Errorf("FormatDistance(%v) = %q, want %q", tc.meters, got, tc.want)
			}
		})
	}
}

// --- FormatDuration ---

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    string
	}{
		{"zero", 0, "0m00s"},
		{"one-minute", 60, "1m00s"},
		{"one-hour", 3600, "1h00m00s"},
		{"mixed", 3661, "1h01m01s"},
		{"5k-ish", 1200, "20m00s"},
		{"half-marathon-ish", 5400, "1h30m00s"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := output.FormatDuration(tc.seconds)
			if got != tc.want {
				t.Errorf("FormatDuration(%d) = %q, want %q", tc.seconds, got, tc.want)
			}
		})
	}
}

// unmarshalAthleteResponse unmarshals JSON into a GetLoggedInAthleteResponse.
func unmarshalAthleteResponse(t *testing.T, raw string) *client.GetLoggedInAthleteResponse {
	t.Helper()
	resp := &client.GetLoggedInAthleteResponse{}
	// We unmarshal into the Body and then parse JSON200 via ParseGetLoggedInAthleteResponse.
	// Simpler: just unmarshal into JSON200 directly.
	if err := json.Unmarshal([]byte(raw), &resp.JSON200); err != nil {
		t.Fatalf("unmarshal athlete response: %v", err)
	}
	return resp
}

// unmarshalActivityResponse unmarshals JSON into a GetActivityByIdResponse.
func unmarshalActivityResponse(t *testing.T, raw string) *client.GetActivityByIdResponse {
	t.Helper()
	resp := &client.GetActivityByIdResponse{}
	if err := json.Unmarshal([]byte(raw), &resp.JSON200); err != nil {
		t.Fatalf("unmarshal activity response: %v", err)
	}
	return resp
}

// unmarshalActivitiesResponse unmarshals JSON into a GetLoggedInAthleteActivitiesResponse.
func unmarshalActivitiesResponse(t *testing.T, raw string) *client.GetLoggedInAthleteActivitiesResponse {
	t.Helper()
	resp := &client.GetLoggedInAthleteActivitiesResponse{}
	if err := json.Unmarshal([]byte(raw), &resp.JSON200); err != nil {
		t.Fatalf("unmarshal activities response: %v", err)
	}
	return resp
}

// --- Athlete output ---

func TestPrinterAthlete_HumanReadable(t *testing.T) {
	resp := unmarshalAthleteResponse(t, `{
		"firstname": "Jane",
		"lastname": "Doe",
		"id": 12345,
		"city": "San Francisco",
		"state": "CA",
		"country": "US",
		"follower_count": 42,
		"friend_count": 13,
		"summit": true,
		"created_at": "2020-03-15T00:00:00Z"
	}`)

	var buf bytes.Buffer
	p := output.New(&buf, false)
	if err := p.Athlete(resp); err != nil {
		t.Fatalf("Athlete() error: %v", err)
	}
	got := buf.String()

	for _, want := range []string{"Jane Doe", "12345", "San Francisco", "42", "13", "Mar 2020"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestPrinterAthlete_JSON(t *testing.T) {
	resp := unmarshalAthleteResponse(t, `{"firstname":"Alice","id":99}`)

	var buf bytes.Buffer
	p := output.New(&buf, true)
	if err := p.Athlete(resp); err != nil {
		t.Fatalf("Athlete() JSON error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `"firstname"`) {
		t.Errorf("expected JSON output, got: %s", got)
	}
}

func TestPrinterAthlete_NilJSON200(t *testing.T) {
	resp := &client.GetLoggedInAthleteResponse{}
	p := output.New(&bytes.Buffer{}, false)
	if err := p.Athlete(resp); err == nil {
		t.Error("expected error for nil JSON200")
	}
}

// --- Activities output ---

func TestPrinterActivities_Empty(t *testing.T) {
	resp := unmarshalActivitiesResponse(t, `[]`)

	var buf bytes.Buffer
	p := output.New(&buf, false)
	if err := p.Activities(resp); err != nil {
		t.Fatalf("Activities() error: %v", err)
	}
	if !strings.Contains(buf.String(), "No activities") {
		t.Errorf("expected 'No activities' message, got: %s", buf.String())
	}
}

func TestPrinterActivities_TableColumns(t *testing.T) {
	resp := unmarshalActivitiesResponse(t, `[
		{
			"id": 99887766,
			"name": "Morning Run",
			"sport_type": "Run",
			"distance": 10000,
			"moving_time": 3600,
			"start_date_local": "2024-05-01T07:30:00Z"
		}
	]`)

	var buf bytes.Buffer
	p := output.New(&buf, false)
	if err := p.Activities(resp); err != nil {
		t.Fatalf("Activities() error: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"99887766", "Morning Run", "Run", "10.00 km", "1h00m00s", "2024-05-01"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestPrinterActivities_JSON(t *testing.T) {
	resp := unmarshalActivitiesResponse(t, `[{"id":1,"name":"Ride"}]`)

	var buf bytes.Buffer
	p := output.New(&buf, true)
	if err := p.Activities(resp); err != nil {
		t.Fatalf("Activities() JSON error: %v", err)
	}
	// Should be valid JSON array
	var out []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot: %s", err, buf.String())
	}
}

// --- Activity detail output ---

func TestPrinterActivity_HumanReadable(t *testing.T) {
	resp := unmarshalActivityResponse(t, `{
		"id": 1234567,
		"name": "Lunch Ride",
		"sport_type": "Ride",
		"distance": 25000,
		"moving_time": 3600,
		"elapsed_time": 3800,
		"total_elevation_gain": 300,
		"average_speed": 6.944,
		"kudos_count": 10,
		"start_date_local": "2024-06-01T12:00:00Z",
		"description": "Nice day out"
	}`)

	var buf bytes.Buffer
	p := output.New(&buf, false)
	if err := p.Activity(resp); err != nil {
		t.Fatalf("Activity() error: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"1234567", "Lunch Ride", "Ride", "25.00 km", "1h00m00s", "300", "Nice day out", "10"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestPrinterActivity_NilJSON200(t *testing.T) {
	resp := &client.GetActivityByIdResponse{}
	p := output.New(&bytes.Buffer{}, false)
	if err := p.Activity(resp); err == nil {
		t.Error("expected error for nil JSON200")
	}
}

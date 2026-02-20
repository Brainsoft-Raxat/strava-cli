package output

// This file contains formatters for all API resources beyond athlete/activities.

import (
	"fmt"
	"strings"
	"time"

	"github.com/Brainsoft-Raxat/strava-cli/internal/client"
)

// Stats prints athlete lifetime and recent statistics.
func (p *Printer) Stats(r *client.GetStatsResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	d := r.JSON200
	type totals struct {
		label string
		v     *struct {
			AchievementCount *int     `json:"achievement_count,omitempty"`
			Count            *int     `json:"count,omitempty"`
			Distance         *float32 `json:"distance,omitempty"`
			ElapsedTime      *int     `json:"elapsed_time,omitempty"`
			ElevationGain    *float32 `json:"elevation_gain,omitempty"`
			MovingTime       *int     `json:"moving_time,omitempty"`
		}
	}
	sections := []struct {
		heading string
		rows    []totals
	}{
		{"Recent (last 4 weeks)", []totals{
			{"Rides", d.RecentRideTotals},
			{"Runs", d.RecentRunTotals},
			{"Swims", d.RecentSwimTotals},
		}},
		{"All time", []totals{
			{"Rides", d.AllRideTotals},
			{"Runs", d.AllRunTotals},
			{"Swims", d.AllSwimTotals},
		}},
	}
	for _, sec := range sections {
		fmt.Fprintf(p.w, "\n%s\n", sec.heading)
		fmt.Fprintln(p.w, strings.Repeat("─", 70))
		fmt.Fprintf(p.w, "  %-10s  %6s  %-10s  %-10s  %-10s\n",
			"Sport", "Count", "Distance", "Moving", "Elevation")
		for _, row := range sec.rows {
			if row.v == nil {
				continue
			}
			fmt.Fprintf(p.w, "  %-10s  %6d  %-10s  %-10s  %.0f m\n",
				row.label,
				intVal(row.v.Count),
				formatDistance(float32Val(row.v.Distance)),
				formatDuration(intVal(row.v.MovingTime)),
				float32Val(row.v.ElevationGain),
			)
		}
	}
	if d.BiggestRideDistance != nil {
		fmt.Fprintf(p.w, "\nBiggest ride distance: %s\n",
			formatDistance(float32(*d.BiggestRideDistance)))
	}
	if d.BiggestClimbElevationGain != nil {
		fmt.Fprintf(p.w, "Biggest climb:         %.0f m\n", *d.BiggestClimbElevationGain)
	}
	return nil
}

// AthleteZones prints the authenticated athlete's HR and power zones.
func (p *Printer) AthleteZones(r *client.GetLoggedInAthleteZonesResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	d := r.JSON200
	if d.HeartRate != nil && d.HeartRate.Zones != nil {
		fmt.Fprintln(p.w, "Heart Rate Zones")
		fmt.Fprintln(p.w, strings.Repeat("─", 35))
		for i, z := range *d.HeartRate.Zones {
			min := intVal(z.Min)
			max := intVal(z.Max)
			if max == -1 {
				fmt.Fprintf(p.w, "  Zone %d  %d+ bpm\n", i+1, min)
			} else {
				fmt.Fprintf(p.w, "  Zone %d  %d–%d bpm\n", i+1, min, max)
			}
		}
	}
	if d.Power != nil && d.Power.Zones != nil {
		fmt.Fprintln(p.w, "\nPower Zones")
		fmt.Fprintln(p.w, strings.Repeat("─", 35))
		for i, z := range *d.Power.Zones {
			min := intVal(z.Min)
			max := intVal(z.Max)
			if max == -1 {
				fmt.Fprintf(p.w, "  Zone %d  %d+ W\n", i+1, min)
			} else {
				fmt.Fprintf(p.w, "  Zone %d  %d–%d W\n", i+1, min, max)
			}
		}
	}
	return nil
}

// Laps prints laps for an activity.
func (p *Printer) Laps(r *client.GetLapsByActivityIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	laps := *r.JSON200
	if len(laps) == 0 {
		fmt.Fprintln(p.w, "No laps recorded.")
		return nil
	}
	fmt.Fprintf(p.w, "%-4s  %-10s  %-10s  %-10s  %s\n",
		"Lap", "Distance", "Time", "Avg Speed", "Start")
	fmt.Fprintln(p.w, strings.Repeat("─", 65))
	for _, lap := range laps {
		fmt.Fprintf(p.w, "%-4d  %-10s  %-10s  %-10s  %s\n",
			intVal(lap.LapIndex),
			formatDistance(float32Val(lap.Distance)),
			formatDuration(intVal(lap.MovingTime)),
			fmt.Sprintf("%.1f km/h", msToKmh(float32Val(lap.AverageSpeed))),
			formatTime(lap.StartDateLocal),
		)
	}
	return nil
}

// ActivityZones prints HR/power zones for an activity.
func (p *Printer) ActivityZones(r *client.GetZonesByActivityIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	zones := *r.JSON200
	if len(zones) == 0 {
		fmt.Fprintln(p.w, "No zone data available.")
		return nil
	}
	for _, z := range zones {
		typ := ""
		if z.Type != nil {
			typ = string(*z.Type)
		}
		score := ""
		if z.Score != nil {
			score = fmt.Sprintf("  score: %d", *z.Score)
		}
		fmt.Fprintf(p.w, "%s%s\n", strings.Title(typ), score)
		fmt.Fprintln(p.w, strings.Repeat("─", 40))
		if z.DistributionBuckets != nil {
			for _, b := range *z.DistributionBuckets {
				fmt.Fprintf(p.w, "  %d–%d bpm: %d s\n",
					intVal(b.Min), intVal(b.Max), intVal(b.Time))
			}
		}
		fmt.Fprintln(p.w)
	}
	return nil
}

// Comments prints comments on an activity.
func (p *Printer) Comments(r *client.GetCommentsByActivityIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	comments := *r.JSON200
	if len(comments) == 0 {
		fmt.Fprintln(p.w, "No comments.")
		return nil
	}
	for _, c := range comments {
		name := "Unknown"
		if c.Athlete != nil {
			name = strings.TrimSpace(strVal(c.Athlete.Firstname) + " " + strVal(c.Athlete.Lastname))
		}
		date := formatTime(c.CreatedAt)
		fmt.Fprintf(p.w, "%s  (%s)\n", name, date)
		if c.Text != nil {
			fmt.Fprintf(p.w, "  %s\n", *c.Text)
		}
		fmt.Fprintln(p.w)
	}
	return nil
}

// Kudos prints athletes who kudoed an activity.
func (p *Printer) Kudos(r *client.GetKudoersByActivityIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	kudoers := *r.JSON200
	if len(kudoers) == 0 {
		fmt.Fprintln(p.w, "No kudos yet.")
		return nil
	}
	fmt.Fprintf(p.w, "%d kudo(s):\n", len(kudoers))
	for _, k := range kudoers {
		fmt.Fprintf(p.w, "  %s %s\n", strVal(k.Firstname), strVal(k.Lastname))
	}
	return nil
}

// Streams prints activity stream data. In human mode it shows a summary table;
// use --json for the full data.
func (p *Printer) Streams(r *client.GetActivityStreamsResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	d := r.JSON200
	// Show a summary of available streams with their lengths.
	type streamInfo struct {
		name string
		n    int
	}
	var available []streamInfo
	if d.Time != nil && d.Time.Data != nil {
		available = append(available, streamInfo{"time (s)", len(*d.Time.Data)})
	}
	if d.Distance != nil && d.Distance.Data != nil {
		available = append(available, streamInfo{"distance (m)", len(*d.Distance.Data)})
	}
	if d.Altitude != nil && d.Altitude.Data != nil {
		available = append(available, streamInfo{"altitude (m)", len(*d.Altitude.Data)})
	}
	if d.Heartrate != nil && d.Heartrate.Data != nil {
		available = append(available, streamInfo{"heartrate (bpm)", len(*d.Heartrate.Data)})
	}
	if d.Cadence != nil && d.Cadence.Data != nil {
		available = append(available, streamInfo{"cadence (rpm)", len(*d.Cadence.Data)})
	}
	if d.Watts != nil && d.Watts.Data != nil {
		available = append(available, streamInfo{"power (W)", len(*d.Watts.Data)})
	}
	if d.VelocitySmooth != nil && d.VelocitySmooth.Data != nil {
		available = append(available, streamInfo{"velocity (m/s)", len(*d.VelocitySmooth.Data)})
	}
	if d.Latlng != nil && d.Latlng.Data != nil {
		available = append(available, streamInfo{"latlng", len(*d.Latlng.Data)})
	}
	if d.Moving != nil && d.Moving.Data != nil {
		available = append(available, streamInfo{"moving", len(*d.Moving.Data)})
	}
	if d.GradeSmooth != nil && d.GradeSmooth.Data != nil {
		available = append(available, streamInfo{"grade (%)", len(*d.GradeSmooth.Data)})
	}
	if d.Temp != nil && d.Temp.Data != nil {
		available = append(available, streamInfo{"temp (°C)", len(*d.Temp.Data)})
	}

	if len(available) == 0 {
		fmt.Fprintln(p.w, "No stream data available.")
		return nil
	}
	fmt.Fprintln(p.w, "Available streams:")
	fmt.Fprintln(p.w, strings.Repeat("─", 35))
	for _, s := range available {
		fmt.Fprintf(p.w, "  %-20s  %d data points\n", s.name, s.n)
	}
	fmt.Fprintln(p.w, "\nUse --json to get the full data.")
	return nil
}

// Clubs prints the list of clubs the athlete belongs to.
func (p *Printer) Clubs(r *client.GetLoggedInAthleteClubsResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	clubs := *r.JSON200
	if len(clubs) == 0 {
		fmt.Fprintln(p.w, "No clubs.")
		return nil
	}
	fmt.Fprintf(p.w, "%-12s  %-35s  %7s  %s\n", "ID", "Name", "Members", "Location")
	fmt.Fprintln(p.w, strings.Repeat("─", 80))
	for _, c := range clubs {
		loc := strings.TrimRight(strVal(c.City)+", "+strVal(c.Country), ", ")
		fmt.Fprintf(p.w, "%-12d  %-35s  %7d  %s\n",
			int64Val(c.Id),
			truncate(strVal(c.Name), 35),
			intVal(c.MemberCount),
			loc,
		)
	}
	return nil
}

// Club prints a single club's detail.
func (p *Printer) Club(r *client.GetClubByIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	d := r.JSON200
	fmt.Fprintf(p.w, "ID:       %d\n", int64Val(d.Id))
	fmt.Fprintf(p.w, "Name:     %s\n", strVal(d.Name))
	fmt.Fprintf(p.w, "City:     %s, %s, %s\n", strVal(d.City), strVal(d.State), strVal(d.Country))
	fmt.Fprintf(p.w, "Members:  %d  (following: %d)\n", intVal(d.MemberCount), intVal(d.FollowingCount))
	fmt.Fprintf(p.w, "Private:  %v\n", boolVal(d.Private))
	if d.Membership != nil {
		fmt.Fprintf(p.w, "Your membership: %s\n", *d.Membership)
	}
	return nil
}

// ClubMembers prints the members of a club.
func (p *Printer) ClubMembers(r *client.GetClubMembersByIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	members := *r.JSON200
	if len(members) == 0 {
		fmt.Fprintln(p.w, "No members.")
		return nil
	}
	fmt.Fprintf(p.w, "%-30s  %s\n", "Name", "Role")
	fmt.Fprintln(p.w, strings.Repeat("─", 45))
	for _, m := range members {
		role := strVal(m.Member)
		if boolVal(m.Admin) {
			role = "admin"
		}
		if boolVal(m.Owner) {
			role = "owner"
		}
		fmt.Fprintf(p.w, "%-30s  %s\n",
			truncate(strVal(m.Firstname)+" "+strVal(m.Lastname), 30),
			role,
		)
	}
	return nil
}

// ClubActivities prints recent activities from a club.
// Note: the API only returns athlete ID (not name) for privacy reasons.
func (p *Printer) ClubActivities(r *client.GetClubActivitiesByIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	acts := *r.JSON200
	if len(acts) == 0 {
		fmt.Fprintln(p.w, "No recent activities.")
		return nil
	}
	fmt.Fprintf(p.w, "%-30s  %-16s  %-10s  %s\n",
		"Name", "Sport", "Distance", "Time")
	fmt.Fprintln(p.w, strings.Repeat("─", 75))
	for _, a := range acts {
		sport := ""
		if a.SportType != nil {
			sport = string(*a.SportType)
		}
		fmt.Fprintf(p.w, "%-30s  %-16s  %-10s  %s\n",
			truncate(strVal(a.Name), 30),
			truncate(sport, 16),
			formatDistance(float32Val(a.Distance)),
			formatDuration(intVal(a.MovingTime)),
		)
	}
	return nil
}

// Gear prints gear detail.
func (p *Printer) Gear(r *client.GetGearByIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	d := r.JSON200
	fmt.Fprintf(p.w, "ID:        %s\n", strVal(d.Id))
	fmt.Fprintf(p.w, "Name:      %s\n", strVal(d.Name))
	fmt.Fprintf(p.w, "Brand:     %s\n", strVal(d.BrandName))
	fmt.Fprintf(p.w, "Model:     %s\n", strVal(d.ModelName))
	fmt.Fprintf(p.w, "Distance:  %s\n", formatDistance(float32Val(d.Distance)))
	fmt.Fprintf(p.w, "Primary:   %v\n", boolVal(d.Primary))
	if d.Description != nil && *d.Description != "" {
		fmt.Fprintf(p.w, "Notes:     %s\n", *d.Description)
	}
	return nil
}

// Routes prints a list of routes.
func (p *Printer) Routes(r *client.GetRoutesByAthleteIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	routes := *r.JSON200
	if len(routes) == 0 {
		fmt.Fprintln(p.w, "No routes found.")
		return nil
	}
	fmt.Fprintf(p.w, "%-12s  %-35s  %-10s  %-8s  %s\n",
		"ID", "Name", "Distance", "Elev", "Est. Time")
	fmt.Fprintln(p.w, strings.Repeat("─", 85))
	for _, r := range routes {
		fmt.Fprintf(p.w, "%-12d  %-35s  %-10s  %-8s  %s\n",
			int64Val(r.Id),
			truncate(strVal(r.Name), 35),
			formatDistance(float32Val(r.Distance)),
			fmt.Sprintf("%.0fm", float32Val(r.ElevationGain)),
			formatDuration(intVal(r.EstimatedMovingTime)),
		)
	}
	return nil
}

// Route prints a single route's detail.
func (p *Printer) Route(r *client.GetRouteByIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	d := r.JSON200
	fmt.Fprintf(p.w, "ID:           %d\n", int64Val(d.Id))
	fmt.Fprintf(p.w, "Name:         %s\n", strVal(d.Name))
	fmt.Fprintf(p.w, "Distance:     %s\n", formatDistance(float32Val(d.Distance)))
	fmt.Fprintf(p.w, "Elevation:    %.0f m\n", float32Val(d.ElevationGain))
	fmt.Fprintf(p.w, "Est. time:    %s\n", formatDuration(intVal(d.EstimatedMovingTime)))
	if d.Description != nil && *d.Description != "" {
		fmt.Fprintf(p.w, "Description:  %s\n", *d.Description)
	}
	if d.CreatedAt != nil {
		fmt.Fprintf(p.w, "Created:      %s\n", d.CreatedAt.Format("2006-01-02"))
	}
	return nil
}

// Segment prints a segment's detail.
func (p *Printer) Segment(r *client.GetSegmentByIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	d := r.JSON200
	fmt.Fprintf(p.w, "ID:           %d\n", int64Val(d.Id))
	fmt.Fprintf(p.w, "Name:         %s\n", strVal(d.Name))
	fmt.Fprintf(p.w, "Location:     %s, %s, %s\n", strVal(d.City), strVal(d.State), strVal(d.Country))
	fmt.Fprintf(p.w, "Distance:     %s\n", formatDistance(float32Val(d.Distance)))
	fmt.Fprintf(p.w, "Avg grade:    %.1f%%\n", float32Val(d.AverageGrade))
	fmt.Fprintf(p.w, "Max grade:    %.1f%%\n", float32Val(d.MaximumGrade))
	fmt.Fprintf(p.w, "Elev high:    %.0f m\n", float32Val(d.ElevationHigh))
	fmt.Fprintf(p.w, "Elev low:     %.0f m\n", float32Val(d.ElevationLow))
	fmt.Fprintf(p.w, "Climb cat:    %d\n", intVal(d.ClimbCategory))
	fmt.Fprintf(p.w, "Efforts:      %d\n", intVal(d.EffortCount))
	fmt.Fprintf(p.w, "Stars:        %d\n", intVal(d.StarCount))
	fmt.Fprintf(p.w, "Athletes:     %d\n", intVal(d.AthleteCount))
	if d.AthletePrEffort != nil && d.AthletePrEffort.PrElapsedTime != nil {
		fmt.Fprintf(p.w, "Your PR:      %s", formatDuration(intVal(d.AthletePrEffort.PrElapsedTime)))
		if d.AthletePrEffort.PrDate != nil {
			fmt.Fprintf(p.w, "  (%s)", d.AthletePrEffort.PrDate.Format("2006-01-02"))
		}
		fmt.Fprintln(p.w)
	}
	return nil
}

// StarredSegments prints the list of starred segments.
func (p *Printer) StarredSegments(r *client.GetLoggedInAthleteStarredSegmentsResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	segs := *r.JSON200
	if len(segs) == 0 {
		fmt.Fprintln(p.w, "No starred segments.")
		return nil
	}
	fmt.Fprintf(p.w, "%-12s  %-35s  %-10s  %6s  %s\n",
		"ID", "Name", "Distance", "Grade", "City")
	fmt.Fprintln(p.w, strings.Repeat("─", 80))
	for _, s := range segs {
		fmt.Fprintf(p.w, "%-12d  %-35s  %-10s  %5.1f%%  %s\n",
			int64Val(s.Id), truncate(strVal(s.Name), 35),
			formatDistance(float32Val(s.Distance)),
			float32Val(s.AverageGrade), strVal(s.City))
	}
	return nil
}

// ExploreSegments prints explored segments.
func (p *Printer) ExploreSegments(r *client.ExploreSegmentsResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	if r.JSON200.Segments == nil || len(*r.JSON200.Segments) == 0 {
		fmt.Fprintln(p.w, "No segments found in this area.")
		return nil
	}
	segs := *r.JSON200.Segments
	fmt.Fprintf(p.w, "%-12s  %-35s  %-10s  %6s  %s\n",
		"ID", "Name", "Distance", "Grade", "Cat")
	fmt.Fprintln(p.w, strings.Repeat("─", 80))
	for _, s := range segs {
		cat := ""
		if s.ClimbCategoryDesc != nil {
			cat = string(*s.ClimbCategoryDesc)
		}
		fmt.Fprintf(p.w, "%-12d  %-35s  %-10s  %5.1f%%  %s\n",
			int64Val(s.Id),
			truncate(strVal(s.Name), 35),
			formatDistance(float32Val(s.Distance)),
			float32Val(s.AvgGrade),
			cat,
		)
	}
	return nil
}

// SegmentEfforts prints a list of efforts on a segment.
func (p *Printer) SegmentEfforts(r *client.GetEffortsBySegmentIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	efforts := *r.JSON200
	if len(efforts) == 0 {
		fmt.Fprintln(p.w, "No efforts found.")
		return nil
	}
	fmt.Fprintf(p.w, "%-12s  %-10s  %s\n", "ID", "Time", "Date")
	fmt.Fprintln(p.w, strings.Repeat("─", 45))
	for _, e := range efforts {
		fmt.Fprintf(p.w, "%-12d  %-10s  %s\n",
			int64Val(e.Id),
			formatDuration(intVal(e.ElapsedTime)),
			formatTime(e.StartDateLocal),
		)
	}
	return nil
}

// SegmentEffort prints a single segment effort.
func (p *Printer) SegmentEffort(r *client.GetSegmentEffortByIdResponse) error {
	if r.JSON200 == nil {
		return fmt.Errorf("unexpected empty response")
	}
	if p.JSON {
		return printJSON(p.w, r.JSON200)
	}
	d := r.JSON200
	segName := strVal(d.Name) // Name field holds the segment name on efforts
	if d.Segment != nil && d.Segment.Name != nil {
		segName = *d.Segment.Name
	}
	fmt.Fprintf(p.w, "ID:           %d\n", int64Val(d.Id))
	fmt.Fprintf(p.w, "Segment:      %s\n", segName)
	fmt.Fprintf(p.w, "Date:         %s\n", formatTime(d.StartDateLocal))
	fmt.Fprintf(p.w, "Elapsed time: %s\n", formatDuration(intVal(d.ElapsedTime)))
	fmt.Fprintf(p.w, "Moving time:  %s\n", formatDuration(intVal(d.MovingTime)))
	fmt.Fprintf(p.w, "Distance:     %s\n", formatDistance(float32Val(d.Distance)))
	if d.AverageHeartrate != nil {
		fmt.Fprintf(p.w, "Avg HR:       %.0f bpm\n", *d.AverageHeartrate)
	}
	if d.AverageWatts != nil {
		fmt.Fprintf(p.w, "Avg power:    %.0f W\n", *d.AverageWatts)
	}
	if d.KomRank != nil {
		fmt.Fprintf(p.w, "KOM rank:     %d\n", *d.KomRank)
	}
	if d.PrRank != nil {
		fmt.Fprintf(p.w, "PR rank:      %d\n", *d.PrRank)
	}
	return nil
}

// --- internal helpers ---


// FormatTime exports the time formatter for use in tests.
func FormatTime(t *time.Time) string { return formatTime(t) }

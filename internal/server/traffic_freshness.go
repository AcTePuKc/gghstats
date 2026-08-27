package server

import (
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

// trafficFreshness is intentionally additive API/page metadata. A nil count in
// chart data means the day was not explicitly reported by GitHub.
type trafficFreshness struct {
	Metric                string   `json:"metric"`
	Status                string   `json:"status"` // fresh, delayed, missing, failed, never
	FetchedAt             string   `json:"fetched_at,omitempty"`
	LatestObservedDay     string   `json:"latest_observed_day,omitempty"`
	LatestCompletedUTCDay string   `json:"latest_completed_utc_day"`
	MissingCompletedDays  []string `json:"missing_completed_days"`
	Error                 string   `json:"error,omitempty"`
}

func (f trafficFreshness) DisplayStatus() string {
	switch f.Status {
	case "fresh":
		return "Fresh"
	case "delayed":
		return "Delayed"
	case "missing":
		return "Missing completed days"
	case "failed":
		return "Fetch failed"
	default:
		return "Never synced"
	}
}

func trafficFreshnessFor(db *store.Store, repo, metric string, now time.Time) (trafficFreshness, error) {
	st, err := db.TrafficMetricState(repo, metric)
	if err != nil {
		return trafficFreshness{}, err
	}
	completed := now.UTC().AddDate(0, 0, -1).Format("2006-01-02")
	f := trafficFreshness{Metric: metric, FetchedAt: st.LastSuccessAt, LatestObservedDay: st.LatestObservedDate, LatestCompletedUTCDay: completed, Error: st.LastError, MissingCompletedDays: []string{}}
	if st.LastStatus == "failed" {
		f.Status = "failed"
		return f, nil
	}
	if st.LastStatus == "never" || st.LastSuccessAt == "" {
		f.Status = "never"
		return f, nil
	}
	coverage, err := db.TrafficCoverageDates(repo, metric, st.LastSuccessAt)
	if err != nil {
		return trafficFreshness{}, err
	}
	if st.CoverageFrom != "" {
		start, parseErr := time.Parse("2006-01-02", st.CoverageFrom)
		if parseErr == nil {
			for d := start.UTC(); !d.After(now.UTC().AddDate(0, 0, -1)); d = d.AddDate(0, 0, 1) {
				day := d.Format("2006-01-02")
				if st.CoverageTo != "" && day > st.CoverageTo {
					break
				}
				if !coverage[day] {
					f.MissingCompletedDays = append(f.MissingCompletedDays, day)
				}
			}
		}
	}
	if len(f.MissingCompletedDays) > 0 {
		f.Status = "missing"
		return f, nil
	}
	// A traffic response that does not reach yesterday is delayed. Today is
	// deliberately excluded because GitHub commonly publishes it late.
	if st.LatestObservedDate < completed {
		f.Status = "delayed"
	} else {
		f.Status = "fresh"
	}
	return f, nil
}

func repoTrafficFreshness(db *store.Store, repo string, now time.Time) (views, clones trafficFreshness, err error) {
	views, err = trafficFreshnessFor(db, repo, "views", now)
	if err != nil {
		return views, clones, err
	}
	clones, err = trafficFreshnessFor(db, repo, "clones", now)
	return views, clones, err
}

type chartTrafficPoint struct {
	Date    string `json:"date"`
	Count   *int   `json:"count"`
	Uniques *int   `json:"uniques"`
}

func denseTrafficChart(rows []store.DayRow, from, to string) []chartTrafficPoint {
	byDate := make(map[string]store.DayRow, len(rows))
	for _, row := range rows {
		byDate[row.Date] = row
	}
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return nil
	}
	out := make([]chartTrafficPoint, 0, int(end.Sub(start).Hours()/24)+1)
	for d := start.UTC(); !d.After(end.UTC()); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")
		point := chartTrafficPoint{Date: date}
		if row, ok := byDate[date]; ok {
			count, uniques := row.Count, row.Uniques
			point.Count, point.Uniques = &count, &uniques
		}
		out = append(out, point)
	}
	return out
}

// denseTrafficChartWithCoverage prevents an older cached row from disguising a
// date omitted by GitHub's latest successful rolling response as confirmed data.
func denseTrafficChartWithCoverage(db *store.Store, repo, metric, from, to string, rows []store.DayRow) ([]chartTrafficPoint, error) {
	st, err := db.TrafficMetricState(repo, metric)
	if err != nil || st.LastSuccessAt == "" || st.CoverageFrom == "" || st.CoverageTo == "" {
		return denseTrafficChart(rows, from, to), err
	}
	coverage, err := db.TrafficCoverageDates(repo, metric, st.LastSuccessAt)
	if err != nil {
		return nil, err
	}
	filtered := make([]store.DayRow, 0, len(rows))
	for _, row := range rows {
		if row.Date >= st.CoverageFrom && row.Date <= st.CoverageTo && !coverage[row.Date] {
			continue
		}
		filtered = append(filtered, row)
	}
	return denseTrafficChart(filtered, from, to), nil
}

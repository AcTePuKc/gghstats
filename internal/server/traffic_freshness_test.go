package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestTrafficFreshnessDistinguishesMissingCompletedDayFromZero(t *testing.T) {
	db := testStore(t)
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	// The 25th was explicitly reported as zero; the 26th was not reported.
	err := db.RecordTrafficMetricSuccess("o/r", "views", []store.DayRow{{Date: "2026-08-25", Count: 0, Uniques: 0}}, now, "2026-08-25", "2026-08-27")
	if err != nil {
		t.Fatal(err)
	}
	f, err := trafficFreshnessFor(db, "o/r", "views", now)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != "missing" || len(f.MissingCompletedDays) != 1 || f.MissingCompletedDays[0] != "2026-08-26" {
		t.Fatalf("freshness=%+v", f)
	}
}

func TestTrafficFreshnessDoesNotTreatCurrentUTCDayAsMissing(t *testing.T) {
	db := testStore(t)
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	err := db.RecordTrafficMetricSuccess("o/r", "clones", []store.DayRow{{Date: "2026-08-26", Count: 4, Uniques: 2}}, now, "2026-08-26", "2026-08-27")
	if err != nil {
		t.Fatal(err)
	}
	f, err := trafficFreshnessFor(db, "o/r", "clones", now)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != "fresh" || len(f.MissingCompletedDays) != 0 {
		t.Fatalf("freshness=%+v", f)
	}
}

func TestTrafficFreshnessReportsFailedBeforeAnySuccessfulFetch(t *testing.T) {
	db := testStore(t)
	if err := db.RecordTrafficMetricFailure("o/r", "views", fmt.Errorf("upstream unavailable")); err != nil {
		t.Fatal(err)
	}
	f, err := trafficFreshnessFor(db, "o/r", "views", time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC))
	if err != nil || f.Status != "failed" || f.Error == "" {
		t.Fatalf("freshness=%+v err=%v", f, err)
	}
}

func TestDenseTrafficChartUsesNullForGaps(t *testing.T) {
	points := denseTrafficChart([]store.DayRow{{Date: "2026-08-25", Count: 0, Uniques: 0}, {Date: "2026-08-27", Count: 3, Uniques: 1}}, "2026-08-25", "2026-08-27")
	if len(points) != 3 || points[0].Count == nil || *points[0].Count != 0 || points[1].Count != nil || points[2].Count == nil || *points[2].Count != 3 {
		t.Fatalf("points=%+v", points)
	}
}

func TestDenseTrafficChartHidesCachedRowOmittedByLatestCoverage(t *testing.T) {
	db := testStore(t)
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	// Older history contains the 26th, but the latest response only confirmed the 25th.
	if err := db.UpsertView("o/r", "2026-08-26", 9, 4); err != nil {
		t.Fatal(err)
	}
	if err := db.RecordTrafficMetricSuccess("o/r", "views", []store.DayRow{{Date: "2026-08-25", Count: 0, Uniques: 0}}, now, "2026-08-25", "2026-08-27"); err != nil {
		t.Fatal(err)
	}
	rows, err := db.ViewsByRange("o/r", "2026-08-25", "2026-08-27")
	if err != nil {
		t.Fatal(err)
	}
	points, err := denseTrafficChartWithCoverage(db, "o/r", "views", "2026-08-25", "2026-08-27", rows)
	if err != nil {
		t.Fatal(err)
	}
	if points[0].Count == nil || *points[0].Count != 0 || points[1].Count != nil {
		t.Fatalf("points=%+v", points)
	}
}

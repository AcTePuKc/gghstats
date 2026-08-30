package server

import (
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/i18n"
	"github.com/hrodrig/gghstats/internal/store"
)

func TestCalculateCloneStatistics(t *testing.T) {
	tests := []struct {
		name    string
		rows    []store.DayRow
		locale  string
		compact bool
		want    *cloneStatistics
	}{
		{
			name: "empty",
			want: nil,
		},
		{
			name:   "single value en",
			rows:   []store.DayRow{{Date: "2026-01-01", Count: 7}},
			locale: "en",
			want: &cloneStatistics{
				Mean: "7.00", Median: "7.00", Variance: "0.00", StandardDeviation: "0.00",
				Minimum: "7", Maximum: "7", P95: "7",
			},
		},
		{
			name:   "zero-filled even window en",
			rows:   []store.DayRow{{Count: 0}, {Count: 2}, {Count: 4}, {Count: 8}},
			locale: "en",
			want: &cloneStatistics{
				Mean: "3.50", Median: "3.00", Variance: "8.75", StandardDeviation: "2.96",
				Minimum: "0", Maximum: "8", P95: "8",
			},
		},
		{
			name:   "odd window nearest-rank p95 en with grouping",
			rows:   []store.DayRow{{Count: 1}, {Count: 3}, {Count: 5}, {Count: 7}, {Count: 100}},
			locale: "en",
			want: &cloneStatistics{
				Mean: "23.20", Median: "5.00", Variance: "1,478.56", StandardDeviation: "38.45",
				Minimum: "1", Maximum: "100", P95: "100",
			},
		},
		{
			name:   "es locale separators",
			rows:   []store.DayRow{{Count: 1000}, {Count: 2000}, {Count: 3000}},
			locale: "es",
			want: &cloneStatistics{
				Mean: "2.000,00", Median: "2.000,00", Variance: "666.666,67", StandardDeviation: "816,50",
				Minimum: "1.000", Maximum: "3.000", P95: "3.000",
			},
		},
		{
			name:    "compact ints",
			rows:    []store.DayRow{{Count: 1200}, {Count: 3400}},
			locale:  "en",
			compact: true,
			want: &cloneStatistics{
				Mean: "2,300.00", Median: "2,300.00", Variance: "1,210,000.00", StandardDeviation: "1,100.00",
				Minimum: "1.2k", Maximum: "3.4k", P95: "3.4k",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCloneStatistics(tt.rows, tt.locale, tt.compact)
			if !cloneStatisticsValuesEqual(got, tt.want) {
				t.Fatalf("calculateCloneStatistics() = %#v, want %#v", got, tt.want)
			}
			if got != nil && (got.MeanHelp == "" || got.P95Help == "") {
				t.Fatalf("expected non-empty help tooltips, got MeanHelp=%q P95Help=%q", got.MeanHelp, got.P95Help)
			}
		})
	}
}

func TestCalculateUniqueCloneStatistics(t *testing.T) {
	rows := []store.DayRow{
		{Count: 100, Uniques: 1},
		{Count: 200, Uniques: 3},
		{Count: 300, Uniques: 5},
	}
	want := &cloneStatistics{
		Mean: "3.00", Median: "3.00", Variance: "2.67", StandardDeviation: "1.63",
		Minimum: "1", Maximum: "5", P95: "5",
	}
	got := calculateUniqueCloneStatistics(rows, "en", false)
	if !cloneStatisticsValuesEqual(got, want) {
		t.Fatalf("calculateUniqueCloneStatistics() = %#v, want %#v", got, want)
	}
	if !strings.Contains(got.MeanHelp, i18n.MustLoad().T("en", "index.stats_unit_unique")) {
		t.Fatalf("unique help should mention unique unit: %q", got.MeanHelp)
	}
}

func TestCloneStatisticsHelpLocalized(t *testing.T) {
	rows := []store.DayRow{{Date: "2026-01-01", Count: 1000}, {Date: "2026-01-02", Count: 2000}}
	got := calculateCloneStatistics(rows, "es", false)
	if got == nil {
		t.Fatal("expected stats")
	}
	if !strings.Contains(got.VarianceHelp, "varianza poblacional") {
		t.Fatalf("es VarianceHelp should be Spanish: %q", got.VarianceHelp)
	}
	if !strings.Contains(got.MeanHelp, "Promedio diario") {
		t.Fatalf("es MeanHelp should be Spanish: %q", got.MeanHelp)
	}
	if strings.Contains(got.VarianceHelp, "Squared dispersion") {
		t.Fatalf("es help must not keep English boilerplate: %q", got.VarianceHelp)
	}
}

func TestCloneStatisticsHelpIncludesDates(t *testing.T) {
	rows := []store.DayRow{
		{Date: "2026-05-08", Count: 9},
		{Date: "2026-06-19", Count: 878},
		{Date: "2026-06-01", Count: 100},
	}
	got := calculateCloneStatistics(rows, "en", false)
	if got == nil {
		t.Fatal("expected stats")
	}
	if !strings.Contains(got.MinimumHelp, "2026-05-08") {
		t.Fatalf("MinimumHelp missing min date: %q", got.MinimumHelp)
	}
	if !strings.Contains(got.MaximumHelp, "2026-06-19") {
		t.Fatalf("MaximumHelp missing max date: %q", got.MaximumHelp)
	}
}

func cloneStatisticsValuesEqual(got, want *cloneStatistics) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Mean == want.Mean &&
		got.Median == want.Median &&
		got.Variance == want.Variance &&
		got.StandardDeviation == want.StandardDeviation &&
		got.Minimum == want.Minimum &&
		got.Maximum == want.Maximum &&
		got.P95 == want.P95
}

package server

import (
	"math"
	"sort"
	"strconv"

	"github.com/hrodrig/gghstats/internal/i18n"
	"github.com/hrodrig/gghstats/internal/store"
)

type cloneStatistics struct {
	Mean              string
	Median            string
	Variance          string
	StandardDeviation string
	Minimum           string
	Maximum           string
	P95               string
	// *Help strings power native title="" tooltips (#24, Option B); localized via i18n.Tfmt.
	MeanHelp              string
	MedianHelp            string
	VarianceHelp          string
	StandardDeviationHelp string
	MinimumHelp           string
	MaximumHelp           string
	P95Help               string
}

func calculateCloneStatistics(rows []store.DayRow, locale string, compact bool) *cloneStatistics {
	return calculateCloneStatisticsFor(rows, locale, compact, "index.stats_unit_clones", func(row store.DayRow) int { return row.Count })
}

func calculateUniqueCloneStatistics(rows []store.DayRow, locale string, compact bool) *cloneStatistics {
	return calculateCloneStatisticsFor(rows, locale, compact, "index.stats_unit_unique", func(row store.DayRow) int { return row.Uniques })
}

func calculateCloneStatisticsFor(rows []store.DayRow, locale string, compact bool, unitKey string, value func(store.DayRow) int) *cloneStatistics {
	if len(rows) == 0 {
		return nil
	}

	values := make([]float64, len(rows))
	var sum float64
	minIdx, maxIdx := 0, 0
	for i, row := range rows {
		values[i] = float64(value(row))
		sum += values[i]
		if values[i] < values[minIdx] {
			minIdx = i
		}
		if values[i] > values[maxIdx] {
			maxIdx = i
		}
	}
	mean := sum / float64(len(values))
	minDate, maxDate := rows[minIdx].Date, rows[maxIdx].Date
	minVal, maxVal := values[minIdx], values[maxIdx]

	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	median := sorted[len(sorted)/2]
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + median) / 2
	}

	var squaredDifferenceSum float64
	for _, v := range values {
		difference := v - mean
		squaredDifferenceSum += difference * difference
	}
	variance := squaredDifferenceSum / float64(len(values))
	stddev := math.Sqrt(variance)
	p95Index := int(math.Ceil(0.95*float64(len(sorted)))) - 1
	p95 := sorted[p95Index]

	meanFmt := i18n.FormatFloat(mean, locale, 2)
	medianFmt := i18n.FormatFloat(median, locale, 2)
	varianceFmt := i18n.FormatFloat(variance, locale, 2)
	stddevFmt := i18n.FormatFloat(stddev, locale, 2)
	minFmt := i18n.FormatCount(int(minVal), locale, compact)
	maxFmt := i18n.FormatCount(int(maxVal), locale, compact)
	p95Fmt := i18n.FormatCount(int(p95), locale, compact)

	low := mean - stddev
	if low < 0 {
		low = 0
	}
	high := mean + stddev
	exceededDays := int(math.Round(0.05 * float64(len(values))))
	if exceededDays < 1 && len(values) >= 20 {
		exceededDays = 1
	}

	bundle := i18n.MustLoad()
	unit := bundle.T(locale, unitKey)
	vars := func(extra map[string]string) map[string]string {
		m := map[string]string{"unit": unit}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	p95Key := "index.stats_help_p95_other"
	if exceededDays == 1 {
		p95Key = "index.stats_help_p95_one"
	}

	return &cloneStatistics{
		Mean:              meanFmt,
		Median:            medianFmt,
		Variance:          varianceFmt,
		StandardDeviation: stddevFmt,
		Minimum:           minFmt,
		Maximum:           maxFmt,
		P95:               p95Fmt,
		MeanHelp:          bundle.Tfmt(locale, "index.stats_help_mean", vars(map[string]string{"value": meanFmt})),
		MedianHelp:        bundle.Tfmt(locale, "index.stats_help_median", vars(map[string]string{"value": medianFmt})),
		VarianceHelp:      bundle.Tfmt(locale, "index.stats_help_variance", vars(map[string]string{"value": varianceFmt})),
		StandardDeviationHelp: bundle.Tfmt(locale, "index.stats_help_stddev", vars(map[string]string{
			"value": stddevFmt,
			"low":   i18n.FormatFloat(low, locale, 0),
			"high":  i18n.FormatFloat(high, locale, 0),
		})),
		MinimumHelp: bundle.Tfmt(locale, "index.stats_help_min", vars(map[string]string{"value": minFmt, "date": dateParen(minDate)})),
		MaximumHelp: bundle.Tfmt(locale, "index.stats_help_max", vars(map[string]string{"value": maxFmt, "date": dateParen(maxDate)})),
		P95Help: bundle.Tfmt(locale, p95Key, vars(map[string]string{
			"value": p95Fmt,
			"days":  strconv.Itoa(exceededDays),
		})),
	}
}

func dateParen(date string) string {
	if date == "" {
		return ""
	}
	return " (" + date + ")"
}

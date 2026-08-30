package server

import (
	"math"
	"sort"

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
}

func calculateCloneStatistics(rows []store.DayRow, locale string, compact bool) *cloneStatistics {
	return calculateCloneStatisticsFor(rows, locale, compact, func(row store.DayRow) int { return row.Count })
}

func calculateUniqueCloneStatistics(rows []store.DayRow, locale string, compact bool) *cloneStatistics {
	return calculateCloneStatisticsFor(rows, locale, compact, func(row store.DayRow) int { return row.Uniques })
}

func calculateCloneStatisticsFor(rows []store.DayRow, locale string, compact bool, value func(store.DayRow) int) *cloneStatistics {
	if len(rows) == 0 {
		return nil
	}

	values := make([]float64, len(rows))
	var sum float64
	for i, row := range rows {
		values[i] = float64(value(row))
		sum += values[i]
	}
	mean := sum / float64(len(values))

	sort.Float64s(values)
	median := values[len(values)/2]
	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + median) / 2
	}

	var squaredDifferenceSum float64
	for _, value := range values {
		difference := value - mean
		squaredDifferenceSum += difference * difference
	}
	variance := squaredDifferenceSum / float64(len(values))
	p95Index := int(math.Ceil(0.95*float64(len(values)))) - 1

	return &cloneStatistics{
		Mean:              i18n.FormatFloat(mean, locale, 2),
		Median:            i18n.FormatFloat(median, locale, 2),
		Variance:          i18n.FormatFloat(variance, locale, 2),
		StandardDeviation: i18n.FormatFloat(math.Sqrt(variance), locale, 2),
		Minimum:           i18n.FormatCount(int(values[0]), locale, compact),
		Maximum:           i18n.FormatCount(int(values[len(values)-1]), locale, compact),
		P95:               i18n.FormatCount(int(values[p95Index]), locale, compact),
	}
}

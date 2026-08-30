package i18n

import (
	"strconv"
	"strings"
)

// Grouping and decimal separators per UI locale. gghstats supports a fixed set
// of locales (see locales/*.json); anything unknown falls back to English.
var localeGroupSep = map[string]string{
	"en":    ",",
	"es":    ".",
	"de":    ".",
	"fr":    " ",
	"pt-br": ".",
}

var localeDecSep = map[string]string{
	"en":    ".",
	"es":    ",",
	"de":    ",",
	"fr":    ",",
	"pt-br": ",",
}

// FormatCount renders a non-negative integer for display according to locale.
//
// When compact is false it emits a thousands-grouped integer using the locale's
// grouping separator (e.g. en "25,746", es "25.746", fr "25 746").
//
// When compact is true it emits a metric-style abbreviation base 1000 with one
// decimal (e.g. 1234 -> "1.2k", 123456 -> "123.5k", 1234567 -> "1.2M").
// Values below 1000 are returned verbatim.
func FormatCount(n int, locale string, compact bool) string {
	if n < 0 {
		n = 0
	}
	locale = NormalizeLocale(locale)
	if compact {
		return compactCount(n, locale)
	}
	return groupedCount(n, locale)
}

// FormatFloat renders a non-negative float with a fixed number of decimal places
// and locale-aware grouping/decimal separators (e.g. en "33,146.93", es "33.146,93").
// Compact mode is not applied: statistics panels need full precision, not "33.1k".
func FormatFloat(f float64, locale string, decimals int) string {
	if f < 0 {
		f = 0
	}
	if decimals < 0 {
		decimals = 0
	}
	locale = NormalizeLocale(locale)
	raw := strconv.FormatFloat(f, 'f', decimals, 64)
	intPart, fracPart, hasFrac := strings.Cut(raw, ".")
	n, err := strconv.Atoi(intPart)
	if err != nil {
		return raw
	}
	out := groupedCount(n, locale)
	if !hasFrac || decimals == 0 {
		return out
	}
	dec := localeDecSep[locale]
	if dec == "" {
		dec = "."
	}
	return out + dec + fracPart
}

func groupedCount(n int, locale string) string {
	s := strconv.Itoa(n)
	sep := localeGroupSep[locale]
	if sep == "" {
		sep = ","
	}
	var b strings.Builder
	start := len(s) % 3
	if start == 0 {
		start = 3
	}
	b.WriteString(s[:start])
	for i := start; i < len(s); i += 3 {
		b.WriteString(sep)
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func compactCount(n int, locale string) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	dec := localeDecSep[locale]
	if dec == "" {
		dec = "."
	}
	units := []struct {
		div int64
		suf string
	}{
		{1_000_000_000_000_000, "P"},
		{1_000_000_000_000, "T"},
		{1_000_000_000, "G"},
		{1_000_000, "M"},
		{1_000, "k"},
	}
	for _, u := range units {
		if int64(n) >= u.div {
			val := float64(n) / float64(u.div)
			s := strconv.FormatFloat(val, 'f', 1, 64)
			if dec != "." {
				s = strings.Replace(s, ".", dec, 1)
			}
			return s + u.suf
		}
	}
	return strconv.Itoa(n)
}

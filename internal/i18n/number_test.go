package i18n

import "testing"

func TestFormatCountGrouped(t *testing.T) {
	cases := []struct {
		n      int
		locale string
		want   string
	}{
		{0, "en", "0"},
		{999, "en", "999"},
		{1000, "en", "1,000"},
		{25746, "en", "25,746"},
		{181937, "en", "181,937"},
		{25746, "es", "25.746"},
		{181937, "es", "181.937"},
		{25746, "de", "25.746"},
		{25746, "fr", "25 746"},
		{25746, "pt-br", "25.746"},
	}
	for _, c := range cases {
		if got := FormatCount(c.n, c.locale, false); got != c.want {
			t.Errorf("FormatCount(%d, %q, false) = %q, want %q", c.n, c.locale, got, c.want)
		}
	}
}

func TestFormatCountCompact(t *testing.T) {
	cases := []struct {
		n      int
		locale string
		want   string
	}{
		{0, "en", "0"},
		{999, "en", "999"},
		{1000, "en", "1.0k"},
		{1234, "en", "1.2k"},
		{34567, "en", "34.6k"},
		{181937, "en", "181.9k"},
		{1234567, "en", "1.2M"},
		{1234, "es", "1,2k"},
		{1234567, "es", "1,2M"},
		{181937, "de", "181,9k"},
		{181937, "fr", "181,9k"},
		{181937, "pt-br", "181,9k"},
	}
	for _, c := range cases {
		if got := FormatCount(c.n, c.locale, true); got != c.want {
			t.Errorf("FormatCount(%d, %q, true) = %q, want %q", c.n, c.locale, got, c.want)
		}
	}
}

func TestFormatCountNegative(t *testing.T) {
	if got := FormatCount(-5, "en", false); got != "0" {
		t.Errorf("negative grouped = %q, want 0", got)
	}
	if got := FormatCount(-5, "en", true); got != "0" {
		t.Errorf("negative compact = %q, want 0", got)
	}
}

func TestFormatFloat(t *testing.T) {
	cases := []struct {
		f        float64
		locale   string
		decimals int
		want     string
	}{
		{7, "en", 2, "7.00"},
		{33146.93, "en", 2, "33,146.93"},
		{33146.93, "es", 2, "33.146,93"},
		{33146.93, "de", 2, "33.146,93"},
		{33146.93, "fr", 2, "33 146,93"},
		{33146.93, "pt-br", 2, "33.146,93"},
		{1478.56, "en", 2, "1,478.56"},
		{-1, "en", 2, "0.00"},
	}
	for _, c := range cases {
		if got := FormatFloat(c.f, c.locale, c.decimals); got != c.want {
			t.Errorf("FormatFloat(%v, %q, %d) = %q, want %q", c.f, c.locale, c.decimals, got, c.want)
		}
	}
}

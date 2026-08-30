package server

import (
	"testing"
	"time"
)

func TestIndexJSONLExportFilename(t *testing.T) {
	got := indexJSONLExportFilename(time.Date(2026, 8, 24, 10, 40, 59, 0, time.UTC))
	want := "gghstats-export-20260824-1040.jsonl"
	if got != want {
		t.Fatalf("indexJSONLExportFilename() = %q, want %q", got, want)
	}
	// Local zone must still format as UTC clock.
	loc := time.FixedZone("UTC-4", -4*3600)
	got = indexJSONLExportFilename(time.Date(2026, 8, 24, 6, 40, 0, 0, loc))
	if got != want {
		t.Fatalf("local input → UTC filename = %q, want %q", got, want)
	}
}

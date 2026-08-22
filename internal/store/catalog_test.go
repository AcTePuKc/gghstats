package store

import (
	"testing"
)

func TestPinCRUD(t *testing.T) {
	s := tempDB(t)

	// list empty
	pins, err := s.ListPins()
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 0 {
		t.Fatalf("ListPins on empty = %d, want 0", len(pins))
	}

	// add
	if err := s.AddPin("owner/repo"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddPin("owner/other"); err != nil {
		t.Fatal(err)
	}

	pins, err = s.ListPins()
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 2 {
		t.Fatalf("ListPins = %d, want 2", len(pins))
	}

	// duplicate add is idempotent
	if err := s.AddPin("owner/repo"); err != nil {
		t.Fatal(err)
	}
	pins, _ = s.ListPins()
	if len(pins) != 2 {
		t.Fatalf("duplicate add changed count: %d, want 2", len(pins))
	}

	// remove
	if rem, err := s.RemovePin("owner/repo"); err != nil || !rem {
		t.Fatalf("RemovePin = %v, %v", rem, err)
	}
	pins, _ = s.ListPins()
	if len(pins) != 1 || pins[0] != "owner/other" {
		t.Fatalf("after rm: %v, want [owner/other]", pins)
	}

	// remove missing reports removed=false
	if rem, err := s.RemovePin("nope/nope"); err != nil || rem {
		t.Fatalf("RemovePin missing = %v, %v", rem, err)
	}
}

func TestFeaturedCRUD(t *testing.T) {
	s := tempDB(t)

	f, err := s.ListFeatured()
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 0 {
		t.Fatalf("ListFeatured empty = %d, want 0", len(f))
	}

	// add (idempotent)
	if err := s.AddFeatured("owner/one"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFeatured("owner/two"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFeatured("owner/one"); err != nil {
		t.Fatal(err)
	}
	f, err = s.ListFeatured()
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 2 {
		t.Fatalf("ListFeatured = %d, want 2", len(f))
	}

	// sort is ascending insertion order (owner/one before owner/two)
	if f[0].Name != "owner/one" || f[1].Name != "owner/two" {
		t.Fatalf("sort order = %v, want [owner/one owner/two]", []string{f[0].Name, f[1].Name})
	}

	// metadata update
	if err := s.UpsertFeaturedMeta("owner/one", "", "up/one", "desc", 42, true); err != nil {
		t.Fatal(err)
	}
	f, _ = s.ListFeatured()
	var got *Featured
	for i := range f {
		if f[i].Name == "owner/one" {
			got = &f[i]
		}
	}
	if got == nil {
		t.Fatal("owner/one missing after meta")
	}
	if got.UpstreamFullName != "up/one" || got.UpstreamDescription != "desc" || got.UpstreamStars != 42 || !got.Fork {
		t.Fatalf("meta = %+v, want upstream=up/one desc stars=42 fork=true", got)
	}

	// remove
	if rem, err := s.RemoveFeatured("owner/one"); err != nil || !rem {
		t.Fatalf("RemoveFeatured = %v, %v", rem, err)
	}
	f, _ = s.ListFeatured()
	if len(f) != 1 || f[0].Name != "owner/two" {
		t.Fatalf("after rm: %v, want [owner/two]", f)
	}
	if rem, err := s.RemoveFeatured("nope/nope"); err != nil || rem {
		t.Fatalf("RemoveFeatured missing = %v, %v", rem, err)
	}
}

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

	// list empty
	checkFeaturedList(t, s, nil)

	// add (idempotent duplicate is fine)
	mustAddFeatured(t, s, "owner/one")
	mustAddFeatured(t, s, "owner/two")
	mustAddFeatured(t, s, "owner/one")
	checkFeaturedList(t, s, []string{"owner/one", "owner/two"})

	// metadata update
	mustUpsertMeta(t, s, "owner/one")
	checkFeaturedMeta(t, s, "owner/one")

	// remove
	checkFeaturedRemove(t, s, "owner/one")
}

func checkFeaturedList(t *testing.T, s *Store, want []string) {
	t.Helper()
	f, err := s.ListFeatured()
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != len(want) {
		t.Fatalf("ListFeatured = %d rows, want %d", len(f), len(want))
	}
	for i, name := range want {
		if f[i].Name != name {
			t.Fatalf("order[%d] = %q, want %q", i, f[i].Name, name)
		}
	}
}

func mustAddFeatured(t *testing.T, s *Store, name string) {
	t.Helper()
	if err := s.AddFeatured(name); err != nil {
		t.Fatal(err)
	}
}

func mustUpsertMeta(t *testing.T, s *Store, name string) {
	t.Helper()
	if err := s.UpsertFeaturedMeta(name, "", "up/one", "desc", 42, true); err != nil {
		t.Fatal(err)
	}
}

func checkFeaturedMeta(t *testing.T, s *Store, name string) {
	t.Helper()
	f, err := s.ListFeatured()
	if err != nil {
		t.Fatal(err)
	}
	var got *Featured
	for i := range f {
		if f[i].Name == name {
			got = &f[i]
		}
	}
	if got == nil {
		t.Fatalf("%s missing after meta", name)
	}
	if got.UpstreamFullName != "up/one" || got.UpstreamDescription != "desc" || got.UpstreamStars != 42 || !got.Fork {
		t.Fatalf("meta = %+v, want upstream=up/one desc stars=42 fork=true", got)
	}
}

func checkFeaturedRemove(t *testing.T, s *Store, name string) {
	t.Helper()
	if rem, err := s.RemoveFeatured(name); err != nil || !rem {
		t.Fatalf("RemoveFeatured(%s) = %v, %v", name, rem, err)
	}
	checkFeaturedList(t, s, []string{"owner/two"})
	if rem, err := s.RemoveFeatured("nope/nope"); err != nil || rem {
		t.Fatalf("RemoveFeatured missing = %v, %v", rem, err)
	}
}

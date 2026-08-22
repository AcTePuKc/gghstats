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

// filterFeatured runs FilterFeatured and returns the names + total, failing
// the test on any store error so callers stay low-complexity.
func filterFeatured(t *testing.T, s *Store, q, sort, dir string, page, perPage int) ([]string, int) {
	t.Helper()
	got, total, err := s.FilterFeatured(q, sort, dir, page, perPage)
	if err != nil {
		t.Fatalf("FilterFeatured(%q,%q,%q,%d,%d): %v", q, sort, dir, page, perPage, err)
	}
	return names(got), total
}

func seedFeatured(t *testing.T, s *Store) {
	t.Helper()
	for _, name := range []string{"zzz/last", "aaa/first", "mmm/mid", "bbb/second"} {
		mustAddFeatured(t, s, name)
	}
	stars := map[string]int{"zzz/last": 10, "aaa/first": 40, "mmm/mid": 30, "bbb/second": 20}
	for name, st := range stars {
		if err := s.UpsertFeaturedMeta(name, "", "up/"+name, "desc", st, false); err != nil {
			t.Fatal(err)
		}
	}
}

func assertFeatureds(t *testing.T, got []string, want string) {
	t.Helper()
	if len(got) == 0 || got[0] != want {
		t.Fatalf("head = %v, want %q first", got, want)
	}
}

func TestFilterFeaturedDefaultOrder(t *testing.T) {
	s := tempDB(t)
	seedFeatured(t, s)
	got, total := filterFeatured(t, s, "", "sort", "asc", 1, 100)
	if total != 4 || len(got) != 4 {
		t.Fatalf("default = %d/%d, want 4/4", len(got), total)
	}
	assertFeatureds(t, got, "zzz/last")
	if got[3] != "bbb/second" {
		t.Fatalf("default tail = %v, want bbb/second", got)
	}
}

func TestFilterFeaturedSortByName(t *testing.T) {
	s := tempDB(t)
	seedFeatured(t, s)
	got, _ := filterFeatured(t, s, "", "name", "asc", 1, 100)
	assertFeatureds(t, got, "aaa/first")
	if got[3] != "zzz/last" {
		t.Fatalf("name asc tail = %v, want zzz/last", got)
	}
}

func TestFilterFeaturedSortByStars(t *testing.T) {
	s := tempDB(t)
	seedFeatured(t, s)
	got, _ := filterFeatured(t, s, "", "stars", "desc", 1, 100)
	assertFeatureds(t, got, "aaa/first")
	if got[3] != "zzz/last" {
		t.Fatalf("stars desc tail = %v, want zzz/last (10 stars)", got)
	}
}

func TestFilterFeaturedSearch(t *testing.T) {
	s := tempDB(t)
	seedFeatured(t, s)
	got, total := filterFeatured(t, s, "MM", "", "asc", 1, 100)
	if total != 1 || len(got) != 1 || got[0] != "mmm/mid" {
		t.Fatalf("search 'MM' = %v (total %d), want [mmm/mid]", got, total)
	}
}

func TestFilterFeaturedPagination(t *testing.T) {
	s := tempDB(t)
	seedFeatured(t, s)
	got, total := filterFeatured(t, s, "", "sort", "asc", 2, 3)
	if total != 4 || len(got) != 1 || got[0] != "bbb/second" {
		t.Fatalf("page2 = %v (total %d), want [bbb/second]", got, total)
	}
}

func TestFilterFeaturedUnknownSort(t *testing.T) {
	s := tempDB(t)
	seedFeatured(t, s)
	got, _ := filterFeatured(t, s, "", "bogus", "asc", 1, 100)
	if len(got) != 4 {
		t.Fatalf("unknown sort returns %d, want 4", len(got))
	}
}

func names(fs []Featured) []string {
	out := make([]string, len(fs))
	for i := range fs {
		out[i] = fs[i].Name
	}
	return out
}

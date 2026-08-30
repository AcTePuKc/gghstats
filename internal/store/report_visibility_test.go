package store

import (
	"path/filepath"
	"testing"
)

func TestReportVisibilityPolicyAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "visibility.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	seedReportPolicyRepos(t, s)
	setReportPolicy(t, s, "private/repo", ReportInclude)
	setReportPolicy(t, s, "public/repo", ReportExclude)
	if got, err := s.ListReportRepos(ReportVisibility{}, "name", "asc"); err != nil || len(got) != 1 || got[0].Name != "private/repo" {
		t.Fatalf("report repos = %#v, %v", got, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	states, err := s.ListRepoReportStates()
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 3 || states[0].ReportPolicy != ReportInclude || states[1].ReportPolicy != ReportExclude {
		t.Fatalf("persisted states = %#v", states)
	}
}

func TestReportVisibilityPrivateOverride(t *testing.T) {
	s := tempDB(t)
	if err := s.UpsertRepoWithVisibility("private/repo", "", 1, 0, 0, 0, 0, false, false, "", VisibilityPrivate); err != nil {
		t.Fatal(err)
	}
	if got, err := s.ReportRepoByName(ReportVisibility{}, "private/repo"); err != nil || got != nil {
		t.Fatalf("private default = %#v, %v", got, err)
	}
	if got, err := s.ReportRepoByName(ReportVisibility{IncludePrivate: true}, "private/repo"); err != nil || got == nil {
		t.Fatalf("private enabled = %#v, %v", got, err)
	}
}

func TestNormalizeGitHubVisibilityFailsClosedForUnknownClass(t *testing.T) {
	if got := NormalizeGitHubVisibility("internal", false); got != VisibilityUnknown {
		t.Fatalf("internal visibility = %q, want unknown", got)
	}
}

func TestReincludingRepositoryRestoresStoredReportData(t *testing.T) {
	s := tempDB(t)
	if err := s.UpsertRepoWithVisibility("o/r", "", 1, 0, 0, 0, 0, false, false, "", VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView("o/r", "2026-08-26", 7, 3); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.SetRepoReportPolicy("o/r", ReportExclude); err != nil || !ok {
		t.Fatalf("exclude = %v, %v", ok, err)
	}
	if repo, err := s.ReportRepoByName(ReportVisibility{}, "o/r"); err != nil || repo != nil {
		t.Fatalf("excluded repo=%+v err=%v", repo, err)
	}
	if ok, err := s.SetRepoReportPolicy("o/r", ReportInclude); err != nil || !ok {
		t.Fatalf("include = %v, %v", ok, err)
	}
	repo, err := s.ReportRepoByName(ReportVisibility{}, "o/r")
	if err != nil || repo == nil || repo.TotalViews != 7 {
		t.Fatalf("re-included repo=%+v err=%v", repo, err)
	}
}

func seedReportPolicyRepos(t *testing.T, s *Store) {
	t.Helper()
	for _, repo := range []struct{ name, visibility string }{
		{"public/repo", VisibilityPublic},
		{"private/repo", VisibilityPrivate},
		{"unknown/repo", VisibilityUnknown},
	} {
		if err := s.UpsertRepoWithVisibility(repo.name, "", 1, 0, 0, 0, 0, false, false, "", repo.visibility); err != nil {
			t.Fatal(err)
		}
	}
}

func setReportPolicy(t *testing.T, s *Store, name, policy string) {
	t.Helper()
	if ok, err := s.SetRepoReportPolicy(name, policy); err != nil || !ok {
		t.Fatalf("set %s=%s: ok=%v err=%v", name, policy, ok, err)
	}
}

func reportTotalsFixture(t *testing.T) *Store {
	t.Helper()
	s := tempDB(t)
	for _, repo := range []struct {
		name   string
		views  int
		clones int
	}{
		{"public/visible", 7, 3},
		{"public/excluded", 11, 5},
	} {
		if err := s.UpsertRepoWithVisibility(repo.name, "", 0, 0, 0, 0, 0, false, false, "", VisibilityPublic); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertView(repo.name, "2026-08-26", repo.views, 1); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertClone(repo.name, "2026-08-26", repo.clones, 1); err != nil {
			t.Fatal(err)
		}
		if err := s.AddFeatured(repo.name); err != nil {
			t.Fatal(err)
		}
	}
	setReportPolicy(t, s, "public/excluded", ReportExclude)
	return s
}

func TestReportScopedTotals(t *testing.T) {
	s := reportTotalsFixture(t)
	scope := ReportVisibility{}
	if got, err := s.ReportRepoCount(scope); err != nil || got != 1 {
		t.Fatalf("report repo count=%d err=%v", got, err)
	}
	if got, err := s.SumReportViewsAll(scope); err != nil || got != 7 {
		t.Fatalf("report views=%d err=%v", got, err)
	}
	if got, err := s.SumReportClonesAll(scope); err != nil || got != 3 {
		t.Fatalf("report clones=%d err=%v", got, err)
	}
}

func TestFeaturedCatalogIgnoresReportScope(t *testing.T) {
	s := reportTotalsFixture(t)
	entries, total, err := s.FilterFeatured("", "sort", "asc", 1, 25)
	if err != nil || total != 2 || len(entries) != 2 {
		t.Fatalf("catalog featured=%+v total=%d err=%v", entries, total, err)
	}
	if got, err := s.FeaturedCount(); err != nil || got != 2 {
		t.Fatalf("featured count=%d err=%v", got, err)
	}
}

func TestFeaturedCatalogWithoutCollectedRepo(t *testing.T) {
	s := tempDB(t)
	if err := s.AddFeatured("avelino/awesome-go"); err != nil {
		t.Fatal(err)
	}
	entries, total, err := s.FilterFeatured("", "sort", "asc", 1, 25)
	if err != nil || total != 1 || len(entries) != 1 || entries[0].Name != "avelino/awesome-go" {
		t.Fatalf("vitrine-only featured=%+v total=%d err=%v", entries, total, err)
	}
}

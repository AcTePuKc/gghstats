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
	if err := s.UpsertRepoWithVisibility("public/repo", "", 1, 0, 0, 0, 0, false, false, "", VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRepoWithVisibility("private/repo", "", 1, 0, 0, 0, 0, false, false, "", VisibilityPrivate); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRepoWithVisibility("unknown/repo", "", 1, 0, 0, 0, 0, false, false, "", VisibilityUnknown); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.SetRepoReportPolicy("private/repo", ReportInclude); err != nil || !ok {
		t.Fatalf("set include = %v, %v", ok, err)
	}
	if ok, err := s.SetRepoReportPolicy("public/repo", ReportExclude); err != nil || !ok {
		t.Fatalf("set exclude = %v, %v", ok, err)
	}
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

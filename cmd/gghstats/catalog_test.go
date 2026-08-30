package main

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
)

func openCatalogTestDB(t *testing.T) (*store.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cat.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}

func TestRepoCLI(t *testing.T) {
	s, path := openCatalogTestDB(t)

	// invalid name
	if err := runCatalog("repo", "add", []string{"--db", path, "bogus"}); err == nil {
		t.Fatal("expected error for invalid name")
	}

	// add
	if err := runCatalog("repo", "add", []string{"--db", path, "owner/repo"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	// duplicate add: idempotent, exit 0
	if err := runCatalog("repo", "add", []string{"--db", path, "owner/repo"}); err != nil {
		t.Fatalf("dup add: %v", err)
	}
	pins, _ := s.ListPins()
	if len(pins) != 1 || pins[0] != "owner/repo" {
		t.Fatalf("pins = %v, want [owner/repo]", pins)
	}

	// ls
	var lsOut strings.Builder
	if err := runCatalogList("repo", []string{"--db", path}, &lsOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lsOut.String(), "owner/repo") {
		t.Fatalf("ls output missing owner/repo: %q", lsOut.String())
	}

	// rm
	if err := runCatalog("repo", "rm", []string{"--db", path, "owner/repo"}); err != nil {
		t.Fatalf("rm: %v", err)
	}
	pins, _ = s.ListPins()
	if len(pins) != 0 {
		t.Fatalf("pins after rm = %v, want empty", pins)
	}

	// rm missing: error
	if err := runCatalog("repo", "rm", []string{"--db", path, "nope/nope"}); err == nil {
		t.Fatal("expected error for rm missing")
	}
}

func TestFeaturedCLI(t *testing.T) {
	s, path := openCatalogTestDB(t)

	if err := runCatalog("featured", "add", []string{"--db", path, "owner/up"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := runCatalog("featured", "add", []string{"--db", path, "owner/up"}); err != nil {
		t.Fatalf("dup add: %v", err)
	}
	f, _ := s.ListFeatured()
	if len(f) != 1 || f[0].Name != "owner/up" {
		t.Fatalf("featured = %v, want [owner/up]", f)
	}

	var lsOut strings.Builder
	if err := runCatalogList("featured", []string{"--db", path}, &lsOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lsOut.String(), "owner/up") {
		t.Fatalf("ls output missing owner/up: %q", lsOut.String())
	}

	if err := runCatalog("featured", "rm", []string{"--db", path, "owner/up"}); err != nil {
		t.Fatalf("rm: %v", err)
	}
	f, _ = s.ListFeatured()
	if len(f) != 0 {
		t.Fatalf("featured after rm = %v, want empty", f)
	}
	if err := runCatalog("featured", "rm", []string{"--db", path, "nope/nope"}); err == nil {
		t.Fatal("expected error for rm missing")
	}
}

func TestCatalogDispatch(t *testing.T) {
	// repo and featured are registered commands
	if cliCommands["repo"] == nil {
		t.Fatal("repo command not registered")
	}
	if cliCommands["featured"] == nil {
		t.Fatal("featured command not registered")
	}
}

func TestRepoReportCLI(t *testing.T) {
	s, path := openCatalogTestDB(t)
	if err := s.UpsertRepoWithVisibility("owner/repo", "", 0, 0, 0, 0, 0, false, false, "", store.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRepoWithVisibility("owner/hidden", "", 0, 0, 0, 0, 0, false, false, "", store.VisibilityUnknown); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runRepoReport([]string{"set", "--db", path, "owner/repo", store.ReportExclude}); err != nil {
		t.Fatalf("set: %v", err)
	}
	s, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if repo, err := s.ReportRepoByName(store.ReportVisibility{}, "owner/repo"); err != nil || repo != nil {
		t.Fatalf("excluded repo=%+v err=%v", repo, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runRepoReport([]string{"ls", "--db", path}); err != nil {
		t.Fatalf("ls: %v", err)
	}
	var jsonOut strings.Builder
	if err := runRepoReportLs([]string{"--db", path, "--json", "--policy", store.ReportExclude}, &jsonOut); err != nil {
		t.Fatalf("ls --json: %v", err)
	}
	var rows []store.RepoReportState
	if err := json.Unmarshal([]byte(jsonOut.String()), &rows); err != nil {
		t.Fatalf("decode json: %v body=%s", err, jsonOut.String())
	}
	if len(rows) != 1 || rows[0].Name != "owner/repo" || rows[0].ReportPolicy != store.ReportExclude {
		t.Fatalf("json filter=%+v", rows)
	}
	var unknownOut strings.Builder
	if err := runRepoReportLs([]string{"--db", path, "--visibility", store.VisibilityUnknown}, &unknownOut); err != nil {
		t.Fatalf("ls --visibility: %v", err)
	}
	if !strings.Contains(unknownOut.String(), "owner/hidden") || strings.Contains(unknownOut.String(), "owner/repo") {
		t.Fatalf("visibility filter output=%q", unknownOut.String())
	}
	if err := runRepoReportLs([]string{"--db", path, "--visibility", "nope"}, io.Discard); err == nil {
		t.Fatal("expected invalid visibility")
	}
	if err := runRepoReport([]string{"set", "--db", path, "owner/repo", "invalid"}); err == nil {
		t.Fatal("expected invalid policy error")
	}
	if err := runRepoReport([]string{"set", "--db", path, "missing/repo", store.ReportInclude}); err == nil {
		t.Fatal("expected missing repository error")
	}
}

package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/store"
)

// resolveTracked resolves the tracked repo set: FILTER ∪ pins (R2/R8), with
// explicit opts.Repos bypassing the pin union (R10).

func TestTrackedUnionNoPins(t *testing.T) {
	// R2: empty catalog → tracked set unchanged (no pin union applied).
	all := []github.Repo{{FullName: "a/1"}, {FullName: "a/2", Fork: true}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(all)
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL
	s := tempStore(t)

	repos, err := resolveTracked(c, s, Options{Filter: "*,!fork"})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].FullName != "a/1" {
		t.Fatalf("tracked = %+v, want [a/1] (fork dropped, no pins)", repos)
	}
}

func TestTrackedUnionAddsPin(t *testing.T) {
	// R8: pin overrides !fork exclusion (union).
	all := []github.Repo{{FullName: "a/1"}, {FullName: "a/2", Fork: true}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(all)
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL
	s := tempStore(t)
	if err := s.AddPin("a/2"); err != nil {
		t.Fatal(err)
	}

	repos, err := resolveTracked(c, s, Options{Filter: "*,!fork"})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("tracked = %+v, want 2 (pin union restores fork)", repos)
	}
	names := map[string]bool{}
	for _, r := range repos {
		names[r.FullName] = true
	}
	if !names["a/1"] || !names["a/2"] {
		t.Fatalf("tracked names = %v, want a/1 + a/2", names)
	}
}

func TestTrackedExplicitIgnoredPins(t *testing.T) {
	// R10: opts.Repos explicit list does not absorb pins.
	s := tempStore(t)
	if err := s.AddPin("pinned/repo"); err != nil {
		t.Fatal(err)
	}
	c := github.NewClient("tok")
	c.BaseURL = "http://should-not-be-called.example"

	repos, err := resolveTracked(c, s, Options{Repos: []string{"acme/a"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].FullName != "acme/a" {
		t.Fatalf("tracked = %+v, want exactly [acme/a] (pin not merged)", repos)
	}
}

func TestSyncFeaturedMeta(t *testing.T) {
	// R11: featured metadata refresh uses only GitHub.Repo (no traffic/*).
	// Fork entry resolves its parent for description + stars; plain entry uses
	// itself as upstream. No /traffic calls may fire for featured-only entries.
	hitTraffic := false
	mux := http.NewServeMux()

	// fork entry: parent carries stars + description
	mux.HandleFunc("/repos/hrodrig/awesome-readme", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"full_name":        "hrodrig/awesome-readme",
			"fork":             true,
			"parent":           map[string]any{"full_name": "matiassingers/awesome-readme"},
			"stargazers_count": 7,
		})
	})
	mux.HandleFunc("/repos/matiassingers/awesome-readme", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"full_name":        "matiassingers/awesome-readme",
			"fork":             false,
			"description":      "A curated list of awesome readmes",
			"stargazers_count": 15000,
		})
	})
	// plain (non-fork) entry: itself is the upstream
	mux.HandleFunc("/repos/hrodrig/plain", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"full_name":        "hrodrig/plain",
			"fork":             false,
			"description":      "plain repo",
			"stargazers_count": 42,
		})
	})
	// traffic must never be hit for featured-only entries
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/traffic/") {
			hitTraffic = true
		}
		http.NotFound(w, r)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL
	s := tempStore(t)

	if err := s.AddFeatured("hrodrig/awesome-readme"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFeatured("hrodrig/plain"); err != nil {
		t.Fatal(err)
	}

	if err := syncFeaturedMeta(c, s); err != nil {
		t.Fatal(err)
	}

	rows, err := s.ListFeatured()
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]store.Featured{}
	for _, f := range rows {
		byName[f.Name] = f
	}

	fork := byName["hrodrig/awesome-readme"]
	if !fork.Fork {
		t.Errorf("fork entry: Fork=false")
	}
	if fork.UpstreamFullName != "matiassingers/awesome-readme" {
		t.Errorf("fork upstream = %q, want matiassingers/awesome-readme", fork.UpstreamFullName)
	}
	if fork.ParentFullName != "matiassingers/awesome-readme" {
		t.Errorf("fork parent = %q", fork.ParentFullName)
	}
	if fork.UpstreamStars != 15000 {
		t.Errorf("fork upstream stars = %d, want 15000", fork.UpstreamStars)
	}
	if fork.UpstreamDescription != "A curated list of awesome readmes" {
		t.Errorf("fork upstream desc = %q", fork.UpstreamDescription)
	}

	plain := byName["hrodrig/plain"]
	if plain.Fork {
		t.Errorf("plain entry: Fork=true")
	}
	if plain.UpstreamFullName != "hrodrig/plain" {
		t.Errorf("plain upstream = %q, want hrodrig/plain", plain.UpstreamFullName)
	}
	if plain.UpstreamStars != 42 {
		t.Errorf("plain stars = %d, want 42", plain.UpstreamStars)
	}

	if hitTraffic {
		t.Errorf("featured metadata sync hit /traffic/* (R11 violation)")
	}
}

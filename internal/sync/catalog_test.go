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

	checkFeaturedEntry(t, byName["hrodrig/awesome-readme"], featuredWant{
		fork: true, upstream: "matiassingers/awesome-readme",
		parent: "matiassingers/awesome-readme", stars: 15000,
		desc: "A curated list of awesome readmes",
	})
	checkFeaturedEntry(t, byName["hrodrig/plain"], featuredWant{
		fork: false, upstream: "hrodrig/plain", stars: 42,
	})

	if hitTraffic {
		t.Errorf("featured metadata sync hit /traffic/* (R11 violation)")
	}
}

type featuredWant struct {
	fork     bool
	upstream string
	parent   string
	stars    int
	desc     string
}

func checkFeaturedEntry(t *testing.T, f store.Featured, want featuredWant) {
	t.Helper()
	if f.Fork != want.fork {
		t.Errorf("entry %q: Fork=%v, want %v", f.Name, f.Fork, want.fork)
	}
	if want.upstream != "" && f.UpstreamFullName != want.upstream {
		t.Errorf("entry %q: upstream = %q, want %q", f.Name, f.UpstreamFullName, want.upstream)
	}
	if want.parent != "" && f.ParentFullName != want.parent {
		t.Errorf("entry %q: parent = %q, want %q", f.Name, f.ParentFullName, want.parent)
	}
	if want.stars != 0 && f.UpstreamStars != want.stars {
		t.Errorf("entry %q: stars = %d, want %d", f.Name, f.UpstreamStars, want.stars)
	}
	if want.desc != "" && f.UpstreamDescription != want.desc {
		t.Errorf("entry %q: desc = %q, want %q", f.Name, f.UpstreamDescription, want.desc)
	}
}

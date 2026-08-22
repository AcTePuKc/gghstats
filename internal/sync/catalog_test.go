package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hrodrig/gghstats/internal/github"
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

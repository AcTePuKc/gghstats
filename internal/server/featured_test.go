package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// R1: empty catalog → GET / has no href="/featured".
func TestFeaturedNavHiddenWhenEmpty(t *testing.T) {
	h := New(Config{Store: testStore(t)})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(w.Body.String(), `href="/featured"`) {
		t.Error("R1: empty catalog must not render a /featured nav link")
	}
}

// R5: API-only → GET /featured is not an HTML dashboard page.
func TestFeaturedAPIOonlyNotFound(t *testing.T) {
	h := New(Config{Store: testStore(t), APIToken: "tok", APIOnly: true, DisableMetrics: true})
	req := httptest.NewRequest(http.MethodGet, "/featured", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("R5: API-only /featured status = %d, want 404", w.Code)
	}
}

// R9: after featured add, GET / has the link; after removing the last row,
// the link is gone again.
func TestFeaturedNavToggle(t *testing.T) {
	db := testStore(t)
	if err := db.AddFeatured("hrodrig/awesome-readme"); err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `href="/featured"`) {
		t.Error("R9: after featured add, GET / must render a /featured nav link")
	}

	if _, err := db.RemoveFeatured("hrodrig/awesome-readme"); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if strings.Contains(w.Body.String(), `href="/featured"`) {
		t.Error("R9: after removing the last featured row, the /featured nav link must disappear")
	}
}

// The /featured page itself: 200 and renders the fork card (upstream link).
func TestFeaturedPageShell(t *testing.T) {
	db := testStore(t)
	if err := db.AddFeatured("hrodrig/awesome-readme"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertFeaturedMeta(
		"hrodrig/awesome-readme",
		"matiassingers/awesome-readme",
		"matiassingers/awesome-readme",
		"A curated list of awesome readmes",
		15000,
		true,
	); err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db})
	req := httptest.NewRequest(http.MethodGet, "/featured", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "matiassingers/awesome-readme") {
		t.Error("expected upstream repo name in featured page")
	}
	if !strings.Contains(body, "15000") {
		t.Error("expected upstream stars in featured page")
	}
}

// The /featured page renders a friendly empty state when the showcase is empty.
func TestFeaturedPageEmpty(t *testing.T) {
	h := New(Config{Store: testStore(t)})
	req := httptest.NewRequest(http.MethodGet, "/featured", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `data-gghstats-role="featured-empty"`) {
		t.Errorf("expected empty-state marker in featured page body")
	}
}

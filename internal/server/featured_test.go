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
	if !strings.Contains(body, "15,000") {
		t.Error("expected formatted upstream stars (thousands separator) in featured page")
	}
	// The card title/link must be an external GitHub URL, not an internal route
	// (https://github.com/{upstream_full_name}). Regression: an earlier draft
	// linked to href="/{name}", which 404s on the dashboard.
	if !strings.Contains(body, `href="https://github.com/matiassingers/awesome-readme"`) {
		t.Error("expected card title/link to point to https://github.com/<upstream>")
	}
	if strings.Contains(body, `href="/matiassingers/awesome-readme"`) {
		t.Error("card title must not link to an internal route")
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

// With GGHSTATS_COMPACT_NUMBERS (CompactNumbers=true), large upstream stars
// render in compact metric notation (e.g. 15.0k) rather than 15,000.
func TestFeaturedPageCompactNumbers(t *testing.T) {
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
	h := New(Config{Store: db, CompactNumbers: true})
	req := httptest.NewRequest(http.MethodGet, "/featured", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "15.0k") {
		t.Errorf("expected compact stars (15.0k) in featured page, got body without it")
	}
	if strings.Contains(body, "15,000") {
		t.Errorf("compact mode must not render thousands separators (15,000)")
	}
}

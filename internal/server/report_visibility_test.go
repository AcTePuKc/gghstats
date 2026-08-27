package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestExcludedRepositoryDoesNotLeakAcrossReportRoutes(t *testing.T) {
	db := testStore(t)
	if err := db.UpsertRepoWithVisibility("public/ok", "shown", 1, 0, 0, 0, 0, false, false, "", store.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertRepoWithVisibility("secret/nope", "must not leak", 99, 0, 0, 0, 0, false, false, "", store.VisibilityPrivate); err != nil {
		t.Fatal(err)
	}
	if ok, err := db.SetRepoReportPolicy("secret/nope", store.ReportExclude); err != nil || !ok {
		t.Fatal(err)
	}
	h := New(Config{Store: db, APIToken: "token", DisableMetrics: true})
	for _, path := range []string{"/", "/export.jsonl", "/sitemap.xml", "/api/repos", "/api/v1/badge/secret/nope"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if strings.HasPrefix(path, "/api/repos") {
			r.Header.Set("x-api-token", "token")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if strings.Contains(w.Body.String(), "secret/nope") || strings.Contains(w.Body.String(), "must not leak") {
			t.Fatalf("%s leaked excluded repo: %s", path, w.Body.String())
		}
	}
	for _, path := range []string{"/secret/nope", "/api/v1/repos/secret/nope", "/api/v1/repos/secret/nope/traffic", "/api/v1/repos/secret/nope/stars", "/api/v1/repos/secret/nope/popular"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if strings.HasPrefix(path, "/api/") {
			r.Header.Set("x-api-token", "token")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d, want 404", path, w.Code)
		}
		if strings.Contains(w.Body.String(), "secret/nope") {
			t.Fatalf("%s echoed excluded repo", path)
		}
	}
	r := httptest.NewRequest(http.MethodGet, "/h2h?a=secret/nope&b=public/ok", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if strings.Contains(w.Body.String(), "secret/nope") {
		t.Fatal("H2H page echoed excluded repo")
	}
}

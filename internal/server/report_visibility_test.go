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

func TestMetricsDoesNotLeakExcludedRepositoryOrFilter(t *testing.T) {
	db := testStore(t)
	const repo = "secret/nope"
	if err := db.UpsertRepoWithVisibility(repo, "must not leak", 1, 0, 0, 0, 0, false, false, "", store.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertView(repo, "2026-08-26", 7, 3); err != nil {
		t.Fatal(err)
	}
	reg, dom := NewMetricsRegistry(MetricsRegistryConfig{
		Store:            db,
		PerRepoEnabled:   true,
		ReportVisibility: store.ReportVisibility{},
	})
	h := New(Config{Store: db, MetricsRegistry: reg, DomainMetrics: dom})

	scrape := func() string {
		t.Helper()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, MetricsPath, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("metrics status=%d", w.Code)
		}
		return w.Body.String()
	}

	if ok, err := db.SetRepoReportPolicy(repo, store.ReportExclude); err != nil || !ok {
		t.Fatalf("exclude repo: ok=%v err=%v", ok, err)
	}
	excluded := scrape()
	if strings.Contains(excluded, repo) || strings.Contains(excluded, `owner="secret"`) || strings.Contains(excluded, `repo="nope"`) || strings.Contains(excluded, `filter=`) {
		t.Fatalf("metrics leaked excluded repo or raw filter: %s", excluded)
	}
	if !strings.Contains(excluded, "gghstats_repos_total 0") {
		t.Fatalf("metrics must retain report-scoped repo total, got: %s", excluded)
	}

	if ok, err := db.SetRepoReportPolicy(repo, store.ReportInclude); err != nil || !ok {
		t.Fatalf("include repo: ok=%v err=%v", ok, err)
	}
	included := scrape()
	if !strings.Contains(included, "gghstats_repos_total 1") {
		t.Fatalf("included repo was not counted, got: %s", included)
	}
	if strings.Contains(included, repo) || strings.Contains(included, `filter=`) {
		t.Fatalf("aggregate metrics must not expose repository configuration: %s", included)
	}
	if !strings.Contains(included, `owner="secret"`) || !strings.Contains(included, `repo="nope"`) {
		t.Fatalf("report-visible per-repo metric labels were not restored: %s", included)
	}
}

func TestFeaturedAPIAndSitemapRespectReportVisibility(t *testing.T) {
	db := testStore(t)
	for _, repo := range []struct {
		name       string
		visibility string
	}{
		{"public/visible", store.VisibilityPublic},
		{"secret/excluded", store.VisibilityPrivate},
	} {
		if err := db.UpsertRepoWithVisibility(repo.name, "", 0, 0, 0, 0, 0, false, false, "", repo.visibility); err != nil {
			t.Fatal(err)
		}
		if err := db.AddFeatured(repo.name); err != nil {
			t.Fatal(err)
		}
		if err := db.UpsertFeaturedMeta(repo.name, "", repo.name, repo.name+" metadata", 1, false); err != nil {
			t.Fatal(err)
		}
	}
	if ok, err := db.SetRepoReportPolicy("secret/excluded", store.ReportExclude); err != nil || !ok {
		t.Fatalf("exclude repo: ok=%v err=%v", ok, err)
	}

	h := New(Config{Store: db, APIToken: "token", DisableMetrics: true, PublicURL: "https://stats.example.com"})
	apiReq := httptest.NewRequest(http.MethodGet, "/api/v1/featured", nil)
	apiReq.Header.Set("x-api-token", "token")
	apiRes := httptest.NewRecorder()
	h.ServeHTTP(apiRes, apiReq)
	if apiRes.Code != http.StatusOK {
		t.Fatalf("featured API status=%d body=%s", apiRes.Code, apiRes.Body.String())
	}
	if strings.Contains(apiRes.Body.String(), "secret/excluded") || strings.Contains(apiRes.Body.String(), "excluded metadata") {
		t.Fatalf("featured API leaked excluded repository: %s", apiRes.Body.String())
	}
	if !strings.Contains(apiRes.Body.String(), `"total_count":1`) || !strings.Contains(apiRes.Body.String(), "public/visible") {
		t.Fatalf("featured API did not retain report-visible entry: %s", apiRes.Body.String())
	}

	sitemapReq := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	sitemapReq.Host = "stats.example.com"
	sitemapRes := httptest.NewRecorder()
	h.ServeHTTP(sitemapRes, sitemapReq)
	if sitemapRes.Code != http.StatusOK {
		t.Fatalf("sitemap status=%d body=%s", sitemapRes.Code, sitemapRes.Body.String())
	}
	if strings.Contains(sitemapRes.Body.String(), "secret/excluded") {
		t.Fatalf("sitemap leaked excluded repository: %s", sitemapRes.Body.String())
	}
	for _, want := range []string{"https://stats.example.com/public/visible", "https://stats.example.com/featured"} {
		if !strings.Contains(sitemapRes.Body.String(), want) {
			t.Fatalf("sitemap missing report-visible URL %q: %s", want, sitemapRes.Body.String())
		}
	}
}

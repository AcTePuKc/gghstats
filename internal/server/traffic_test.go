package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestAPIRepoTrafficUnauthorized(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAPIRepoTrafficDisabledWithoutToken(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, APIToken: ""})

	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestAPIRepoTrafficReturnsSeries(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	db.UpsertView("a/b", today, 50, 20)
	db.UpsertClone("a/b", yesterday, 5, 2)
	db.UpsertClone("a/b", today, 12, 4)

	handler := New(Config{Store: db, APIToken: "secret"})
	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=30", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d body=%q", w.Code, w.Body.String())
	}
	var resp repoTrafficResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Name != "a/b" || resp.Days != 30 {
		t.Fatalf("resp = %+v", resp)
	}
	if len(resp.Clones) != 2 || len(resp.Views) != 1 {
		t.Fatalf("clones=%d views=%d", len(resp.Clones), len(resp.Views))
	}
	if resp.Clones[0].Date != yesterday || resp.Clones[0].Count != 5 {
		t.Fatalf("clones[0] = %+v, want date %s count 5", resp.Clones[0], yesterday)
	}
}

func TestAPIRepoTrafficNotFound(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/repos/missing/r/traffic", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAPIRepoTrafficInvalidDays(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=abc", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid days") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestAPIRepoTrafficDaysExceedsMax(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=99999", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAPIRepoTrafficEmptyHistory(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=7", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var resp repoTrafficResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Clones) != 0 || len(resp.Views) != 0 {
		t.Fatalf("expected empty series, got clones=%d views=%d", len(resp.Clones), len(resp.Views))
	}
}

func TestParseTrafficDays(t *testing.T) {
	got, err := parseTrafficDays("")
	if err != nil || got != defaultTrafficDays {
		t.Fatalf("default: got %d err %v", got, err)
	}
	if _, err := parseTrafficDays("x"); err == nil {
		t.Fatal("expected error for non-numeric")
	}
	if _, err := parseTrafficDays("-1"); err == nil {
		t.Fatal("expected error for negative")
	}
}

func TestTrafficDateRangeUTC(t *testing.T) {
	from, to, err := trafficDateRangeUTC(7, "2026-01-01", true)
	if err != nil {
		t.Fatal(err)
	}
	if to == "" || from == "" {
		t.Fatalf("from=%q to=%q", from, to)
	}
	from0, to0, err := trafficDateRangeUTC(0, "", false)
	if err != nil || from0 != to0 {
		t.Fatalf("no extent: from=%q to=%q err=%v", from0, to0, err)
	}
}

func TestAPIRepoTrafficAllTime(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	db.UpsertClone("a/b", "2026-01-01", 1, 1)
	db.UpsertView("a/b", "2026-02-01", 2, 1)

	handler := New(Config{Store: db, APIToken: "secret"})
	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=0", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var resp repoTrafficResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.From != "2026-01-01" {
		t.Errorf("from = %q, want 2026-01-01", resp.From)
	}
	if len(resp.Clones) != 1 || len(resp.Views) != 1 {
		t.Fatalf("clones=%d views=%d", len(resp.Clones), len(resp.Views))
	}
}

func TestAPIRepoTrafficOmitsCachedRowOutsideLatestCoverage(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	// An earlier revision exists for the 26th, but the latest response only
	// confirms 25th and 27th, making 26th unknown to both API and chart.
	if err := db.UpsertView("a/b", "2026-08-26", 9, 4); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := db.RecordTrafficMetricSuccess("a/b", "views", []store.DayRow{{Date: "2026-08-25", Count: 0, Uniques: 0}, {Date: "2026-08-27", Count: 3, Uniques: 1}}, now, "2026-08-25", "2026-08-27"); err != nil {
		t.Fatal(err)
	}
	handler := New(Config{Store: db, APIToken: "secret"})
	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=0", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp repoTrafficResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Views) != 2 || resp.Views[0].Date != "2026-08-25" || resp.Views[0].Count != 0 || resp.Views[1].Date != "2026-08-27" {
		t.Fatalf("sparse views=%+v", resp.Views)
	}
	rows, err := db.ViewsByRange("a/b", "2026-08-25", "2026-08-27")
	if err != nil {
		t.Fatal(err)
	}
	chart, err := denseTrafficChartWithCoverage(db, "a/b", "views", "2026-08-25", "2026-08-27", rows)
	if err != nil || chart[1].Count != nil {
		t.Fatalf("dense chart=%+v err=%v", chart, err)
	}
}

func TestAPIRepoTrafficDenseFillsCalendarGaps(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	db.UpsertClone("a/b", yesterday, 5, 2)
	db.UpsertClone("a/b", today, 12, 4)

	handler := New(Config{Store: db, APIToken: "secret"})
	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=3&dense=1", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp repoTrafficDenseResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Dense || len(resp.Clones) != 3 {
		t.Fatalf("dense=%v clones=%d, want 3", resp.Dense, len(resp.Clones))
	}
	// Oldest day in a 3-day window has no row → null count/uniques.
	if resp.Clones[0].Count != nil || resp.Clones[0].Uniques != nil {
		t.Fatalf("gap day should be null: %+v", resp.Clones[0])
	}
	if resp.Clones[1].Count == nil || *resp.Clones[1].Count != 5 {
		t.Fatalf("yesterday = %+v, want count 5", resp.Clones[1])
	}
	if resp.Clones[2].Count == nil || *resp.Clones[2].Count != 12 {
		t.Fatalf("today = %+v, want count 12", resp.Clones[2])
	}
}

func TestAPIRepoTrafficDownloadImpliesDenseAndDisposition(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("owner/repo", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, APIToken: "secret"})
	req := httptest.NewRequest("GET", "/api/v1/repos/owner/repo/traffic?days=2&download=1", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}
	cd := w.Header().Get("Content-Disposition")
	wantName := trafficJSONFilename("owner/repo", time.Now().UTC())
	if !strings.Contains(cd, wantName) {
		t.Fatalf("Content-Disposition=%q, want filename %q", cd, wantName)
	}
	var resp repoTrafficDenseResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Dense || len(resp.Clones) != 2 {
		t.Fatalf("dense=%v len=%d", resp.Dense, len(resp.Clones))
	}
}

func TestRepoTrafficJSONExportPublicWithoutAPIToken(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, APIToken: ""})
	req := httptest.NewRequest("GET", "/a/b/traffic.json?days=2", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, trafficJSONFilename("a/b", time.Now().UTC())) {
		t.Fatalf("Content-Disposition=%q", cd)
	}
	var resp repoTrafficDenseResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Dense || resp.Name != "a/b" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestRepoTrafficJSONExportRequiresTokenWhenConfigured(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/a/b/traffic.json", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no token: status=%d, want 401", w.Code)
	}

	req = httptest.NewRequest("GET", "/a/b/traffic.json", nil)
	req.Header.Set("x-api-token", "wrong")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad token: status=%d, want 401", w.Code)
	}

	req = httptest.NewRequest("GET", "/a/b/traffic.json?days=1", nil)
	req.Header.Set("x-api-token", "secret")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("good token: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestRepoTrafficJSONExportExcludedIs404(t *testing.T) {
	db := testStore(t)
	if err := db.UpsertRepoWithVisibility("secret/nope", "hidden", 1, 0, 0, 0, 0, false, false, "", store.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	setTestReportPolicy(t, db, "secret/nope", store.ReportExclude)
	handler := New(Config{Store: db, APIToken: ""})
	req := httptest.NewRequest("GET", "/secret/nope/traffic.json", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", w.Code)
	}
	if strings.Contains(w.Body.String(), "secret/nope") {
		t.Fatalf("leaked name: %s", w.Body.String())
	}
}

func TestTrafficJSONFilename(t *testing.T) {
	got := trafficJSONFilename("hrodrig/gghstats", time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC))
	want := "gghstats-hrodrig-gghstats-traffic-20260830.json"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

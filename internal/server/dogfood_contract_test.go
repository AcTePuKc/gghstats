package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func dogfoodJSON(t *testing.T, h http.Handler, token, path string) map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("x-api-token", token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", path, w.Code, w.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s decode: %v", path, err)
	}
	return out
}

func requireJSONField(t *testing.T, m map[string]interface{}, key string) {
	t.Helper()
	if m[key] == nil {
		t.Fatalf("missing %s", key)
	}
}

// TestDogfoodContract_APIOnly rebuilds index, repo, H2H, and Featured from documented JSON endpoints alone.
func TestDogfoodContract_APIOnly(t *testing.T) {
	db := seedH2HRepos(t)
	today := time.Now().UTC().Format("2006-01-02")
	_ = db.UpsertStar("a/one", today, 10)
	_ = db.UpsertStar("b/two", today, 5)

	const token = "dogfood-token"
	h := New(Config{
		Store:          db,
		APIToken:       token,
		APIOnly:        true,
		DisableMetrics: true,
	})

	index := dogfoodJSON(t, h, token, "/api/repos?sort=name&dir=asc")
	if index["total_count"].(float64) < 2 {
		t.Fatalf("index total_count = %v", index["total_count"])
	}
	requireJSONField(t, index, "items")
	requireJSONField(t, dogfoodJSON(t, h, token, "/api/v1/charts/index-clones"), "series")

	repo := dogfoodJSON(t, h, token, "/api/v1/repos/a/one")
	requireJSONField(t, repo, "repo")
	requireJSONField(t, repo, "momentum_7d")

	traffic := dogfoodJSON(t, h, token, "/api/v1/repos/a/one/traffic?days=30")
	requireJSONField(t, traffic, "clones")
	requireJSONField(t, traffic, "views")
	requireJSONField(t, dogfoodJSON(t, h, token, "/api/v1/repos/a/one/stars"), "stars")

	popular := dogfoodJSON(t, h, token, "/api/v1/repos/a/one/popular")
	requireJSONField(t, popular, "referrers")
	requireJSONField(t, popular, "paths")

	h2hResp := dogfoodJSON(t, h, token, "/api/v1/h2h?a=a/one&b=b/two&w=7d")
	requireJSONField(t, h2hResp, "result")
	requireJSONField(t, h2hResp, "charts")

	featured := dogfoodJSON(t, h, token, "/api/v1/featured")
	requireJSONField(t, featured, "items")
	requireJSONField(t, featured, "total_count")
}

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func featuredHandler(t *testing.T) http.Handler {
	t.Helper()
	db := testStore(t)
	if err := db.AddFeatured("z/zebra"); err != nil {
		t.Fatal(err)
	}
	if err := db.AddFeatured("a/alpha"); err != nil {
		t.Fatal(err)
	}
	makeFeaturedReportable(t, db, "a/alpha")
	makeFeaturedReportable(t, db, "z/zebra")
	if err := db.UpsertFeaturedMeta("a/alpha", "", "up/alpha", "Alpha desc", 50, false); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertFeaturedMeta("z/zebra", "up/parent", "up/zebra", "Zebra", 10, true); err != nil {
		t.Fatal(err)
	}
	return New(Config{Store: db, APIToken: "tok", DisableMetrics: true})
}

func decodeFeaturedJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestAPIFeaturedListSortsByName(t *testing.T) {
	h := featuredHandler(t)
	body := decodeFeaturedJSON(t, apiGET(t, h, "/api/v1/featured?sort=name&dir=asc", "tok"))
	if body["total_count"].(float64) != 2 {
		t.Fatalf("total_count = %v", body["total_count"])
	}
	items := body["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("items len = %d", len(items))
	}
	first := items[0].(map[string]interface{})
	if first["name"] != "a/alpha" {
		t.Fatalf("sort name asc first = %v", first["name"])
	}
	if first["upstream_stars"].(float64) != 50 {
		t.Fatalf("upstream_stars = %v", first["upstream_stars"])
	}
	if _, hasTraffic := first["total_clones"]; hasTraffic {
		t.Fatal("featured items must not include traffic fields")
	}
}

func TestAPIFeaturedQuery(t *testing.T) {
	h := featuredHandler(t)
	body := decodeFeaturedJSON(t, apiGET(t, h, "/api/v1/featured?q=zebra&sort=name&dir=asc", "tok"))
	if body["total_count"].(float64) != 1 {
		t.Fatalf("q filter count = %v", body["total_count"])
	}
}

func TestAPIFeaturedPagination(t *testing.T) {
	h := featuredHandler(t)
	body := decodeFeaturedJSON(t, apiGET(t, h, "/api/v1/featured?sort=name&dir=asc&page=1&per_page=1", "tok"))
	if body["page"].(float64) != 1 {
		t.Fatalf("page = %v", body["page"])
	}
	if body["per_page"].(float64) != 1 {
		t.Fatalf("per_page = %v", body["per_page"])
	}
	if body["total_pages"].(float64) != 2 {
		t.Fatalf("total_pages = %v", body["total_pages"])
	}
	items := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("page size = %d", len(items))
	}
}

func TestAPIFeaturedEmpty(t *testing.T) {
	db := testStore(t)
	h := New(Config{Store: db, APIToken: "tok", DisableMetrics: true, APIOnly: true})
	body := decodeFeaturedJSON(t, apiGET(t, h, "/api/v1/featured", "tok"))
	if body["total_count"].(float64) != 0 {
		t.Fatalf("total_count = %v", body["total_count"])
	}
	items, ok := body["items"].([]interface{})
	if !ok {
		t.Fatalf("items = %#v", body["items"])
	}
	if len(items) != 0 {
		t.Fatalf("items len = %d", len(items))
	}
}

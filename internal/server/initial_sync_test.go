package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/sync"
)

func TestIndexShowsInitialSyncStateForEmptyDatabase(t *testing.T) {
	block := make(chan struct{})
	released := false
	release := func() {
		if !released {
			close(block)
			released = true
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			<-block
			_ = json.NewEncoder(w).Encode([]github.Repo{})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(func() {
		release()
		srv.Close()
	})

	db := testStore(t)
	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	coord := sync.NewCoordinator(gh, db, sync.Options{Filter: "*"})
	if err := coord.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for !coord.Status().Running && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !coord.Status().Running {
		t.Fatal("sync did not start")
	}

	h := New(Config{
		Store:              db,
		SyncCoordinator:    coord,
		SyncOnStartup:      true,
		InitialSyncPending: true,
		DisableMetrics:     true,
	})
	if err := db.UpsertRepo("partial/repo", "", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "Initial sync in progress") {
		t.Fatal("index should explain that the initial sync is running")
	}
	if !strings.Contains(body, "The first results are arriving") {
		t.Fatal("partial index should explain that the initial sync is still running")
	}
	if !strings.Contains(body, `data-sync-running="true"`) {
		t.Fatal("partial index should keep the automatic refresh marker")
	}
	if strings.Contains(body, "Sync one repository with:") {
		t.Fatal("initial sync state should not show the manual empty-state hint")
	}

	release()
	deadline = time.Now().Add(time.Second)
	for coord.Status().Running && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
}

package api

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/realtime-logs/internal/db"
	"github.com/example/realtime-logs/internal/models"
	"github.com/gin-gonic/gin"
)

type captureStore struct {
	appended []models.LogEntry
	params   db.QueryParams
	queried  bool
	err      error
}

func (s *captureStore) Append(entries []models.LogEntry) error {
	if s.err != nil {
		return s.err
	}
	s.appended = append(s.appended, entries...)
	return nil
}

func (s *captureStore) Query(params db.QueryParams) ([]models.LogEntry, int) {
	s.queried = true
	s.params = params
	return nil, 0
}

func TestIngestRejectsInvalidBatchItem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &captureStore{}
	router := gin.New()
	(&IngestHandler{Store: store}).Register(router)

	body := []byte(`{"items":[{"org_id":"acme","level":"info","message":"ok"},{"org_id":"acme","level":"error"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}
	if len(store.appended) != 0 {
		t.Fatalf("expected no appended entries, got %d", len(store.appended))
	}
}

func TestIngestReturnsUnavailableWhenQueueIsFull(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &captureStore{}
	queue := make(chan []models.LogEntry, 1)
	queue <- []models.LogEntry{{OrgID: "acme", Level: "info", Message: "queued"}}

	router := gin.New()
	(&IngestHandler{Store: store, queue: queue}).Register(router)

	body := []byte(`{"org_id":"acme","level":"info","message":"new"}`)
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}
}

func TestIngestSyncPathReportsStoreError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	(&IngestHandler{Store: &captureStore{err: errors.New("boom")}}).Register(router)

	body := []byte(`{"org_id":"acme","level":"info","message":"new"}`)
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.Code)
	}
}

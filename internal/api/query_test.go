package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestQueryRejectsInvalidTimeFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &captureStore{}
	router := gin.New()
	(&QueryHandler{Store: store}).Register(router)

	req := httptest.NewRequest(http.MethodGet, "/query?from=not-a-date", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}
	if store.queried {
		t.Fatal("expected store not to be queried")
	}
}

func TestQueryNormalizesPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &captureStore{}
	router := gin.New()
	(&QueryHandler{Store: store}).Register(router)

	req := httptest.NewRequest(http.MethodGet, "/query?limit=5000&offset=-8", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}
	if store.params.Limit != 1000 {
		t.Fatalf("expected limit 1000, got %d", store.params.Limit)
	}
	if store.params.Offset != 0 {
		t.Fatalf("expected offset 0, got %d", store.params.Offset)
	}
}

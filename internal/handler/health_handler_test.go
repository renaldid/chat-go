package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockDatabasePinger struct {
	err error
}

func (m *mockDatabasePinger) Ping(ctx context.Context) error {
	return m.err
}

func TestHealthCheckSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &mockDatabasePinger{}

	healthHandler := NewHealthHandler(db)

	router := gin.New()
	router.GET("/health", healthHandler.Check)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	expected := `{"database":"ok","status":"ok"}`

	if rec.Body.String() != expected {
		t.Fatalf("expected body %s, got %s", expected, rec.Body.String())
	}
}

func TestHealthCheckDatabaseUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &mockDatabasePinger{
		err: errors.New("database unavailable"),
	}

	healthHandler := NewHealthHandler(db)

	router := gin.New()
	router.GET("/health", healthHandler.Check)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			rec.Code,
		)
	}

	expected := `{"database":"unavailable","status":"error"}`

	if rec.Body.String() != expected {
		t.Fatalf("expected body %s, got %s", expected, rec.Body.String())
	}
}

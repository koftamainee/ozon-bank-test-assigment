package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLiveness(t *testing.T) {
	h := New(0)
	rec := httptest.NewRecorder()
	h.Liveness().ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestReadinessAllOK(t *testing.T) {
	h := New(0)
	h.Check("db", func(ctx context.Context) error { return nil })
	h.Check("cache", func(ctx context.Context) error { return nil })

	rec := httptest.NewRecorder()
	h.Readiness().ServeHTTP(rec, httptest.NewRequest("GET", "/readyz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Status string                 `json:"status"`
		Checks map[string]CheckResult `json:"checks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Checks["db"].Status != "ok" || body.Checks["cache"].Status != "ok" {
		t.Errorf("checks = %+v, want all ok", body.Checks)
	}
}

func TestReadinessDegraded(t *testing.T) {
	h := New(0)
	h.Check("db", func(ctx context.Context) error { return errors.New("down") })

	rec := httptest.NewRecorder()
	h.Readiness().ServeHTTP(rec, httptest.NewRequest("GET", "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body struct {
		Status string                 `json:"status"`
		Checks map[string]CheckResult `json:"checks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body.Status != "degraded" {
		t.Errorf("status = %q, want degraded", body.Status)
	}
	check := body.Checks["db"]
	if check.Status != "error" {
		t.Errorf("db check = %+v, want error status", check)
	}
}

func TestReadinessTimeout(t *testing.T) {
	h := New(10 * time.Millisecond)
	h.Check("slow", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	rec := httptest.NewRecorder()
	start := time.Now()
	h.Readiness().ServeHTTP(rec, httptest.NewRequest("GET", "/readyz", nil))

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Readiness took %v, want ~10ms timeout", elapsed)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestDefaultTimeout(t *testing.T) {
	h := New(0)
	if h.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", h.timeout)
	}
}

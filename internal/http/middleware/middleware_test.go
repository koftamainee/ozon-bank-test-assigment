package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDGeneratesAndSetsHeader(t *testing.T) {
	var gotID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = RequestIDFromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if gotID == "" {
		t.Error("request id was not generated")
	}
	if rec.Header().Get("X-Request-ID") != gotID {
		t.Errorf("response header id = %q, want %q", rec.Header().Get("X-Request-ID"), gotID)
	}
}

func TestRequestIDUsesIncomingHeader(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := RequestIDFromContext(r.Context()); id != "client-provided" {
			t.Errorf("context id = %q, want client-provided", id)
		}
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "client-provided")
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != "client-provided" {
		t.Errorf("response header id = %q, want client-provided", rec.Header().Get("X-Request-ID"))
	}
}

func TestRequestIDFromContextEmpty(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("RequestIDFromContext() = %q, want empty", got)
	}
}

func TestLoggerFromContextDefault(t *testing.T) {
	if got := LoggerFromContext(context.Background()); got != slog.Default() {
		t.Error("LoggerFromContext() should return default logger")
	}
}

func TestLoggerRecordsStatus(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	defer slog.SetDefault(old)

	handler := RequestID(Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/some/path", nil))

	out := buf.String()
	if !strings.Contains(out, `status=418`) {
		t.Errorf("expected status=418 in log, got %q", out)
	}
	if !strings.Contains(out, `method=GET`) || !strings.Contains(out, `path=/some/path`) {
		t.Errorf("expected method and path in log, got %q", out)
	}
}

func TestRecoveryCatchesPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("kaboom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if body := rec.Body.String(); !strings.Contains(body, "kaboom") {
		t.Errorf("body = %q, want panic message", body)
	}
}

func TestRecoveryPassesThrough(t *testing.T) {
	called := false
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if !called {
		t.Error("handler was not called")
	}
	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
}

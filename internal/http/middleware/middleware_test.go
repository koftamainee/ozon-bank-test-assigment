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

func TestRequestIDSanitizesIncomingHeader(t *testing.T) {
	for _, bad := range []string{"a\nb", "a b", "x\u0000y", "a/b", ">x", strings.Repeat("a", 65)} {
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-ID", bad)
		handler.ServeHTTP(rec, req)

		if id := rec.Header().Get("X-Request-ID"); id == bad {
			t.Errorf("unsanitized request id accepted: %q", bad)
		}
	}

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "a_b.c-1")
	handler.ServeHTTP(rec, req)
	if id := rec.Header().Get("X-Request-ID"); id != "a_b.c-1" {
		t.Errorf("valid request id = %q, want a_b.c-1", id)
	}
}

func TestGeneratedRequestIDIsHex(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	id := rec.Header().Get("X-Request-ID")
	if len(id) != 16 {
		t.Fatalf("generated id %q, want 16 hex chars", id)
	}
	for _, c := range id {
		if !isHexDigit(c) {
			t.Fatalf("generated id %q is not hex", id)
		}
	}
}

func isHexDigit(c rune) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f'
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
	if body := rec.Body.String(); !strings.Contains(body, http.StatusText(http.StatusInternalServerError)) {
		t.Errorf("body = %q, want generic message", body)
	}
	if strings.Contains(rec.Body.String(), "kaboom") {
		t.Errorf("body leaks panic message: %q", rec.Body.String())
	}
}

func TestStatusWriterHijacks(t *testing.T) {
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("statusWriter does not implement http.Hijacker")
		}
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
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

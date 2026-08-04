package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func serve(handler http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestNewRegistersHealthEndpoints(t *testing.T) {
	_, srv, _ := New(Config{Addr: ":0"}, Default())

	if rec := serve(srv.Handler, "GET", "/healthz"); rec.Code != http.StatusOK {
		t.Errorf("/healthz = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec := serve(srv.Handler, "GET", "/readyz"); rec.Code != http.StatusOK {
		t.Errorf("/readyz = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNewWithoutHealth(t *testing.T) {
	_, srv, _ := New(Config{Addr: ":0"}, None())

	if rec := serve(srv.Handler, "GET", "/healthz"); rec.Code != http.StatusNotFound {
		t.Errorf("/healthz = %d, want %d (not registered)", rec.Code, http.StatusNotFound)
	}
}

func TestNewNilOptionsUsesDefault(t *testing.T) {
	_, srv, _ := New(Config{Addr: ":0"}, nil)

	if rec := serve(srv.Handler, "GET", "/healthz"); rec.Code != http.StatusOK {
		t.Errorf("/healthz = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNewAppliesCustomMiddleware(t *testing.T) {
	called := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	_, srv, _ := New(Config{Addr: ":0"}, None().WithMiddleware(mw))
	serve(srv.Handler, "GET", "/")

	if !called {
		t.Error("custom middleware was not applied")
	}
}

func TestNewDefaultTimeouts(t *testing.T) {
	_, srv, _ := New(Config{Addr: ":0"}, None())

	if srv.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout = %v, want 10s", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 10*time.Second {
		t.Errorf("WriteTimeout = %v, want 10s", srv.WriteTimeout)
	}
}

func TestNewCustomTimeouts(t *testing.T) {
	_, srv, _ := New(Config{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 7 * time.Second,
	}, None())

	if srv.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", srv.Addr)
	}
	if srv.ReadTimeout != 5*time.Second {
		t.Errorf("ReadTimeout = %v, want 5s", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 7*time.Second {
		t.Errorf("WriteTimeout = %v, want 7s", srv.WriteTimeout)
	}
}

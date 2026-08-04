package httpserver

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
)

func TestChainAppliesMiddlewaresInOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string

	app := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				order = append(order, name)
				mu.Unlock()
				next.ServeHTTP(w, r)
			})
		}
	}

	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			order = append(order, "handler")
			mu.Unlock()
		}),
		app("a"), app("b"), app("c"),
	)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	want := []string{"a", "b", "c", "handler"}
	mu.Lock()
	defer mu.Unlock()
	if !slices.Equal(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestChainEmpty(t *testing.T) {
	called := false
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	if !called {
		t.Error("handler was not called")
	}
}

package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/json"
)

type CheckFunc func(ctx context.Context) error

type CheckResult struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type Health struct {
	mu      sync.RWMutex
	checks  map[string]CheckFunc
	timeout time.Duration
}

func New(timeout time.Duration) *Health {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &Health{
		checks:  make(map[string]CheckFunc),
		timeout: timeout,
	}
}

func (h *Health) Check(name string, fn CheckFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = fn
}

func (h *Health) Liveness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		data, err := json.Marshal(map[string]string{"status": "ok"})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(data)
	}
}

func (h *Health) Readiness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		checks := make(map[string]CheckFunc, len(h.checks))
		for k, v := range h.checks {
			checks[k] = v
		}
		h.mu.RUnlock()

		ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
		defer cancel()

		results := make(map[string]CheckResult, len(checks))
		allOK := true

		for name, check := range checks {
			if err := check(ctx); err != nil {
				results[name] = CheckResult{Status: "error", Error: err.Error()}
				allOK = false
			} else {
				results[name] = CheckResult{Status: "ok"}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if !allOK {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		data, err := json.Marshal(map[string]any{
			"status": statusStr(allOK),
			"checks": results,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(data)
	}
}

func statusStr(ok bool) string {
	if ok {
		return "ok"
	}
	return "degraded"
}

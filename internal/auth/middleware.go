package auth

import (
	"context"
	"net/http"
)

type contextKey int

const userIDKey contextKey = iota

func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		userID, err := m.Verify(cookie.Value)
		if err != nil {
			http.SetCookie(w, m.ClearCookie())
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}

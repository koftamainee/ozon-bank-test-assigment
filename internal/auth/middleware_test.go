package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func userIDAfterMiddleware(m *Manager, r *http.Request) (int64, bool) {
	rec := httptest.NewRecorder()
	var got int64
	var ok bool
	h := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(rec, r)
	return got, ok
}

func newCookieRequest(value string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: value})
	return r
}

func TestMiddlewareNoCookieIsAnonymous(t *testing.T) {
	m := newTestManager(time.Hour, false)
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	if _, ok := userIDAfterMiddleware(m, r); ok {
		t.Fatal("must be anonymous without cookie")
	}
}

func TestMiddlewareValidCookie(t *testing.T) {
	m := newTestManager(time.Hour, false)
	token, err := m.Sign(5, "alice")
	if err != nil {
		t.Fatal(err)
	}

	userID, ok := userIDAfterMiddleware(m, newCookieRequest(token))
	if !ok {
		t.Fatal("must be authenticated")
	}
	if userID != 5 {
		t.Fatalf("userID = %d, want 5", userID)
	}
}

func TestMiddlewareInvalidCookieIsAnonymous(t *testing.T) {
	m := newTestManager(time.Hour, false)
	token, err := m.Sign(5, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := userIDAfterMiddleware(m, newCookieRequest(token+"x")); ok {
		t.Fatal("tampered cookie must be anonymous")
	}
}

func TestMiddlewareMalformedCookieIsAnonymous(t *testing.T) {
	m := newTestManager(time.Hour, false)

	if _, ok := userIDAfterMiddleware(m, newCookieRequest("garbage")); ok {
		t.Fatal("malformed cookie must be anonymous")
	}
}

func TestMiddlewareExpiredCookieIsAnonymous(t *testing.T) {
	m := newTestManager(-time.Hour, false)
	token, err := m.Sign(5, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := userIDAfterMiddleware(m, newCookieRequest(token)); ok {
		t.Fatal("expired cookie must be anonymous")
	}
}

func TestMiddlewareClearsInvalidCookie(t *testing.T) {
	m := newTestManager(time.Hour, false)
	rec := httptest.NewRecorder()
	h := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(rec, newCookieRequest("garbage"))

	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, CookieName) || !strings.Contains(setCookie, "Max-Age=0") {
		t.Fatalf("Set-Cookie = %q, want clearing cookie on invalid session", setCookie)
	}
}

func TestSessionCookieHasExpires(t *testing.T) {
	m := newTestManager(time.Hour, false)
	c := m.SessionCookie("tok")
	if c.Expires.IsZero() {
		t.Fatal("session cookie must carry Expires")
	}
	if got, want := time.Until(c.Expires), time.Hour; got < want-time.Minute {
		t.Fatalf("Expires = %v from now, want ~%v", got, want)
	}
}

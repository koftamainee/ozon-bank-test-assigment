package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/json"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/service"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store/memory"
)

func newAuthHandlers(t *testing.T) (*Manager, *LoginHandler, *LogoutHandler) {
	t.Helper()
	store := memory.New()
	svc := service.NewAuth(store.Users())
	m := New([]byte(testSecret), time.Hour, false)
	return m, NewLoginHandler(m, svc, nil), NewLogoutHandler(m)
}

func TestLoginHappyPath(t *testing.T) {
	m, handler, _ := newAuthHandlers(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"alice"}`))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var body map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body["ok"] {
		t.Fatalf("body = %v, want ok=true", body)
	}

	cookies := rec.Result().Cookies()
	var session *http.Cookie
	for _, c := range cookies {
		if c.Name == CookieName {
			session = c
		}
	}
	if session == nil {
		t.Fatal("session cookie not set")
	}
	if !session.HttpOnly || session.Path != "/" || session.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie = %+v, want HttpOnly Path=/ SameSite=Lax", session)
	}
	if session.MaxAge == 0 {
		t.Fatal("cookie must have Max-Age")
	}

	userID, err := m.Verify(session.Value)
	if err != nil {
		t.Fatal(err)
	}
	if userID == 0 {
		t.Fatal("cookie must contain a real user")
	}
}

func TestLoginTrimsUsernameAndIsIdempotent(t *testing.T) {
	m, handler, _ := newAuthHandlers(t)
	login := func(body string) int64 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (body %s)", rec.Code, rec.Body.String())
		}
		for _, c := range rec.Result().Cookies() {
			if c.Name == CookieName {
				id, err := m.Verify(c.Value)
				if err != nil {
					t.Fatal(err)
				}
				return id
			}
		}
		t.Fatal("session cookie not set")
		return 0
	}

	first := login(`{"username":"  alice  "}`)
	second := login(`{"username":"alice"}`)
	if first != second {
		t.Fatalf("same username must map to same user: %d vs %d", first, second)
	}
}

func TestLoginInvalidUsername(t *testing.T) {
	_, handler, _ := newAuthHandlers(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":""}`))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "VALIDATION_ERROR") {
		t.Fatalf("body = %s, want VALIDATION_ERROR", rec.Body.String())
	}
}

func TestLoginInvalidJSON(t *testing.T) {
	_, handler, _ := newAuthHandlers(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":`))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestLoginBodyTooLarge(t *testing.T) {
	_, handler, _ := newAuthHandlers(t)
	body := `{"username":"` + strings.Repeat("a", 1500) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestLoginMethodNotAllowed(t *testing.T) {
	_, handler, _ := newAuthHandlers(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("Allow = %q, want POST", allow)
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	_, _, handler := newAuthHandlers(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, CookieName) || !strings.Contains(setCookie, "Max-Age=0") {
		t.Fatalf("Set-Cookie = %q, want clearing cookie", setCookie)
	}
}

func TestLogoutMethodNotAllowed(t *testing.T) {
	_, _, handler := newAuthHandlers(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestSameOrigin(t *testing.T) {
	req := func(headers map[string]string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"alice"}`))
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		return r
	}

	tests := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{"no origin, no sec-fetch-site", nil, true},
		{"same origin", map[string]string{"Origin": "http://example.com"}, true},
		{"origin with different port", map[string]string{"Origin": "http://example.com:8080"}, false},
		{"cross origin", map[string]string{"Origin": "https://evil.example"}, false},
		{"malformed origin", map[string]string{"Origin": "://bad"}, false},
		{"cross-site sec-fetch-site", map[string]string{"Sec-Fetch-Site": "cross-site"}, false},
		{"same-site sec-fetch-site", map[string]string{"Sec-Fetch-Site": "same-site"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameOrigin(req(tt.headers)); got != tt.want {
				t.Errorf("sameOrigin = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoginRejectsCrossOrigin(t *testing.T) {
	_, handler, _ := newAuthHandlers(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"alice"}`))
	req.Header.Set("Origin", "https://evil.example")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/http/response"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/json"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/service"
)

const maxLoginBodyBytes = 1 << 10

func sameOrigin(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return strings.EqualFold(u.Host, r.Host)
	}
	return r.Header.Get("Sec-Fetch-Site") != "cross-site"
}

type loginRequest struct {
	Username string `json:"username"`
}

type LoginHandler struct {
	manager *Manager
	svc     *service.AuthService
	log     *slog.Logger
}

func NewLoginHandler(manager *Manager, svc *service.AuthService, log *slog.Logger) *LoginHandler {
	if log == nil {
		log = slog.Default()
	}
	return &LoginHandler{manager: manager, svc: svc, log: log}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		if err := response.Errorf(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed"); err != nil {
			return
		}
		return
	}
	if !sameOrigin(r) {
		if err := response.Forbidden(w, "cross-site request"); err != nil {
			return
		}
		return
	}

	var req loginRequest
	if err := json.Decode(http.MaxBytesReader(w, r.Body, maxLoginBodyBytes), &req); err != nil {
		if err := response.BadRequest(w, "invalid request body"); err != nil {
			return
		}
		return
	}

	user, err := h.svc.Login(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidUsername) {
			if err := response.ValidationError(w, err); err != nil {
				return
			}
			return
		}
		h.log.Error("login failed", "err", err)
		if err := response.Internal(w); err != nil {
			return
		}
		return
	}

	token, err := h.manager.Sign(user.ID, user.Username)
	if err != nil {
		h.log.Error("sign token", "err", err)
		if err := response.Internal(w); err != nil {
			return
		}
		return
	}

	http.SetCookie(w, h.manager.SessionCookie(token))
	if err := response.Ok(w, map[string]bool{"ok": true}); err != nil {
		return
	}
}

type LogoutHandler struct {
	manager *Manager
}

func NewLogoutHandler(manager *Manager) *LogoutHandler {
	return &LogoutHandler{manager: manager}
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		if err := response.Errorf(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed"); err != nil {
			return
		}
		return
	}
	if !sameOrigin(r) {
		if err := response.Forbidden(w, "cross-site request"); err != nil {
			return
		}
		return
	}

	http.SetCookie(w, h.manager.ClearCookie())
	if err := response.NoContent(w); err != nil {
		return
	}
}

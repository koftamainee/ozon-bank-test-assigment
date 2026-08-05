package auth

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
)

const (
	CookieName = "forum_session"
	issuer     = "forum"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Manager struct {
	secret []byte
	ttl    time.Duration
	secure bool
}

func New(secret []byte, ttl time.Duration, secure bool) *Manager {
	return &Manager{secret: secret, ttl: ttl, secure: secure}
}

type claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (m *Manager) Sign(userID int64, username domain.Username) (string, error) {
	now := time.Now().UTC()
	c := claims{
		Username: username.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(m.secret)
}

func (m *Manager) Verify(tokenStr string) (int64, error) {
	c := &claims{}
	_, err := jwt.ParseWithClaims(tokenStr, c,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, ErrExpiredToken
		}
		return 0, ErrInvalidToken
	}
	if c.Subject == "" {
		return 0, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(c.Subject, 10, 64)
	if err != nil {
		return 0, ErrInvalidToken
	}
	return userID, nil
}

func (m *Manager) SessionCookie(token string) *http.Cookie {
	//nolint:gosec // Secure is configurable (false only in dev; see jwt_cookie_secure)
	return &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m.secure,
		MaxAge:   int(m.ttl.Seconds()),
		Expires:  time.Now().UTC().Add(m.ttl),
	}
}

func (m *Manager) ClearCookie() *http.Cookie {
	//nolint:gosec // Secure is configurable (false only in dev; see jwt_cookie_secure)
	return &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m.secure,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	}
}

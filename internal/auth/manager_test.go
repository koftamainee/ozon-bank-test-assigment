package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
)

const testSecret = "test-secret"

func newTestManager(ttl time.Duration, secure bool) *Manager {
	return New([]byte(testSecret), ttl, secure)
}

func signWith(t *testing.T, m *Manager, c claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	s, err := token.SignedString(m.secret)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSignVerifyRoundTrip(t *testing.T) {
	m := newTestManager(time.Hour, false)

	token, err := m.Sign(42, domain.Username("alice"))
	if err != nil {
		t.Fatal(err)
	}

	userID, err := m.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if userID != 42 {
		t.Fatalf("userID = %d, want 42", userID)
	}
}

func TestSignEmbedsSubjectAndUsername(t *testing.T) {
	m := newTestManager(time.Hour, false)

	token, err := m.Sign(7, domain.Username("bob"))
	if err != nil {
		t.Fatal(err)
	}

	c := &claims{}
	_, err = jwt.ParseWithClaims(token, c,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithoutClaimsValidation())
	if err != nil {
		t.Fatal(err)
	}
	if c.Subject != "7" || c.Username != "bob" {
		t.Fatalf("claims = %+v, want subject 7 username bob", c)
	}
	if c.Issuer != issuer {
		t.Fatalf("issuer = %q, want %q", c.Issuer, issuer)
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	m := newTestManager(-time.Hour, false)

	token, err := m.Sign(1, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Verify(token); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("err = %v, want ErrExpiredToken", err)
	}
}

func TestVerifyWrongSecret(t *testing.T) {
	m := newTestManager(time.Hour, false)
	other := New([]byte("other-secret"), time.Hour, false)

	token, err := m.Sign(1, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := other.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyTamperedToken(t *testing.T) {
	m := newTestManager(time.Hour, false)

	token, err := m.Sign(1, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Verify(token + "x"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyMalformedToken(t *testing.T) {
	m := newTestManager(time.Hour, false)

	if _, err := m.Verify("not.a.token"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyWrongIssuer(t *testing.T) {
	m := newTestManager(time.Hour, false)
	c := claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "1",
		Issuer:    "other",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token := signWith(t, m, c)

	if _, err := m.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyMissingExpiration(t *testing.T) {
	m := newTestManager(time.Hour, false)
	c := claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject: "1",
		Issuer:  issuer,
	}}
	token := signWith(t, m, c)

	if _, err := m.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

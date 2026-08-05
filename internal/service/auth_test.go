package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store/memory"
)

func newAuthService(t *testing.T) (*AuthService, *memory.Store) {
	t.Helper()
	s := memory.New()
	return NewAuth(s.Users()), s
}

func TestLoginCreatesUser(t *testing.T) {
	svc, _ := newAuthService(t)
	u, err := svc.Login(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == 0 || u.Username != "alice" || u.CreatedAt.IsZero() {
		t.Fatalf("got %+v", u)
	}
}

func TestLoginTrimsUsername(t *testing.T) {
	svc, _ := newAuthService(t)
	u, err := svc.Login(context.Background(), "  alice  ")
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "alice" {
		t.Fatalf("username = %q, want alice", u.Username)
	}
	again, err := svc.Login(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != u.ID {
		t.Fatalf("same user must have same id: %d vs %d", u.ID, again.ID)
	}
}

func TestLoginExistingUserIsIdempotent(t *testing.T) {
	svc, _ := newAuthService(t)
	first, err := svc.Login(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Login(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("ids differ: %d vs %d", first.ID, second.ID)
	}
}

func TestLoginInvalidUsername(t *testing.T) {
	svc, _ := newAuthService(t)
	ctx := context.Background()

	if _, err := svc.Login(ctx, ""); !errors.Is(err, domain.ErrInvalidUsername) {
		t.Fatalf("empty: err = %v, want ErrInvalidUsername", err)
	}
	if _, err := svc.Login(ctx, "   "); !errors.Is(err, domain.ErrInvalidUsername) {
		t.Fatalf("whitespace: err = %v, want ErrInvalidUsername", err)
	}
	if _, err := svc.Login(ctx, strings.Repeat("a", domain.MaxUsernameLength+1)); !errors.Is(err, domain.ErrInvalidUsername) {
		t.Fatalf("too long: err = %v, want ErrInvalidUsername", err)
	}
}

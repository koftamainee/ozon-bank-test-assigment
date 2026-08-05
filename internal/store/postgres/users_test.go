package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/pashagolub/pgxmock/v5"
)

func userCols() []string {
	return []string{"id", "username", "created_at"}
}

func TestUsersCreateInsert(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pool.ExpectQuery("INSERT INTO users").
		WithArgs("alice").
		WillReturnRows(pgxmock.NewRows(userCols()).AddRow(int64(1), "alice", now))

	u, err := New(pool).Users().Create(context.Background(), domain.Username("alice"))
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 1 || u.Username != "alice" {
		t.Fatalf("got %+v", u)
	}
	if !u.CreatedAt.Equal(now) {
		t.Fatalf("created_at = %v, want %v", u.CreatedAt, now)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUsersCreateIdempotentOnConflict(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pool.ExpectQuery("INSERT INTO users").
		WithArgs("alice").
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("SELECT id, username, created_at").
		WithArgs("alice").
		WillReturnRows(pgxmock.NewRows(userCols()).AddRow(int64(1), "alice", now))

	u, err := New(pool).Users().Create(context.Background(), domain.Username("alice"))
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 1 {
		t.Fatalf("got id %d, want 1", u.ID)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUsersByUsernameNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	pool.ExpectQuery("FROM users").
		WithArgs("nobody").
		WillReturnError(pgx.ErrNoRows)

	_, err = New(pool).Users().ByUsername(context.Background(), domain.Username("nobody"))
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}

func TestUsersByIDNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	pool.ExpectQuery("FROM users").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = New(pool).Users().ByID(context.Background(), 999)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}

func TestUsersByIDs(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pool.ExpectQuery("id = ANY").
		WithArgs([]int64{1, 2}).
		WillReturnRows(pgxmock.NewRows(userCols()).
			AddRow(int64(1), "alice", now).
			AddRow(int64(2), "bob", now))

	got, err := New(pool).Users().ByIDs(context.Background(), []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Username != "alice" || got[1].Username != "bob" {
		t.Fatalf("got %+v", got)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

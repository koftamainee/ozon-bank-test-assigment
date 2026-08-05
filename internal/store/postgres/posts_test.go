package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
	"github.com/pashagolub/pgxmock/v5"
)

func postCols() []string {
	return []string{"id", "author_id", "title", "body", "comments_allowed", "deleted_at", "created_at", "updated_at"}
}

func TestPostsCreate(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pool.ExpectQuery("INSERT INTO posts").
		WithArgs(int64(1), "title", "body").
		WillReturnRows(pgxmock.NewRows(postCols()).
			AddRow(int64(1), int64(1), "title", "body", true, nil, now, now))

	got, err := New(pool).Posts().Create(context.Background(), domain.Post{AuthorID: 1, Title: "title", Body: "body"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 1 || !got.CommentsAllowed {
		t.Fatalf("got %+v", got)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostsByIDNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	pool.ExpectQuery("FROM posts").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = New(pool).Posts().ByID(context.Background(), 999)
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound", err)
	}
}

func TestPostsListFirstPage(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	ts3 := time.Date(2026, 8, 4, 12, 3, 0, 0, time.UTC)
	ts2 := time.Date(2026, 8, 4, 12, 2, 0, 0, time.UTC)
	ts1 := time.Date(2026, 8, 4, 12, 1, 0, 0, time.UTC)
	pool.ExpectQuery("ORDER BY created_at DESC, id DESC").
		WithArgs(int32(3)).
		WillReturnRows(pgxmock.NewRows(postCols()).
			AddRow(int64(3), int64(1), "t3", "b3", true, nil, ts3, ts3).
			AddRow(int64(2), int64(1), "t2", "b2", true, nil, ts2, ts2).
			AddRow(int64(1), int64(1), "t1", "b1", true, nil, ts1, ts1))

	page, err := New(pool).Posts().List(context.Background(), 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].ID != 3 || page.Items[1].ID != 2 {
		t.Fatalf("items = %+v", page.Items)
	}
	if page.Next == nil {
		t.Fatal("want Next cursor")
	}
	createdAt, id, err := store.DecodePostCursor(page.Next.String())
	if err != nil {
		t.Fatal(err)
	}
	if !createdAt.Equal(ts2) || id != 2 {
		t.Fatalf("cursor = %v/%d, want %v/2", createdAt, id, ts2)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostsListAfterCursor(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	ts2 := time.Date(2026, 8, 4, 12, 2, 0, 0, time.UTC)
	ts1 := time.Date(2026, 8, 4, 12, 1, 0, 0, time.UTC)
	ts0 := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	after := store.NewCursor(store.EncodePostCursor(ts2, 2))
	pool.ExpectQuery("created_at <").
		WithArgs(pgtype.Timestamptz{Time: ts2, Valid: true}, int64(2), int32(3)).
		WillReturnRows(pgxmock.NewRows(postCols()).
			AddRow(int64(1), int64(1), "t1", "b1", true, nil, ts1, ts1).
			AddRow(int64(0), int64(1), "t0", "b0", true, nil, ts0, ts0))

	page, err := New(pool).Posts().List(context.Background(), 2, after)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Next != nil {
		t.Fatalf("items=%+v next=%v", page.Items, page.Next)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostsListInvalidCursor(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(pool).Posts().List(context.Background(), 2, store.NewCursor("not-a-cursor"))
	if err == nil {
		t.Fatal("want error for invalid cursor")
	}
}

func TestPostsSetCommentsAllowed(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pool.ExpectQuery("UPDATE posts").
		WithArgs(int64(1), int64(7), false).
		WillReturnRows(pgxmock.NewRows(postCols()).
			AddRow(int64(1), int64(7), "t", "b", false, nil, now, now))

	got, err := New(pool).Posts().SetCommentsAllowed(context.Background(), 1, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.CommentsAllowed {
		t.Fatal("comments_allowed must be false")
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostsSetCommentsAllowedNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	pool.ExpectQuery("UPDATE posts").
		WithArgs(int64(999), int64(7), true).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("SELECT .* FROM posts WHERE id").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = New(pool).Posts().SetCommentsAllowed(context.Background(), 999, 7, true)
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound", err)
	}
}

func TestPostsSetCommentsAllowedForbidden(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pool.ExpectQuery("UPDATE posts").
		WithArgs(int64(1), int64(7), true).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("SELECT .* FROM posts WHERE id").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows(postCols()).
			AddRow(int64(1), int64(999), "t", "b", true, nil, now, now))

	_, err = New(pool).Posts().SetCommentsAllowed(context.Background(), 1, 7, true)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

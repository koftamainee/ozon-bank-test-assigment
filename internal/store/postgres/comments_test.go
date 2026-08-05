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

func commentCols() []string {
	return []string{"id", "post_id", "author_id", "parent_id", "path", "body", "deleted_at", "created_at"}
}

func TestCommentsCreateRoot(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	cols := commentCols()
	pool.ExpectBegin()
	pool.ExpectQuery("FROM posts").
		WithArgs(int64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"comments_allowed", "deleted_at"}).AddRow(true, nil))
	pool.ExpectQuery("INSERT INTO comments").
		WithArgs(int64(10), int64(5), pgtype.Int8{}, "hello").
		WillReturnRows(pgxmock.NewRows(cols).
			AddRow(int64(8), int64(10), int64(5), nil, "", "hello", nil, now))
	pool.ExpectQuery("UPDATE comments").
		WithArgs("0000000000000000008", int64(8)).
		WillReturnRows(pgxmock.NewRows(cols).
			AddRow(int64(8), int64(10), int64(5), nil, "0000000000000000008", "hello", nil, now))
	pool.ExpectCommit()

	got, err := New(pool).Comments().Create(context.Background(), domain.Comment{PostID: 10, AuthorID: 5, Body: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "0000000000000000008" || got.Depth != 0 || got.ParentID != nil {
		t.Fatalf("got %+v", got)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCommentsCreateChild(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	parentID := int64(7)
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	cols := commentCols()
	pool.ExpectBegin()
	pool.ExpectQuery("FROM posts").
		WithArgs(int64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"comments_allowed", "deleted_at"}).AddRow(true, nil))
	pool.ExpectQuery("FROM comments").
		WithArgs(parentID).
		WillReturnRows(pgxmock.NewRows([]string{"post_id", "path", "deleted_at"}).
			AddRow(int64(10), "0000000000000000001", nil))
	pool.ExpectQuery("INSERT INTO comments").
		WithArgs(int64(10), int64(5), pgtype.Int8{Int64: parentID, Valid: true}, "hello").
		WillReturnRows(pgxmock.NewRows(cols).
			AddRow(int64(8), int64(10), int64(5), parentID, "", "hello", nil, now))
	pool.ExpectQuery("UPDATE comments").
		WithArgs("0000000000000000001.0000000000000000008", int64(8)).
		WillReturnRows(pgxmock.NewRows(cols).
			AddRow(int64(8), int64(10), int64(5), parentID, "0000000000000000001.0000000000000000008", "hello", nil, now))
	pool.ExpectCommit()

	got, err := New(pool).Comments().Create(context.Background(), domain.Comment{
		PostID: 10, AuthorID: 5, ParentID: &parentID, Body: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "0000000000000000001.0000000000000000008" || got.Depth != 1 {
		t.Fatalf("got %+v", got)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCommentsCreatePostNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	pool.ExpectBegin()
	pool.ExpectQuery("FROM posts").
		WithArgs(int64(10)).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectRollback()

	_, err = New(pool).Comments().Create(context.Background(), domain.Comment{PostID: 10, AuthorID: 5, Body: "x"})
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound", err)
	}
}

func TestCommentsCreatePostDeleted(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pool.ExpectBegin()
	pool.ExpectQuery("FROM posts").
		WithArgs(int64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"comments_allowed", "deleted_at"}).AddRow(true, now))
	pool.ExpectRollback()

	_, err = New(pool).Comments().Create(context.Background(), domain.Comment{PostID: 10, AuthorID: 5, Body: "x"})
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound", err)
	}
}

func TestCommentsCreateDisabled(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	pool.ExpectBegin()
	pool.ExpectQuery("FROM posts").
		WithArgs(int64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"comments_allowed", "deleted_at"}).AddRow(false, nil))
	pool.ExpectRollback()

	_, err = New(pool).Comments().Create(context.Background(), domain.Comment{PostID: 10, AuthorID: 5, Body: "x"})
	if !errors.Is(err, domain.ErrCommentsDisabled) {
		t.Fatalf("err = %v, want ErrCommentsDisabled", err)
	}
}

func TestCommentsCreateParentNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	parentID := int64(99)
	pool.ExpectBegin()
	pool.ExpectQuery("FROM posts").
		WithArgs(int64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"comments_allowed", "deleted_at"}).AddRow(true, nil))
	pool.ExpectQuery("FROM comments").
		WithArgs(parentID).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectRollback()

	_, err = New(pool).Comments().Create(context.Background(), domain.Comment{
		PostID: 10, AuthorID: 5, ParentID: &parentID, Body: "x",
	})
	if !errors.Is(err, domain.ErrCommentNotFound) {
		t.Fatalf("err = %v, want ErrCommentNotFound", err)
	}
}

func TestCommentsCreateParentFromOtherPost(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	parentID := int64(7)
	pool.ExpectBegin()
	pool.ExpectQuery("FROM posts").
		WithArgs(int64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"comments_allowed", "deleted_at"}).AddRow(true, nil))
	pool.ExpectQuery("FROM comments").
		WithArgs(parentID).
		WillReturnRows(pgxmock.NewRows([]string{"post_id", "path", "deleted_at"}).
			AddRow(int64(99), "0000000000000000007", nil))
	pool.ExpectRollback()

	_, err = New(pool).Comments().Create(context.Background(), domain.Comment{
		PostID: 10, AuthorID: 5, ParentID: &parentID, Body: "x",
	})
	if !errors.Is(err, domain.ErrParentNotInPost) {
		t.Fatalf("err = %v, want ErrParentNotInPost", err)
	}
}

func TestCommentsCreateParentDeleted(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	parentID := int64(7)
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pool.ExpectBegin()
	pool.ExpectQuery("FROM posts").
		WithArgs(int64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"comments_allowed", "deleted_at"}).AddRow(true, nil))
	pool.ExpectQuery("FROM comments").
		WithArgs(parentID).
		WillReturnRows(pgxmock.NewRows([]string{"post_id", "path", "deleted_at"}).
			AddRow(int64(10), "0000000000000000007", now))
	pool.ExpectRollback()

	_, err = New(pool).Comments().Create(context.Background(), domain.Comment{
		PostID: 10, AuthorID: 5, ParentID: &parentID, Body: "x",
	})
	if !errors.Is(err, domain.ErrParentDeleted) {
		t.Fatalf("err = %v, want ErrParentDeleted", err)
	}
}

func TestCommentsByIDNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	pool.ExpectQuery("FROM comments").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = New(pool).Comments().ByID(context.Background(), 999)
	if !errors.Is(err, domain.ErrCommentNotFound) {
		t.Fatalf("err = %v, want ErrCommentNotFound", err)
	}
}

func TestCommentsListByPostFirstPage(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	cols := commentCols()
	pool.ExpectQuery("ORDER BY path ASC").
		WithArgs(int64(10), int32(3)).
		WillReturnRows(pgxmock.NewRows(cols).
			AddRow(int64(1), int64(10), int64(5), nil, "0000000000000000001", "c1", nil, now).
			AddRow(int64(2), int64(10), int64(5), nil, "0000000000000000002", "c2", nil, now).
			AddRow(int64(3), int64(10), int64(5), nil, "0000000000000000003", "c3", nil, now))

	page, err := New(pool).Comments().ListByPost(context.Background(), 10, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].ID != 1 || page.Items[1].ID != 2 {
		t.Fatalf("items = %+v", page.Items)
	}
	if page.Next == nil {
		t.Fatal("want Next cursor")
	}
	path, err := store.DecodeCommentCursor(page.Next.String())
	if err != nil {
		t.Fatal(err)
	}
	if path != "0000000000000000002" {
		t.Fatalf("cursor path = %q", path)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCommentsListByPostAfterCursor(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	after := store.NewCursor(store.EncodeCommentCursor("0000000000000000002"))
	pool.ExpectQuery("path >").
		WithArgs(int64(10), "0000000000000000002", int32(3)).
		WillReturnRows(pgxmock.NewRows(commentCols()).
			AddRow(int64(3), int64(10), int64(5), nil, "0000000000000000003", "c3", nil, now).
			AddRow(int64(4), int64(10), int64(5), nil, "0000000000000000004", "c4", nil, now))

	page, err := New(pool).Comments().ListByPost(context.Background(), 10, 2, after)
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

func TestCommentsListByPostInvalidCursor(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(pool).Comments().ListByPost(context.Background(), 10, 2, store.NewCursor("!!!"))
	if err == nil {
		t.Fatal("want error for invalid cursor")
	}
}

package store

import (
	"context"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
)

type Cursor struct {
	value string
}

func NewCursor(value string) *Cursor {
	return &Cursor{value: value}
}

func ParseCursor(s string) *Cursor {
	if s == "" {
		return nil
	}
	return &Cursor{value: s}
}

func (c *Cursor) String() string {
	if c == nil {
		return ""
	}
	return c.value
}

type Page[T any] struct {
	Items []T
	Next  *Cursor
}

type UserStore interface {
	Create(ctx context.Context, username domain.Username) (domain.User, error)
	ByUsername(ctx context.Context, username domain.Username) (domain.User, error)
	ByID(ctx context.Context, id int64) (domain.User, error)
	ByIDs(ctx context.Context, ids []int64) ([]domain.User, error)
}

type PostStore interface {
	Create(ctx context.Context, p domain.Post) (domain.Post, error)
	ByID(ctx context.Context, id int64) (domain.Post, error)
	List(ctx context.Context, limit int, after *Cursor) (Page[domain.Post], error)
	SetCommentsAllowed(ctx context.Context, id, authorID int64, allowed bool) (domain.Post, error)
}

type CommentStore interface {
	Create(ctx context.Context, c domain.Comment) (domain.Comment, error)
	ByID(ctx context.Context, id int64) (domain.Comment, error)
	ListByPost(ctx context.Context, postID int64, limit int, after *Cursor) (Page[domain.Comment], error)
}

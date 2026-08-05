package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	storepg "github.com/koftamainee/ozon-bank-test-assigment/internal/store/postgres/gen"
)

func toDomainUser(u storepg.User) domain.User {
	return domain.User{
		ID:        u.ID,
		Username:  domain.Username(u.Username),
		CreatedAt: u.CreatedAt.Time,
	}
}

func toDomainPost(p storepg.Post) domain.Post {
	return domain.Post{
		ID:              p.ID,
		AuthorID:        p.AuthorID,
		Title:           p.Title,
		Body:            p.Body,
		CommentsAllowed: p.CommentsAllowed,
		DeletedAt:       toTimePtr(p.DeletedAt),
		CreatedAt:       p.CreatedAt.Time,
		UpdatedAt:       p.UpdatedAt.Time,
	}
}

func toDomainComment(c storepg.Comment) domain.Comment {
	return domain.Comment{
		ID:        c.ID,
		PostID:    c.PostID,
		AuthorID:  c.AuthorID,
		ParentID:  toIDPtr(c.ParentID),
		Path:      c.Path,
		Body:      c.Body,
		DeletedAt: toTimePtr(c.DeletedAt),
		CreatedAt: c.CreatedAt.Time,
	}
}

func toTimePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func toIDPtr(id pgtype.Int8) *int64 {
	if !id.Valid {
		return nil
	}
	v := id.Int64
	return &v
}

func toPgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/pg"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
	storepg "github.com/koftamainee/ozon-bank-test-assigment/internal/store/postgres/gen"
)

type Comments struct {
	pool Pool
}

func (s *Comments) Create(ctx context.Context, c domain.Comment) (domain.Comment, error) {
	var created domain.Comment
	err := pg.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := storepg.New(tx)

		lock, err := q.LockPostForComments(ctx, c.PostID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrPostNotFound
		}
		if err != nil {
			return err
		}
		if lock.DeletedAt.Valid {
			return domain.ErrPostNotFound
		}
		if !lock.CommentsAllowed {
			return domain.ErrCommentsDisabled
		}

		parentPath := ""
		if c.ParentID != nil {
			parent, err := q.LockCommentForReply(ctx, *c.ParentID)
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrCommentNotFound
			}
			if err != nil {
				return err
			}
			if parent.PostID != c.PostID {
				return domain.ErrParentNotInPost
			}
			if parent.DeletedAt.Valid {
				return domain.ErrParentDeleted
			}
			parentPath = parent.Path
		}

		parentID := pgtype.Int8{}
		if c.ParentID != nil {
			parentID = pgtype.Int8{Int64: *c.ParentID, Valid: true}
		}

		ins, err := q.InsertComment(ctx, storepg.InsertCommentParams{
			PostID:   c.PostID,
			AuthorID: c.AuthorID,
			ParentID: parentID,
			Body:     c.Body,
		})
		if err != nil {
			return err
		}

		path := padID(ins.ID)
		if c.ParentID != nil {
			path = parentPath + "." + padID(ins.ID)
		}

		upd, err := q.UpdateCommentPath(ctx, storepg.UpdateCommentPathParams{
			ID:   ins.ID,
			Path: path,
		})
		if err != nil {
			return err
		}

		created = toDomainComment(upd)
		return nil
	})
	if err != nil {
		return domain.Comment{}, err
	}
	return created, nil
}

func (s *Comments) ByID(ctx context.Context, id int64) (domain.Comment, error) {
	c, err := storepg.New(s.pool).GetCommentByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Comment{}, domain.ErrCommentNotFound
	}
	if err != nil {
		return domain.Comment{}, err
	}
	return toDomainComment(c), nil
}

func (s *Comments) ListByPost(ctx context.Context, postID int64, limit int, after *store.Cursor) (store.Page[domain.Comment], error) {
	if limit < 1 {
		limit = 1
	}

	var (
		rows []storepg.Comment
		err  error
	)
	if after != nil {
		path, decodeErr := store.DecodeCommentCursor(after.String())
		if decodeErr != nil {
			return store.Page[domain.Comment]{}, decodeErr
		}
		rows, err = storepg.New(s.pool).ListCommentsByPostAfter(ctx, storepg.ListCommentsByPostAfterParams{
			PostID: postID,
			Path:   path,
			Limit:  int32(limit + 1),
		})
	} else {
		rows, err = storepg.New(s.pool).ListCommentsByPost(ctx, storepg.ListCommentsByPostParams{
			PostID: postID,
			Limit:  int32(limit + 1),
		})
	}
	if err != nil {
		return store.Page[domain.Comment]{}, err
	}

	items := make([]domain.Comment, 0, len(rows))
	for _, r := range rows {
		items = append(items, toDomainComment(r))
	}

	page := store.Page[domain.Comment]{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := items[limit-1]
		page.Next = store.NewCursor(store.EncodeCommentCursor(last.Path))
	}
	return page, nil
}

func padID(id int64) string {
	return fmt.Sprintf("%019d", id)
}

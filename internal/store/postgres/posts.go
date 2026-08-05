package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
	storepg "github.com/koftamainee/ozon-bank-test-assigment/internal/store/postgres/gen"
)

type Posts struct {
	pool Pool
}

func (s *Posts) Create(ctx context.Context, p domain.Post) (domain.Post, error) {
	post, err := storepg.New(s.pool).InsertPost(ctx, storepg.InsertPostParams{
		AuthorID: p.AuthorID,
		Title:    p.Title,
		Body:     p.Body,
	})
	if err != nil {
		return domain.Post{}, err
	}
	return toDomainPost(post), nil
}

func (s *Posts) ByID(ctx context.Context, id int64) (domain.Post, error) {
	p, err := storepg.New(s.pool).GetPostByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Post{}, domain.ErrPostNotFound
	}
	if err != nil {
		return domain.Post{}, err
	}
	return toDomainPost(p), nil
}

func (s *Posts) List(ctx context.Context, limit int, after *store.Cursor) (store.Page[domain.Post], error) {
	if limit < 1 {
		limit = 1
	}

	var (
		rows []storepg.Post
		err  error
	)
	if after != nil {
		createdAt, id, decodeErr := store.DecodePostCursor(after.String())
		if decodeErr != nil {
			return store.Page[domain.Post]{}, decodeErr
		}
		rows, err = storepg.New(s.pool).ListPostsAfter(ctx, storepg.ListPostsAfterParams{
			CreatedAt: toPgTime(createdAt),
			ID:        id,
			Limit:     int32(limit + 1),
		})
	} else {
		rows, err = storepg.New(s.pool).ListPosts(ctx, int32(limit+1))
	}
	if err != nil {
		return store.Page[domain.Post]{}, err
	}

	items := make([]domain.Post, 0, len(rows))
	for _, r := range rows {
		items = append(items, toDomainPost(r))
	}

	page := store.Page[domain.Post]{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := items[limit-1]
		page.Next = store.NewCursor(store.EncodePostCursor(last.CreatedAt, last.ID))
	}
	return page, nil
}

func (s *Posts) SetCommentsAllowed(ctx context.Context, id, authorID int64, allowed bool) (domain.Post, error) {
	p, err := storepg.New(s.pool).UpdatePostCommentsAllowed(ctx, storepg.UpdatePostCommentsAllowedParams{
		ID:              id,
		AuthorID:        authorID,
		CommentsAllowed: allowed,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		post, getErr := s.ByID(ctx, id)
		if getErr != nil {
			return domain.Post{}, getErr
		}
		if post.AuthorID != authorID {
			return domain.Post{}, domain.ErrForbidden
		}
		return domain.Post{}, domain.ErrPostNotFound
	}
	if err != nil {
		return domain.Post{}, err
	}
	return toDomainPost(p), nil
}

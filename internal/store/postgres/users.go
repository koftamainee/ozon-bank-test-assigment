package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	storepg "github.com/koftamainee/ozon-bank-test-assigment/internal/store/postgres/gen"
)

type Users struct {
	pool Pool
}

func (s *Users) Create(ctx context.Context, username domain.Username) (domain.User, error) {
	q := storepg.New(s.pool)
	u, err := q.InsertUser(ctx, username.String())
	if err == nil {
		return toDomainUser(u), nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		u, err := q.GetUserByUsername(ctx, username.String())
		if err != nil {
			return domain.User{}, err
		}
		return toDomainUser(u), nil
	}
	return domain.User{}, err
}

func (s *Users) ByUsername(ctx context.Context, username domain.Username) (domain.User, error) {
	u, err := storepg.New(s.pool).GetUserByUsername(ctx, username.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(u), nil
}

func (s *Users) ByID(ctx context.Context, id int64) (domain.User, error) {
	u, err := storepg.New(s.pool).GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(u), nil
}

func (s *Users) ByIDs(ctx context.Context, ids []int64) ([]domain.User, error) {
	rows, err := storepg.New(s.pool).GetUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(u))
	}
	return out, nil
}

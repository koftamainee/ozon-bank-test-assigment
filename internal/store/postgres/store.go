package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
	storepg "github.com/koftamainee/ozon-bank-test-assigment/internal/store/postgres/gen"
)

type Pool interface {
	storepg.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Store struct {
	pool Pool
}

func New(pool Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Users() *Users       { return &Users{pool: s.pool} }
func (s *Store) Posts() *Posts       { return &Posts{pool: s.pool} }
func (s *Store) Comments() *Comments { return &Comments{pool: s.pool} }

var (
	_ store.UserStore    = (*Users)(nil)
	_ store.PostStore    = (*Posts)(nil)
	_ store.CommentStore = (*Comments)(nil)
)

package pg

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	cfg.applyDefaults()

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, err
	}

	poolCfg.MaxConns = cfg.MaxOpenConns
	poolCfg.MinConns = cfg.MaxIdleConns
	if poolCfg.MinConns > poolCfg.MaxConns {
		poolCfg.MinConns = poolCfg.MaxConns
	}
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime

	return pgxpool.NewWithConfig(ctx, poolCfg)
}

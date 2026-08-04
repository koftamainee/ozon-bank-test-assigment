package pg

import (
	"context"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

func HealthCheck(pool Pinger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return pool.Ping(ctx)
	}
}

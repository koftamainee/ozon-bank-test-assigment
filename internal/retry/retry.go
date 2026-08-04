package retry

import (
	"context"
	"time"
)

func Get[T any](ctx context.Context, fn func() (T, error), attempts int, delay time.Duration) (T, error) {
	var zero T

	for i := 0; i < attempts; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		if i == attempts-1 {
			return zero, err
		}

		backoff := delay * time.Duration(1<<uint(i))
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(backoff):
		}
	}

	return zero, nil
}

func Do(ctx context.Context, fn func() error, attempts int, delay time.Duration) error {
	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else if i == attempts-1 {
			return err
		}

		backoff := delay * time.Duration(1<<uint(i))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	return nil
}

func MustGet[T any](ctx context.Context, fn func() (T, error), attempts int, delay time.Duration) T {
	result, err := Get(ctx, fn, attempts, delay)
	if err != nil {
		panic(err)
	}
	return result
}

func MustDo(ctx context.Context, fn func() error, attempts int, delay time.Duration) {
	if err := Do(ctx, fn, attempts, delay); err != nil {
		panic(err)
	}
}

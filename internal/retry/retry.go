package retry

import (
	"context"
	"time"
)

const maxBackoffShift = 30

func backoff(delay time.Duration, i int) time.Duration {
	if i > maxBackoffShift {
		i = maxBackoffShift
	}
	return delay * time.Duration(1<<uint(i))
}

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

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(backoff(delay, i)):
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

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff(delay, i)):
		}
	}

	return nil
}

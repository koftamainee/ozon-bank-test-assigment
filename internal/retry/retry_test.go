package retry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

const tiny = time.Millisecond

func TestDoSucceedsOnFirstAttempt(t *testing.T) {
	var calls atomic.Int32

	err := Do(context.Background(), func() error {
		calls.Add(1)
		return nil
	}, 3, tiny)

	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}

func TestDoRetriesUntilSuccess(t *testing.T) {
	var calls atomic.Int32

	err := Do(context.Background(), func() error {
		if calls.Add(1) < 3 {
			return errBoom
		}
		return nil
	}, 5, tiny)

	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestDoExhaustsAttempts(t *testing.T) {
	err := Do(context.Background(), func() error {
		return errBoom
	}, 3, tiny)

	if !errors.Is(err, errBoom) {
		t.Fatalf("Do() error = %v, want %v", err, errBoom)
	}
}

func TestDoContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Do(ctx, func() error {
		return errBoom
	}, 5, time.Second)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do() error = %v, want context.Canceled", err)
	}
}

func TestGetReturnsValue(t *testing.T) {
	got, err := Get(context.Background(), func() (string, error) {
		return "ok", nil
	}, 3, tiny)

	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != "ok" {
		t.Errorf("Get() = %q, want ok", got)
	}
}

func TestGetRetriesThenReturnsValue(t *testing.T) {
	var calls atomic.Int32

	got, err := Get(context.Background(), func() (int, error) {
		n := calls.Add(1)
		if n < 3 {
			return 0, errBoom
		}
		return int(n), nil
	}, 5, tiny)

	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != 3 {
		t.Errorf("Get() = %d, want 3", got)
	}
}

func TestGetReturnsError(t *testing.T) {
	got, err := Get(context.Background(), func() (int, error) {
		return 42, errBoom
	}, 3, tiny)

	if !errors.Is(err, errBoom) {
		t.Fatalf("Get() error = %v, want %v", err, errBoom)
	}
	if got != 0 {
		t.Errorf("Get() = %d, want zero value on error", got)
	}
}

func TestGetContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Get(ctx, func() (int, error) {
		return 0, errBoom
	}, 5, time.Second)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Get() error = %v, want context.Canceled", err)
	}
}

func TestMustGet(t *testing.T) {
	got := MustGet(context.Background(), func() (string, error) {
		return "ok", nil
	}, 3, tiny)

	if got != "ok" {
		t.Errorf("MustGet() = %q, want ok", got)
	}
}

func TestMustGetPanicsOnError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustGet() did not panic")
		}
	}()

	MustGet(context.Background(), func() (int, error) {
		return 0, errBoom
	}, 3, tiny)
}

func TestMustDo(t *testing.T) {
	MustDo(context.Background(), func() error {
		return nil
	}, 3, tiny)
}

func TestMustDoPanicsOnError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustDo() did not panic")
		}
	}()

	MustDo(context.Background(), func() error {
		return errBoom
	}, 3, tiny)
}

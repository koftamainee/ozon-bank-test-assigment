package shutdown

import (
	"context"
	"os"
	"os/signal"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestRunStopsServicesInReverseOrder(t *testing.T) {
	m := NewManager()

	var mu sync.Mutex
	var order []string

	for _, name := range []string{"first", "second", "third"} {
		m.Register(name, func(ctx context.Context) error {
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			return nil
		})
	}

	done := make(chan struct{})
	go func() {
		m.run(context.Background(), 0, quitChan())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not complete")
	}

	want := []string{"third", "second", "first"}
	mu.Lock()
	defer mu.Unlock()
	if !slices.Equal(order, want) {
		t.Errorf("shutdown order = %v, want %v", order, want)
	}
}

func TestRunWithTimeout(t *testing.T) {
	m := NewManager()

	start := time.Now()
	m.Register("blocking", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	done := make(chan struct{})
	go func() {
		m.run(context.Background(), 50*time.Millisecond, quitChan())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not complete")
	}

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("run() took %v, want ~50ms", elapsed)
	}
}

func TestRunWithoutTimeoutBlocksIndefinitely(t *testing.T) {
	m := NewManager()
	released := make(chan struct{})

	m.Register("release", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-released:
			return nil
		}
	})

	done := make(chan struct{})
	go func() {
		m.run(context.Background(), 0, quitChan())
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("run() returned while service is still blocking")
	case <-time.After(100 * time.Millisecond):
	}

	close(released)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not complete after service released")
	}
}

func TestRunWithSignals(t *testing.T) {
	m := NewManager()
	var stopped atomic.Bool

	m.Register("svc", func(ctx context.Context) error {
		stopped.Store(true)
		return nil
	})

	watch := make(chan os.Signal, 1)
	signal.Notify(watch, syscall.SIGTERM)
	defer signal.Reset(syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		m.Run(context.Background())
		close(done)
	}()

	for {
		select {
		case <-done:
			if !stopped.Load() {
				t.Error("registered service was not stopped")
			}
			return
		case <-time.After(20 * time.Millisecond):
			_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		}
	}
}

func TestRegisterNilSafe(t *testing.T) {
	m := NewManager()
	m.Register("nil", nil)
	m.Register("empty-name", func(ctx context.Context) error { return nil })

	if len(m.services) != 2 {
		t.Errorf("services = %d, want 2", len(m.services))
	}
}

func quitChan() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	ch <- syscall.SIGTERM
	return ch
}

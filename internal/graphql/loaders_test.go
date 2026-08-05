package graphql

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
)

func TestUserLoaderLoadsUser(t *testing.T) {
	fn := func(_ context.Context, ids []int64) ([]domain.User, error) {
		return []domain.User{{ID: ids[0], Username: "u"}}, nil
	}
	l := newUserLoader(fn)

	u, err := l.Load(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 1 {
		t.Fatalf("got %+v", u)
	}
}

func TestUserLoaderBatchesIDs(t *testing.T) {
	old := batchWindow
	batchWindow = 50 * time.Millisecond
	defer func() { batchWindow = old }()

	var mu sync.Mutex
	var calls [][]int64
	fn := func(_ context.Context, ids []int64) ([]domain.User, error) {
		mu.Lock()
		calls = append(calls, append([]int64(nil), ids...))
		mu.Unlock()
		users := make([]domain.User, 0, len(ids))
		for _, id := range ids {
			users = append(users, domain.User{ID: id, Username: "u"})
		}
		return users, nil
	}
	l := newUserLoader(fn)

	var wg sync.WaitGroup
	for _, id := range []int64{1, 2} {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			if _, err := l.Load(context.Background(), id); err != nil {
				t.Error(err)
			}
		}(id)
	}
	time.Sleep(10 * time.Millisecond)
	l.flush(context.Background())
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 1 || len(calls[0]) != 2 {
		t.Fatalf("calls = %v, want one batch of 2 ids", calls)
	}
}

func TestUserLoaderDedup(t *testing.T) {
	old := batchWindow
	batchWindow = 50 * time.Millisecond
	defer func() { batchWindow = old }()

	var calls int
	fn := func(_ context.Context, ids []int64) ([]domain.User, error) {
		calls++
		users := make([]domain.User, 0, len(ids))
		for _, id := range ids {
			users = append(users, domain.User{ID: id, Username: "u"})
		}
		return users, nil
	}
	l := newUserLoader(fn)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := l.Load(context.Background(), 1); err != nil {
				t.Error(err)
			}
		}()
	}
	time.Sleep(10 * time.Millisecond)
	l.flush(context.Background())
	wg.Wait()

	if calls != 1 {
		t.Fatalf("fn calls = %d, want 1", calls)
	}
}

func TestUserLoaderNotFound(t *testing.T) {
	fn := func(_ context.Context, _ []int64) ([]domain.User, error) {
		return nil, nil
	}
	l := newUserLoader(fn)

	_, err := l.Load(context.Background(), 42)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}

func TestUserLoaderPropagatesError(t *testing.T) {
	want := domain.ErrForbidden
	fn := func(_ context.Context, _ []int64) ([]domain.User, error) {
		return nil, want
	}
	l := newUserLoader(fn)

	if _, err := l.Load(context.Background(), 1); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

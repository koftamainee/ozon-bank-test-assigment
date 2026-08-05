package graphql

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
)

var batchWindow = time.Millisecond

type userLoader struct {
	mu         sync.Mutex
	fn         func(ctx context.Context, ids []int64) ([]domain.User, error)
	pending    map[int64]*userCall
	timer      *time.Timer
	closed     bool
	inFlight   bool
	flightDone chan struct{}
}

type userCall struct {
	done chan struct{}
	user domain.User
	err  error
}

func newUserLoader(fn func(ctx context.Context, ids []int64) ([]domain.User, error)) *userLoader {
	return &userLoader{
		fn:      fn,
		pending: make(map[int64]*userCall),
	}
}

func (l *userLoader) Load(ctx context.Context, id int64) (domain.User, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return domain.User{}, errors.New("user loader is closed")
	}
	call, ok := l.pending[id]
	if !ok {
		call = &userCall{done: make(chan struct{})}
		l.pending[id] = call
	}
	if l.timer == nil && !l.inFlight {
		l.timer = time.AfterFunc(batchWindow, func() { l.flush(context.Background()) })
	}
	l.mu.Unlock()

	select {
	case <-call.done:
		return call.user, call.err
	case <-ctx.Done():
		return domain.User{}, ctx.Err()
	}
}

func (l *userLoader) flush(ctx context.Context) {
	l.mu.Lock()
	if l.inFlight || len(l.pending) == 0 {
		l.timer = nil
		l.mu.Unlock()
		return
	}
	ids := make([]int64, 0, len(l.pending))
	calls := make([]*userCall, 0, len(l.pending))
	for id, call := range l.pending {
		ids = append(ids, id)
		calls = append(calls, call)
	}
	l.pending = make(map[int64]*userCall)
	l.timer = nil
	l.inFlight = true
	flightDone := make(chan struct{})
	l.flightDone = flightDone
	l.mu.Unlock()

	users, err := l.fn(ctx, ids)
	byID := make(map[int64]domain.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	for i, id := range ids {
		user, found := byID[id]
		var callErr error
		if err != nil {
			callErr = err
		} else if !found {
			callErr = domain.ErrUserNotFound
		}
		calls[i].user = user
		calls[i].err = callErr
		close(calls[i].done)
	}

	l.mu.Lock()
	l.inFlight = false
	l.flightDone = nil
	close(flightDone)
	if len(l.pending) > 0 && !l.closed {
		l.timer = time.AfterFunc(batchWindow, func() { l.flush(context.Background()) })
	}
	l.mu.Unlock()
}

func (l *userLoader) Shutdown() {
	l.mu.Lock()
	if l.timer != nil {
		l.timer.Stop()
		l.timer = nil
	}
	l.closed = true
	flightDone := l.flightDone
	l.mu.Unlock()

	if flightDone != nil {
		<-flightDone
	}
	l.flush(context.Background())
}

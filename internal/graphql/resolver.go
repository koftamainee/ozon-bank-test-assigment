package graphql

import (
	"context"
	"sync"

	"github.com/99designs/gqlgen/graphql"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/notifier"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/service"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
)

type Resolver struct {
	forum    *service.ForumService
	notifier *notifier.Broadcaster
	users    *userLoader

	subMu  sync.Mutex
	subSeq int64
	subs   map[int64]context.CancelFunc
}

func NewResolver(forum *service.ForumService, b *notifier.Broadcaster) *Resolver {
	return &Resolver{
		forum:    forum,
		notifier: b,
		users: newUserLoader(func(ctx context.Context, ids []int64) ([]domain.User, error) {
			return forum.UsersByIDs(ctx, ids)
		}),
	}
}

func (r *Resolver) trackSub(ctx context.Context) (context.Context, context.CancelFunc) {
	subCtx, cancel := context.WithCancel(ctx)

	r.subMu.Lock()
	if r.subs == nil {
		r.subs = make(map[int64]context.CancelFunc)
	}
	r.subSeq++
	id := r.subSeq
	r.subs[id] = cancel
	r.subMu.Unlock()

	go func() {
		<-subCtx.Done()
		r.subMu.Lock()
		delete(r.subs, id)
		r.subMu.Unlock()
	}()

	return subCtx, cancel
}

func (r *Resolver) Shutdown(_ context.Context) error {
	r.subMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(r.subs))
	for _, c := range r.subs {
		cancels = append(cancels, c)
	}
	r.subMu.Unlock()

	for _, c := range cancels {
		c()
	}

	r.users.Shutdown()
	return nil
}

func NewSchema(resolver *Resolver) graphql.ExecutableSchema {
	return NewExecutableSchema(Config{Resolvers: resolver})
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func nextCursor(next *store.Cursor) *string {
	if next == nil {
		return nil
	}
	s := next.String()
	return &s
}

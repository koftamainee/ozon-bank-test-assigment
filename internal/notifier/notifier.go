package notifier

import (
	"log/slog"
	"sync"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
)

const bufferSize = 32

type Broadcaster struct {
	mu   sync.Mutex
	seq  int64
	subs map[int64]map[int64]chan domain.Comment
}

func New() *Broadcaster {
	return &Broadcaster{subs: make(map[int64]map[int64]chan domain.Comment)}
}

func (b *Broadcaster) Subscribe(postID int64) (<-chan domain.Comment, func()) {
	ch := make(chan domain.Comment, bufferSize)

	b.mu.Lock()
	b.seq++
	id := b.seq
	if b.subs[postID] == nil {
		b.subs[postID] = make(map[int64]chan domain.Comment)
	}
	b.subs[postID][id] = ch
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs[postID], id)
			if len(b.subs[postID]) == 0 {
				delete(b.subs, postID)
			}
			close(ch)
			b.mu.Unlock()
		})
	}

	return ch, unsubscribe
}

func (b *Broadcaster) Publish(postID int64, c domain.Comment) {
	b.mu.Lock()
	for id, ch := range b.subs[postID] {
		select {
		case ch <- c:
		default:
			slog.Debug("dropping comment event for slow subscriber", "subscriber", id, "post_id", postID, "comment_id", c.ID)
		}
	}
	b.mu.Unlock()
}

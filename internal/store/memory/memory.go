package memory

import (
	"fmt"
	"sync"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
)

type core struct {
	mu sync.RWMutex

	usersByID       map[int64]domain.User
	usersByUsername map[domain.Username]domain.User
	nextUserID      int64

	posts      []domain.Post
	postsByID  map[int64]domain.Post
	nextPostID int64

	comments      map[int64][]domain.Comment
	commentsByID  map[int64]domain.Comment
	nextCommentID int64
}

func newCore() *core {
	return &core{
		usersByID:       make(map[int64]domain.User),
		usersByUsername: make(map[domain.Username]domain.User),
		postsByID:       make(map[int64]domain.Post),
		comments:        make(map[int64][]domain.Comment),
		commentsByID:    make(map[int64]domain.Comment),
	}
}

type Store struct {
	*core
}

func New() *Store {
	return &Store{core: newCore()}
}

func (s *Store) Users() *Users       { return &Users{core: s.core} }
func (s *Store) Posts() *Posts       { return &Posts{core: s.core} }
func (s *Store) Comments() *Comments { return &Comments{core: s.core} }

func padID(id int64) string {
	return fmt.Sprintf("%019d", id)
}

func paginate[T any](items []T, limit int, cursorOf func(T) *store.Cursor) store.Page[T] {
	if limit < 1 {
		limit = 1
	}

	page := store.Page[T]{Items: items}
	if len(items) > limit {
		page.Next = cursorOf(items[limit-1])
		page.Items = items[:limit]
	}
	return page
}

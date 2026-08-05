package memory

import (
	"context"
	"sort"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
)

type Posts struct {
	*core
}

func (s *Posts) Create(ctx context.Context, p domain.Post) (domain.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.nextPostID++
	p.ID = s.nextPostID
	p.CreatedAt = now
	p.UpdatedAt = now
	p.CommentsAllowed = true
	p.DeletedAt = nil

	s.postsByID[p.ID] = p
	s.posts = insertSortedPosts(s.posts, p)
	return p, nil
}

func (s *Posts) ByID(ctx context.Context, id int64) (domain.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.postsByID[id]
	if !ok {
		return domain.Post{}, domain.ErrPostNotFound
	}
	return p, nil
}

func (s *Posts) List(ctx context.Context, limit int, after *store.Cursor) (store.Page[domain.Post], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]domain.Post, 0, len(s.posts))
	for _, p := range s.posts {
		if !p.IsDeleted() {
			items = append(items, p)
		}
	}

	if after != nil {
		createdAt, id, err := store.DecodePostCursor(after.String())
		if err != nil {
			return store.Page[domain.Post]{}, err
		}
		idx := sort.Search(len(items), func(i int) bool {
			return strictlyOlder(items[i], createdAt, id)
		})
		items = items[idx:]
	}

	return paginate(items, limit, func(p domain.Post) *store.Cursor {
		return store.NewCursor(store.EncodePostCursor(p.CreatedAt, p.ID))
	}), nil
}

func (s *Posts) SetCommentsAllowed(ctx context.Context, id, authorID int64, allowed bool) (domain.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.postsByID[id]
	if !ok || p.IsDeleted() {
		return domain.Post{}, domain.ErrPostNotFound
	}
	if p.AuthorID != authorID {
		return domain.Post{}, domain.ErrForbidden
	}

	p.CommentsAllowed = allowed
	p.UpdatedAt = time.Now().UTC()
	s.postsByID[id] = p

	for i := range s.posts {
		if s.posts[i].ID == id {
			s.posts[i] = p
			break
		}
	}

	return p, nil
}

func newer(a, b domain.Post) bool {
	if a.CreatedAt.Equal(b.CreatedAt) {
		return a.ID > b.ID
	}
	return a.CreatedAt.After(b.CreatedAt)
}

func strictlyOlder(p domain.Post, createdAt time.Time, id int64) bool {
	if p.CreatedAt.Equal(createdAt) {
		return p.ID < id
	}
	return p.CreatedAt.Before(createdAt)
}

func insertSortedPosts(posts []domain.Post, p domain.Post) []domain.Post {
	i := sort.Search(len(posts), func(i int) bool {
		return newer(p, posts[i])
	})
	posts = append(posts, domain.Post{})
	copy(posts[i+1:], posts[i:])
	posts[i] = p
	return posts
}

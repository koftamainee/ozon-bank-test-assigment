package memory

import (
	"context"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
)

type Users struct {
	*core
}

func (s *Users) Create(ctx context.Context, username domain.Username) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u, ok := s.usersByUsername[username]; ok {
		return u, nil
	}

	s.nextUserID++
	u := domain.User{
		ID:        s.nextUserID,
		Username:  username,
		CreatedAt: time.Now().UTC(),
	}
	s.usersByID[u.ID] = u
	s.usersByUsername[username] = u
	return u, nil
}

func (s *Users) ByUsername(ctx context.Context, username domain.Username) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.usersByUsername[username]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (s *Users) ByID(ctx context.Context, id int64) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.usersByID[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (s *Users) ByIDs(ctx context.Context, ids []int64) ([]domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.User, 0, len(ids))
	for _, id := range ids {
		if u, ok := s.usersByID[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}

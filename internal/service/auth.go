package service

import (
	"context"
	"errors"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
)

type AuthService struct {
	users store.UserStore
}

func NewAuth(users store.UserStore) *AuthService {
	return &AuthService{users: users}
}

func (s *AuthService) Login(ctx context.Context, username string) (domain.User, error) {
	u, err := domain.NewUsername(username)
	if err != nil {
		return domain.User{}, err
	}

	user, err := s.users.ByUsername(ctx, u)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, err
	}

	return s.users.Create(ctx, u)
}

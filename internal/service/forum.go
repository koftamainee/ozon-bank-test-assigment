package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
)

const (
	defaultFirst = 20
	maxFirst     = 100
)

type Notifier interface {
	Publish(postID int64, c domain.Comment)
}

type ForumService struct {
	users    store.UserStore
	posts    store.PostStore
	comments store.CommentStore
	notifier Notifier
}

func NewForum(users store.UserStore, posts store.PostStore, comments store.CommentStore, n Notifier) *ForumService {
	return &ForumService{users: users, posts: posts, comments: comments, notifier: n}
}

func (s *ForumService) CreatePost(ctx context.Context, authorID int64, in domain.CreatePostInput) (domain.Post, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" || utf8.RuneCountInString(title) > domain.MaxPostTitleLength {
		return domain.Post{}, domain.ErrInvalidPostTitle
	}
	body := strings.TrimSpace(in.Body)
	if body == "" || utf8.RuneCountInString(body) > domain.MaxPostBodyLength {
		return domain.Post{}, domain.ErrInvalidPostBody
	}

	return s.posts.Create(ctx, domain.Post{AuthorID: authorID, Title: title, Body: body})
}

func (s *ForumService) GetPost(ctx context.Context, id int64) (domain.Post, error) {
	if id <= 0 {
		return domain.Post{}, domain.ErrInvalidID
	}
	p, err := s.posts.ByID(ctx, id)
	if err != nil {
		return domain.Post{}, err
	}
	if p.IsDeleted() {
		return domain.Post{}, domain.ErrPostNotFound
	}
	return p, nil
}

func (s *ForumService) ListPosts(ctx context.Context, first int, after *store.Cursor) (store.Page[domain.Post], error) {
	return s.posts.List(ctx, clampFirst(first), after)
}

func (s *ForumService) SetCommentsAllowed(ctx context.Context, postID, userID int64, allowed bool) (domain.Post, error) {
	if postID <= 0 {
		return domain.Post{}, domain.ErrInvalidID
	}
	return s.posts.SetCommentsAllowed(ctx, postID, userID, allowed)
}

func (s *ForumService) CreateComment(ctx context.Context, authorID int64, in domain.CreateCommentInput) (domain.Comment, error) {
	body, err := domain.NewCommentBody(in.Body.String())
	if err != nil {
		return domain.Comment{}, err
	}
	if in.PostID <= 0 {
		return domain.Comment{}, domain.ErrInvalidID
	}
	if in.ParentID != nil {
		if *in.ParentID <= 0 {
			return domain.Comment{}, domain.ErrInvalidID
		}
		if _, err := s.comments.ByID(ctx, *in.ParentID); err != nil {
			return domain.Comment{}, err
		}
	}

	c, err := s.comments.Create(ctx, domain.Comment{
		PostID:   in.PostID,
		AuthorID: authorID,
		ParentID: in.ParentID,
		Body:     body.String(),
	})
	if err != nil {
		return domain.Comment{}, err
	}

	s.notifier.Publish(in.PostID, c)
	return c, nil
}

func (s *ForumService) ListComments(ctx context.Context, postID int64, first int, after *store.Cursor) (store.Page[domain.Comment], error) {
	if _, err := s.GetPost(ctx, postID); err != nil {
		return store.Page[domain.Comment]{}, err
	}
	return s.comments.ListByPost(ctx, postID, clampFirst(first), after)
}

func (s *ForumService) UsersByIDs(ctx context.Context, ids []int64) ([]domain.User, error) {
	return s.users.ByIDs(ctx, ids)
}

func clampFirst(first int) int {
	if first <= 0 {
		return defaultFirst
	}
	if first > maxFirst {
		return maxFirst
	}
	return first
}

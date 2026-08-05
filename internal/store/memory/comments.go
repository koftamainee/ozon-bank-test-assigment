package memory

import (
	"context"
	"sort"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
)

type Comments struct {
	*core
}

func (s *Comments) Create(ctx context.Context, c domain.Comment) (domain.Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	post, ok := s.postsByID[c.PostID]
	if !ok || post.IsDeleted() {
		return domain.Comment{}, domain.ErrPostNotFound
	}
	if !post.CommentsAllowed {
		return domain.Comment{}, domain.ErrCommentsDisabled
	}

	var parentPath string
	if c.ParentID != nil {
		parent, ok := s.commentsByID[*c.ParentID]
		if !ok {
			return domain.Comment{}, domain.ErrCommentNotFound
		}
		if parent.PostID != c.PostID {
			return domain.Comment{}, domain.ErrParentNotInPost
		}
		if parent.IsDeleted() {
			return domain.Comment{}, domain.ErrParentDeleted
		}
		parentPath = parent.Path
	}

	s.nextCommentID++
	c.ID = s.nextCommentID
	c.CreatedAt = time.Now().UTC()
	c.DeletedAt = nil
	if c.ParentID != nil {
		c.Path = parentPath + "." + padID(c.ID)
	} else {
		c.Path = padID(c.ID)
	}

	c = cloneComment(c)
	s.commentsByID[c.ID] = c
	s.comments[c.PostID] = insertSortedComments(s.comments[c.PostID], c)
	return cloneComment(c), nil
}

func (s *Comments) ByID(ctx context.Context, id int64) (domain.Comment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.commentsByID[id]
	if !ok {
		return domain.Comment{}, domain.ErrCommentNotFound
	}
	return cloneComment(c), nil
}

func (s *Comments) ListByPost(ctx context.Context, postID int64, limit int, after *store.Cursor) (store.Page[domain.Comment], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]domain.Comment, 0, len(s.comments[postID]))
	for _, c := range s.comments[postID] {
		if !c.IsDeleted() {
			items = append(items, cloneComment(c))
		}
	}

	if after != nil {
		path, err := store.DecodeCommentCursor(after.String())
		if err != nil {
			return store.Page[domain.Comment]{}, err
		}
		idx := sort.Search(len(items), func(i int) bool {
			return items[i].Path > path
		})
		items = items[idx:]
	}

	return paginate(items, limit, func(c domain.Comment) *store.Cursor {
		return store.NewCursor(store.EncodeCommentCursor(c.Path))
	}), nil
}

func insertSortedComments(items []domain.Comment, c domain.Comment) []domain.Comment {
	i := sort.Search(len(items), func(i int) bool {
		return items[i].Path > c.Path
	})
	items = append(items, domain.Comment{})
	copy(items[i+1:], items[i:])
	items[i] = c
	return items
}

func cloneComment(c domain.Comment) domain.Comment {
	if c.ParentID != nil {
		pid := *c.ParentID
		c.ParentID = &pid
	}
	return c
}

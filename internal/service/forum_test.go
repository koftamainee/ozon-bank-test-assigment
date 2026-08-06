package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store/memory"
)

type notifyEvent struct {
	postID  int64
	comment domain.Comment
}

type fakeNotifier struct {
	mu     sync.Mutex
	events []notifyEvent
}

func (f *fakeNotifier) Publish(postID int64, c domain.Comment) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, notifyEvent{postID: postID, comment: c})
}

func (f *fakeNotifier) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

func newServices(t *testing.T) (auth *AuthService, forum *ForumService, s *memory.Store, n *fakeNotifier) {
	t.Helper()
	s = memory.New()
	n = &fakeNotifier{}
	auth = NewAuth(s.Users())
	forum = NewForum(s.Users(), s.Posts(), s.Comments(), n)
	return
}

func mustLogin(t *testing.T, auth *AuthService, username string) domain.User {
	t.Helper()
	u, err := auth.Login(context.Background(), username)
	if err != nil {
		t.Fatalf("login %q: %v", username, err)
	}
	return u
}

func mustCreatePost(t *testing.T, forum *ForumService, authorID int64, title, body string) domain.Post {
	t.Helper()
	p, err := forum.CreatePost(context.Background(), authorID, domain.CreatePostInput{Title: title, Body: body})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	return p
}

func mustCreateComment(t *testing.T, forum *ForumService, authorID int64, postID int64, parentID *int64, body string) domain.Comment {
	t.Helper()
	c, err := forum.CreateComment(context.Background(), authorID, domain.CreateCommentInput{
		PostID: postID, ParentID: parentID, Body: domain.CommentBody(body),
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	return c
}

func int64Ptr(v int64) *int64 { return &v }

func TestCreatePostHappyPath(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")

	p := mustCreatePost(t, forum, alice.ID, "  Title  ", "  Body  ")
	if p.Title != "Title" || p.Body != "Body" {
		t.Fatalf("title/body not trimmed: %q %q", p.Title, p.Body)
	}
	if p.AuthorID != alice.ID || p.ID == 0 {
		t.Fatalf("got %+v", p)
	}
	if !p.CommentsAllowed {
		t.Fatal("comments_allowed must default to true")
	}
}

func TestCreatePostValidation(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	ctx := context.Background()

	tests := []struct {
		name  string
		title string
		body  string
		want  error
	}{
		{"empty title", "", "body", domain.ErrInvalidPostTitle},
		{"whitespace title", "   ", "body", domain.ErrInvalidPostTitle},
		{"title too long", strings.Repeat("t", domain.MaxPostTitleLength+1), "body", domain.ErrInvalidPostTitle},
		{"empty body", "title", "", domain.ErrInvalidPostBody},
		{"whitespace body", "title", "   ", domain.ErrInvalidPostBody},
		{"body too long", "title", strings.Repeat("b", domain.MaxPostBodyLength+1), domain.ErrInvalidPostBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := forum.CreatePost(ctx, alice.ID, domain.CreatePostInput{Title: tt.title, Body: tt.body})
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCreatePostMaxLengthsAreAllowed(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")

	p := mustCreatePost(t, forum, alice.ID,
		strings.Repeat("t", domain.MaxPostTitleLength),
		strings.Repeat("b", domain.MaxPostBodyLength))
	if len(p.Title) != domain.MaxPostTitleLength || len(p.Body) != domain.MaxPostBodyLength {
		t.Fatalf("max length post rejected: %d/%d", len(p.Title), len(p.Body))
	}
}

func TestGetPost(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")

	got, err := forum.GetPost(context.Background(), post.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != post.ID || got.AuthorID != alice.ID {
		t.Fatalf("got %+v", got)
	}
}

func TestGetPostNotFound(t *testing.T) {
	_, forum, _, _ := newServices(t)
	_, err := forum.GetPost(context.Background(), 999)
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound", err)
	}
}

func TestGetPostNegativeIDIsInvalid(t *testing.T) {
	_, forum, _, _ := newServices(t)
	for _, id := range []int64{-1, 0} {
		_, err := forum.GetPost(context.Background(), id)
		if !errors.Is(err, domain.ErrInvalidID) {
			t.Fatalf("GetPost(%d) err = %v, want ErrInvalidID", id, err)
		}
	}
}

func TestNegativeIDsAreInvalidEverywhere(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")

	if _, err := forum.SetCommentsAllowed(context.Background(), -1, alice.ID, false); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("SetCommentsAllowed(-1) err = %v, want ErrInvalidID", err)
	}
	if _, err := forum.CreateComment(context.Background(), alice.ID, domain.CreateCommentInput{PostID: -1, Body: "x"}); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("CreateComment(postID=-1) err = %v, want ErrInvalidID", err)
	}
	if _, err := forum.CreateComment(context.Background(), alice.ID, domain.CreateCommentInput{PostID: 1, ParentID: int64Ptr(-2), Body: "x"}); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("CreateComment(parentID=-2) err = %v, want ErrInvalidID", err)
	}
	if _, err := forum.ListComments(context.Background(), -1, 20, nil); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("ListComments(-1) err = %v, want ErrInvalidID", err)
	}
}

func TestGetPostDeleted(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")

	forum.posts = deletedPostStore{PostStore: forum.posts, deleted: postWithDeletedAt(post)}
	_, err := forum.GetPost(context.Background(), post.ID)
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound", err)
	}
}

func TestListPostsPagination(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	p1 := mustCreatePost(t, forum, alice.ID, "t1", "b1")
	p2 := mustCreatePost(t, forum, alice.ID, "t2", "b2")
	p3 := mustCreatePost(t, forum, alice.ID, "t3", "b3")
	ctx := context.Background()

	page1, err := forum.ListPosts(ctx, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 2 || page1.Items[0].ID != p3.ID || page1.Items[1].ID != p2.ID {
		t.Fatalf("page1 = %+v, want [p3 p2]", page1.Items)
	}
	if page1.Next == nil {
		t.Fatal("page1 must have Next")
	}

	page2, err := forum.ListPosts(ctx, 2, page1.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 1 || page2.Items[0].ID != p1.ID || page2.Next != nil {
		t.Fatalf("page2 = %+v, want [p1] no Next", page2.Items)
	}
}

func TestListPostsClampsFirst(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	for i := 0; i < 5; i++ {
		mustCreatePost(t, forum, alice.ID, "t", "b")
	}
	ctx := context.Background()

	for _, first := range []int{0, -1} {
		page, err := forum.ListPosts(ctx, first, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 5 {
			t.Fatalf("first=%d must clamp to default 20, got %d items", first, len(page.Items))
		}
	}

	page, err := forum.ListPosts(ctx, 1000, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 5 {
		t.Fatalf("first=1000 must clamp to 100, got %d items", len(page.Items))
	}
}

func TestSetCommentsAllowedByAuthor(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")

	got, err := forum.SetCommentsAllowed(context.Background(), post.ID, alice.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.CommentsAllowed {
		t.Fatal("comments_allowed must be false")
	}

	got, err = forum.SetCommentsAllowed(context.Background(), post.ID, alice.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CommentsAllowed {
		t.Fatal("comments_allowed must be true")
	}
}

func TestSetCommentsAllowedByNonAuthor(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	bob := mustLogin(t, auth, "bob")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")

	_, err := forum.SetCommentsAllowed(context.Background(), post.ID, bob.ID, false)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestSetCommentsAllowedNotFound(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")

	_, err := forum.SetCommentsAllowed(context.Background(), 999, alice.ID, false)
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound", err)
	}
}

func TestSetCommentsAllowedDeletedPost(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")

	forum.posts = deletedPostStore{PostStore: forum.posts, deleted: postWithDeletedAt(post)}
	_, err := forum.SetCommentsAllowed(context.Background(), post.ID, alice.ID, false)
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound", err)
	}
}

func TestCreateCommentPublishesEvent(t *testing.T) {
	auth, forum, _, n := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")

	c := mustCreateComment(t, forum, alice.ID, post.ID, nil, "hello")
	if c.ID == 0 || c.PostID != post.ID || c.AuthorID != alice.ID || c.Path == "" {
		t.Fatalf("got %+v", c)
	}

	if n.count() != 1 {
		t.Fatalf("notifier calls = %d, want 1", n.count())
	}
	n.mu.Lock()
	ev := n.events[0]
	n.mu.Unlock()
	if ev.postID != post.ID || ev.comment.ID != c.ID {
		t.Fatalf("event = %+v, want postID=%d comment.ID=%d", ev, post.ID, c.ID)
	}
}

func TestCreateCommentNestedPublishesEvent(t *testing.T) {
	auth, forum, _, n := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")
	root := mustCreateComment(t, forum, alice.ID, post.ID, nil, "root")

	child := mustCreateComment(t, forum, alice.ID, post.ID, int64Ptr(root.ID), "child")
	if strings.Count(child.Path, ".") != 1 || !strings.Contains(child.Path, root.Path+".") {
		t.Fatalf("child = %+v", child)
	}
	if n.count() != 2 {
		t.Fatalf("notifier calls = %d, want 2", n.count())
	}
}

func TestCreateCommentErrorsDoNotPublish(t *testing.T) {
	auth, forum, _, n := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")
	other := mustCreatePost(t, forum, alice.ID, "other", "b")
	root := mustCreateComment(t, forum, alice.ID, post.ID, nil, "root")

	ctx := context.Background()
	if _, err := forum.SetCommentsAllowed(ctx, post.ID, alice.ID, false); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		authorID int64
		in       domain.CreateCommentInput
		want     error
	}{
		{"post not found", alice.ID, domain.CreateCommentInput{PostID: 999, Body: "x"}, domain.ErrPostNotFound},
		{"comments disabled", alice.ID, domain.CreateCommentInput{PostID: post.ID, Body: "x"}, domain.ErrCommentsDisabled},
		{"parent from other post", alice.ID, domain.CreateCommentInput{PostID: other.ID, ParentID: int64Ptr(root.ID), Body: "x"}, domain.ErrParentNotInPost},
		{"empty body", alice.ID, domain.CreateCommentInput{PostID: post.ID, Body: ""}, domain.ErrInvalidCommentBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := forum.CreateComment(ctx, tt.authorID, tt.in)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}

	if n.count() != 1 {
		t.Fatalf("notifier calls = %d, want 1 (only the root)", n.count())
	}
}

func TestCreateCommentParentNotFound(t *testing.T) {
	auth, forum, _, n := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")

	_, err := forum.CreateComment(context.Background(), alice.ID, domain.CreateCommentInput{
		PostID: post.ID, ParentID: int64Ptr(999), Body: "x",
	})
	if !errors.Is(err, domain.ErrCommentNotFound) {
		t.Fatalf("err = %v, want ErrCommentNotFound", err)
	}
	if n.count() != 0 {
		t.Fatalf("notifier calls = %d, want 0", n.count())
	}
}

func TestCreateCommentDeepNesting(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")

	parent := mustCreateComment(t, forum, alice.ID, post.ID, nil, "root")
	const levels = 150
	for i := 0; i < levels; i++ {
		parent = mustCreateComment(t, forum, alice.ID, post.ID, int64Ptr(parent.ID), "level")
	}

	got, err := forum.CreateComment(context.Background(), alice.ID, domain.CreateCommentInput{
		PostID: post.ID, ParentID: int64Ptr(parent.ID), Body: "deep",
	})
	if err != nil {
		t.Fatalf("deep nesting: %v", err)
	}
	if strings.Count(got.Path, ".") != levels+1 {
		t.Fatalf("depth = %d, want %d", strings.Count(got.Path, "."), levels+1)
	}
}

func TestListCommentsDepthFirst(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")
	root1 := mustCreateComment(t, forum, alice.ID, post.ID, nil, "r1")
	mustCreateComment(t, forum, alice.ID, post.ID, int64Ptr(root1.ID), "r1-child")
	mustCreateComment(t, forum, alice.ID, post.ID, nil, "r2")

	page, err := forum.ListComments(context.Background(), post.ID, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(page.Items))
	for _, c := range page.Items {
		got = append(got, c.Body)
	}
	want := []string{"r1", "r1-child", "r2"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestListCommentsPagination(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")
	for i := 0; i < 5; i++ {
		mustCreateComment(t, forum, alice.ID, post.ID, nil, "c")
	}
	ctx := context.Background()

	page1, err := forum.ListComments(ctx, post.ID, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 2 || page1.Next == nil {
		t.Fatalf("page1 = %+v, want 2 items + Next", page1.Items)
	}
	page2, err := forum.ListComments(ctx, post.ID, 2, page1.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 2 || page2.Next == nil {
		t.Fatalf("page2 = %+v, want 2 items + Next", page2.Items)
	}
	page3, err := forum.ListComments(ctx, post.ID, 2, page2.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(page3.Items) != 1 || page3.Next != nil {
		t.Fatalf("page3 = %+v, want 1 item, no Next", page3.Items)
	}
}

func TestListCommentsClampsFirst(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	post := mustCreatePost(t, forum, alice.ID, "t", "b")
	for i := 0; i < 3; i++ {
		mustCreateComment(t, forum, alice.ID, post.ID, nil, "c")
	}
	ctx := context.Background()

	page, err := forum.ListComments(ctx, post.ID, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 3 {
		t.Fatalf("first=0 must clamp, got %d", len(page.Items))
	}
}

func TestUsersByIDs(t *testing.T) {
	auth, forum, _, _ := newServices(t)
	alice := mustLogin(t, auth, "alice")
	bob := mustLogin(t, auth, "bob")

	got, err := forum.UsersByIDs(context.Background(), []int64{alice.ID, 999, bob.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != alice.ID || got[1].ID != bob.ID {
		t.Fatalf("got %+v", got)
	}
}

type deletedPostStore struct {
	store.PostStore
	deleted domain.Post
}

func (s deletedPostStore) ByID(_ context.Context, _ int64) (domain.Post, error) {
	return s.deleted, nil
}

func (s deletedPostStore) SetCommentsAllowed(_ context.Context, id, authorID int64, allowed bool) (domain.Post, error) {
	if s.deleted.IsDeleted() {
		return domain.Post{}, domain.ErrPostNotFound
	}
	return s.PostStore.SetCommentsAllowed(context.Background(), id, authorID, allowed)
}

func postWithDeletedAt(p domain.Post) domain.Post {
	now := time.Now().UTC()
	p.DeletedAt = &now
	return p
}

package graphql

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/service"
)

func TestQueryPostsEmpty(t *testing.T) {
	env := newTestEnv(t)

	page, err := env.resolver.Query().Posts(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 || page.Next != nil {
		t.Fatalf("page = %+v, want empty", page)
	}
}

func TestQueryPostsPagination(t *testing.T) {
	env := newTestEnv(t)
	author, _ := env.login(t, "alice")
	env.createPost(t, author.ID, "t1", "b1")
	env.createPost(t, author.ID, "t2", "b2")
	env.createPost(t, author.ID, "t3", "b3")
	ctx := context.Background()

	page1, err := env.resolver.Query().Posts(ctx, intPtr(2), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 2 || page1.Items[0].Title != "t3" || page1.Items[1].Title != "t2" {
		t.Fatalf("page1 = %+v, want [t3 t2]", page1.Items)
	}
	if page1.Next == nil {
		t.Fatal("page1 must have Next")
	}

	page2, err := env.resolver.Query().Posts(ctx, intPtr(2), page1.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 1 || page2.Items[0].Title != "t1" || page2.Next != nil {
		t.Fatalf("page2 = %+v, want [t1] no Next", page2.Items)
	}
}

func TestQueryPostFound(t *testing.T) {
	env := newTestEnv(t)
	author, _ := env.login(t, "alice")
	post := env.createPost(t, author.ID, "t", "b")

	got, err := env.resolver.Query().Post(context.Background(), post.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != post.ID || got.Title != "t" {
		t.Fatalf("got %+v", got)
	}
}

func TestQueryPostNotFound(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.resolver.Query().Post(context.Background(), 999)
	if code := errorCode(t, err); code != "NOT_FOUND" {
		t.Fatalf("code = %q, want NOT_FOUND", code)
	}
}

func TestQueryPostNegativeIDIsValidationError(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.resolver.Query().Post(context.Background(), -1)
	if code := errorCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("code = %q, want VALIDATION_ERROR", code)
	}

	_, err = env.resolver.Subscription().CommentAdded(context.Background(), 0)
	if code := errorCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("subscription code = %q, want VALIDATION_ERROR", code)
	}
}

func TestQueryPostDeleted(t *testing.T) {
	env := newTestEnv(t)
	author, _ := env.login(t, "alice")
	post := env.createPost(t, author.ID, "t", "b")

	now := time.Now().UTC()
	deleted := post
	deleted.DeletedAt = &now
	forum := service.NewForum(env.store.Users(), deletedPostStore{PostStore: env.store.Posts(), deleted: deleted}, env.store.Comments(), env.notifier)
	resolver := NewResolver(forum, env.notifier)

	_, err := resolver.Query().Post(context.Background(), post.ID)
	if code := errorCode(t, err); code != "NOT_FOUND" {
		t.Fatalf("code = %q, want NOT_FOUND", code)
	}
}

func TestQueryCommentsDepthFirst(t *testing.T) {
	env := newTestEnv(t)
	author, _ := env.login(t, "alice")
	post := env.createPost(t, author.ID, "t", "b")
	root1 := env.createComment(t, author.ID, post.ID, nil, "r1")
	env.createComment(t, author.ID, post.ID, int64Ptr(root1.ID), "r1-child")
	env.createComment(t, author.ID, post.ID, nil, "r2")

	page, err := env.resolver.Query().Comments(context.Background(), post.ID, nil, nil)
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

func TestQueryCommentsPagination(t *testing.T) {
	env := newTestEnv(t)
	author, _ := env.login(t, "alice")
	post := env.createPost(t, author.ID, "t", "b")
	for i := 0; i < 5; i++ {
		env.createComment(t, author.ID, post.ID, nil, "c")
	}
	ctx := context.Background()

	page1, err := env.resolver.Query().Comments(ctx, post.ID, intPtr(2), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 2 || page1.Next == nil {
		t.Fatalf("page1 = %+v, want 2 items + Next", page1.Items)
	}
	page2, err := env.resolver.Query().Comments(ctx, post.ID, intPtr(2), page1.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 2 || page2.Next == nil {
		t.Fatalf("page2 = %+v, want 2 items + Next", page2.Items)
	}
	page3, err := env.resolver.Query().Comments(ctx, post.ID, intPtr(2), page2.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(page3.Items) != 1 || page3.Next != nil {
		t.Fatalf("page3 = %+v, want 1 item, no Next", page3.Items)
	}
}

func TestMutationCreatePostUnauthorized(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.resolver.Mutation().CreatePost(context.Background(), CreatePostInput{Title: "t", Body: "b"})
	if code := errorCode(t, err); code != "UNAUTHORIZED" {
		t.Fatalf("code = %q, want UNAUTHORIZED", code)
	}
}

func TestMutationCreatePostAuthorized(t *testing.T) {
	env := newTestEnv(t)
	author, ctx := env.login(t, "alice")

	got, err := env.resolver.Mutation().CreatePost(ctx, CreatePostInput{Title: "  t  ", Body: "  b  "})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == 0 || got.AuthorID != author.ID || got.Title != "t" {
		t.Fatalf("got %+v", got)
	}
}

func TestMutationSetCommentsAllowedForbidden(t *testing.T) {
	env := newTestEnv(t)
	author, _ := env.login(t, "alice")
	post := env.createPost(t, author.ID, "t", "b")
	_, bobCtx := env.login(t, "bob")

	_, err := env.resolver.Mutation().SetCommentsAllowed(bobCtx, post.ID, false)
	if code := errorCode(t, err); code != "FORBIDDEN" {
		t.Fatalf("code = %q, want FORBIDDEN", code)
	}
}

func TestMutationSetCommentsAllowedByAuthor(t *testing.T) {
	env := newTestEnv(t)
	author, ctx := env.login(t, "alice")
	post := env.createPost(t, author.ID, "t", "b")

	got, err := env.resolver.Mutation().SetCommentsAllowed(ctx, post.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.CommentsAllowed {
		t.Fatal("comments_allowed must be false")
	}
}

func TestMutationCreateCommentHappyPath(t *testing.T) {
	env := newTestEnv(t)
	author, ctx := env.login(t, "alice")
	post := env.createPost(t, author.ID, "t", "b")

	ch, unsubscribe := env.notifier.Subscribe(post.ID)
	defer unsubscribe()

	c, err := env.resolver.Mutation().CreateComment(ctx, CreateCommentInput{PostID: post.ID, Body: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == 0 || c.AuthorID != author.ID || c.PostID != post.ID {
		t.Fatalf("got %+v", c)
	}

	select {
	case ev := <-ch:
		if ev.ID != c.ID {
			t.Fatalf("event = %+v, want comment %d", ev, c.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("no notifier event received")
	}
}

func TestMutationCreateCommentInvalidBody(t *testing.T) {
	env := newTestEnv(t)
	author, ctx := env.login(t, "alice")
	post := env.createPost(t, author.ID, "t", "b")

	_, err := env.resolver.Mutation().CreateComment(ctx, CreateCommentInput{PostID: post.ID, Body: strings.Repeat("x", 2001)})
	if code := errorCode(t, err); code != "VALIDATION_ERROR" {
		t.Fatalf("code = %q, want VALIDATION_ERROR", code)
	}
}

func TestMutationCreateCommentParentFromOtherPost(t *testing.T) {
	env := newTestEnv(t)
	author, ctx := env.login(t, "alice")
	post1 := env.createPost(t, author.ID, "p1", "b")
	post2 := env.createPost(t, author.ID, "p2", "b")
	root := env.createComment(t, author.ID, post1.ID, nil, "root")

	_, err := env.resolver.Mutation().CreateComment(ctx, CreateCommentInput{
		PostID:   post2.ID,
		ParentID: int64Ptr(root.ID),
		Body:     "x",
	})
	if code := errorCode(t, err); code != "PARENT_NOT_IN_POST" {
		t.Fatalf("code = %q, want PARENT_NOT_IN_POST", code)
	}
}

func TestMutationCreateCommentUnauthorized(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.resolver.Mutation().CreateComment(context.Background(), CreateCommentInput{PostID: 1, Body: "x"})
	if code := errorCode(t, err); code != "UNAUTHORIZED" {
		t.Fatalf("code = %q, want UNAUTHORIZED", code)
	}
}

func TestPostAuthorResolved(t *testing.T) {
	env := newTestEnv(t)
	alice, _ := env.login(t, "alice")
	post := env.createPost(t, alice.ID, "t", "b")

	u, err := env.resolver.Post().Author(context.Background(), &post)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != alice.ID || u.Username.String() != "alice" {
		t.Fatalf("got %+v, want alice", u)
	}
}

func TestSubscriptionCommentAdded(t *testing.T) {
	env := newTestEnv(t)
	author, _ := env.login(t, "alice")
	post := env.createPost(t, author.ID, "t", "b")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := env.resolver.Subscription().CommentAdded(ctx, post.ID)
	if err != nil {
		t.Fatal(err)
	}

	env.createComment(t, author.ID, post.ID, nil, "event")

	select {
	case c := <-ch:
		if c.Body != "event" {
			t.Fatalf("got %+v, want body event", c)
		}
	case <-time.After(time.Second):
		t.Fatal("no event received")
	}

	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel must close after cancel")
		}
	case <-time.After(time.Second):
		t.Fatal("channel did not close on cancel")
	}
}

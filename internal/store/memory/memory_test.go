package memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
)

func newTestStore(t *testing.T) (*Store, *Users, *Posts, *Comments) {
	t.Helper()
	s := New()
	return s, s.Users(), s.Posts(), s.Comments()
}

func mustCreateUser(t *testing.T, u *Users, name string) domain.User {
	t.Helper()
	user, err := u.Create(context.Background(), domain.Username(name))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func mustCreatePost(t *testing.T, p *Posts, authorID int64, title, body string) domain.Post {
	t.Helper()
	post, err := p.Create(context.Background(), domain.Post{AuthorID: authorID, Title: title, Body: body})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	return post
}

func mustCreateComment(t *testing.T, c *Comments, postID, authorID int64, parentID *int64, body string) domain.Comment {
	t.Helper()
	comment, err := c.Create(context.Background(), domain.Comment{
		PostID: postID, AuthorID: authorID, ParentID: parentID, Body: body,
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	return comment
}

func int64Ptr(v int64) *int64 { return &v }

func TestUsersCreateIsIdempotentByUsername(t *testing.T) {
	_, u, _, _ := newTestStore(t)
	ctx := context.Background()

	alice1, err := u.Create(ctx, domain.Username("alice"))
	if err != nil {
		t.Fatal(err)
	}
	alice2, err := u.Create(ctx, domain.Username("alice"))
	if err != nil {
		t.Fatal(err)
	}
	bob, err := u.Create(ctx, domain.Username("bob"))
	if err != nil {
		t.Fatal(err)
	}

	if alice1.ID != alice2.ID {
		t.Fatalf("duplicate username produced different ids: %d vs %d", alice1.ID, alice2.ID)
	}
	if bob.ID <= alice1.ID {
		t.Fatalf("ids must be increasing: alice=%d bob=%d", alice1.ID, bob.ID)
	}
	if alice1.CreatedAt.IsZero() {
		t.Fatal("created_at must be set")
	}
}

func TestUsersLookup(t *testing.T) {
	_, u, _, _ := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")

	if got, err := u.ByUsername(ctx, "alice"); err != nil || got.ID != alice.ID {
		t.Fatalf("ByUsername: got %+v err %v", got, err)
	}
	if _, err := u.ByUsername(ctx, "nobody"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("ByUsername(nobody) = %v, want ErrUserNotFound", err)
	}
	if got, err := u.ByID(ctx, alice.ID); err != nil || got.Username != "alice" {
		t.Fatalf("ByID: got %+v err %v", got, err)
	}
	if _, err := u.ByID(ctx, 999); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("ByID(999) = %v, want ErrUserNotFound", err)
	}

	bob := mustCreateUser(t, u, "bob")
	got, err := u.ByIDs(ctx, []int64{alice.ID, bob.ID, 999})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("ByIDs returned %d users, want 2", len(got))
	}
}

func TestPostsCreateAndByID(t *testing.T) {
	_, u, p, _ := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")

	post := mustCreatePost(t, p, alice.ID, "title", "body")
	if post.ID == 0 {
		t.Fatal("id must be set")
	}
	if !post.CommentsAllowed {
		t.Fatal("comments_allowed must default to true")
	}
	if post.CreatedAt.IsZero() || post.UpdatedAt.IsZero() {
		t.Fatal("timestamps must be set")
	}

	if got, err := p.ByID(ctx, post.ID); err != nil || got.Title != "title" {
		t.Fatalf("ByID: got %+v err %v", got, err)
	}
	if _, err := p.ByID(ctx, 999); !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("ByID(999) = %v, want ErrPostNotFound", err)
	}
}

func TestPostsListPagination(t *testing.T) {
	_, u, p, _ := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")

	p1 := mustCreatePost(t, p, alice.ID, "t1", "b1")
	p2 := mustCreatePost(t, p, alice.ID, "t2", "b2")
	p3 := mustCreatePost(t, p, alice.ID, "t3", "b3")

	page1, err := p.List(ctx, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("page1 len = %d, want 2", len(page1.Items))
	}
	if page1.Items[0].ID != p3.ID || page1.Items[1].ID != p2.ID {
		t.Fatalf("page1 order wrong: %d, %d; want %d, %d", page1.Items[0].ID, page1.Items[1].ID, p3.ID, p2.ID)
	}
	if page1.Next == nil {
		t.Fatal("page1 must have Next")
	}

	page2, err := p.List(ctx, 2, page1.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 1 || page2.Items[0].ID != p1.ID {
		t.Fatalf("page2 = %+v, want [p1]", page2.Items)
	}
	if page2.Next != nil {
		t.Fatal("page2 must not have Next")
	}
}

func TestPostsListHidesDeleted(t *testing.T) {
	s, u, p, _ := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")
	post := mustCreatePost(t, p, alice.ID, "t", "b")

	now := time.Now().UTC()
	deleted := post
	deleted.DeletedAt = &now
	s.postsByID[post.ID] = deleted
	for i := range s.posts {
		if s.posts[i].ID == post.ID {
			s.posts[i] = deleted
		}
	}

	page, err := p.List(ctx, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("deleted post must be hidden, got %d", len(page.Items))
	}
}

func TestPostsSetCommentsAllowed(t *testing.T) {
	_, u, p, _ := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")
	post := mustCreatePost(t, p, alice.ID, "t", "b")

	got, err := p.SetCommentsAllowed(ctx, post.ID, alice.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.CommentsAllowed {
		t.Fatal("comments_allowed must be false")
	}
	if got.UpdatedAt.Before(post.UpdatedAt) {
		t.Fatal("updated_at must move forward")
	}

	if _, err := p.SetCommentsAllowed(ctx, 999, alice.ID, false); !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("SetCommentsAllowed(999) = %v, want ErrPostNotFound", err)
	}
	if _, err := p.SetCommentsAllowed(ctx, post.ID, 999, false); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("SetCommentsAllowed(by non-author) = %v, want ErrForbidden", err)
	}
}

func TestPostsSetCommentsAllowedDeleted(t *testing.T) {
	_, u, p, _ := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")
	post := mustCreatePost(t, p, alice.ID, "t", "b")

	now := time.Now().UTC()
	post.DeletedAt = &now
	p.postsByID[post.ID] = post

	if _, err := p.SetCommentsAllowed(ctx, post.ID, alice.ID, false); !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("SetCommentsAllowed(deleted) = %v, want ErrPostNotFound", err)
	}
}

func TestCommentsRootAndChildPaths(t *testing.T) {
	_, u, p, c := newTestStore(t)
	alice := mustCreateUser(t, u, "alice")
	post := mustCreatePost(t, p, alice.ID, "t", "b")

	root := mustCreateComment(t, c, post.ID, alice.ID, nil, "root")
	if root.Path != padID(root.ID) || root.Depth != 0 {
		t.Fatalf("root path/depth = %q/%d", root.Path, root.Depth)
	}

	child := mustCreateComment(t, c, post.ID, alice.ID, int64Ptr(root.ID), "child")
	wantPath := root.Path + "." + padID(child.ID)
	if child.Path != wantPath || child.Depth != 1 {
		t.Fatalf("child path/depth = %q/%d, want %q/1", child.Path, child.Depth, wantPath)
	}
}

func TestCommentsParentIDDeepCopy(t *testing.T) {
	_, u, p, c := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")
	post := mustCreatePost(t, p, alice.ID, "t", "b")
	root := mustCreateComment(t, c, post.ID, alice.ID, nil, "root")

	child := mustCreateComment(t, c, post.ID, alice.ID, int64Ptr(root.ID), "child")

	*child.ParentID = 42

	got, err := c.ByID(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ParentID == nil || *got.ParentID != root.ID {
		t.Fatalf("ByID parent = %v, want %d (deep copy)", got.ParentID, root.ID)
	}

	page, err := c.ListByPost(ctx, post.ID, 20, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range page.Items {
		if item.ID == child.ID && (item.ParentID == nil || *item.ParentID != root.ID) {
			t.Fatalf("ListByPost parent = %v, want %d (deep copy)", item.ParentID, root.ID)
		}
	}

	if *child.ParentID != 42 {
		t.Fatal("returned comment should not share pointer with stored comment")
	}
}

func TestCommentsDeepNesting(t *testing.T) {
	_, u, p, c := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")
	post := mustCreatePost(t, p, alice.ID, "t", "b")

	var parent *int64
	var last domain.Comment
	for i := 0; i < 10; i++ {
		last = mustCreateComment(t, c, post.ID, alice.ID, parent, "level")
		parent = int64Ptr(last.ID)
	}
	if last.Depth != 9 {
		t.Fatalf("depth = %d, want 9", last.Depth)
	}

	page, err := c.ListByPost(ctx, post.ID, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 10 {
		t.Fatalf("len = %d, want 10", len(page.Items))
	}
	for i, cm := range page.Items {
		if cm.Depth != i {
			t.Fatalf("item %d depth = %d, want %d", i, cm.Depth, i)
		}
	}
}

func TestCommentsDepthFirstOrder(t *testing.T) {
	_, u, p, c := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")
	post := mustCreatePost(t, p, alice.ID, "t", "b")

	root1 := mustCreateComment(t, c, post.ID, alice.ID, nil, "r1")
	mustCreateComment(t, c, post.ID, alice.ID, int64Ptr(root1.ID), "r1-child")
	mustCreateComment(t, c, post.ID, alice.ID, nil, "r2")

	page, err := c.ListByPost(ctx, post.ID, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(page.Items))
	for _, cm := range page.Items {
		got = append(got, cm.Body)
	}
	want := []string{"r1", "r1-child", "r2"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestCommentsPagination(t *testing.T) {
	_, u, p, c := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")
	post := mustCreatePost(t, p, alice.ID, "t", "b")

	for i := 0; i < 5; i++ {
		mustCreateComment(t, c, post.ID, alice.ID, nil, "c")
	}

	page1, err := c.ListByPost(ctx, post.ID, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 2 || page1.Next == nil {
		t.Fatalf("page1 = %+v, want 2 items + Next", page1.Items)
	}

	page2, err := c.ListByPost(ctx, post.ID, 2, page1.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 2 || page2.Next == nil {
		t.Fatalf("page2 = %+v, want 2 items + Next", page2.Items)
	}

	page3, err := c.ListByPost(ctx, post.ID, 2, page2.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(page3.Items) != 1 || page3.Next != nil {
		t.Fatalf("page3 = %+v, want 1 item, no Next", page3.Items)
	}

	if page1.Items[0].ID == page2.Items[0].ID || page2.Items[0].ID == page3.Items[0].ID {
		t.Fatal("pages must not overlap")
	}
}

func TestCommentsCreateErrors(t *testing.T) {
	t.Run("post not found", func(t *testing.T) {
		_, u, _, c := newTestStore(t)
		alice := mustCreateUser(t, u, "alice")
		_, err := c.Create(context.Background(), domain.Comment{PostID: 999, AuthorID: alice.ID, Body: "x"})
		if !errors.Is(err, domain.ErrPostNotFound) {
			t.Fatalf("err = %v, want ErrPostNotFound", err)
		}
	})

	t.Run("post deleted", func(t *testing.T) {
		s, u, p, c := newTestStore(t)
		alice := mustCreateUser(t, u, "alice")
		post := mustCreatePost(t, p, alice.ID, "t", "b")

		now := time.Now().UTC()
		deleted := post
		deleted.DeletedAt = &now
		s.postsByID[post.ID] = deleted

		_, err := c.Create(context.Background(), domain.Comment{PostID: post.ID, AuthorID: alice.ID, Body: "x"})
		if !errors.Is(err, domain.ErrPostNotFound) {
			t.Fatalf("err = %v, want ErrPostNotFound", err)
		}
	})

	t.Run("comments disabled", func(t *testing.T) {
		_, u, p, c := newTestStore(t)
		alice := mustCreateUser(t, u, "alice")
		post := mustCreatePost(t, p, alice.ID, "t", "b")
		if _, err := p.SetCommentsAllowed(context.Background(), post.ID, alice.ID, false); err != nil {
			t.Fatal(err)
		}

		_, err := c.Create(context.Background(), domain.Comment{PostID: post.ID, AuthorID: alice.ID, Body: "x"})
		if !errors.Is(err, domain.ErrCommentsDisabled) {
			t.Fatalf("err = %v, want ErrCommentsDisabled", err)
		}
	})

	t.Run("parent not found", func(t *testing.T) {
		_, u, p, c := newTestStore(t)
		alice := mustCreateUser(t, u, "alice")
		post := mustCreatePost(t, p, alice.ID, "t", "b")

		_, err := c.Create(context.Background(), domain.Comment{PostID: post.ID, AuthorID: alice.ID, ParentID: int64Ptr(999), Body: "x"})
		if !errors.Is(err, domain.ErrCommentNotFound) {
			t.Fatalf("err = %v, want ErrCommentNotFound", err)
		}
	})

	t.Run("parent from other post", func(t *testing.T) {
		_, u, p, c := newTestStore(t)
		alice := mustCreateUser(t, u, "alice")
		post := mustCreatePost(t, p, alice.ID, "t", "b")
		root := mustCreateComment(t, c, post.ID, alice.ID, nil, "root")
		otherPost := mustCreatePost(t, p, alice.ID, "other", "b")

		_, err := c.Create(context.Background(), domain.Comment{PostID: otherPost.ID, AuthorID: alice.ID, ParentID: int64Ptr(root.ID), Body: "x"})
		if !errors.Is(err, domain.ErrParentNotInPost) {
			t.Fatalf("err = %v, want ErrParentNotInPost", err)
		}
	})

	t.Run("parent deleted", func(t *testing.T) {
		s, u, p, c := newTestStore(t)
		alice := mustCreateUser(t, u, "alice")
		post := mustCreatePost(t, p, alice.ID, "t", "b")
		root := mustCreateComment(t, c, post.ID, alice.ID, nil, "root")

		now := time.Now().UTC()
		deletedRoot := root
		deletedRoot.DeletedAt = &now
		s.commentsByID[root.ID] = deletedRoot
		for i := range s.comments[post.ID] {
			if s.comments[post.ID][i].ID == root.ID {
				s.comments[post.ID][i] = deletedRoot
			}
		}

		_, err := c.Create(context.Background(), domain.Comment{PostID: post.ID, AuthorID: alice.ID, ParentID: int64Ptr(root.ID), Body: "x"})
		if !errors.Is(err, domain.ErrParentDeleted) {
			t.Fatalf("err = %v, want ErrParentDeleted", err)
		}
	})
}

func TestCommentsListByPostHidesDeleted(t *testing.T) {
	s, u, p, c := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")
	post := mustCreatePost(t, p, alice.ID, "t", "b")

	c1 := mustCreateComment(t, c, post.ID, alice.ID, nil, "c1")
	mustCreateComment(t, c, post.ID, alice.ID, nil, "c2")

	now := time.Now().UTC()
	deleted := c1
	deleted.DeletedAt = &now
	s.commentsByID[c1.ID] = deleted
	for i := range s.comments[post.ID] {
		if s.comments[post.ID][i].ID == c1.ID {
			s.comments[post.ID][i] = deleted
		}
	}

	page, err := c.ListByPost(ctx, post.ID, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != c1.ID+1 {
		t.Fatalf("items = %+v, want only c2", page.Items)
	}
}

func TestCommentsConcurrentCreate(t *testing.T) {
	_, u, p, c := newTestStore(t)
	ctx := context.Background()
	alice := mustCreateUser(t, u, "alice")
	post := mustCreatePost(t, p, alice.ID, "t", "b")

	const n = 50
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Create(ctx, domain.Comment{PostID: post.ID, AuthorID: alice.ID, Body: "body"})
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}

	page, err := c.ListByPost(ctx, post.ID, n, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != n {
		t.Fatalf("got %d comments, want %d", len(page.Items), n)
	}

	seen := make(map[string]bool, n)
	prev := ""
	for _, cm := range page.Items {
		if cm.Path <= prev {
			t.Fatalf("paths not strictly increasing: %q <= %q", cm.Path, prev)
		}
		if seen[cm.Path] {
			t.Fatalf("duplicate path %q", cm.Path)
		}
		seen[cm.Path] = true
		prev = cm.Path
	}
}

var (
	_ store.UserStore    = (*Users)(nil)
	_ store.PostStore    = (*Posts)(nil)
	_ store.CommentStore = (*Comments)(nil)
)

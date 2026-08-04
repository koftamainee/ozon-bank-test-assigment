package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewUsername(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		u, err := NewUsername("  alice  ")
		if err != nil {
			t.Fatalf("NewUsername() error = %v", err)
		}
		if u.String() != "alice" {
			t.Errorf("NewUsername() = %q, want alice", u)
		}
	})

	t.Run("empty", func(t *testing.T) {
		if _, err := NewUsername("   "); !errors.Is(err, ErrInvalidUsername) {
			t.Errorf("NewUsername() error = %v, want ErrInvalidUsername", err)
		}
	})

	t.Run("too long", func(t *testing.T) {
		if _, err := NewUsername(strings.Repeat("a", MaxUsernameLength+1)); !errors.Is(err, ErrInvalidUsername) {
			t.Errorf("NewUsername() error = %v, want ErrInvalidUsername", err)
		}
	})

	t.Run("exact max length", func(t *testing.T) {
		if _, err := NewUsername(strings.Repeat("a", MaxUsernameLength)); err != nil {
			t.Errorf("NewUsername() error = %v, want nil", err)
		}
	})
}

func TestNewCommentBody(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		b, err := NewCommentBody("  hello  ")
		if err != nil {
			t.Fatalf("NewCommentBody() error = %v", err)
		}
		if b.String() != "hello" {
			t.Errorf("NewCommentBody() = %q, want hello", b)
		}
	})

	t.Run("empty", func(t *testing.T) {
		if _, err := NewCommentBody(" \n\t "); !errors.Is(err, ErrInvalidCommentBody) {
			t.Errorf("NewCommentBody() error = %v, want ErrInvalidCommentBody", err)
		}
	})

	t.Run("too long", func(t *testing.T) {
		if _, err := NewCommentBody(strings.Repeat("x", MaxCommentLength+1)); !errors.Is(err, ErrInvalidCommentBody) {
			t.Errorf("NewCommentBody() error = %v, want ErrInvalidCommentBody", err)
		}
	})

	t.Run("exact max length", func(t *testing.T) {
		if _, err := NewCommentBody(strings.Repeat("x", MaxCommentLength)); err != nil {
			t.Errorf("NewCommentBody() error = %v, want nil", err)
		}
	})

	t.Run("multibyte runes count characters", func(t *testing.T) {
		body := strings.Repeat("я", MaxCommentLength)
		if _, err := NewCommentBody(body); err != nil {
			t.Errorf("NewCommentBody() error = %v, want nil (runes not bytes)", err)
		}
	})
}

func TestPostIsDeleted(t *testing.T) {
	if (Post{}).IsDeleted() {
		t.Error("zero Post IsDeleted() = true, want false")
	}

	now := time.Now()
	if !(Post{DeletedAt: &now}).IsDeleted() {
		t.Error("deleted Post IsDeleted() = false, want true")
	}
}

func TestCommentIsDeleted(t *testing.T) {
	if (Comment{}).IsDeleted() {
		t.Error("zero Comment IsDeleted() = true, want false")
	}

	now := time.Now()
	if !(Comment{DeletedAt: &now}).IsDeleted() {
		t.Error("deleted Comment IsDeleted() = false, want true")
	}
}

func TestBuildCommentTree(t *testing.T) {
	comments := []Comment{
		{ID: 1, PostID: 10, Path: "00000000000000000001"},
		{ID: 2, PostID: 10, ParentID: new(int64(1)), Path: "00000000000000000001.00000000000000000002"},
		{ID: 3, PostID: 10, ParentID: new(int64(2)), Path: "00000000000000000001.00000000000000000002.00000000000000000003"},
		{ID: 4, PostID: 10, Path: "00000000000000000004"},
	}

	roots := BuildCommentTree(comments)

	if len(roots) != 2 {
		t.Fatalf("roots = %d, want 2", len(roots))
	}
	if roots[0].ID != 1 {
		t.Errorf("roots[0].ID = %d, want 1", roots[0].ID)
	}
	if roots[1].ID != 4 {
		t.Errorf("roots[1].ID = %d, want 4", roots[1].ID)
	}

	children := roots[0].Children
	if len(children) != 1 || children[0].ID != 2 {
		t.Fatalf("roots[0] children = %+v, want [2]", children)
	}
	if len(children[0].Children) != 1 || children[0].Children[0].ID != 3 {
		t.Errorf("children[0] children = %+v, want [3]", children[0].Children)
	}
}

func TestBuildCommentTreeEmpty(t *testing.T) {
	if roots := BuildCommentTree(nil); len(roots) != 0 {
		t.Errorf("BuildCommentTree(nil) = %d roots, want 0", len(roots))
	}
}

func TestBuildCommentTreeOrphanBecomesRoot(t *testing.T) {
	comments := []Comment{
		{ID: 1, ParentID: new(int64(999)), Path: "00000000000000000001"},
	}

	roots := BuildCommentTree(comments)
	if len(roots) != 1 || roots[0].ID != 1 {
		t.Errorf("orphan handling: roots = %+v, want [1]", roots)
	}
}

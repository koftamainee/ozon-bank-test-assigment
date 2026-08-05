package graphql

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/auth"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/notifier"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/service"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store/memory"
)

type testEnv struct {
	forum    *service.ForumService
	auth     *service.AuthService
	notifier *notifier.Broadcaster
	store    *memory.Store
	manager  *auth.Manager
	resolver *Resolver
	schema   graphql.ExecutableSchema
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	s := memory.New()
	n := notifier.New()
	authSvc := service.NewAuth(s.Users())
	forum := service.NewForum(s.Users(), s.Posts(), s.Comments(), n)
	manager := auth.New([]byte("test-secret"), time.Hour, false)
	r := NewResolver(forum, n)
	return &testEnv{
		forum:    forum,
		auth:     authSvc,
		notifier: n,
		store:    s,
		manager:  manager,
		resolver: r,
		schema:   NewSchema(r),
	}
}

func (e *testEnv) login(t *testing.T, username string) (domain.User, context.Context) {
	t.Helper()
	u, err := e.auth.Login(context.Background(), username)
	if err != nil {
		t.Fatal(err)
	}
	token, err := e.manager.Sign(u.ID, u.Username)
	if err != nil {
		t.Fatal(err)
	}
	return u, authedContext(t, e.manager, token)
}

func (e *testEnv) cookieFor(t *testing.T, username string) *http.Cookie {
	t.Helper()
	u, err := e.auth.Login(context.Background(), username)
	if err != nil {
		t.Fatal(err)
	}
	token, err := e.manager.Sign(u.ID, u.Username)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: auth.CookieName, Value: token}
}

func authedContext(t *testing.T, m *auth.Manager, token string) context.Context {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	var ctx context.Context
	m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})).ServeHTTP(rec, req)
	return ctx
}

func errorCode(t *testing.T, err error) string {
	t.Helper()
	var gqlErr *gqlerror.Error
	if !errors.As(err, &gqlErr) {
		t.Fatalf("err = %v, want *gqlerror.Error", err)
	}
	code, _ := gqlErr.Extensions["code"].(string)
	return code
}

func (e *testEnv) createPost(t *testing.T, authorID int64, title, body string) domain.Post {
	t.Helper()
	p, err := e.forum.CreatePost(context.Background(), authorID, domain.CreatePostInput{Title: title, Body: body})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (e *testEnv) createComment(t *testing.T, authorID, postID int64, parentID *int64, body string) domain.Comment {
	t.Helper()
	c, err := e.forum.CreateComment(context.Background(), authorID, domain.CreateCommentInput{
		PostID:   postID,
		ParentID: parentID,
		Body:     domain.CommentBody(body),
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func intPtr(v int) *int       { return &v }
func int64Ptr(v int64) *int64 { return &v }

type deletedPostStore struct {
	store.PostStore
	deleted domain.Post
}

func (s deletedPostStore) ByID(_ context.Context, _ int64) (domain.Post, error) {
	return s.deleted, nil
}

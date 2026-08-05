package graphql

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/json"
)

func graphqlPost(t *testing.T, handler http.Handler, query string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	srv := handler
	body := `{"query":` + strconv.Quote(query) + `}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func graphqlServer(env *testEnv) http.Handler {
	return env.manager.Middleware(handler.NewDefaultServer(env.schema))
}

func firstErrorCode(t *testing.T, body []byte) string {
	t.Helper()
	var resp struct {
		Errors []struct {
			Extensions struct {
				Code string `json:"code"`
			} `json:"extensions"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal response: %v, body %s", err, body)
	}
	if len(resp.Errors) == 0 {
		t.Fatalf("no errors in response, body %s", body)
	}
	return resp.Errors[0].Extensions.Code
}

func TestSmokeQueryPosts(t *testing.T) {
	env := newTestEnv(t)

	rec := graphqlPost(t, graphqlServer(env), `{ posts { items { id title } } }`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"posts"`) {
		t.Fatalf("body = %s, want posts field", rec.Body.String())
	}
}

func TestSmokeMutationUnauthorized(t *testing.T) {
	env := newTestEnv(t)

	rec := graphqlPost(t, graphqlServer(env), `mutation { createPost(input: {title: "t", body: "b"}) { id } }`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if code := firstErrorCode(t, rec.Body.Bytes()); code != "UNAUTHORIZED" {
		t.Fatalf("code = %q, want UNAUTHORIZED", code)
	}
}

func TestSmokePostNotFound(t *testing.T) {
	env := newTestEnv(t)

	rec := graphqlPost(t, graphqlServer(env), `{ post(id: 999) { id } }`)
	if code := firstErrorCode(t, rec.Body.Bytes()); code != "NOT_FOUND" {
		t.Fatalf("code = %q, want NOT_FOUND", code)
	}
}

func TestSmokeMutationAuthorizedWithAuthor(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.cookieFor(t, "alice")

	rec := graphqlPost(t, graphqlServer(env),
		`mutation { createPost(input: {title: "hello", body: "world"}) { id title author { username } } }`,
		cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"title":"hello"`) || !strings.Contains(body, `"username":"alice"`) {
		t.Fatalf("body = %s, want post with author alice", body)
	}
}

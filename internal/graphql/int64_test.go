package graphql

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	appjson "github.com/koftamainee/ozon-bank-test-assigment/internal/json"
)

func TestMarshalInt64AsString(t *testing.T) {
	big := int64(1)<<53 + 1
	m := MarshalInt64(big)
	w := httptest.NewRecorder()
	m.MarshalGQL(w)

	if got := w.Body.String(); got != `"9007199254740993"` {
		t.Errorf("MarshalInt64 = %s, want JSON string \"9007199254740993\"", got)
	}
}

func TestUnmarshalInt64(t *testing.T) {
	big := int64(1)<<53 + 1

	tests := []struct {
		name string
		in   any
		want int64
		ok   bool
	}{
		{"int64", int64(42), 42, true},
		{"int", 42, 42, true},
		{"float64 integral", float64(42), 42, true},
		{"json.Number", json.Number("42"), 42, true},
		{"json.Number big", json.Number("9007199254740993"), big, true},
		{"string", "42", 42, true},
		{"string big", "9007199254740993", big, true},
		{"float64 fractional", float64(42.5), 0, false},
		{"json.Number fractional", json.Number("42.5"), 0, false},
		{"json.Number out of range", json.Number("99999999999999999999"), 0, false},
		{"string garbage", "abc", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalInt64(tt.in)
			if tt.ok && err != nil {
				t.Fatalf("UnmarshalInt64(%v) error = %v", tt.in, err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("UnmarshalInt64(%v) = %d, want error", tt.in, got)
			}
			if tt.ok && got != tt.want {
				t.Fatalf("UnmarshalInt64(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestInt64ViaVariables(t *testing.T) {
	env := newTestEnv(t)
	author, _ := env.login(t, "alice")
	post := env.createPost(t, author.ID, "var", "b")
	srv := graphqlServer(env)

	body := `{"query":"query($id: Int64!) { post(id: $id) { id title } }","variables":{"id":` + strconv.FormatInt(post.ID, 10) + `}}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"title":"var"`) {
		t.Fatalf("body = %s, want post with title var", rec.Body.String())
	}

	var resp struct {
		Data struct {
			Post struct {
				ID string `json:"id"`
			} `json:"post"`
		} `json:"data"`
	}
	if err := appjson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v, body %s", err, rec.Body.String())
	}
	if resp.Data.Post.ID != strconv.FormatInt(post.ID, 10) {
		t.Errorf("post id = %q, want %d", resp.Data.Post.ID, post.ID)
	}
}

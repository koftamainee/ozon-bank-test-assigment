package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) Error {
	t.Helper()

	var e Error
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	return e
}

func TestOk(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := Ok(rec, map[string]string{"a": "b"}); err != nil {
		t.Fatalf("Ok() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := rec.Body.String(); got != `{"a":"b"}` {
		t.Errorf("body = %q", got)
	}
}

func TestCreated(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := Created(rec, nil); err != nil {
		t.Fatalf("Created() error = %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := NoContent(rec); err != nil {
		t.Fatalf("NoContent() error = %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestErrorHelpers(t *testing.T) {
	cases := []struct {
		name       string
		call       func(w http.ResponseWriter, msg string) error
		wantStatus int
		wantCode   string
	}{
		{"NotFound", NotFound, http.StatusNotFound, "NOT_FOUND"},
		{"Conflict", Conflict, http.StatusConflict, "CONFLICT"},
		{"BadRequest", BadRequest, http.StatusBadRequest, "BAD_REQUEST"},
		{"Unauthorized", Unauthorized, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"Forbidden", Forbidden, http.StatusForbidden, "FORBIDDEN"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			if err := c.call(rec, "some message"); err != nil {
				t.Fatalf("%s() error = %v", c.name, err)
			}

			if rec.Code != c.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, c.wantStatus)
			}

			e := decodeError(t, rec)
			if e.Code != c.wantCode {
				t.Errorf("code = %q, want %q", e.Code, c.wantCode)
			}
			if e.Msg != "some message" {
				t.Errorf("msg = %q, want %q", e.Msg, "some message")
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	rec := httptest.NewRecorder()
	origErr := &jsonError{msg: "invalid input"}
	if err := ValidationError(rec, origErr); err != nil {
		t.Fatalf("ValidationError() error = %v", err)
	}

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	e := decodeError(t, rec)
	if e.Code != "VALIDATION_ERROR" {
		t.Errorf("code = %q, want VALIDATION_ERROR", e.Code)
	}
	if e.Msg != "invalid input" {
		t.Errorf("msg = %q, want %q", e.Msg, "invalid input")
	}
}

func TestInternal(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := Internal(rec); err != nil {
		t.Fatalf("Internal() error = %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	e := decodeError(t, rec)
	if e.Code != "INTERNAL_ERROR" {
		t.Errorf("code = %q, want INTERNAL_ERROR", e.Code)
	}
	if !strings.Contains(e.Msg, "internal server error") {
		t.Errorf("msg = %q, want internal server error", e.Msg)
	}
}

func TestErrorf(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := Errorf(rec, http.StatusBadGateway, "UPSTREAM", "upstream failed"); err != nil {
		t.Fatalf("Errorf() error = %v", err)
	}

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	e := decodeError(t, rec)
	if e.Code != "UPSTREAM" || e.Msg != "upstream failed" {
		t.Errorf("error = %+v", e)
	}
}

func TestErrorString(t *testing.T) {
	e := Error{Code: "TEA", Msg: "short and stout"}
	if got := e.Error(); got != "TEA: short and stout" {
		t.Errorf("Error() = %q, want %q", got, "TEA: short and stout")
	}
}

type jsonError struct {
	msg string
}

func (e *jsonError) Error() string {
	return e.msg
}

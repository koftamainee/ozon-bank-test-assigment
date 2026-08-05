package response

import (
	"fmt"
	"net/http"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/json"
)

type Error struct {
	Code string `json:"code"`
	Msg  string `json:"error"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

func encode(w http.ResponseWriter, status int, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(b)
	return err
}

func Ok(w http.ResponseWriter, data any) error {
	return encode(w, http.StatusOK, data)
}

func Created(w http.ResponseWriter, data any) error {
	return encode(w, http.StatusCreated, data)
}

func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func NotFound(w http.ResponseWriter, msg string) error {
	return Errorf(w, http.StatusNotFound, "NOT_FOUND", msg)
}

func Conflict(w http.ResponseWriter, msg string) error {
	return Errorf(w, http.StatusConflict, "CONFLICT", msg)
}

func BadRequest(w http.ResponseWriter, msg string) error {
	return Errorf(w, http.StatusBadRequest, "BAD_REQUEST", msg)
}

func ValidationError(w http.ResponseWriter, err error) error {
	return Errorf(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
}

func Unauthorized(w http.ResponseWriter, msg string) error {
	return Errorf(w, http.StatusUnauthorized, "UNAUTHORIZED", msg)
}

func Forbidden(w http.ResponseWriter, msg string) error {
	return Errorf(w, http.StatusForbidden, "FORBIDDEN", msg)
}

func Internal(w http.ResponseWriter) error {
	return Errorf(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func Errorf(w http.ResponseWriter, status int, code string, msg string) error {
	return encode(w, status, Error{
		Code: code,
		Msg:  msg,
	})
}

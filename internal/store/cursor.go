package store

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const postCursorSeparator = "|"

var ErrInvalidCursor = errors.New("store: invalid cursor")

func EncodePostCursor(createdAt time.Time, id int64) string {
	return encode(fmt.Sprintf("%s%s%d", createdAt.Format(time.RFC3339Nano), postCursorSeparator, id))
}

func DecodePostCursor(s string) (time.Time, int64, error) {
	raw, err := decode(s)
	if err != nil {
		return time.Time{}, 0, err
	}

	createdAtStr, idStr, ok := strings.Cut(raw, postCursorSeparator)
	if !ok {
		return time.Time{}, 0, ErrInvalidCursor
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtStr)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("%w: bad timestamp", ErrInvalidCursor)
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("%w: bad id", ErrInvalidCursor)
	}

	return createdAt, id, nil
}

func EncodeCommentCursor(path string) string {
	return encode(path)
}

func DecodeCommentCursor(s string) (string, error) {
	return decode(s)
}

func encode(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func decode(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("%w: not base64", ErrInvalidCursor)
	}
	return string(b), nil
}

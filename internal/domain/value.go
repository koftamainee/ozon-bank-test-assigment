package domain

import (
	"strings"
	"unicode/utf8"
)

const (
	MaxUsernameLength = 32
	MaxCommentLength  = 2000
)

type Username string

func NewUsername(s string) (Username, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > MaxUsernameLength {
		return "", ErrInvalidUsername
	}
	return Username(s), nil
}

func (u Username) String() string {
	return string(u)
}

type CommentBody string

func NewCommentBody(s string) (CommentBody, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > MaxCommentLength {
		return "", ErrInvalidCommentBody
	}
	return CommentBody(s), nil
}

func (b CommentBody) String() string {
	return string(b)
}

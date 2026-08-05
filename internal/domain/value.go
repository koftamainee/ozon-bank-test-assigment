package domain

import (
	"strings"
	"unicode/utf8"
)

const (
	MaxUsernameLength  = 32
	MaxCommentLength   = 2000
	MaxPostTitleLength = 300
	MaxPostBodyLength  = 40000
)

type Username string

func NewUsername(s string) (Username, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > MaxUsernameLength {
		return "", ErrInvalidUsername
	}
	for _, r := range s {
		if !isValidUsernameRune(r) {
			return "", ErrInvalidUsername
		}
	}
	return Username(s), nil
}

func isValidUsernameRune(r rune) bool {
	if r == '_' || r == '-' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
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

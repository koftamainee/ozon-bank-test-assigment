package domain

import "errors"

var (
	ErrInvalidID          = errors.New("id must be positive")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidCommentBody = errors.New("comment body must be 1-2000 characters")
	ErrInvalidPostTitle   = errors.New("post title must be 1-300 characters")
	ErrInvalidPostBody    = errors.New("post body must be 1-40000 characters")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrUserNotFound       = errors.New("user not found")
	ErrPostNotFound       = errors.New("post not found")
	ErrCommentNotFound    = errors.New("comment not found")
	ErrCommentsDisabled   = errors.New("comments are disabled for this post")
	ErrParentNotInPost    = errors.New("parent comment does not belong to this post")
	ErrParentDeleted      = errors.New("parent comment has been deleted")
)

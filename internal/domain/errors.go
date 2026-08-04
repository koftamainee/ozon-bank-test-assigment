package domain

import "errors"

var (
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidCommentBody = errors.New("comment body must be 1-2000 characters")
	ErrPostNotFound       = errors.New("post not found")
	ErrCommentNotFound    = errors.New("comment not found")
	ErrCommentsDisabled   = errors.New("comments are disabled for this post")
	ErrParentNotInPost    = errors.New("parent comment does not belong to this post")
	ErrParentDeleted      = errors.New("parent comment has been deleted")
)

package graphql

import (
	"context"
	"errors"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/http/middleware"
	"github.com/koftamainee/ozon-bank-test-assigment/internal/store"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func gqlError(ctx context.Context, err error) *gqlerror.Error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return withCode(err, "UNAUTHORIZED")
	case errors.Is(err, domain.ErrForbidden):
		return withCode(err, "FORBIDDEN")
	case errors.Is(err, domain.ErrInvalidID),
		errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrInvalidCommentBody),
		errors.Is(err, domain.ErrInvalidPostTitle),
		errors.Is(err, domain.ErrInvalidPostBody),
		errors.Is(err, store.ErrInvalidCursor):
		return withCode(err, "VALIDATION_ERROR")
	case errors.Is(err, domain.ErrUserNotFound):
		return withCode(err, "USER_NOT_FOUND")
	case errors.Is(err, domain.ErrPostNotFound),
		errors.Is(err, domain.ErrCommentNotFound):
		return withCode(err, "NOT_FOUND")
	case errors.Is(err, domain.ErrCommentsDisabled):
		return withCode(err, "COMMENTS_DISABLED")
	case errors.Is(err, domain.ErrParentNotInPost):
		return withCode(err, "PARENT_NOT_IN_POST")
	case errors.Is(err, domain.ErrParentDeleted):
		return withCode(err, "PARENT_DELETED")
	default:
		logger := middleware.LoggerFromContext(ctx)
		if reqID := middleware.RequestIDFromContext(ctx); reqID != "" {
			logger = logger.With("request_id", reqID)
		}
		logger.Error("unexpected graphql error", "err", err)
		return &gqlerror.Error{
			Message:    "internal server error",
			Extensions: map[string]any{"code": "INTERNAL"},
		}
	}
}

func withCode(err error, code string) *gqlerror.Error {
	return &gqlerror.Error{
		Message:    err.Error(),
		Extensions: map[string]any{"code": code},
	}
}

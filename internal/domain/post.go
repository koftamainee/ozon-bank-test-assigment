package domain

import "time"

type Post struct {
	ID              int64
	AuthorID        int64
	Title           string
	Body            string
	CommentsAllowed bool
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (p Post) IsDeleted() bool {
	return p.DeletedAt != nil
}

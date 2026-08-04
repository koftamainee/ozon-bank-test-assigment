package domain

import "time"

type Comment struct {
	ID        int64
	PostID    int64
	AuthorID  int64
	ParentID  *int64
	Path      string
	Depth     int
	Body      string
	DeletedAt *time.Time
	CreatedAt time.Time
}

func (c Comment) IsDeleted() bool {
	return c.DeletedAt != nil
}

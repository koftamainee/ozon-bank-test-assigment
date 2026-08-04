package domain

import "time"

type User struct {
	ID        int64
	Username  Username
	CreatedAt time.Time
}

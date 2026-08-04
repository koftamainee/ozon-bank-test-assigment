package pg

import "time"

type Config struct {
	DSN             string
	MaxOpenConns    int32         `default:"10"`
	MaxIdleConns    int32         `default:"5"`
	ConnMaxLifetime time.Duration `default:"30m"`
}

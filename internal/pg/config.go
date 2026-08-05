package pg

import "time"

type Config struct {
	DSN             string
	MaxOpenConns    int32         `default:"10"`
	MaxIdleConns    int32         `default:"5"`
	ConnMaxLifetime time.Duration `default:"30m"`
}

func (c *Config) applyDefaults() {
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 10
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 5
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 30 * time.Minute
	}
}

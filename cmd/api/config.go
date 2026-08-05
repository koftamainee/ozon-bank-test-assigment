package main

import "time"

type Config struct {
	Env             string        `default:"dev"`
	Port            int           `default:"8080"`
	StorageType     string        `default:"memory"`
	DatabaseURL     string        `optional:"true"`
	JWTSecret       string        `optional:"true"`
	JWTTTL          time.Duration `default:"24h"`
	JWTCookieSecure bool          `default:"false"`
}

func (c *Config) GetEnv() string { return c.Env }

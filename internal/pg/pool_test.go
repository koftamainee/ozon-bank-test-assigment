package pg

import (
	"context"
	"testing"
	"time"
)

func TestNewWithInvalidDSN(t *testing.T) {
	_, err := New(context.Background(), Config{DSN: "::not-a-dsn::"})
	if err == nil {
		t.Fatal("New() with invalid DSN: expected error, got nil")
	}
}

func TestNewAppliesPoolConfig(t *testing.T) {
	pool, err := New(context.Background(), Config{
		DSN:             "postgres://user:pass@localhost:5432/db",
		MaxOpenConns:    7,
		MaxIdleConns:    3,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer pool.Close()

	cfg := pool.Config()
	if cfg.MaxConns != 7 {
		t.Errorf("MaxConns = %d, want 7", cfg.MaxConns)
	}
	if cfg.MinConns != 3 {
		t.Errorf("MinConns = %d, want 3", cfg.MinConns)
	}
	if cfg.MaxConnLifetime != time.Minute {
		t.Errorf("MaxConnLifetime = %v, want 1m", cfg.MaxConnLifetime)
	}
}

func TestNewAppliesDefaults(t *testing.T) {
	pool, err := New(context.Background(), Config{DSN: "postgres://user:pass@localhost:5432/db"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer pool.Close()

	cfg := pool.Config()
	if cfg.MaxConns != 10 {
		t.Errorf("MaxConns = %d, want default 10", cfg.MaxConns)
	}
	if cfg.MinConns != 5 {
		t.Errorf("MinConns = %d, want default 5", cfg.MinConns)
	}
	if cfg.MaxConnLifetime != 30*time.Minute {
		t.Errorf("MaxConnLifetime = %v, want default 30m", cfg.MaxConnLifetime)
	}
}

func TestNewClampsMinConns(t *testing.T) {
	pool, err := New(context.Background(), Config{
		DSN:          "postgres://user:pass@localhost:5432/db",
		MaxOpenConns: 3,
		MaxIdleConns: 9,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer pool.Close()

	cfg := pool.Config()
	if cfg.MinConns > cfg.MaxConns {
		t.Fatalf("MinConns = %d > MaxConns = %d, want clamped", cfg.MinConns, cfg.MaxConns)
	}
	if cfg.MinConns != 3 {
		t.Errorf("MinConns = %d, want clamped to 3", cfg.MinConns)
	}
}

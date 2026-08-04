package log

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestTerminalHandlerEnabled(t *testing.T) {
	h := newTerminalHandler(&bytes.Buffer{}, slog.LevelWarn, false)

	cases := []struct {
		level slog.Level
		want  bool
	}{
		{slog.LevelDebug, false},
		{slog.LevelInfo, false},
		{slog.LevelWarn, true},
		{slog.LevelError, true},
	}

	for _, c := range cases {
		if got := h.Enabled(context.Background(), c.level); got != c.want {
			t.Errorf("Enabled(%v) = %v, want %v", c.level, got, c.want)
		}
	}
}

func TestTerminalHandlerPlain(t *testing.T) {
	var buf bytes.Buffer
	h := newTerminalHandler(&buf, slog.LevelInfo, false)

	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "hello", 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "INFO hello" {
		t.Errorf("output = %q, want %q", got, "INFO hello")
	}
}

func TestTerminalHandlerColor(t *testing.T) {
	var buf bytes.Buffer
	h := newTerminalHandler(&buf, slog.LevelInfo, true)

	r := slog.NewRecord(time.Time{}, slog.LevelWarn, "careful", 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "\033[33m") || !strings.Contains(out, "\033[0m") {
		t.Errorf("expected color escape codes in output, got %q", out)
	}
}

func TestTerminalHandlerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	var h slog.Handler = newTerminalHandler(&buf, slog.LevelInfo, false)
	h = h.WithAttrs([]slog.Attr{slog.String("key", "value")})

	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "key=value") {
		t.Errorf("expected key=value in output, got %q", out)
	}
}

func TestTerminalHandlerWithAttrsImmutability(t *testing.T) {
	var buf bytes.Buffer
	h := newTerminalHandler(&buf, slog.LevelInfo, false)

	h.WithAttrs([]slog.Attr{slog.String("a", "1")})

	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if strings.Contains(buf.String(), "a=1") {
		t.Errorf("original handler was mutated, output = %q", buf.String())
	}
}

func TestTerminalHandlerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	var h slog.Handler = newTerminalHandler(&buf, slog.LevelInfo, false)
	h = h.WithGroup("db")

	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("host", "h"))

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "host=h") {
		t.Errorf("expected host=h in output, got %q", out)
	}
}

func TestTerminalHandlerHandleError(t *testing.T) {
	h := newTerminalHandler(errWriter{}, slog.LevelInfo, false)
	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0)
	if err := h.Handle(context.Background(), r); err == nil {
		t.Fatal("Handle() = nil, want writer error")
	}
}

type errWriter struct{}

func (errWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

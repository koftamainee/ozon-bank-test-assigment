package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

var levelColors = map[slog.Level]string{
	slog.LevelDebug: "\033[36m",
	slog.LevelInfo:  "\033[32m",
	slog.LevelWarn:  "\033[33m",
	slog.LevelError: "\033[31m",
}

const reset = "\033[0m"

type terminalHandler struct {
	out    io.Writer
	level  slog.Level
	color  bool
	attrs  []slog.Attr
	groups []string
}

func newTerminalHandler(out io.Writer, level slog.Level, color bool) *terminalHandler {
	return &terminalHandler{
		out:   out,
		level: level,
		color: color,
	}
}

func (h *terminalHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

func (h *terminalHandler) Handle(_ context.Context, r slog.Record) error {
	level := r.Level.String()
	msg := r.Message

	if h.color {
		if c, ok := levelColors[r.Level]; ok {
			level = c + level + reset
		}
	}

	parts := make([]string, 0, 4)
	if !r.Time.IsZero() {
		parts = append(parts, r.Time.Format(time.RFC3339Nano))
	}
	parts = append(parts, level, msg)
	if len(h.attrs) > 0 || r.NumAttrs() > 0 {
		parts = append(parts, strings.TrimSpace(h.formatAttrs(r)))
	}

	_, err := fmt.Fprintln(h.out, strings.Join(parts, " "))
	return err
}

func (h *terminalHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(clone.attrs, h.attrs)
	copy(clone.attrs[len(h.attrs):], attrs)
	return &clone
}

func (h *terminalHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.groups = make([]string, len(h.groups)+1)
	copy(clone.groups, h.groups)
	clone.groups[len(h.groups)] = name
	return &clone
}

func (h *terminalHandler) formatAttrs(r slog.Record) string {
	prefix := strings.Join(h.groups, ".")
	var buf []byte
	r.Attrs(func(a slog.Attr) bool {
		appendAttr(&buf, prefix, a)
		return true
	})
	for _, a := range h.attrs {
		appendAttr(&buf, prefix, a)
	}
	return string(buf)
}

func appendAttr(buf *[]byte, prefix string, a slog.Attr) {
	if a.Equal(slog.Attr{}) {
		return
	}
	key := a.Key
	if prefix != "" {
		key = prefix + "." + key
	}
	if a.Value.Kind() == slog.KindGroup {
		inner := key
		if prefix == "" {
			inner = a.Key
		}
		for _, child := range a.Value.Group() {
			appendAttr(buf, inner, child)
		}
		return
	}
	*buf = append(*buf, fmt.Sprintf("%s=%v ", key, a.Value.Any())...)
}

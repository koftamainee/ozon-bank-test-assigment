package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

var levelColors = map[slog.Level]string{
	slog.LevelDebug: "\033[36m", // cyan
	slog.LevelInfo:  "\033[32m", // green
	slog.LevelWarn:  "\033[33m", // yellow
	slog.LevelError: "\033[31m", // red
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

	var line string
	if len(h.attrs) > 0 || r.NumAttrs() > 0 {
		line = fmt.Sprintf("%s %s %s", level, msg, h.formatAttrs(r))
	} else {
		line = fmt.Sprintf("%s %s", level, msg)
	}

	_, err := fmt.Fprintln(h.out, line)
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
	var buf []byte
	r.Attrs(func(a slog.Attr) bool {
		buf = append(buf, fmt.Sprintf("%s=%v ", a.Key, a.Value)...)
		return true
	})
	for _, a := range h.attrs {
		buf = append(buf, fmt.Sprintf("%s=%v ", a.Key, a.Value)...)
	}
	return string(buf)
}

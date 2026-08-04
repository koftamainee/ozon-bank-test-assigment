package log

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func withStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old
	b, _ := io.ReadAll(r)
	return string(b)
}

func withDefaultLogger(fn func()) {
	old := slog.Default()
	defer slog.SetDefault(old)
	fn()
}

func TestInitJSON(t *testing.T) {
	withDefaultLogger(func() {
		out := withStdout(func() {
			Init(Options{Level: LevelInfo, JSON: true})
			slog.Info("hello", "key", "value")
		})

		if !strings.Contains(out, `"msg":"hello"`) {
			t.Errorf("expected JSON msg hello, got %q", out)
		}
		if !strings.Contains(out, `"key":"value"`) {
			t.Errorf("expected attribute key=value, got %q", out)
		}
		if !strings.Contains(out, `"level":"INFO"`) {
			t.Errorf("expected INFO level, got %q", out)
		}
	})
}

func TestInitLevelFiltering(t *testing.T) {
	withDefaultLogger(func() {
		out := withStdout(func() {
			Init(Options{Level: LevelError, JSON: true})
			slog.Info("should not appear")
			slog.Error("boom")
		})

		if strings.Contains(out, "should not appear") {
			t.Errorf("Info log must be filtered, got %q", out)
		}
		if !strings.Contains(out, "boom") {
			t.Errorf("expected Error log, got %q", out)
		}
	})
}

func TestInitTerminal(t *testing.T) {
	withDefaultLogger(func() {
		out := withStdout(func() {
			Init(Options{Level: LevelInfo, Color: true})
			slog.Info("plain text")
		})

		if !strings.Contains(out, "INFO") || !strings.Contains(out, "plain text") {
			t.Errorf("expected terminal output, got %q", out)
		}
		if strings.Contains(out, `"msg"`) {
			t.Errorf("expected non-JSON output, got %q", out)
		}
	})
}

func TestInitColorOff(t *testing.T) {
	withDefaultLogger(func() {
		out := withStdout(func() {
			Init(Options{Level: LevelInfo, Color: false})
			slog.Info("no color")
		})

		if strings.Contains(out, "\033[") {
			t.Errorf("expected no color codes, got %q", out)
		}
	})
}

func TestPackageHelpers(t *testing.T) {
	withDefaultLogger(func() {
		var buf strings.Builder
		slog.SetDefault(slog.New(newTerminalHandler(&stringWriter{&buf}, slog.LevelDebug, false)))

		Debug("dbg", "a", 1)
		Info("inf", "b", 2)
		Warn("wrn")
		Error("err")

		out := buf.String()
		for _, want := range []string{"DEBUG", "INFO", "WARN", "ERROR", "a=1", "b=2"} {
			if !strings.Contains(out, want) {
				t.Errorf("expected %q in output, got %q", want, out)
			}
		}
	})
}

func TestWith(t *testing.T) {
	withDefaultLogger(func() {
		l := With("k", "v")
		if l == slog.Default() {
			t.Error("With() returned the default logger")
		}
	})
}

type stringWriter struct {
	sb *strings.Builder
}

func (w *stringWriter) Write(p []byte) (int, error) {
	return w.sb.Write(p)
}

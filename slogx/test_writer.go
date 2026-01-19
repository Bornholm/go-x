package slogx

import (
	"log/slog"
	"testing"
)

type TestWriter struct {
	t testing.TB
}

func (w *TestWriter) Write(p []byte) (n int, err error) {
	if len(p) > 0 && p[len(p)-1] == '\n' {
		p = p[:len(p)-1]
	}

	w.t.Logf("%s", p)

	return len(p), nil
}

func NewTestLogger(t testing.TB) *slog.Logger {
	t.Helper()

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}

	handler := slog.NewTextHandler(&TestWriter{t: t}, opts)

	return slog.New(handler)
}

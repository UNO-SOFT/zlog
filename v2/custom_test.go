package zlog

import (
	"testing"

	"golang.org/x/exp/slog"
)

func TestCustomSource(t *testing.T) {
	opts := DefaultHandlerOptions
	opts.AddSource = true
	logger := slog.New(&customSourceHandler{Handler: opts.NewJSONHandler(testWriter{T: t})})
	logger.Debug("Debug")
	logger.Info("no attrs")
}

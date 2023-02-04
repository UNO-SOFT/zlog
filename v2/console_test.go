package zlog_test

import (
	"errors"
	"os"
	"testing"

	"github.com/UNO-SOFT/zlog/v2"
	"golang.org/x/exp/slog"
)

func TestConsole(t *testing.T) {
	logHandler := zlog.NewConsoleHandler(zlog.InfoLevel, os.Stderr)

	logger := slog.New(logHandler)
	logger.Debug("Debug message", "hello", "world", "bad kv")
	logger = logger.
		With("with_key_1", "with_value_1").
		WithGroup("group_1").
		With("with_key_2", "with_value_2")
	logger.Info("Info message", "hello", "world")
	logger.Warn("Warn message", "hello", "world")
	logger.Error("Error message", errors.New("an error"), "hello", "world")
}

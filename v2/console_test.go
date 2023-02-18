package zlog_test

import (
	"errors"
	"testing"

	"github.com/UNO-SOFT/zlog/v2"
	"golang.org/x/exp/slog"
)

func TestConsole(t *testing.T) {
	logHandler := zlog.NewT(t).SLog().Handler()

	logger := slog.New(logHandler)
	logger.Debug("Debug message", "hello", "world", "bad kv")
	logger.Info("no attrs")
	logger = logger.
		With("with_key_1", "with_value_1").
		WithGroup("group_1").
		With("with_key_2", "with_value_2")
	logger.Info("Info message", "hello", "world")
	logger.Warn("Warn message", "hello", "world")
	logger.Error("Error message", errors.New("an error"), "hello", "world")
}

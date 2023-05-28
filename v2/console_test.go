package zlog_test

import (
	"errors"
	"testing"

	"github.com/UNO-SOFT/zlog/v2"
)

func TestConsole(t *testing.T) {
	logger := zlog.NewT(t).SLog()

	logger.Debug("Debug message", "hello", "world", "bad kv")
	logger.Info("no attrs")
	logger = logger.
		With("with_key_1", "with_value_1").
		WithGroup("group_1").
		With("with_key_2", "with_value_2")
	logger.Info("Info message", "hello", "world")
	logger.Warn("Warn message", "hello", "world")
	logger.Error("Error message", "error", errors.New("an error"), "hello", "world")
}

func TestConsoleWithEmptyAttrs(t *testing.T) {
	logger := zlog.NewT(t).SLog() //.With("", "", "", "", "", "")
	logger.Info("two empty attrs, but nothing else", "", "", "", "")
	logger.Info("three empty attrs, plus one", "", "", "", "", "", "", "one", 1)
}

// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package zlog contains some very simple go-logr / zerologr helper functions.
// This sets the default timestamp format to time.RFC3339 with ms precision.
package zlog

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slog"
)

// Logger is a helper type for logr.Logger -like slog.Logger.
type Logger struct{ p atomic.Pointer[slog.Logger] }

func (lgr Logger) load() *slog.Logger {
	if l := lgr.p.Load(); l != nil {
		return l
	}
	if l := slog.Default(); l != nil {
		return l
	}
	return discard()
}

// Discard returns a Logger that does not log at all.
func Discard() Logger {
	var lgr Logger
	lgr.p.Store(discard())
	return lgr
}

func discard() *slog.Logger {
	return slog.New(slog.HandlerOptions{
		Level: slog.LevelError,
	}.NewJSONHandler(io.Discard))
}

type contextKey struct{}

// NewContext returns a new context with the given logger embedded.
func NewContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext returns the Logger embedded into the Context, or the default logger otherwise.
func FromContext(ctx context.Context) Logger {
	if lgr, ok := ctx.Value(contextKey{}).(Logger); ok {
		return lgr
	}
	var lgr Logger
	lgr.p.Store(slog.Default())
	return lgr
}

// Log emulates go-kit/log.
func (lgr Logger) Log(keyvals ...interface{}) error {
	if !lgr.load().Enabled(context.Background(), slog.LevelInfo) {
		return nil
	}
	var msg string
	for i := 0; i < len(keyvals)-1; i++ {
		if keyvals[i] == "msg" {
			var ok bool
			if msg, ok = keyvals[i+1].(string); !ok {
				msg = fmt.Sprintf("%v", keyvals[i+1])
			}
			keyvals[i], keyvals[i+1] = keyvals[0], keyvals[1]
			keyvals = keyvals[2:]
			break
		}
	}
	lgr.Info(msg, keyvals...)
	return nil
}

func (lgr Logger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	if l := lgr.load(); l.Enabled(ctx, level) {
		l.Log(ctx, level, msg, args...)
	}
}

// Debug calls Debug if enabled.
func (lgr Logger) Debug(msg string, args ...any) {
	lgr.log(context.Background(), slog.LevelDebug, msg, args...)
}

// DebugCtx calls DebugCtx if enabled.
func (lgr Logger) DebugCtx(ctx context.Context, msg string, args ...any) {
	lgr.log(ctx, slog.LevelDebug, msg, args...)
}

// Info calls Info if enabled.
func (lgr Logger) Info(msg string, args ...any) {
	lgr.log(context.Background(), slog.LevelInfo, msg, args...)
}

// InfoCtx calls InfoCtx if enabled.
func (lgr Logger) InfoCtx(ctx context.Context, msg string, args ...any) {
	lgr.log(ctx, slog.LevelInfo, msg, args...)
}

// Warn calls Warn if enabled.
func (lgr Logger) Warn(msg string, args ...any) {
	lgr.log(context.Background(), slog.LevelWarn, msg, args...)
}

// WarnCtx calls WarCtx if enabled.
func (lgr Logger) WarnCtx(ctx context.Context, msg string, args ...any) {
	lgr.log(ctx, slog.LevelWarn, msg, args...)
}

// Error calls Error with ErrorLevel, always.
func (lgr Logger) Error(err error, msg string, args ...any) {
	lgr.load().Error(msg, err, args...)
}

// ErrorCtx calls Error with ErrorLevel, always.
func (lgr Logger) ErrorCtx(ctx context.Context, err error, msg string, args ...any) {
	lgr.load().ErrorCtx(ctx, msg, err, args...)
}

// V offsets the logging levels by off (emulates logr.Logger.V).
func (lgr Logger) V(off int) Logger {
	if off == 0 {
		return lgr
	}
	h := lgr.load().Handler()
	level := slog.LevelInfo
	if lh, ok := h.(*LevelHandler); ok {
		level = lh.level.Level()
	}
	var lgr2 Logger
	lgr2.p.Store(slog.New(&LevelHandler{level: level - slog.Level(off), handler: h}))
	return lgr2
}

// WithValues emulates logr.Logger.WithValues with slog.WithAttrs.
func (lgr Logger) WithValues(args ...any) Logger {
	var lgr2 Logger
	lgr2.p.Store(lgr.load().With(args...))
	return lgr2
}

// SetLevel on the underlying LevelHandler.
func (lgr Logger) SetLevel(level slog.Leveler) {
	if lh, ok := lgr.load().Handler().(*LevelHandler); ok {
		lh.SetLevel(level)
	} else {
		lgr.p.Store(slog.New(&LevelHandler{level: level, handler: lgr.load().Handler()}))
	}
}

// WithName implements logr.WithName with slog.WithGroup
func (lgr Logger) WithName(s string) Logger { return lgr.WithGroup(s) }

// WithGroup is slog.WithGroup
func (lgr Logger) WithGroup(s string) Logger {
	var lgr2 Logger
	lgr2.p.Store(lgr.load().WithGroup(s))
	return lgr2
}

// SetOutput sets the output to a new Logger.
func (lgr Logger) SetOutput(w io.Writer) { lgr.p.Store(New(w).load()) }

// SetHandler sets the Handler.
func (lgr Logger) SetHandler(h slog.Handler) { lgr.p.Store(slog.New(h)) }

// SLog returns the underlying slog.Logger
func (lgr Logger) SLog() *slog.Logger { return lgr.load() }

// Logr returns a go-logr/logr.Logger, using this Logger as LogSink
func (lgr Logger) Logr() logr.Logger { return logr.New(SLogSink{lgr.SLog()}) }

// SLogSink is an logr.LogSink for an slog.Logger.
type SLogSink struct{ *slog.Logger }

// Init receives optional information about the logr library for LogSink
// implementations that need it.
func (ls SLogSink) Init(info logr.RuntimeInfo) {}

// Enabled tests whether this LogSink is enabled at the specified V-level.
// For example, commandline flags might be used to set the logging
// verbosity and disable some info logs.
func (ls SLogSink) Enabled(level int) bool {
	return ls.Logger.Enabled(context.Background(), LogrLevel(level).Level())
}

// Info logs a non-error message with the given key/value pairs as context.
// The level argument is provided for optional logging.  This method will
// only be called when Enabled(level) is true. See Logger.Info for more
// details.
func (ls SLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	ls.Logger.Log(context.Background(), slog.LevelInfo, msg, keysAndValues...)
}

// Error logs an error, with the given message and key/value pairs as
// context.  See Logger.Error for more details.
func (ls SLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	ls.Logger.Log(context.Background(), slog.LevelError, msg, append(keysAndValues, slog.Any(slog.ErrorKey, err))...)
}

// WithValues returns a new LogSink with additional key/value pairs.  See
// Logger.WithValues for more details.
func (ls SLogSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return SLogSink{ls.Logger.With(keysAndValues...)}
}

// WithName returns a new LogSink with the specified name appended.  See
// Logger.WithName for more details.
func (ls SLogSink) WithName(name string) logr.LogSink { return SLogSink{ls.Logger.WithGroup(name)} }

var _ logr.LogSink = SLogSink{}

// SetLevel sets the level on the given Logger.
func SetLevel(lgr Logger, level slog.Leveler) { lgr.SetLevel(level) }

// SetOutput sets the output on the given Logger.
func SetOutput(lgr Logger, w io.Writer) { lgr.SetOutput(w) }

// SetHandler sets the handler on the given Logger.
func SetHandler(lgr Logger, h slog.Handler) { lgr.SetHandler(h) }

// NewLogger returns a new Logger writing to w.
func NewLogger(h slog.Handler) Logger {
	var lgr Logger
	lgr.p.Store(slog.New(h))
	return lgr
}

// New returns a new logr.Logger writing to w as a zerolog.Logger, at LevelInfo.
func New(w io.Writer) Logger {
	return NewLogger(NewLevelHandler(
		&slog.LevelVar{},
		MaybeConsoleHandler(InfoLevel, w),
	))
}

var _ slog.Leveler = (*VerboseVar)(nil)
var _ flag.Value = (*VerboseVar)(nil)

type VerboseVar bool

func (vv *VerboseVar) Level() slog.Level {
	if vv != nil && *vv {
		return slog.LevelInfo
	}
	return slog.LevelWarn
}

func (vv *VerboseVar) IsBoolFlag() bool { return true }
func (vv *VerboseVar) String() string {
	if vv != nil && *vv {
		return "true"
	}
	return "false"
}
func (vv *VerboseVar) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	*vv = VerboseVar(b)
	return nil
}

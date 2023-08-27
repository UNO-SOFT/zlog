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
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/UNO-SOFT/zlog/v2/slog"
	"github.com/go-logr/logr"
)

// Logger is a helper type for logr.Logger -like slog.Logger.
type Logger struct{ p *atomic.Pointer[slog.Logger] }

func newLogger() Logger { return Logger{p: &atomic.Pointer[slog.Logger]{}} }

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
	lgr := newLogger()
	lgr.p.Store(discard())
	return lgr
}

func discard() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

type contextKey struct{}

// NewContext returns a new context with the given logger embedded.
func NewContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// NewSContext returns a new context with the given logger embedded.
func NewSContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext returns the Logger embedded into the Context, or the default logger otherwise.
func FromContext(ctx context.Context) Logger {
	val := ctx.Value(contextKey{})
	switch lgr := val.(type) {
	case Logger:
		return lgr
	case *slog.Logger:
		return NewLogger(lgr.Handler())
	}
	lgr := newLogger()
	lgr.p.Store(slog.Default())
	return lgr
}

// SFromContext returns the Logger embedded into the Context, or the default logger otherwise.
func SFromContext(ctx context.Context) *slog.Logger {
	val := ctx.Value(contextKey{})
	switch lgr := val.(type) {
	case *slog.Logger:
		return lgr
	case Logger:
		return lgr.SLog()
	}
	return slog.Default()
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
	l := lgr.load()
	if !l.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	if ctx == nil {
		ctx = context.Background()
	}
	_ = l.Handler().Handle(ctx, r)
}

// Debug calls Debug if enabled.
func (lgr Logger) Debug(msg string, args ...any) {
	lgr.log(context.Background(), slog.LevelDebug, msg, args...)
}

// DebugContext calls DebugContext if enabled.
func (lgr Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	lgr.log(ctx, slog.LevelDebug, msg, args...)
}

// Info calls Info if enabled.
func (lgr Logger) Info(msg string, args ...any) {
	lgr.log(context.Background(), slog.LevelInfo, msg, args...)
}

// InfoContext calls InfoContext if enabled.
func (lgr Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	lgr.log(ctx, slog.LevelInfo, msg, args...)
}

// Warn calls Warn if enabled.
func (lgr Logger) Warn(msg string, args ...any) {
	lgr.log(context.Background(), slog.LevelWarn, msg, args...)
}

// WarnContext calls WarContext if enabled.
func (lgr Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	lgr.log(ctx, slog.LevelWarn, msg, args...)
}

// Error calls Error with ErrorLevel, always.
func (lgr Logger) Error(err error, msg string, args ...any) {
	lgr.load().Error(msg, append(args, slog.String("error", err.Error()))...)
}

// ErrorContext calls Error with ErrorLevel, always.
func (lgr Logger) ErrorContext(ctx context.Context, err error, msg string, args ...any) {
	lgr.load().ErrorContext(ctx, msg, append(args, slog.String("error", err.Error()))...)
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
	lgr2 := newLogger()
	lgr2.p.Store(slog.New(&LevelHandler{level: level - slog.Level(off), handler: h}))
	return lgr2
}

// WithValues emulates logr.Logger.WithValues with slog.WithAttrs.
func (lgr Logger) WithValues(args ...any) Logger {
	lgr2 := newLogger()
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
	lgr2 := newLogger()
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
	ls.Logger.Info(msg, keysAndValues...)
}

// Error logs an error, with the given message and key/value pairs as
// context.  See Logger.Error for more details.
func (ls SLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	ls.Logger.Error(msg, err, keysAndValues)
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
	lgr := Logger{p: &atomic.Pointer[slog.Logger]{}}
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

type VerboseVar uint8

func (vv *VerboseVar) Level() slog.Level {
	if vv != nil {
		if *vv > 1 {
			return slog.LevelDebug
		} else if *vv > 0 {
			return slog.LevelInfo
		}
	}
	return slog.LevelWarn
}

func (vv *VerboseVar) IsBoolFlag() bool { return true }
func (vv *VerboseVar) String() string {
	if vv != nil {
		return strconv.FormatUint(uint64(*vv), 10)
	}
	return "0"
}
func (vv *VerboseVar) Set(s string) error {
	switch s {
	case "true", "":
		*vv = 1
	case "false":
		*vv = 0
	default:
		b, err := strconv.ParseUint(s, 10, 8)
		if err != nil {
			return err
		}
		*vv = VerboseVar(b)
	}
	return nil
}

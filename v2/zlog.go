// Copyright 2022 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package zlog contains some very simple go-logr / zerologr helper functions.
// This sets the default timestamp format to time.RFC3339 with ms precision.
package zlog

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slog"
	"golang.org/x/term"
)

var _ slog.Leveler = LogrLevel(0)

// LogrLevel is an slog.Leveler that converts from github.com/go-logr/logr levels to slog levels.
type LogrLevel int

// Level returns the slog.Level, converted from the logr level.
func (l LogrLevel) Level() slog.Level { return slog.LevelInfo }

/*
	DebugLevel Level = -4
	LevelInfo  Level = 0
	WarnLevel  Level = 4
	ErrorLevel Level = 8
*/
const (
	TraceLevel = LogrLevel(1)
	LevelInfo  = LogrLevel(0)
	ErrorLevel = LogrLevel(-1)
)

var _ = slog.Handler((*MultiHandler)(nil))

// MultiHandler writes to all the specified handlers.
//
// goroutine-safe.
type MultiHandler struct{ ws atomic.Value }

// NewMultiHandler returns a new slog.Handler that writes to all the specified Handlers.
func NewMultiHandler(hs ...slog.Handler) *MultiHandler {
	lw := MultiHandler{}
	lw.ws.Store(hs)
	return &lw
}

// Add an additional writer to the targets.
func (lw *MultiHandler) Add(w slog.Handler) { lw.ws.Store(append(lw.ws.Load().([]slog.Handler), w)) }

// Swap the current writers with the defined.
func (lw *MultiHandler) Swap(ws ...slog.Handler) { lw.ws.Store(ws) }

// Handle the record.
func (lw *MultiHandler) Handle(r slog.Record) error {
	var firstErr error
	for _, h := range lw.ws.Load().([]slog.Handler) {
		if err := h.Handle(r); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// WithAttrs returns a new slog.Handler with the given attrs set on all underlying handlers.
func (lw *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hs := append([]slog.Handler(nil), lw.ws.Load().([]slog.Handler)...)
	for i, h := range hs {
		hs[i] = h.WithAttrs(attrs)
	}
	return NewMultiHandler(hs...)
}

// WithGroup returns a new slog.Handler with the given group set on all underlying handlers.
func (lw *MultiHandler) WithGroup(name string) slog.Handler {
	hs := append([]slog.Handler(nil), lw.ws.Load().([]slog.Handler)...)
	for i, h := range hs {
		hs[i] = h.WithGroup(name)
	}
	return NewMultiHandler(hs...)
}

// Enabled reports whether any of the underlying handlers is enabled for the given level.
func (lw *MultiHandler) Enabled(level slog.Level) bool {
	for _, h := range lw.ws.Load().([]slog.Handler) {
		if h.Enabled(level) {
			return true
		}
	}
	return false
}

// Logger is a helper type for logr.Logger -like slog.Logger.
type Logger struct{ p atomic.Pointer[slog.Logger] }

func (lgr Logger) load() *slog.Logger {
	if l := lgr.p.Load(); l != nil {
		return l
	}
	if l := slog.Default(); l != nil {
		return l
	}
	return slog.New(slog.HandlerOptions{
		Level: slog.LevelError,
	}.NewJSONHandler(io.Discard))
}

type contextKey struct{}

// NewContext returns a new context with the given logger embedded.
func NewContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(slog.NewContext(ctx, logger.load()), contextKey{}, logger)
}

// FromContext returns the Logger embedded into the Context, or the default logger otherwise.
func FromContext(ctx context.Context) Logger {
	if lgr, ok := ctx.Value(contextKey{}).(Logger); ok {
		return lgr
	}
	var lgr Logger
	lgr.p.Store(slog.FromContext(ctx))
	return lgr
}

const callDepth = 0

// Log emulates go-kit/log.
func (lgr Logger) Log(keyvals ...interface{}) error {
	if !lgr.load().Enabled(slog.LevelInfo) {
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

// Info calls Info if enabled.
func (lgr Logger) Info(msg string, args ...any) {
	if l := lgr.load(); l.Enabled(slog.LevelInfo) {
		l.LogDepth(callDepth, slog.LevelInfo, msg, args...)
	}
}

// Error calls Info with ErrorLevel, always.
func (lgr Logger) Error(err error, msg string, args ...any) {
	lgr.load().LogDepth(callDepth, slog.LevelError, msg, append(args, slog.Any("error", err))...)
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
		lh.SetLevel(slog.Level(level.Level()))
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

// AsLogr returns a go-logr/logr.Logger, using this Logger as LogSink
func (lgr Logger) AsLogr() logr.Logger { return logr.New(SLogSink{lgr.SLog()}) }

type SLogSink struct{ *slog.Logger }

// Init receives optional information about the logr library for LogSink
// implementations that need it.
func (ls SLogSink) Init(info logr.RuntimeInfo) {}

// Enabled tests whether this LogSink is enabled at the specified V-level.
// For example, commandline flags might be used to set the logging
// verbosity and disable some info logs.
func (ls SLogSink) Enabled(level int) bool { return ls.Logger.Enabled(LogrLevel(level).Level()) }

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
	ls.Logger.Error(msg, err, keysAndValues...)
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
		MaybeConsoleHandler(w),
	))
}

// DefaultHandlerOptions adds the source.
var DefaultHandlerOptions = slog.HandlerOptions{AddSource: true}

// MaybeConsoleHandler returns an slog.JSONHandler if w is a terminal, and slog.TextHandler otherwise.
func MaybeConsoleHandler(w io.Writer) slog.Handler {
	if IsTerminal(w) {
		return NewConsoleHandler(w)
	}
	return DefaultHandlerOptions.NewJSONHandler(w)
}

// IsTerminal returns whether the io.Writer is a terminal or not.
func IsTerminal(w io.Writer) bool {
	if fder, ok := w.(interface{ Fd() uintptr }); ok {
		return term.IsTerminal(int(fder.Fd()))
	}
	return false
}

// A LevelHandler wraps a Handler with an Enabled method
// that returns false for levels below a minimum.
type LevelHandler struct {
	level   slog.Leveler
	handler slog.Handler
}

// NewLevelHandler returns a LevelHandler with the given level.
// All methods except Enabled delegate to h.
func NewLevelHandler(level slog.Leveler, h slog.Handler) *LevelHandler {
	// Optimization: avoid chains of LevelHandlers.
	if lh, ok := h.(*LevelHandler); ok {
		h = lh.Handler()
	}
	return &LevelHandler{level, h}
}

// Enabled implements Handler.Enabled by reporting whether
// level is at least as large as h's level.
func (h *LevelHandler) Enabled(level slog.Level) bool {
	return level >= h.level.Level()
}

// SetLevel on the LevelHandler.
func (h *LevelHandler) SetLevel(level slog.Leveler) {
	if lv, ok := h.level.(interface{ Set(l slog.Level) }); ok {
		lv.Set(level.Level())
	}
}

// Handle implements Handler.Handle.
func (h *LevelHandler) Handle(r slog.Record) error {
	if h == nil || h.handler == nil {
		return nil
	}
	return h.handler.Handle(r)
}

// WithAttrs implements Handler.WithAttrs.
func (h *LevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewLevelHandler(h.level, h.handler.WithAttrs(attrs))
}

// WithGroup implements Handler.WithGroup.
func (h *LevelHandler) WithGroup(name string) slog.Handler {
	return NewLevelHandler(h.level, h.handler.WithGroup(name))
}

// Handler returns the Handler wrapped by h.
func (h *LevelHandler) Handler() slog.Handler { return h.handler }

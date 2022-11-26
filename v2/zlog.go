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

	"golang.org/x/exp/slog"
	"golang.org/x/term"
)

const (
	TraceLevel = 7 - slog.DebugLevel
	InfoLevel  = 7 - slog.InfoLevel
	ErrorLevel = 7 - slog.ErrorLevel
)

var _ = slog.Handler((*multiHandler)(nil))

type multiHandler struct{ ws atomic.Value }

// NewMultiHandler returns a new slog.Handler that writes to all the specified Handlers.
func NewMultiHandler(hs ...slog.Handler) *multiHandler {
	lw := multiHandler{}
	lw.ws.Store(hs)
	return &lw
}

// Add an additional writer to the targets.
func (lw *multiHandler) Add(w slog.Handler) { lw.ws.Store(append(lw.ws.Load().([]slog.Handler), w)) }

// Swap the current writers with the defined.
func (lw *multiHandler) Swap(ws ...slog.Handler) { lw.ws.Store(ws) }
func (lw *multiHandler) Handle(r slog.Record) error {
	var firstErr error
	for _, h := range lw.ws.Load().([]slog.Handler) {
		if err := h.Handle(r); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
func (lw *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hs := append([]slog.Handler(nil), lw.ws.Load().([]slog.Handler)...)
	for i, h := range hs {
		hs[i] = h.WithAttrs(attrs)
	}
	return NewMultiHandler(hs...)
}
func (lw *multiHandler) WithGroup(name string) slog.Handler {
	hs := append([]slog.Handler(nil), lw.ws.Load().([]slog.Handler)...)
	for i, h := range hs {
		hs[i] = h.WithGroup(name)
	}
	return NewMultiHandler(hs...)
}
func (lw *multiHandler) Enabled(level slog.Level) bool {
	for _, h := range lw.ws.Load().([]slog.Handler) {
		if h.Enabled(level) {
			return true
		}
	}
	return false
}

type Logger struct{ *slog.Logger }

type contextKey struct{}

func NewContext(ctx context.Context, logger Logger) context.Context {
	if logger.Logger == nil {
		return ctx
	}
	return context.WithValue(slog.NewContext(ctx, logger.Logger), contextKey{}, logger)
}
func FromContext(ctx context.Context) Logger {
	if lgr, ok := ctx.Value(contextKey{}).(Logger); ok {
		return lgr
	}
	return Logger{Logger: slog.FromContext(ctx)}
}

const callDepth = 0

func (lgr Logger) Log(keyvals ...interface{}) error {
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

func (lgr Logger) Info(msg string, args ...any) {
	if lgr.Logger != nil && lgr.Logger.Enabled(slog.InfoLevel) {
		lgr.Logger.LogDepth(callDepth, slog.ErrorLevel, msg, args...)
	}
}
func (lgr Logger) Error(err error, msg string, args ...any) {
	if lgr.Logger != nil {
		lgr.Logger.LogDepth(callDepth, slog.ErrorLevel, msg, args...)
	}
}
func (lgr Logger) V(off int) Logger {
	if lgr.Logger == nil {
		return lgr
	}
	h := lgr.Logger.Handler()
	level := slog.Level(7)
	if lh, ok := h.(*LevelHandler); ok {
		level = lh.level.Level()
	}
	return Logger{Logger: slog.New(&LevelHandler{level: level - slog.Level(off), handler: h})}
}
func (lgr Logger) WithValues(args ...any) Logger {
	if lgr.Logger == nil {
		return lgr
	}
	return Logger{Logger: lgr.Logger.With(args...)}
}
func (lgr Logger) SetLevel(level slog.Level) {
	if lh, ok := lgr.Logger.Handler().(*LevelHandler); ok {
		lh.SetLevel(slog.Level(level))
	}
}
func (lgr Logger) SetOutput(w io.Writer) {
	lgr.Logger = New(w).Logger
}
func (lgr Logger) SetHandler(h slog.Handler) {
	lgr.Logger = slog.New(h)
}

func SetLevel(lgr Logger, level slog.Level) { lgr.SetLevel(level) }
func SetOutput(lgr Logger, w io.Writer)     { lgr.SetOutput(w) }
func SetHandler(lgr Logger, h slog.Handler) { lgr.SetHandler(h) }

// NewZerolog returns a new zerolog.Logger writing to w.
func NewLogger(h slog.Handler) Logger { return Logger{Logger: slog.New(h)} }

// New returns a new logr.Logger writing to w as a zerolog.Logger,
// at InfoLevel.
func New(w io.Writer) Logger {
	return NewLogger(NewLevelHandler(&slog.LevelVar{}, MaybeConsoleHandler(w)))
}

var DefaultHandlerOptions = slog.HandlerOptions{
	AddSource: true,
}

// MaybeConsoleHandler returns an slog.JSONHandler if w is a terminal, and slog.TextHandler otherwise.
func MaybeConsoleHandler(w io.Writer) slog.Handler {
	if IsTerminal(w) {
		return DefaultHandlerOptions.NewTextHandler(w)
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
func (h *LevelHandler) SetLevel(level slog.Level) {
	if lv, ok := h.level.(interface{ Set(l slog.Level) }); ok {
		lv.Set(level)
	}
}

// Handle implements Handler.Handle.
func (h *LevelHandler) Handle(r slog.Record) error {
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
func (h *LevelHandler) Handler() slog.Handler {
	return h.handler
}

// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package zlog contains some very simple go-logr / zerologr helper functions.
// This sets the default timestamp format to time.RFC3339 with ms precision.
package zlog

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"golang.org/x/exp/slog"
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
func (lw *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range lw.ws.Load().([]slog.Handler) {
		if h.Enabled(ctx, level) {
			return true
		}
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
func (h *LevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
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

type testWriter struct {
	T interface {
		Log(...any)
		Logf(string, ...any)
	}
}

var _ = io.Writer(testWriter{})

// NewT return a new text writer for a testing.T
func NewT(t testing.TB) Logger { return NewLogger(slog.NewTextHandler(testWriter{T: t})) }

func (t testWriter) Write(p []byte) (int, error) {
	t.T.Log(string(p))
	return len(p), nil
}

// SyncWriter syncs each Write.
type SyncWriter struct {
	w  io.Writer
	mu sync.Mutex
}

var _ = io.Writer((*SyncWriter)(nil))

// NewSyncWriter returns an io.Writer that syncs each io.Write
func NewSyncWriter(w io.Writer) *SyncWriter { return &SyncWriter{w: w} }
func (sw *SyncWriter) Write(p []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

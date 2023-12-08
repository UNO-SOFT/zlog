// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog

import (
	"context"
	"sync/atomic"

	"github.com/UNO-SOFT/zlog/v2/slog"
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
func (lw *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range lw.ws.Load().([]slog.Handler) {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		if err := h.Handle(ctx, r); err != nil && firstErr == nil {
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

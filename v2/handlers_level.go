// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog

import (
	"context"

	"github.com/UNO-SOFT/zlog/v2/slog"
)

var _ = slog.Handler((*LevelHandler)(nil))

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
	} else {
		h.level = level.Level()
	}
}

func (h *LevelHandler) GetLevel() slog.Leveler { return h.level }

// Handle implements Handler.Handle.
func (h *LevelHandler) Handle(ctx context.Context, r slog.Record) error {
	if h == nil || h.handler == nil {
		return nil
	}
	return h.handler.Handle(ctx, r)
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

// Copyright 2024 Tamas Gulacsi. All rights reserved.

package slog

import "context"

func TransformLevel(f func(Level) Level, logger *Logger) *Logger {
	return New(levelTransformer{Handler: logger.Handler(), t: f})
}

type levelTransformer struct {
	Handler
	t func(Level) Level
}

func (h levelTransformer) Handle(ctx context.Context, rec Record) error {
	rec.Level = h.t(rec.Level)
	return h.Handler.Handle(ctx, rec)
}

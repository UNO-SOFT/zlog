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
	"testing"
	"time"

	"github.com/UNO-SOFT/zlog/v2/slog"
)

var _ slog.Leveler = LogrLevel(0)

// LogrLevel is an slog.Leveler that converts from github.com/go-logr/logr levels to slog levels.
type LogrLevel int

// Level returns the slog.Level, converted from the logr level.
func (l LogrLevel) Level() slog.Level { return -slog.Level(l << 1) }

/*
DebugLevel Level = -4
LevelInfo  Level = 0
WarnLevel  Level = 4
ErrorLevel Level = 8
*/
const (
	TraceLevel = slog.LevelDebug - 1
	DebugLevel = slog.LevelDebug
	InfoLevel  = slog.LevelInfo
	ErrorLevel = slog.LevelError
)

type testWriter struct {
	T interface {
		Log(...any)
		Logf(string, ...any)
	}
}

var _ = io.Writer(testWriter{})

// NewT return a new text writer for a testing.T
func NewT(t testing.TB) Logger {
	return NewLogger(slog.NewTextHandler(testWriter{T: t}, &slog.HandlerOptions{Level: TraceLevel}))
}

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

// NewBatchingHandler returns a BatchingHandler that sends the record to the given Handler
// periodically (iff interval > 0) or when the backlog is full.
func NewBatchingHandler(hndl slog.Handler, interval time.Duration, size int) slog.Handler {
	return &batchingHandler{h: hndl, interval: interval, size: size}
}

type batchingHandler struct {
	h        slog.Handler
	initOnce sync.Once
	backlog  []slog.Record
	interval time.Duration
	size     int
	// guards backlog
	mu sync.Mutex
}

// Enabled returns whether the underlying Handler returns Enabled.
func (bh *batchingHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return bh.h != nil && bh.h.Enabled(ctx, lvl)
}

// WithAttrs returns a new BatchingHandler with the underlying handlers' attrs set.
// Implies a Flush.
func (bh *batchingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return bh
	}
	bh.Flush(context.Background())
	return NewBatchingHandler(bh.h.WithAttrs(attrs), bh.interval, bh.size)
}

// WithGroup returns a new BatchingHandler with the underlying handlers' group set.
// Implies a Flush.
func (bh *batchingHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return bh
	}
	bh.Flush(context.Background())
	return NewBatchingHandler(bh.h.WithGroup(name), bh.interval, bh.size)
}

// Handle the record.
func (bh *batchingHandler) Handle(ctx context.Context, record slog.Record) error {
	bh.mu.Lock()
	defer bh.mu.Unlock()
	bh.backlog = append(bh.backlog, record)
	if bh.size >= 0 && len(bh.backlog) >= bh.size {
		bh.flush(ctx)
		return nil
	}
	if bh.interval > 0 {
		bh.initOnce.Do(func() {
			ticker := time.NewTicker(bh.interval)
			ctx := ctx
			go func() {
				defer ticker.Stop()
				if err := ctx.Err(); err != nil {
					ctx = context.Background()
				}
				for range ticker.C {
					bh.Flush(ctx)
				}
			}()
		})
	}
	return nil
}

// Flush the records in the backlog to  the underlying Handler.
func (bh *batchingHandler) Flush(ctx context.Context) error {
	bh.mu.Lock()
	err := bh.flush(ctx)
	bh.mu.Unlock()
	return err
}

// flush the records (no lock is held).
func (bh *batchingHandler) flush(ctx context.Context) error {
	var firstErr error
	for _, rec := range bh.backlog {
		if err := bh.h.Handle(ctx, rec); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	bh.backlog = bh.backlog[:0]
	return firstErr
}

// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package zlog contains some very simple go-logr / zerologr helper functions.
// This sets the default timestamp format to time.RFC3339 with ms precision.
package zlog

import (
	"io"
	"sync"
	"testing"

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

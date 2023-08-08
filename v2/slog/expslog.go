//go:build !go1.21

// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package slog

import (
	"io"
	"time"

	"golang.org/x/exp/slog"
)

type (
	Attr           = slog.Attr
	Handler        = slog.Handler
	HandlerOptions = slog.HandlerOptions
	Level          = slog.Level
	Leveler        = slog.Leveler
	LevelVar       = slog.LevelVar
	Logger         = slog.Logger
	Record         = slog.Record
)

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

func Default() *slog.Logger           { return slog.Default() }
func New(h slog.Handler) *slog.Logger { return slog.New(h) }
func NewRecord(t time.Time, lvl slog.Level, s string, p uintptr) slog.Record {
	return slog.NewRecord(t, lvl, s, p)
}

func String(k, v string) slog.Attr                             { return slog.String(k, v) }
func StringValue(value string) slog.Value                      { return slog.StringValue(value) }
func NewJSONHandler(w io.Writer, opts *HandlerOptions) Handler { return slog.NewJSONHandler(w, opts) }

func Any(k string, v any) slog.Attr                  { return slog.Any(k, v) }
func Bool(k string, v bool) slog.Attr                { return slog.Bool(k, v) }
func Duration(key string, v time.Duration) slog.Attr { return slog.Duration(k, v) }
func Float64(key string, v float64) slog.Attr        { return slog.Float64(k, v) }
func Group(key string, args ...any) slog.Attr        { return slog.Group(k, args...) }
func Int(k string, v int) slog.Attr                  { return slog.Int(k, v) }
func Int64(k string, v int64) slog.Attr              { return slog.Int64(k, v) }
func Time(k string, v time.Time) slog.Attr           { return slog.Time(k, v) }
func Uint64(k string, v uint64) slog.Attr            { return slog.Int64(k, v) }

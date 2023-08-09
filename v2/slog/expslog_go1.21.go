//go:build go1.21

// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package slog

import (
	"context"
	"io"
	"log"
	"log/slog"
	"time"
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
	JSONHandler    = slog.JSONHandler
	Kind           = slog.Kind
	LogValuer      = slog.LogValuer
	Source         = slog.Source
	TextHandler    = slog.TextHandler
	Value          = slog.Value
)

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

func Default() *slog.Logger           { return slog.Default() }
func SetDefault(l *slog.Logger)       { slog.SetDefault(l) }
func New(h slog.Handler) *slog.Logger { return slog.New(h) }
func NewRecord(t time.Time, lvl slog.Level, s string, p uintptr) slog.Record {
	return slog.NewRecord(t, lvl, s, p)
}

func NewJSONHandler(w io.Writer, opts *HandlerOptions) Handler { return slog.NewJSONHandler(w, opts) }
func NewTextHandler(w io.Writer, opts *HandlerOptions) *TextHandler {
	return slog.NewTextHandler(w, opts)
}

func Debug(msg string, args ...any)                             { slog.Debug(msg, args...) }
func DebugContext(ctx context.Context, msg string, args ...any) { slog.DebugContext(ctx, msg, args...) }
func Error(msg string, args ...any)                             { slog.Error(msg, args...) }
func ErrorContext(ctx context.Context, msg string, args ...any) { slog.ErrorContext(ctx, msg, args...) }
func Info(msg string, args ...any)                              { slog.Info(msg, args...) }
func InfoContext(ctx context.Context, msg string, args ...any)  { slog.InfoContext(ctx, msg, args...) }
func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	slog.Log(ctx, level, msg, args...)
}
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...Attr) {
	slog.LogAttrs(ctx, level, msg, attrs...)
}
func NewLogLogger(h Handler, level slog.Level) *log.Logger     { return slog.NewLogLogger(h, level) }
func Warn(msg string, args ...any)                             { slog.Warn(msg, args...) }
func WarnContext(ctx context.Context, msg string, args ...any) { slog.WarnContext(ctx, msg, args...) }
func With(args ...any) *slog.Logger                            { return slog.With(args...) }

func Any(k string, v any) slog.Attr                { return slog.Any(k, v) }
func Bool(k string, v bool) slog.Attr              { return slog.Bool(k, v) }
func Duration(k string, v time.Duration) slog.Attr { return slog.Duration(k, v) }
func Float64(k string, v float64) slog.Attr        { return slog.Float64(k, v) }
func Group(k string, args ...any) slog.Attr        { return slog.Group(k, args...) }
func Int(k string, v int) slog.Attr                { return slog.Int(k, v) }
func Int64(k string, v int64) slog.Attr            { return slog.Int64(k, v) }
func String(k, v string) slog.Attr                 { return slog.String(k, v) }
func Time(k string, v time.Time) slog.Attr         { return slog.Time(k, v) }
func Uint64(k string, v uint64) slog.Attr          { return slog.Uint64(k, v) }

func AnyValue(v any) slog.Value                { return slog.AnyValue(v) }
func BoolValue(v bool) slog.Value              { return slog.BoolValue(v) }
func DurationValue(v time.Duration) slog.Value { return slog.DurationValue(v) }
func Float64Value(v float64) slog.Value        { return slog.Float64Value(v) }
func GroupValue(as ...Attr) slog.Value         { return slog.GroupValue(as...) }
func Int64Value(v int64) slog.Value            { return slog.Int64Value(v) }
func IntValue(v int) slog.Value                { return slog.IntValue(v) }
func StringValue(value string) slog.Value      { return slog.StringValue(value) }
func TimeValue(v time.Time) slog.Value         { return slog.TimeValue(v) }
func Uint64Value(v uint64) slog.Value          { return slog.Uint64Value(v) }

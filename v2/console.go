// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog

// Mostly copied from https://gist.github.com/wijayaerick/de3de10c47a79d5310968ba5ff101a19

// ConsoleHandler formats slog.Logger output in console format, a bit similar with rs/zlog.ConsoleHandler
// The log format is designed to be human-readable.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slog"
	"golang.org/x/term"
)

// DefaultTimeFormat is a "precise" KitchenTime.
const DefaultTimeFormat = "15:04:05.999"

var (
	// TimeFormat is the format used to print the time (padded with zeros if it is the DefaultTimeFormat).
	TimeFormat = DefaultTimeFormat

	pathSep = string([]rune{filepath.Separator})
	modPart = pathSep + "mod" + pathSep
	srcPart = pathSep + "src" + pathSep

	emptyAttr = slog.Attr{Key: "", Value: slog.StringValue("")}
	nilValue  = slog.StringValue("")
)

func trimRootPath(p string) string {
	//fmt.Printf("\ntrimRootPath(%q) modPart=%d srcPart=%d\n", p, strings.Index(p, modPart), strings.Index(p, srcPart))
	if i := strings.Index(p, modPart); i >= 0 && strings.IndexByte(p[i+len(modPart):], '@') >= 0 {
		return p[i+len(modPart):]
	} else if i := strings.Index(p, srcPart); i >= 0 {
		return p[i+len(srcPart):]
	}
	return p
}

// ConsoleHandler prints to the console
type ConsoleHandler struct {
	slog.HandlerOptions
	UseColor bool

	mu          sync.Mutex
	textHandler slog.Handler
	w           io.Writer
}

// NewConsoleHandler returns a new ConsoleHandler which writes to w.
func NewConsoleHandler(level slog.Leveler, w io.Writer) *ConsoleHandler {
	opts := DefaultConsoleHandlerOptions
	opts.Level = level
	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) != 0 {
			return a
		}
		switch a.Key {
		case "", "time", "level", "source", "msg":
			// These are handled directly
			return emptyAttr
		default:
			if a.Value.Kind() == slog.KindAny && a.Value.Any() == nil {
				return emptyAttr
			}
			return slog.Attr{Key: a.Key, Value: nilValue}
		}
		return a
	}
	return &ConsoleHandler{
		UseColor:       true,
		HandlerOptions: opts,

		w:           w,
		textHandler: opts.NewTextHandler(w),
	}
}

// DefaultHandlerOptions adds the source.
var DefaultHandlerOptions = slog.HandlerOptions{AddSource: true}

// DefaultConsoleHandlerOptions *does not* add the source.
var DefaultConsoleHandlerOptions = slog.HandlerOptions{}

// MaybeConsoleHandler returns an slog.JSONHandler if w is a terminal, and slog.TextHandler otherwise.
func MaybeConsoleHandler(level slog.Leveler, w io.Writer) slog.Handler {
	if IsTerminal(w) {
		return NewConsoleHandler(level, w)
	}
	opts := DefaultHandlerOptions
	opts.Level = level
	return opts.NewJSONHandler(w)
}

// IsTerminal returns whether the io.Writer is a terminal or not.
func IsTerminal(w io.Writer) bool {
	if fder, ok := w.(interface{ Fd() uintptr }); ok {
		return term.IsTerminal(int(fder.Fd()))
	}
	return false
}

// Enabled implements slog.Handler.Enabled.
func (h *ConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.HandlerOptions.Level.Level() <= level && h.textHandler.Enabled(ctx, level)
}

var bufPool = sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 128)) }}

// Handle implements slog.Handler.Handle.
func (h *ConsoleHandler) Handle(r slog.Record) error {
	if h == nil || h.textHandler == nil {
		return nil
	}
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()
	tmp := make([]byte, 0, len(TimeFormat)+len(r.Message))
	buf.Write(r.Time.AppendFormat(tmp[:0], TimeFormat))
	if TimeFormat == DefaultTimeFormat {
		for n := len(DefaultTimeFormat) - buf.Len(); n > 0; n-- {
			buf.WriteByte('0')
		}
	}
	buf.WriteString(" ")

	var level string
	if r.Level < slog.LevelInfo {
		level = "DBG"
	} else if r.Level < slog.LevelWarn {
		level = "INF"
	} else if r.Level < slog.LevelError {
		level = "WRN"
	} else {
		level = "ERR"
	}
	if h.UseColor {
		level = addColorToLevel(level)
	}
	buf.WriteString(level)
	buf.WriteString(" ")

	if h.AddSource && r.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		file, line := frame.File, frame.Line
		if file != "" {
			buf.WriteByte('[')
			buf.WriteString(trimRootPath(file))
			buf.WriteString(":")
			buf.Write([]byte(strconv.Itoa(line)))
			buf.WriteString("] ")
		}
	}

	buf.Write(strconv.AppendQuote(tmp[:0], r.Message))
	buf.WriteByte(' ')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())
	if err != nil {
		return err
	}
	r.Time, r.Level, r.PC, r.Message = time.Time{}, 0, 0, ""
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("\nPANIC: %+v rec: %#v\n", rec, r)
		}
	}()
	return h.textHandler.Handle(r)
}

// WithAttrs implements slog.Handler.WithAttrs.
func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ConsoleHandler{
		UseColor:       h.UseColor,
		HandlerOptions: h.HandlerOptions,
		w:              h.w,
		textHandler:    h.textHandler.WithAttrs(attrs),
	}
}

// WithGroup implements slog.Handler.WithGroup.
func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	return &ConsoleHandler{
		UseColor:       h.UseColor,
		HandlerOptions: h.HandlerOptions,
		w:              h.w,
		textHandler:    h.textHandler.WithGroup(name),
	}
}

// Color is a color.
type Color uint8

// Colors
const (
	Black Color = iota + 30
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// Add adds the coloring to the given string.
func (c Color) Add(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(c), s)
}

var (
	levelToColor = map[string]Color{
		"DBG": Magenta,
		"INF": Blue,
		"WRN": Yellow,
		"ERR": Red,
	}
	unknownLevelColor = Red
)

func addColorToLevel(level string) string {
	color, ok := levelToColor[level]
	if !ok {
		color = unknownLevelColor
	}
	return color.Add(level)
}

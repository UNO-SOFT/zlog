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
	HandlerOptions
	textHandler slog.Handler
	w           io.Writer
	buf         bytes.Buffer

	mu       sync.Mutex
	UseColor bool
}

// HandlerOptions wraps slog.HandlerOptions, stripping source prefix.
type HandlerOptions struct{ slog.HandlerOptions }

func newConsoleHandlerOptions() HandlerOptions {
	opts := DefaultConsoleHandlerOptions
	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) != 0 {
			return a
		}
		switch a.Key {
		case "time", "level", "source", "msg":
			// These are handled directly
			return emptyAttr
		}
		return a
	}
	return opts
}

// NewConsoleHandler returns a new ConsoleHandler which writes to w.
func NewConsoleHandler(level slog.Leveler, w io.Writer) *ConsoleHandler {
	opts := newConsoleHandlerOptions()
	opts.Level = level
	h := ConsoleHandler{
		UseColor:       true,
		HandlerOptions: opts,

		w: w,
	}
	h.textHandler = opts.NewJSONHandler(&h.buf)
	return &h
}

// DefaultHandlerOptions adds the source.
var DefaultHandlerOptions = HandlerOptions{HandlerOptions: slog.HandlerOptions{AddSource: true}}

// DefaultConsoleHandlerOptions *does not* add the source.
var DefaultConsoleHandlerOptions = HandlerOptions{}

// MaybeConsoleHandler returns an slog.JSONHandler if w is a terminal, and slog.TextHandler otherwise.
func MaybeConsoleHandler(level slog.Leveler, w io.Writer) slog.Handler {
	if IsTerminal(w) {
		return NewConsoleHandler(level, w)
	}
	opts := DefaultHandlerOptions
	opts.Level = level
	return opts.NewJSONHandler(w)
}

type customSourceHandler struct {
	slog.Handler
	buf bytes.Buffer
}

func (opts HandlerOptions) NewJSONHandler(w io.Writer) slog.Handler {
	o := opts.HandlerOptions
	if !o.AddSource {
		return o.NewJSONHandler(w)
	}
	o.AddSource = false
	return &customSourceHandler{Handler: o.NewJSONHandler(w)}
}

func (h *customSourceHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Handler.Enabled(ctx, r.Level) {
		return nil
	}
	//fmt.Printf("customSourceHandler.Handle r=%+v PC=%d\n", r, r.PC)
	if r.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		if file, line := frame.File, frame.Line; file != "" {
			h.buf.Reset()
			r.AddAttrs(slog.String("source", trimRootPath(file)+":"+strconv.Itoa(line)))
		}
	}
	return h.Handler.Handle(ctx, r)
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

// Handle implements slog.Handler.Handle.
func (h *ConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.buf.Reset()
	tmp := make([]byte, 0, len(TimeFormat)+len(r.Message))
	h.buf.Write(r.Time.AppendFormat(tmp[:0], TimeFormat))
	if TimeFormat == DefaultTimeFormat {
		for n := len(DefaultTimeFormat) - h.buf.Len(); n > 0; n-- {
			h.buf.WriteByte('0')
		}
	}
	h.buf.WriteString(" ")

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
	h.buf.WriteString(level)
	h.buf.WriteString(" ")

	if h.AddSource && r.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		file, line := frame.File, frame.Line
		if file != "" {
			h.buf.WriteByte('[')
			h.buf.WriteString(trimRootPath(file))
			h.buf.WriteString(":")
			h.buf.Write([]byte(strconv.Itoa(line)))
			h.buf.WriteString("] ")
		}
	}

	h.buf.Write(strconv.AppendQuote(tmp[:0], r.Message))

	var err error
	var addEOL bool
	if h.textHandler == nil || r.NumAttrs() == 0 {
		addEOL = true
	} else {
		r.Time, r.Level, r.PC, r.Message = time.Time{}, 0, 0, ""
		h.buf.WriteString(" attrs=")
		err = h.textHandler.Handle(ctx, r)
	}

	if _, wErr := h.w.Write(h.buf.Bytes()); wErr != nil && err == nil {
		err = wErr
	}
	if addEOL {
		h.w.Write([]byte{'\n'})
	}
	return err
}

// WithAttrs implements slog.Handler.WithAttrs.
func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := ConsoleHandler{
		UseColor:       h.UseColor,
		HandlerOptions: h.HandlerOptions,
		w:              h.w,
	}
	h2.textHandler = newConsoleHandlerOptions().
		NewJSONHandler(&h2.buf).
		WithAttrs(attrs)
	return &h2
}

// WithGroup implements slog.Handler.WithGroup.
func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	h2 := ConsoleHandler{
		UseColor:       h.UseColor,
		HandlerOptions: h.HandlerOptions,
		w:              h.w,
	}
	h2.textHandler = newConsoleHandlerOptions().
		NewJSONHandler(&h2.buf).
		WithGroup(name)
	return &h2
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

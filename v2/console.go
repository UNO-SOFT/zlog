// Copyright 2022, 2024 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog

// Mostly copied from https://gist.github.com/wijayaerick/de3de10c47a79d5310968ba5ff101a19

// ConsoleHandler formats slog.Logger output in console format, a bit similar with rs/zlog.ConsoleHandler
// The log format is designed to be human-readable.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/UNO-SOFT/zlog/v2/slog"
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

	zeroAttr slog.Attr
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

var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// ConsoleHandler prints to the console
type ConsoleHandler struct {
	HandlerOptions
	w           io.Writer
	withGroup   []string
	withAttrs   []slog.Attr
	UseColor    bool
	attrHandler *slog.TextHandler

	mu      *sync.Mutex
	attrBuf bytes.Buffer
}

// HandlerOptions wraps slog.HandlerOptions, stripping source prefix.
type HandlerOptions struct{ slog.HandlerOptions }

var (
	jsonMarshalableMu  sync.Mutex
	jsonMarshalableBuf bytes.Buffer
	jsonMarshalableEnc = json.NewEncoder(&jsonMarshalableBuf)
)

func ensurePrintableValueIsEmpty(value *slog.Value) (isEmpty bool) {
	if value.Kind() != slog.KindAny {
		return false
	}

	var ok bool
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
		if !isEmpty && !ok {
			*value = slog.StringValue(fmt.Sprintf("%v", *value))
		}
	}()
	v := value.Any()
	if v == nil {
		ok = true
		return true
	}
	switch x := v.(type) {
	case string:
		ok = true
		*value = slog.StringValue(x)
		return x == ""
	case error:
		ok = true
		if x == nil {
			return true
		}
		*value = slog.StringValue(x.Error())
		return false
	case json.Marshaler:
		ok = true
		return x == nil
	case fmt.Stringer:
		ok = true
		s := x.String()
		*value = slog.StringValue(s)
		return s == ""
	case bool:
		ok = true
		*value = slog.BoolValue(x)
		return false

	case int:
		ok = true
		*value = slog.IntValue(x)
	case int8:
		ok = true
		*value = slog.IntValue(int(x))
		return false
	case int16:
		ok = true
		*value = slog.IntValue(int(x))
		return false
	case int32:
		ok = true
		*value = slog.IntValue(int(x))
		return false
	case int64:
		ok = true
		*value = slog.Int64Value(x)
		return false

	case uint:
		ok = true
		*value = slog.Int64Value(int64(x))
	case uint8:
		ok = true
		*value = slog.IntValue(int(x))
		return false
	case uint16:
		ok = true
		*value = slog.IntValue(int(x))
		return false
	case uint32:
		ok = true
		*value = slog.Int64Value(int64(x))
		return false
	case uint64:
		ok = true
		if x > 1<<63 {
			*value = slog.StringValue(strconv.FormatUint(x, 10))
		} else {
			*value = slog.Uint64Value(x)
		}
		return false

	case float32:
		ok = true
		*value = slog.Float64Value(float64(x))
		return false
	case float64:
		ok = true
		*value = slog.Float64Value(x)
		return false

	case complex64:
		ok = true
		*value = slog.StringValue(strconv.FormatComplex(complex128(x), 'f', -1, 64))
		return false
	case complex128:
		ok = true
		*value = slog.StringValue(strconv.FormatComplex(x, 'f', -1, 128))
		return false

	default:

		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Invalid:
			ok = false
			return true
		case reflect.Chan, reflect.Func:
			ok = false
			return rv.IsNil()
		default:
			jsonMarshalableMu.Lock()
			defer jsonMarshalableMu.Unlock()
			jsonMarshalableBuf.Reset()
			if ok = jsonMarshalableEnc.Encode(v) == nil; ok {
				switch x := jsonMarshalableBuf.String(); x {
				case `""`, `[]`, `{}`, "null":
					return true
				default:
					*value = slog.StringValue(x)
					return false
				}
			}
		}
		return rv.IsZero()
	}
	return false
}

func newConsoleHandlerOptions() HandlerOptions {
	opts := DefaultConsoleHandlerOptions
	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case "time", "level", "source", "msg":
			// These are handled directly
			return zeroAttr
		default:
			if a.Value.Kind() == slog.KindAny {
				if ensurePrintableValueIsEmpty(&a.Value) {
					return zeroAttr
				}
			}
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
		w:              w,
		mu:             new(sync.Mutex),
	}
	h.initAttrHandler()
	return &h
}

// DefaultHandlerOptions adds the source.
var DefaultHandlerOptions = HandlerOptions{HandlerOptions: slog.HandlerOptions{
	AddSource: true,
	ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case "time", "level", "source":
			return a
		default:
			if a.Value.Kind() == slog.KindAny {
				if ensurePrintableValueIsEmpty(&a.Value) {
					return zeroAttr
				}
			}
		}
		return a
	}},
}

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

func (opts HandlerOptions) NewJSONHandler(w io.Writer) slog.Handler {
	o := opts.HandlerOptions
	addSource := o.AddSource
	o.AddSource = false
	hndl := slog.NewJSONHandler(w, &o)
	if !addSource {
		return hndl
	}
	return customSourceHandler{Handler: &syncHandler{Handler: hndl}}
}

type syncHandler struct {
	mu sync.Mutex
	slog.Handler
}

func (h *syncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	return &syncHandler{Handler: h.Handler.WithAttrs(attrs)}
}
func (h *syncHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	return &syncHandler{Handler: h.Handler.WithGroup(name)}
}
func (h *syncHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.Handler.Handle(ctx, r)
}

type customSourceHandler struct {
	slog.Handler
}

func (h customSourceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return customSourceHandler{Handler: h.Handler.WithAttrs(attrs)}
}
func (h customSourceHandler) WithGroup(name string) slog.Handler {
	return customSourceHandler{Handler: h.Handler.WithGroup(name)}
}
func (h customSourceHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Handler.Enabled(ctx, r.Level) {
		return nil
	}
	//fmt.Printf("customSourceHandler.Handle r=%+v PC=%d\n", r, r.PC)
	if r.PC != 0 {
		// https://pkg.go.dev/log/slog#example-package-Wrapping
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		if file, line := frame.File, frame.Line; file != "" {
			buf := bufPool.Get().(*bytes.Buffer)
			defer bufPool.Put(buf)
			buf.Reset()
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
	return level >= h.HandlerOptions.Level.Level()
}

// Handle implements slog.Handler.Handle.
func (h *ConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	if h == nil {
		return nil
	}
	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)
	buf.Reset()
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

	var err error
	if r.NumAttrs() != 0 {
		func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.attrBuf.Reset()

			r.Time, r.Level, r.PC, r.Message = time.Time{}, 0, 0, ""
			err = h.attrHandler.Handle(ctx, r)
			if h.attrBuf.Len() != 0 {
				buf.WriteByte(' ')
				buf.Write(h.attrBuf.Bytes())
			}
		}()
	}
	if buf.Len() != 0 && buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	if _, wErr := h.w.Write(buf.Bytes()); wErr != nil && err == nil {
		err = wErr
	}

	return err
}

func (h *ConsoleHandler) initAttrHandler() {
	h.attrHandler = slog.NewTextHandler(&h.attrBuf, &h.HandlerOptions.HandlerOptions)
	if len(h.withAttrs) != 0 {
		h.attrHandler = h.attrHandler.WithAttrs(h.withAttrs).(*slog.TextHandler)
	}
	if len(h.withGroup) != 0 {
		for _, g := range h.withGroup {
			h.attrHandler = h.attrHandler.WithGroup(g).(*slog.TextHandler)
		}
	}
}

// WithAttrs implements slog.Handler.WithAttrs.
func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	h2.withAttrs = append(append(make([]slog.Attr, 0, len(h2.withAttrs)+len(attrs)), h2.withAttrs...), attrs...)
	h2.initAttrHandler()
	return &h2
}

// WithGroup implements slog.Handler.WithGroup.
func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	h2 := *h
	h2.withGroup = append(append(make([]string, 0, len(h2.withGroup)+1), h2.withGroup...), name)
	h2.initAttrHandler()
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

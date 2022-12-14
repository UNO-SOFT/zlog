package zlog

// Mostly copied from https://gist.github.com/wijayaerick/de3de10c47a79d5310968ba5ff101a19

// ConsoleHandler formats slog.Logger output in console format, a bit similar with rs/zlog.ConsoleHandler
// The log format is designed to be human-readable.

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/exp/slog"
)

var (
	TimeFormat = "15:04:05.999"

	_, callerFile, _, _ = runtime.Caller(0)
	rootPath            = filepath.Dir(callerFile)
)

func trimRootPath(p string) string {
	return strings.Replace(p, rootPath, ".", 1)
}

// ConsoleHandler prints to the console
type ConsoleHandler struct {
	slog.HandlerOptions
	UseColor bool

	mu          sync.Mutex
	textHandler slog.Handler
	buf         bytes.Buffer
	w           io.Writer
}

// NewConsoleHandler returns a new ConsoleHandler which writes to w.
func NewConsoleHandler(w io.Writer) *ConsoleHandler {
	opts := DefaultHandlerOptions
	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) != 0 {
			return a
		}
		switch a.Key {
		case "time", "level", "source", "msg":
			// These are handled directly
			return slog.Attr{}
		}
		return a
	}
	return &ConsoleHandler{
		UseColor:       IsTerminal(w),
		HandlerOptions: opts,

		w:           w,
		textHandler: opts.NewTextHandler(w),
	}
}

func (h *ConsoleHandler) Enabled(level slog.Level) bool { return h.textHandler.Enabled(level) }

var bufPool = sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 128)) }}

func (h *ConsoleHandler) Handle(r slog.Record) error {
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()
	buf.WriteString(r.Time.Format(TimeFormat))
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

	if h.AddSource {
		file, line := r.SourceLine()
		if file != "" {
			buf.WriteByte('[')
			buf.WriteString(trimRootPath(file))
			buf.WriteString(":")
			buf.Write([]byte(strconv.Itoa(line)))
			buf.WriteString("] ")
		}
	}

	buf.WriteString(r.Message)
	buf.WriteByte(' ')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())
	if err != nil {
		return err
	}
	return h.textHandler.Handle(r)
}

func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ConsoleHandler{
		UseColor:       h.UseColor,
		HandlerOptions: h.HandlerOptions,
		w:              h.w,
		textHandler:    h.textHandler.WithAttrs(attrs),
	}
}

func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	return &ConsoleHandler{
		UseColor:       h.UseColor,
		HandlerOptions: h.HandlerOptions,
		w:              h.w,
		textHandler:    h.textHandler.WithGroup(name),
	}
}

type Color uint8

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

// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/UNO-SOFT/zlog/v2"
	"github.com/UNO-SOFT/zlog/v2/slog"
)

func TestMultiConsoleLevel(t *testing.T) {
	var bufInfo, bufAll bytes.Buffer
	verbose := zlog.VerboseVar(0)
	zl := zlog.NewConsoleHandler(&verbose, &bufInfo)
	zlMulti := zlog.NewMultiHandler(zl)
	logger := zlog.NewLogger(zlMulti)
	zlMulti.Add(slog.NewJSONHandler(&bufAll, nil))
	//t.Logf("SetLevel(%v)", zlog.ErrorLevel)
	//logger.SetLevel(zlog.ErrorLevel)
	t.Logf("logger: %#v slog: %#v",
		logger,
		logger.SLog())
	logger.Info("info")
	logger.Error(io.EOF, "error")

	t.Log(bufAll.String())
	if !check(t,
		parse(bufAll.Bytes()),
		map[string]int{"info": 1, "error": 1},
	) {
		return
	}

	t.Log(bufInfo.String())
	if !check(t,
		parse(bufInfo.Bytes()),
		map[string]int{"info": 0, "error": 1},
	) {
		return
	}
}

func TestMultiHandlerLevel(t *testing.T) {
	var bufInfo, bufAll bytes.Buffer
	zl := zlog.NewLevelHandler(zlog.ErrorLevel, slog.NewJSONHandler(&bufInfo, nil))
	zlMulti := zlog.NewMultiHandler(zl)
	logger := zlog.NewLogger(zlMulti)
	zlMulti.Add(slog.NewJSONHandler(&bufAll, nil))
	//t.Logf("SetLevel(%v)", zlog.ErrorLevel)
	//logger.SetLevel(zlog.ErrorLevel)
	t.Logf("logger: %#v slog: %#v",
		logger,
		logger.SLog())
	logger.Info("info")
	logger.Error(io.EOF, "error")

	t.Log(bufAll.String())
	if !check(t,
		parse(bufAll.Bytes()),
		map[string]int{"info": 1, "error": 1},
	) {
		return
	}

	t.Log(bufInfo.String())
	if !check(t,
		parse(bufInfo.Bytes()),
		map[string]int{"info": 0, "error": 1},
	) {
		return
	}
}

func TestGroup(t *testing.T) {
	do := func(logger *slog.Logger) {
		logger.Info("naked", "a", 0)
		{
			logger := logger.WithGroup("group")
			t.Logf("WithGroup: %#v", logger)
			logger.Info("justGroup", "a", 1)
		}
		logger = logger.With("with", "value")
		t.Logf("WithValue: %#v", logger)
		logger.Info("withValue", "a", 2)
		logger = logger.WithGroup("group")
		t.Logf("WithValueGroup: %#v", logger)
		var emptyFunc func()
		logger.Info("withValueGroup", "a", 3, "emptyFunc", emptyFunc, "func", logger.Info)
	}

	t.Run("console", func(t *testing.T) {
		verbose := zlog.VerboseVar(2)
		var buf bytes.Buffer
		zl := zlog.NewConsoleHandler(&verbose, &buf)
		logger := zlog.NewLogger(zl).SLog()

		do(logger)
		t.Log(buf.String())

		want := []struct {
			Msg   string
			Attrs map[string]any
		}{
			{Msg: "naked", Attrs: map[string]any{"a": 0}},
			{Msg: "justGroup", Attrs: map[string]any{"group": map[string]any{"a": 1}}},
			{Msg: "withValue", Attrs: map[string]any{"with": "value", "group": map[string]any{"a": 2}}},
			{Msg: "withValueGroup", Attrs: map[string]any{"group": map[string]any{"a": 3}}},
		}
		for i, line := range bytes.Split(buf.Bytes(), []byte{'\n'}) {
			if len(line) == 0 {
				continue
			}
			var m map[string]any
			if _, after, found := bytes.Cut(line, []byte(" \x1b[34mINF\x1b[0m ")); !found {
				t.Errorf("line %q does not contain INF", string(line))
			} else if j := bytes.IndexByte(after, '{'); j < 0 {
				t.Errorf("%d. no { in %q", i+1, string(after))
			} else if msg, err := strconv.Unquote(string(bytes.TrimSpace(bytes.TrimSuffix(after[:j], []byte("attrs="))))); err != nil {
				t.Errorf("%d. unquote %q: %+v", i+1, string(after[:j]), err)
			} else if err = json.Unmarshal(after[j:], &m); err != nil {
				t.Errorf("%d. unmarshal %q: %+v", i+1, string(after[:j]), err)
			} else if want[i].Msg != msg {
				t.Errorf("%d. got %q, wanted %q", i+1, msg, want[i].Msg)
			} else {
				t.Logf("%d. %q %+v", i+1, msg, m)
			}
		}
	})

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		zl := zlog.DefaultHandlerOptions.NewJSONHandler(&buf)
		t.Logf("zl: %#v", zl)
		logger := zlog.NewLogger(zl).SLog()

		do(logger)
		t.Log(buf.String())
		type Line struct {
			Time   time.Time      `json:"time"`
			Level  string         `json:"level"`
			Msg    string         `json:"msg"`
			Source string         `json:"source"`
			Group  map[string]any `json:"group"`
			A      int            `json:"a"`
			With   string         `json:"with"`
		}
		dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
		for {
			var line Line
			if err := dec.Decode(&line); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				t.Fatal(err)
			}
			t.Log(line)
			if line.Source == "" && line.Group["source"] == "" {
				t.Error("no source")
			}
		}

	})
}

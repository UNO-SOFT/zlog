// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
			logger.Info("justGroup", "a", 1)
		}
		logger = logger.With("with", "value")
		logger.Info("withValue", "a", 2)
		logger = logger.WithGroup("group")
		logger.Info("withValueGroup", "a", 3)
	}

	t.Run("console", func(t *testing.T) {
		verbose := zlog.VerboseVar(2)
		var buf bytes.Buffer
		zl := zlog.NewConsoleHandler(&verbose, &buf)
		logger := zlog.NewLogger(zl).SLog()

		do(logger)
		t.Log(buf.String())
	})

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		zl := zlog.DefaultHandlerOptions.NewJSONHandler(&buf)
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
			if line.Source == "" {
				t.Error("no source")
			}
		}

	})
}

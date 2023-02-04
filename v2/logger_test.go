// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog_test

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/UNO-SOFT/zlog/v2"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"golang.org/x/exp/slog"
)

func TestLoggerLevel(t *testing.T) {
	var buf strings.Builder
	logger := zlog.New(&buf)
	t.Logf("SetLevel(%v)", zlog.ErrorLevel)
	logger.SetLevel(zlog.ErrorLevel)
	t.Logf("logger: %#v slog: %#v level: %v",
		logger,
		logger.SLog(),
		logger.SLog().Handler().(*zlog.LevelHandler).GetLevel())
	logger.Info("info")
	logger.Error(io.EOF, "error")
	t.Log(buf.String())
	recs := parse(strings.NewReader(buf.String()))
	if !check(t, recs, map[string]int{"info": 0, "error": 1}) {
		return
	}
}

func TestMultiHandlerLevel(t *testing.T) {
	var bufInfo, bufAll strings.Builder
	zl := zlog.NewLevelHandler(zlog.ErrorLevel, slog.NewJSONHandler(&bufInfo))
	zlMulti := zlog.NewMultiHandler(zl)
	logger := zlog.NewLogger(zlMulti)
	zlMulti.Add(slog.NewJSONHandler(&bufAll))
	//t.Logf("SetLevel(%v)", zlog.ErrorLevel)
	//logger.SetLevel(zlog.ErrorLevel)
	t.Logf("logger: %#v slog: %#v",
		logger,
		logger.SLog())
	logger.Info("info")
	logger.Error(io.EOF, "error")

	t.Log(bufAll.String())
	if !check(t,
		parse(strings.NewReader(bufAll.String())),
		map[string]int{"info": 1, "error": 1},
	) {
		return
	}

	t.Log(bufInfo.String())
	if !check(t,
		parse(strings.NewReader(bufInfo.String())),
		map[string]int{"info": 0, "error": 1},
	) {
		return
	}

}

func TestLogrLevel(t *testing.T) {
	var buf strings.Builder
	zlogger := zerolog.New(&buf).Level(zerolog.ErrorLevel)
	zerolog.MessageFieldName = "msg"
	logger := zerologr.New(&zlogger)
	logger.Info("info")
	logger.Error(io.EOF, "error")
	t.Log(buf.String())
	recs := parse(strings.NewReader(buf.String()))
	if !check(t, recs, map[string]int{"info": 0, "error": 1}) {
		return
	}
}

func TestSLogLevel(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.HandlerOptions{Level: slog.LevelError}.NewJSONHandler(&buf))
	logger.Info("info")
	logger.Error("error", io.EOF)
	t.Log(buf.String())
	recs := parse(strings.NewReader(buf.String()))
	if !check(t, recs, map[string]int{"info": 0, "error": 1}) {
		return
	}
}

type record struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
	Line  int    `json:"-"`
}

func check(t *testing.T, recs map[string][]record, want map[string]int) bool {
	t.Log("recs:", recs)
	got := make(map[string]int, len(want))
	ok := true
	for k, v := range want {
		if got[k] = len(recs[k]); got[k] != v {
			t.Errorf("got %d %q, wanted %d", got[k], k, v)
			ok = false
		}
	}
	return ok
}

func parse(r io.Reader) map[string][]record {
	records := make(map[string][]record)
	dec := json.NewDecoder(r)
	var lineNo int
	for {
		var rec record
		if err := dec.Decode(&rec); err != nil {
			return records
		}
		lineNo++
		rec.Line = lineNo
		records[rec.Msg] = append(records[rec.Msg], rec)
	}
}

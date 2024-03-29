// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog_test

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"testing"

	"github.com/UNO-SOFT/zlog/v2"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"golang.org/x/exp/slog"
)

func TestLoggerLevel(t *testing.T) {
	var buf bytes.Buffer
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
	recs := parse(buf.Bytes())
	if !check(t, recs, map[string]int{"info": 0, "error": 1}) {
		return
	}
}

func TestLogrLevel(t *testing.T) {
	var buf bytes.Buffer
	zlogger := zerolog.New(&buf).Level(zerolog.ErrorLevel)
	zerolog.MessageFieldName = "msg"
	logger := zerologr.New(&zlogger)
	logger.Info("info")
	logger.Error(io.EOF, "error")
	t.Log(buf.String())
	recs := parse(buf.Bytes())
	if !check(t, recs, map[string]int{"info": 0, "error": 1}) {
		return
	}
}

func TestSLogLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))
	logger.Info("info")
	logger.Error("error", io.EOF)
	t.Log(buf.String())
	recs := parse(buf.Bytes())
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

func parse(b []byte) map[string][]record {
	records := make(map[string][]record)
	for lineNo, line := range bytes.Split(b, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var rec record
		if line[0] == '{' && line[len(line)-1] == '}' {
			if err := json.Unmarshal(line, &rec); err != nil {
				return records
			}
		} else if i := bytes.IndexByte(line, '"'); i >= 0 {
			if j := bytes.IndexByte(line[i+1:], '"'); j >= 0 {
				rec.Msg, _ = strconv.Unquote(string(line[i : i+1+j+1]))
			}
		}
		rec.Line = lineNo + 1
		records[rec.Msg] = append(records[rec.Msg], rec)
	}
	return records
}

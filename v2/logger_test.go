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
)

func TestLogLevel(t *testing.T) {
	var buf strings.Builder
	logger := zlog.New(&buf)
	logger.SetLevel(zlog.ErrorLevel)
	logger.Info("info")
	logger.Error(io.EOF, "error")
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

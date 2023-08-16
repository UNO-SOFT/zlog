//go:build go1.21

// Copyright 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"testing/slogtest"

	"github.com/UNO-SOFT/zlog/v2"
	"github.com/UNO-SOFT/zlog/v2/slog"
)

func TestSLogTest(t *testing.T) {
	var buf bytes.Buffer
	var level slog.LevelVar
	h := zlog.MaybeConsoleHandler(&level, &buf)

	results := func() []map[string]any {
		var ms []map[string]any
		for _, line := range bytes.Split(buf.Bytes(), []byte{'\n'}) {
			if len(line) == 0 {
				continue
			}
			var m map[string]any
			if err := json.Unmarshal(line, &m); err != nil {
				t.Fatal(err) // In a real test, use t.Fatal.
			}
			ms = append(ms, m)
		}
		return ms
	}
	if err := slogtest.TestHandler(h, results); err != nil {
		if strings.Contains(err.Error(), "a Handler should not output groups for an empty Record") {
			t.Log(err)
		} else {
			t.Fatal(err)
		}
	}
}

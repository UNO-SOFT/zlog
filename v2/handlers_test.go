// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/UNO-SOFT/zlog/v2"
	"golang.org/x/exp/slog"
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

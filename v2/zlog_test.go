// Copyright 2022 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package zlog_test

import (
	"strings"
	"testing"

	"github.com/UNO-SOFT/zlog/v2"
)

func TestLevels(t *testing.T) {
	var buf strings.Builder
	logger := zlog.New(&buf)
	logger.Info("a")
	t.Logf("got: %q", buf.String())
}

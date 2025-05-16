// Copyright 2024 Tamas Gulacsi. All rights reserved.
// Copyright 2023 Jamie Tama. All rights reserved.
//
// Package loghttp is from https://www.jvt.me/posts/2023/03/11/go-debug-http/
package loghttp

import (
	"crypto/sha256"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"

	"github.com/UNO-SOFT/zlog/v2"
)

type option func(*LoggingTransport)

// WithLevel allows seting then log level.
func WithLevel(lvl slog.Leveler) option {
	return func(tr *LoggingTransport) { tr.LogLevel = lvl }
}

// Transport returns a transport that logs requests and responses.
func Transport(tr http.RoundTripper, opts ...option) LoggingTransport {
	ltr := LoggingTransport{Transport: tr, seen: new(sync.Map), size: new(atomic.Uint32)}
	for _, o := range opts {
		o(&ltr)
	}
	return ltr
}

type LoggingTransport struct {
	LogLevel  slog.Leveler
	Transport http.RoundTripper
	seen      *sync.Map
	size      *atomic.Uint32
}

func (s LoggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()
	logger := zlog.SFromContext(ctx)
	level := slog.LevelDebug
	if s.LogLevel != nil {
		level = s.LogLevel.Level()
	}
	enabled := logger.Enabled(ctx, level)
	var reqBytes []byte
	if enabled {
		var err error
		if reqBytes, err = httputil.DumpRequestOut(r, true); err != nil {
			logger.Error("DumpRequestOut", "error", err)
		}
	}

	tr := http.DefaultTransport
	if s.Transport != nil {
		tr = s.Transport
	}
	resp, err := tr.RoundTrip(r)
	// err is returned after dumping the response
	if !enabled {
		return resp, err
	}

	var respBytes []byte
	if resp != nil {
		var dumpErr error
		if respBytes, dumpErr = httputil.DumpResponse(resp, true); dumpErr != nil {
			logger.Error("DumpResponse", "error", dumpErr)
		}
	}
	var skip bool
	if s.seen != nil {
		h := sha256.New()
		h.Write(reqBytes)
		h.Write(respBytes)
		if _, skip = s.seen.LoadOrStore(h.Sum(nil), nil); !skip {
			if s.size.Add(1) > 1000 {
				s.seen.Clear()
				s.seen.Store(h.Sum(nil), nil)
			}
		}
	}

	if !skip {
		logger.Log(ctx, level, "RoundTrip", "request", string(reqBytes), "response", string(respBytes))
	}

	return resp, err
}

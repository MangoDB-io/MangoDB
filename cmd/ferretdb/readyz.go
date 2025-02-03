// Copyright 2021 FerretDB Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"

	"github.com/FerretDB/wire"
	"github.com/FerretDB/wire/wireclient"

	"github.com/FerretDB/FerretDB/v2/internal/util/logging"
	"github.com/FerretDB/FerretDB/v2/internal/util/must"
)

// ReadyZ represents the Readiness probe.
//
// It connects to FerretDB with Go driver and sends `ping` command.
type ReadyZ struct {
	l *slog.Logger
}

// Probe implements [debug.Probe].
func (r *ReadyZ) Probe(ctx context.Context) bool {
	var urls []string

	if cli.Listen.Addr != "" {
		host, port, err := net.SplitHostPort(cli.Listen.Addr)
		if err != nil {
			r.l.ErrorContext(ctx, "Getting host and port failed", logging.Error(err))
			return false
		}

		if host == "" {
			host = "127.0.0.1"
		}

		u := &url.URL{
			Scheme: "mongodb",
			Host:   net.JoinHostPort(host, port),
			Path:   "/",
		}

		urls = append(urls, u.String())
	}

	if cli.Listen.TLS != "" {
		// TODO https://github.com/FerretDB/FerretDB/issues/4427
		r.l.WarnContext(ctx, "TLS ping is not implemented yet")
	}

	if cli.Listen.Unix != "" {
		urls = append(urls, "mongodb://"+url.PathEscape(cli.Listen.Unix))
	}

	if len(urls) == 0 {
		r.l.InfoContext(ctx, "Nothing to ping")
		return true
	}

	for _, u := range urls {
		r.l.DebugContext(ctx, fmt.Sprintf("Pinging %s", u))

		conn, err := wireclient.Connect(ctx, u, r.l)
		if err != nil {
			r.l.ErrorContext(ctx, "Connection failed", logging.Error(err))
			return false
		}

		defer func() {
			if err = conn.Close(); err != nil {
				r.l.ErrorContext(ctx, "Closing connection failed", logging.Error(err))
			}
		}()

		_, resBody, err := conn.Request(ctx, wire.MustOpMsg(
			"ping", int32(1),
			"$db", "admin",
		))
		if err != nil {
			r.l.ErrorContext(ctx, "Ping request failed", logging.Error(err))
			return false
		}

		res, err := must.NotFail(resBody.(*wire.OpMsg).RawDocument()).Decode()
		if err != nil {
			r.l.ErrorContext(ctx, "Decoding ping response failed", logging.Error(err))
			return false
		}

		if res.Get("ok").(float64) == 1 {
			r.l.InfoContext(ctx, "Ping successful", slog.String("url", u))
			continue
		}

		attrs := []any{
			slog.Int("code", int(res.Get("code").(int32))),
			slog.String("code_name", res.Get("codeName").(string)),
			slog.String("errmsg", res.Get("errmsg").(string)),
		}
		r.l.ErrorContext(ctx, "Ping failed", attrs...)

		return false
	}

	return true
}

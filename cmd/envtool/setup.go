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
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/FerretDB/FerretDB/v2/internal/util/debug"
	"github.com/FerretDB/FerretDB/v2/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/v2/internal/util/logging"
)

// setup runs all setup commands.
func setup(ctx context.Context, logger *slog.Logger) error {
	h, err := debug.Listen(&debug.ListenOpts{
		TCPAddr: "127.0.0.1:8089",
		L:       logging.WithName(logger, "debug"),
		R:       prometheus.DefaultRegisterer,
	})
	if err != nil {
		return lazyerrors.Error(err)
	}

	go h.Serve(ctx)

	logger.InfoContext(ctx, "Done.")
	return nil
}

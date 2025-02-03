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

package setup

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/FerretDB/FerretDB/v2/internal/clientconn/connmetrics"
	"github.com/FerretDB/FerretDB/v2/internal/util/debug"
	"github.com/FerretDB/FerretDB/v2/internal/util/logging"
	"github.com/FerretDB/FerretDB/v2/internal/util/observability"

	"github.com/FerretDB/FerretDB/v2/integration/shareddata"
)

// listenerMetrics are shared between tests.
var listenerMetrics = connmetrics.NewListenerMetrics()

// shutdown cancels context passed to startup components.
var shutdown context.CancelFunc

// startupWG waits for all startup components to finish.
var startupWG sync.WaitGroup

// Startup initializes things that should be initialized only once.
func Startup() {
	opts := &logging.NewHandlerOpts{
		Base:          "console",
		Level:         slog.LevelDebug,
		RemoveTime:    true,
		RemoveLevel:   true,
		CheckMessages: false, // TODO https://github.com/FerretDB/FerretDB/issues/4511
	}
	logging.Setup(opts, "")
	l := slog.Default()

	ctx := context.Background()

	// https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables
	if t, _ := strconv.ParseBool(os.Getenv("RUNNER_DEBUG")); t {
		l.InfoContext(ctx, "Enabling setup debug logging on GitHub Actions")
		*debugSetupF = true
	}

	prometheus.DefaultRegisterer.MustRegister(listenerMetrics)

	// use any available port to allow running different configurations in parallel
	h, err := debug.Listen(&debug.ListenOpts{
		TCPAddr: "127.0.0.1:0",
		L:       logging.WithName(l, "debug"),
		R:       prometheus.DefaultRegisterer,
	})
	if err != nil {
		l.LogAttrs(ctx, logging.LevelFatal, "Failed to create debug handler", logging.Error(err))
	}

	ot, err := observability.NewOTelTraceExporter(&observability.OTelTraceExporterOpts{
		Logger:  logging.WithName(l, "otel"),
		Service: "integration-tests",
		URL:     "http://127.0.0.1:4318/v1/traces",
	})
	if err != nil {
		l.LogAttrs(ctx, logging.LevelFatal, "Failed to create Otel tracer", logging.Error(err))
	}

	ctx, shutdown = context.WithCancel(ctx)

	startupWG.Add(1)

	go func() {
		defer startupWG.Done()
		h.Serve(ctx)
	}()

	startupWG.Add(1)

	go func() {
		defer startupWG.Done()
		ot.Run(ctx)
	}()

	clientCtx, clientCancel := context.WithTimeout(ctx, 5*time.Second)
	defer clientCancel()

	// do basic flags validation earlier, before all tests

	if *benchDocsF <= 0 {
		l.LogAttrs(ctx, logging.LevelFatal, "-bench-docs must be > 0")
	}

	for _, p := range shareddata.AllBenchmarkProviders() {
		if g, ok := p.(shareddata.BenchmarkGenerator); ok {
			g.Init(*benchDocsF)
		}
	}

	if *targetBackendF == "" {
		l.LogAttrs(ctx, logging.LevelFatal, "-target-backend must be set")
	}

	if !slices.Contains(allBackends, *targetBackendF) {
		l.LogAttrs(ctx, logging.LevelFatal, "Unknown target backend", slog.String("target_backend", *targetBackendF))
	}

	if *targetURLF != "" {
		*targetURLF, err = setClientPaths(*targetURLF)
		if err != nil {
			l.LogAttrs(ctx, logging.LevelFatal, "Failed to set target client path", logging.Error(err))
		}

		var client *mongo.Client

		client, err = makeClient(clientCtx, *targetURLF, false)
		if err != nil {
			l.LogAttrs(ctx, logging.LevelFatal, "Failed to connect to target system", slog.String("target_url", *targetURLF), logging.Error(err))
		}

		_ = client.Disconnect(clientCtx)

		l.InfoContext(ctx, "Target system", slog.String("target_backend", *targetBackendF), slog.String("target_url", *targetURLF))
	} else {
		l.InfoContext(ctx, "Target system (built-in)", slog.String("target_backend", *targetBackendF))
	}

	if *compatURLF != "" {
		*compatURLF, err = setClientPaths(*compatURLF)
		if err != nil {
			l.LogAttrs(ctx, logging.LevelFatal, "Failed to set compat client path", logging.Error(err))
		}

		var client *mongo.Client

		client, err = makeClient(clientCtx, *compatURLF, false)
		if err != nil {
			l.LogAttrs(ctx, logging.LevelFatal, "Failed to connect to compat system", slog.String("compat_url", *compatURLF), logging.Error(err))
		}

		_ = client.Disconnect(clientCtx)

		l.InfoContext(ctx, "Compat system: MongoDB", slog.String("compat_url", *compatURLF))
	} else {
		l.InfoContext(ctx, "Compat system: none, compatibility tests will be skipped")
	}
}

// Shutdown cleans up after all tests.
func Shutdown() {
	shutdown()

	startupWG.Wait()

	// to increase a chance of resource finalizers to spot problems
	runtime.GC()
	runtime.GC()
}

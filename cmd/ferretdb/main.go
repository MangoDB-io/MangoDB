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
	"log"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alecthomas/kong"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"go.uber.org/automaxprocs/maxprocs"
	_ "golang.org/x/crypto/x509roots/fallback" // register root TLS certificates for production Docker image

	"github.com/FerretDB/FerretDB/v2/build/version"
	"github.com/FerretDB/FerretDB/v2/internal/clientconn"
	"github.com/FerretDB/FerretDB/v2/internal/clientconn/connmetrics"
	"github.com/FerretDB/FerretDB/v2/internal/dataapi"
	"github.com/FerretDB/FerretDB/v2/internal/documentdb"
	"github.com/FerretDB/FerretDB/v2/internal/handler"
	"github.com/FerretDB/FerretDB/v2/internal/util/ctxutil"
	"github.com/FerretDB/FerretDB/v2/internal/util/debug"
	"github.com/FerretDB/FerretDB/v2/internal/util/devbuild"
	"github.com/FerretDB/FerretDB/v2/internal/util/logging"
	"github.com/FerretDB/FerretDB/v2/internal/util/must"
	"github.com/FerretDB/FerretDB/v2/internal/util/observability"
	"github.com/FerretDB/FerretDB/v2/internal/util/state"
	"github.com/FerretDB/FerretDB/v2/internal/util/telemetry"
)

// The cli struct represents all command-line commands, fields and flags.
// It's used for parsing the user input.
//
// Keep order in sync with documentation.
var cli struct {
	// We hide `run` command to show only `ping` in the help message.
	Run  struct{} `cmd:"" default:"1"                             hidden:""`
	Ping struct{} `cmd:"" help:"Ping existing FerretDB instance."`

	Version     bool   `default:"false"           help:"Print version to stdout and exit."      env:"-"`
	Mode        string `default:"${default_mode}" help:"${help_mode}"                           enum:"${enum_mode}"`
	Auth        bool   `default:"true"            help:"Enable authentication (on by default)." negatable:""`
	StateDir    string `default:"."               help:"Process state directory."`
	ReplSetName string `default:""                help:"Replica set name."`

	Listen struct {
		Addr        string `default:"127.0.0.1:27017" help:"Listen TCP address for MongoDB protocol."`
		Unix        string `default:""                help:"Listen Unix domain socket path for MongoDB protocol."`
		TLS         string `default:""                help:"Listen TLS address for MongoDB protocol."`
		TLSCertFile string `default:""                help:"TLS cert file path."`
		TLSKeyFile  string `default:""                help:"TLS key file path."`
		TLSCaFile   string `default:""                help:"TLS CA file path."`
		DataAPIAddr string `default:""                help:"Listen TCP address for HTTP Data API."`
	} `embed:"" prefix:"listen-"`

	Proxy struct {
		Addr        string `default:"" help:"Proxy address."`
		TLSCertFile string `default:"" help:"Proxy TLS cert file path."`
		TLSKeyFile  string `default:"" help:"Proxy TLS key file path."`
		TLSCaFile   string `default:"" help:"Proxy TLS CA file path."`
	} `embed:"" prefix:"proxy-"`

	DebugAddr string `default:"127.0.0.1:8088" help:"Listen address for HTTP handlers for metrics, pprof, etc."`

	Log struct {
		Level  string `default:"${default_log_level}" help:"${help_log_level}"`
		Format string `default:"console"              help:"${help_log_format}"                     enum:"${enum_log_format}"`
		UUID   bool   `default:"false"                help:"Add instance UUID to all log messages." negatable:""`
	} `embed:"" prefix:"log-"`

	PostgreSQLURL string `name:"postgresql-url" default:"postgres://127.0.0.1:5432/postgres" help:"PostgreSQL URL."`

	MetricsUUID bool `default:"false" help:"Add instance UUID to all metrics." negatable:""`

	OTel struct {
		Traces struct {
			URL string `default:"" help:"OpenTelemetry OTLP/HTTP traces endpoint URL (e.g. 'http://host:4318/v1/traces')."`
		} `embed:"" prefix:"traces-"`
	} `embed:"" prefix:"otel-"`

	Telemetry telemetry.Flag `default:"undecided" help:"${help_telemetry}"`

	Dev struct {
		RecordsDir string `hidden:""`

		Telemetry struct {
			URL            string        `default:"https://beacon.ferretdb.com/" hidden:""`
			UndecidedDelay time.Duration `default:"1h"                           hidden:""`
			ReportInterval time.Duration `default:"24h"                          hidden:""`
			Package        string        `default:""                             hidden:""`
		} `embed:"" prefix:"telemetry-"`
	} `embed:"" prefix:"dev-"`
}

// Additional variables for [kong.Parse].
var (
	logLevels = []string{
		slog.LevelDebug.String(),
		slog.LevelInfo.String(),
		slog.LevelWarn.String(),
		slog.LevelError.String(),
	}

	logFormats = []string{"console", "text", "json"}

	kongOptions = []kong.Option{
		kong.Vars{
			"default_log_level": defaultLogLevel().String(),
			"default_mode":      clientconn.AllModes[0],

			"enum_log_format": strings.Join(logFormats, ","),
			"enum_mode":       strings.Join(clientconn.AllModes, ","),

			"help_log_format": fmt.Sprintf("Log format: '%s'.", strings.Join(logFormats, "', '")),
			"help_log_level":  fmt.Sprintf("Log level: '%s'.", strings.Join(logLevels, "', '")),
			"help_mode":       fmt.Sprintf("Operation mode: '%s'.", strings.Join(clientconn.AllModes, "', '")),
			"help_telemetry":  "Enable or disable basic telemetry reporting. See https://beacon.ferretdb.com.",
		},
		kong.DefaultEnvars("FERRETDB"),
	}
)

func main() {
	ctx := kong.Parse(&cli, kongOptions...)

	switch ctx.Command() {
	case "run":
		run()

	case "ping":
		logger := setupLogger(cli.Log.Format, "")
		checkFlags(logger)

		ready := ReadyZ{
			l: logger,
		}

		ctx, stop := ctxutil.SigTerm(context.Background())
		defer stop()

		if !ready.Probe(ctx) {
			os.Exit(1)
		}

	default:
		panic("unknown sub-command")
	}
}

// defaultLogLevel returns the default log level.
func defaultLogLevel() slog.Level {
	if version.Get().DevBuild {
		return slog.LevelDebug
	}

	return slog.LevelInfo
}

// setupState setups state provider.
func setupState() *state.Provider {
	if cli.StateDir == "" || cli.StateDir == "-" {
		log.Fatal("State directory must be set.")
	}

	f, err := filepath.Abs(filepath.Join(cli.StateDir, "state.json"))
	if err != nil {
		log.Fatalf("Failed to get path for state file: %s.", err)
	}

	sp, err := state.NewProvider(f)
	if err != nil {
		log.Fatal(stateFileProblem(f, err))
	}

	return sp
}

// setupMetrics setups Prometheus metrics registerer with some metrics.
func setupMetrics(stateProvider *state.Provider) prometheus.Registerer {
	r := prometheus.DefaultRegisterer
	m := stateProvider.MetricsCollector(true)

	// we don't do it by default due to
	// https://prometheus.io/docs/instrumenting/writing_exporters/#target-labels-not-static-scraped-labels
	if cli.MetricsUUID {
		r = prometheus.WrapRegistererWith(
			prometheus.Labels{"uuid": stateProvider.Get().UUID},
			prometheus.DefaultRegisterer,
		)
		m = stateProvider.MetricsCollector(false)
	}

	r.MustRegister(m)

	return r
}

// setupLogger setups slog logger.
func setupLogger(format string, uuid string) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cli.Log.Level)); err != nil {
		log.Fatal(err)
	}

	opts := &logging.NewHandlerOpts{
		Base:          format,
		Level:         level,
		CheckMessages: false, // TODO https://github.com/FerretDB/FerretDB/issues/4511
	}
	logging.Setup(opts, uuid)
	logger := slog.Default()

	return logger
}

// checkFlags checks that CLI flags are not self-contradictory.
func checkFlags(logger *slog.Logger) {
	ctx := context.Background()

	if devbuild.Enabled {
		logger.WarnContext(ctx, "This is a development build. The performance will be affected.")
	}

	if logger.Enabled(ctx, slog.LevelDebug) {
		logger.WarnContext(ctx, "Debug logging enabled. The performance will be affected.")
	}

	if !cli.Auth {
		logger.WarnContext(ctx, "Authentication is disabled. The server will accept any connection.")
	}
}

// dumpMetrics dumps all Prometheus metrics to stderr.
func dumpMetrics() {
	mfs := must.NotFail(prometheus.DefaultGatherer.Gather())

	for _, mf := range mfs {
		must.NotFail(expfmt.MetricFamilyToText(os.Stderr, mf))
	}
}

// run sets up environment based on provided flags and runs FerretDB.
func run() {
	// to increase a chance of resource finalizers to spot problems
	if devbuild.Enabled {
		defer func() {
			runtime.GC()
			runtime.GC()
		}()
	}

	info := version.Get()

	if p := cli.Dev.Telemetry.Package; p != "" {
		info.Package = p
	}

	if cli.Version {
		_, _ = fmt.Fprintln(os.Stdout, "version:", info.Version)
		_, _ = fmt.Fprintln(os.Stdout, "commit:", info.Commit)
		_, _ = fmt.Fprintln(os.Stdout, "branch:", info.Branch)
		_, _ = fmt.Fprintln(os.Stdout, "dirty:", info.Dirty)
		_, _ = fmt.Fprintln(os.Stdout, "package:", info.Package)
		_, _ = fmt.Fprintln(os.Stdout, "devBuild:", info.DevBuild)

		return
	}

	// safe to always enable
	runtime.SetBlockProfileRate(10000)

	stateProvider := setupState()

	metricsRegisterer := setupMetrics(stateProvider)

	startupFields := []slog.Attr{
		slog.String("version", info.Version),
		slog.String("commit", info.Commit),
		slog.String("branch", info.Branch),
		slog.Bool("dirty", info.Dirty),
		slog.String("package", info.Package),
		slog.Bool("devBuild", info.DevBuild),
		slog.Any("buildEnvironment", info.BuildEnvironment),
	}
	logUUID := stateProvider.Get().UUID

	// Similarly to Prometheus, unless requested, don't add UUID to all messages, but log it once at startup.
	if !cli.Log.UUID {
		startupFields = append(startupFields, slog.String("uuid", logUUID))
		logUUID = ""
	}

	logger := setupLogger(cli.Log.Format, logUUID)

	logger.LogAttrs(context.Background(), slog.LevelInfo, "Starting FerretDB "+info.Version+"...", startupFields...)

	checkFlags(logger)

	maxprocsOpts := []maxprocs.Option{
		maxprocs.Min(2),
		maxprocs.RoundQuotaFunc(func(v float64) int {
			return int(math.Ceil(v))
		}),
		maxprocs.Logger(func(format string, a ...any) {
			logger.Info(fmt.Sprintf(format, a...))
		}),
	}
	if _, err := maxprocs.Set(maxprocsOpts...); err != nil {
		logger.Warn("Failed to set GOMAXPROCS", logging.Error(err))
	}

	ctx, stop := ctxutil.SigTerm(context.Background())

	go func() {
		<-ctx.Done()
		logger.InfoContext(ctx, "Stopping")

		// second SIGTERM should immediately stop the process
		stop()
	}()

	// used to start debug handler with probes as soon as possible, even before listener is created
	var listener atomic.Pointer[clientconn.Listener]

	var wg sync.WaitGroup

	if addr := cli.DebugAddr; addr != "" && addr != "-" {
		wg.Add(1)

		go func() {
			defer wg.Done()

			l := logging.WithName(logger, "debug")
			ready := ReadyZ{
				l: l,
			}

			h, err := debug.Listen(&debug.ListenOpts{
				TCPAddr: addr,
				L:       l,
				R:       metricsRegisterer,
				Livez: func(context.Context) bool {
					if listener.Load() == nil {
						return false
					}

					return listener.Load().Listening()
				},

				Readyz: ready.Probe,
			})
			if err != nil {
				l.LogAttrs(ctx, logging.LevelFatal, "Failed to create debug handler", logging.Error(err))
			}

			h.Serve(ctx)
		}()
	}

	if u := cli.OTel.Traces.URL; u != "" && u != "-" {
		wg.Add(1)

		go func() {
			defer wg.Done()

			l := logging.WithName(logger, "otel")

			ot, err := observability.NewOTelTraceExporter(&observability.OTelTraceExporterOpts{
				Logger:  l,
				Service: "ferretdb",
				Version: version.Get().Version,
				URL:     u,
			})
			if err != nil {
				l.LogAttrs(ctx, logging.LevelFatal, "Failed to create Otel tracer", logging.Error(err))
			}

			ot.Run(ctx)
		}()
	}

	metrics := connmetrics.NewListenerMetrics()

	{
		wg.Add(1)

		go func() {
			defer wg.Done()

			l := logging.WithName(logger, "telemetry")

			file, err := filepath.Abs(filepath.Join(cli.StateDir, "telemetry.json"))
			if err != nil {
				l.LogAttrs(ctx, logging.LevelFatal, "Failed to get path for local telemetry report file", logging.Error(err))
			}

			r, err := telemetry.NewReporter(&telemetry.NewReporterOpts{
				URL:            cli.Dev.Telemetry.URL,
				File:           file,
				F:              &cli.Telemetry,
				DNT:            os.Getenv("DO_NOT_TRACK"),
				ExecName:       os.Args[0],
				P:              stateProvider,
				ConnMetrics:    metrics.ConnMetrics,
				L:              l,
				UndecidedDelay: cli.Dev.Telemetry.UndecidedDelay,
				ReportInterval: cli.Dev.Telemetry.ReportInterval,
			})
			if err != nil {
				l.LogAttrs(ctx, logging.LevelFatal, "Failed to create telemetry reporter", logging.Error(err))
			}

			r.Run(ctx)
		}()
	}

	p, err := documentdb.NewPool(cli.PostgreSQLURL, logging.WithName(logger, "pool"), stateProvider)
	if err != nil {
		logger.LogAttrs(ctx, logging.LevelFatal, "Failed to construct pool", logging.Error(err))
	}

	defer p.Close()

	handlerOpts := &handler.NewOpts{
		Pool: p,
		Auth: cli.Auth,

		TCPHost:     cli.Listen.Addr,
		ReplSetName: cli.ReplSetName,

		L:             logging.WithName(logger, "handler"),
		ConnMetrics:   metrics.ConnMetrics,
		StateProvider: stateProvider,
	}

	h, err := handler.New(handlerOpts)
	if err != nil {
		handlerOpts.L.LogAttrs(ctx, logging.LevelFatal, "Failed to construct handler", logging.Error(err))
	}

	lis, err := clientconn.Listen(&clientconn.NewListenerOpts{
		TCP:  cli.Listen.Addr,
		Unix: cli.Listen.Unix,

		TLS:         cli.Listen.TLS,
		TLSCertFile: cli.Listen.TLSCertFile,
		TLSKeyFile:  cli.Listen.TLSKeyFile,
		TLSCAFile:   cli.Listen.TLSCaFile,

		ProxyAddr:        cli.Proxy.Addr,
		ProxyTLSCertFile: cli.Proxy.TLSCertFile,
		ProxyTLSKeyFile:  cli.Proxy.TLSKeyFile,
		ProxyTLSCAFile:   cli.Proxy.TLSCaFile,

		Mode:           clientconn.Mode(cli.Mode),
		Metrics:        metrics,
		Handler:        h,
		Logger:         logger,
		TestRecordsDir: cli.Dev.RecordsDir,
	})
	if err != nil {
		logger.LogAttrs(ctx, logging.LevelFatal, "Failed to construct listener", logging.Error(err))
	}

	if addr := cli.Listen.DataAPIAddr; addr != "" && addr != "-" {
		wg.Add(1)

		go func() {
			defer wg.Done()

			l := logging.WithName(logger, "dataapi")

			var lis *dataapi.Listener

			lis, err = dataapi.Listen(&dataapi.ListenOpts{
				TCPAddr: addr,
				L:       l,
				Handler: h,
			})
			if err != nil {
				l.LogAttrs(ctx, logging.LevelFatal, "Failed to construct DataAPI listener", logging.Error(err))
			}

			lis.Run(ctx)
		}()
	}

	listener.Store(lis)

	metricsRegisterer.MustRegister(lis)

	lis.Run(ctx)

	wg.Wait()

	if info.DevBuild {
		dumpMetrics()
	}
}

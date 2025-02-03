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

// Package main contains code generator for DocumentDB APIs.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strings"

	"github.com/FerretDB/FerretDB/v2/internal/util/logging"
)

func main() {
	debugF := flag.Bool("debug", false, "enable debug logging")
	schemasF := flag.String("schemas", "", "comma-separated list of schemas")
	flag.Parse()

	opts := &logging.NewHandlerOpts{
		Base:          "console",
		Level:         slog.LevelInfo,
		CheckMessages: true,
	}

	if *debugF {
		opts.Level = slog.LevelDebug
	}

	logging.Setup(opts, "")

	l := slog.Default()
	ctx := context.Background()

	if *schemasF == "" {
		l.Log(ctx, logging.LevelFatal, "-schemas flag is empty.")
	}

	// DOCUMENTDB_GEN_URL=postgres://username:password@127.0.0.1:5432/postgres
	uri := os.Getenv("DOCUMENTDB_GEN_URL")
	if uri == "" {
		l.InfoContext(ctx, "DOCUMENTDB_GEN_URL not set, skipping code generation.")
		os.Exit(0)
	}

	rows, err := Extract(ctx, uri, strings.Split(*schemasF, ","))
	if err != nil {
		l.Log(ctx, logging.LevelFatal, err.Error())
	}

	schemaRoutines := Convert(rows, l)

	if err = Generate(schemaRoutines); err != nil {
		l.Log(ctx, logging.LevelFatal, err.Error())
	}
}

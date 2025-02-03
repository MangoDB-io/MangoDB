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
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"

	"github.com/FerretDB/FerretDB/v2/build/version"
	"github.com/FerretDB/FerretDB/v2/internal/util/ctxutil"
	"github.com/FerretDB/FerretDB/v2/internal/util/logging"
)

var (
	//go:embed error.tmpl
	errorTemplateB []byte

	// Parsed error template.
	errorTemplate = template.Must(template.New("error").Option("missingkey=error").Parse(string(errorTemplateB)))
)

// versionFile contains version information with leading v.
const versionFile = "build/version/version.txt"

// runCommand runs command with given arguments.
func runCommand(command string, args []string, stdout io.Writer, logger *slog.Logger) error {
	bin, err := exec.LookPath(command)
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, args...)
	logger.Debug(fmt.Sprintf("Running %s", strings.Join(cmd.Args, " ")))

	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %s", strings.Join(cmd.Args, " "), err)
	}

	return nil
}

// printDiagnosticData prints diagnostic data and error template on stdout.
func printDiagnosticData(w io.Writer, setupError error, logger *slog.Logger) error {
	_ = runCommand("docker", []string{"compose", "logs"}, w, logger)

	_ = runCommand("docker", []string{"compose", "ps", "--all"}, w, logger)

	_ = runCommand("docker", []string{"stats", "--all", "--no-stream"}, w, logger)

	var buf bytes.Buffer

	var gitVersion string
	if err := runCommand("git", []string{"version"}, &buf, logger); err != nil {
		gitVersion = err.Error()
	} else {
		gitVersion = buf.String()
	}

	buf.Reset()

	var dockerVersion string
	if err := runCommand("docker", []string{"version"}, &buf, logger); err != nil {
		dockerVersion = err.Error()
	} else {
		dockerVersion = buf.String()
	}

	buf.Reset()

	var composeVersion string
	if err := runCommand("docker", []string{"compose", "version"}, &buf, logger); err != nil {
		composeVersion = err.Error()
	} else {
		composeVersion = buf.String()
	}

	info := version.Get()

	return errorTemplate.Execute(w, map[string]any{
		"Error": setupError,

		"GOOS":   runtime.GOOS,
		"GOARCH": runtime.GOARCH,

		"Version":  info.Version,
		"Commit":   info.Commit,
		"Branch":   info.Branch,
		"Dirty":    info.Dirty,
		"Package":  info.Package,
		"DevBuild": info.DevBuild,

		"GoVersion":      runtime.Version(),
		"GitVersion":     strings.TrimSpace(gitVersion),
		"DockerVersion":  strings.TrimSpace(dockerVersion),
		"ComposeVersion": strings.TrimSpace(composeVersion),

		"NewIssueURL": "https://github.com/FerretDB/FerretDB/issues/new/choose",
	})
}

// shellMkDir creates all directories from given paths.
func shellMkDir(paths ...string) error {
	var errs error

	for _, path := range paths {
		if err := os.MkdirAll(path, 0o777); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

// shellRmDir removes all directories from given paths.
func shellRmDir(paths ...string) error {
	var errs error

	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

// shellRead will show the content of a file.
func shellRead(w io.Writer, paths ...string) error {
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		fmt.Fprint(w, string(b))
	}

	return nil
}

// packageVersion will print out FerretDB's package version (omitting leading v).
func packageVersion(w io.Writer, file string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	v := string(b)
	v = strings.TrimPrefix(v, "v")

	_, err = fmt.Fprint(w, v)

	return err
}

// TestsRunParams represents `envtool tests run` parameters.
//
//nolint:vet // for readability
type TestsRunParams struct {
	ShardIndex uint   `help:"Shard index, starting from 1."`
	ShardTotal uint   `help:"Total number of shards."`
	Run        string `help:"Run only tests matching the regexp."`
	Skip       string `help:"Skip tests matching the regexp."`
	RawPrefix  string `help:"Prefix for raw output files."`

	Args []string `arg:"" help:"Other arguments and flags for 'go test'." passthrough:""`
}

// cli struct represents all command-line commands, fields and flags.
// It's used for parsing the user input.
//
//nolint:vet // for readability
var cli struct {
	Debug bool `help:"Enable debug mode."`

	Setup struct{} `cmd:"" help:"Setup development environment."`

	PackageVersion struct{} `cmd:"" help:"Print package version."`

	Shell struct {
		Mkdir struct {
			Paths []string `name:"path" arg:"" help:"Paths to create." type:"path"`
		} `cmd:"" help:"Create directories if they do not already exist."`
		Rmdir struct {
			Paths []string `name:"path" arg:"" help:"Paths to remove." type:"path"`
		} `cmd:"" help:"Remove directories."`
		Read struct {
			Paths []string `name:"path" arg:"" help:"Paths to read." type:"path"`
		} `cmd:"" help:"Read files."`
	} `cmd:""`

	Tests struct {
		Run TestsRunParams `cmd:"" help:"Run tests."`
	} `cmd:""`

	Fuzz struct {
		Corpus struct {
			Src string `arg:"" help:"Source, one of: 'seed', 'generated', or collected corpus' directory."`
			Dst string `arg:"" help:"Destination, one of: 'seed', 'generated', or collected corpus' directory."`
		} `cmd:"" help:"Sync fuzz corpora."`
	} `cmd:""`
}

func main() {
	kongCtx := kong.Parse(&cli, kong.DefaultEnvars("ENVTOOL"))

	// https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables
	if t, _ := strconv.ParseBool(os.Getenv("RUNNER_DEBUG")); t {
		cli.Debug = true
	}

	level := slog.LevelInfo
	if cli.Debug {
		level = slog.LevelDebug
	}

	opts := &logging.NewHandlerOpts{
		Base:          "console",
		Level:         level,
		RemoveTime:    true,
		RemoveSource:  true,
		CheckMessages: false, // TODO https://github.com/FerretDB/FerretDB/issues/4511
	}
	logging.Setup(opts, "")
	logger := slog.Default()

	cmd := kongCtx.Command()
	logger.Debug(fmt.Sprintf("Command: %q", cmd))

	var err error

	switch cmd {
	case "setup":
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		err = setup(ctx, logger)

	case "package-version":
		err = packageVersion(os.Stdout, versionFile)

	case "shell mkdir <path>":
		err = shellMkDir(cli.Shell.Mkdir.Paths...)
	case "shell rmdir <path>":
		err = shellRmDir(cli.Shell.Rmdir.Paths...)
	case "shell read <path>":
		err = shellRead(os.Stdout, cli.Shell.Read.Paths...)

	case "tests run <args>":
		ctx, stop := ctxutil.SigTerm(context.Background())
		defer stop()

		err = testsRun(ctx, &cli.Tests.Run, logger)

	case "fuzz corpus <src> <dst>":
		var seedCorpus, generatedCorpus string

		if seedCorpus, err = os.Getwd(); err != nil {
			logger.LogAttrs(context.Background(), logging.LevelFatal, "Failed to get current directory", logging.Error(err))
		}

		if generatedCorpus, err = fuzzGeneratedCorpus(); err != nil {
			logger.LogAttrs(context.Background(), logging.LevelFatal, "Failed to generate fuzz corpus", logging.Error(err))
		}

		var src, dst string

		switch cli.Fuzz.Corpus.Src {
		case "seed":
			src = seedCorpus
		case "generated":
			src = generatedCorpus
		default:
			if src, err = filepath.Abs(cli.Fuzz.Corpus.Src); err != nil {
				logger.LogAttrs(context.Background(), logging.LevelFatal, "Unknown fuzz corpus source", logging.Error(err))
			}
		}

		switch cli.Fuzz.Corpus.Dst {
		case "seed":
			// Because we would need to add `/testdata/fuzz` back, and that's not very easy.
			logger.LogAttrs(
				context.Background(),
				logging.LevelFatal,
				"Copying to seed corpus is not supported",
				logging.Error(err),
			)
		case "generated":
			dst = generatedCorpus
		default:
			dst, err = filepath.Abs(cli.Fuzz.Corpus.Dst)
			if err != nil {
				logger.LogAttrs(context.Background(), logging.LevelFatal, "Unknown fuzz corpus destination", logging.Error(err))
			}
		}

		err = fuzzCopyCorpus(src, dst, logger)

	default:
		err = fmt.Errorf("unknown command: %s", cmd)
	}

	if err != nil {
		if cmd == "setup" {
			_ = printDiagnosticData(os.Stderr, err, logger)
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			logger.Error("Failed", logging.Error(exitErr))
			os.Exit(exitErr.ExitCode())
		}

		logger.LogAttrs(context.Background(), logging.LevelFatal, "Failed unexpectedly", logging.Error(err))
	}
}

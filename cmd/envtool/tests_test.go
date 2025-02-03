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
	"log/slog"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FerretDB/FerretDB/v2/internal/util/logging"
	"github.com/FerretDB/FerretDB/v2/internal/util/testutil"
)

var (
	cleanupTimingRe    = regexp.MustCompile(`(\d+\.\d+)s`)                  // duration are different
	cleanupGoroutineRe = regexp.MustCompile(`goroutine (\d+)`)              // goroutine IDs are different
	cleanupPathRe      = regexp.MustCompile(`\t(.+)/cmd/envtool/testdata/`) // absolute file paths are different
	cleanupAddrRe      = regexp.MustCompile(` (\+0x[0-9a-f]+)$`)            // addresses are different
)

// toLogLines takes buffer and removes variable parts of the output.
func toLogLines(buf *bytes.Buffer) []string {
	s := strings.TrimSpace(buf.String())
	lines := strings.Split(s, "\n")

	for i, line := range lines {
		if loc := cleanupTimingRe.FindStringSubmatchIndex(line); loc != nil {
			line = line[:loc[2]] + "<SEC>" + line[loc[3]:]
		}

		if loc := cleanupGoroutineRe.FindStringSubmatchIndex(line); loc != nil {
			line = line[:loc[2]] + "<ID>" + line[loc[3]:]
		}

		if loc := cleanupPathRe.FindStringSubmatchIndex(line); loc != nil {
			line = line[:loc[2]] + "<DIR>" + line[loc[3]:]
		}

		if loc := cleanupAddrRe.FindStringSubmatchIndex(line); loc != nil {
			line = line[:loc[2]] + "<ADDR>" + line[loc[3]:]
		}

		lines[i] = line
	}

	return lines
}

// bufLogger returns logger that writes to buffer.
func bufLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	h := logging.NewHandler(&buf, &logging.NewHandlerOpts{
		Base:          "console",
		Level:         slog.LevelInfo,
		RemoveTime:    true,
		RemoveSource:  true,
		CheckMessages: false, // TODO https://github.com/FerretDB/FerretDB/issues/4511
	})

	return slog.New(h), &buf
}

func TestTestArgs(t *testing.T) {
	t.Parallel()

	ctx := testutil.Ctx(t)

	list, listErr := listTestFuncs(ctx, "testdata", ".", testutil.Logger(t))
	require.NoError(t, listErr)
	require.Equal(
		t,
		[]string{"TestError1", "TestError2", "TestNormal1", "TestNormal2", "TestPanic1", "TestSkip1", "TestWithSubtest"},
		list,
	)

	t.Run("All", func(t *testing.T) {
		t.Parallel()

		actual, total, err := testArgs(ctx, "testdata", 0, 0, "", "", testutil.Logger(t))
		require.NoError(t, err)
		expected := []string{"-run=^(TestError1|TestError2|TestNormal1|TestNormal2|TestPanic1|TestSkip1|TestWithSubtest)$"}
		assert.Equal(t, expected, actual)
		assert.EqualValues(t, 7, total)
	})

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		actual, total, err := testArgs(ctx, "testdata", 0, 0, "(?i)Normal", "", testutil.Logger(t))
		require.NoError(t, err)
		expected := []string{"-run=^(TestNormal1|TestNormal2)$"}
		assert.Equal(t, expected, actual)
		assert.EqualValues(t, 2, total)
	})

	t.Run("Subtest", func(t *testing.T) {
		t.Parallel()

		actual, total, err := testArgs(ctx, "testdata", 0, 0, "TestWithSubtest/Second", "", testutil.Logger(t))
		require.NoError(t, err)
		expected := []string{"-run=TestWithSubtest/Second"}
		assert.Equal(t, expected, actual)
		assert.EqualValues(t, 0, total)
	})

	t.Run("Shard", func(t *testing.T) {
		t.Parallel()

		actual, total, err := testArgs(ctx, "testdata", 1, 2, "", "", testutil.Logger(t))
		require.NoError(t, err)
		expected := []string{"-run=^(TestError1|TestNormal1|TestPanic1|TestWithSubtest)$"}
		assert.Equal(t, expected, actual)
		assert.EqualValues(t, 4, total)
	})
}

func TestRunGoTest(t *testing.T) {
	t.Parallel()

	ctx := testutil.Ctx(t)

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		l, buf := bufLogger()

		err := runGoTest(ctx, &runGoTestOpts{
			args:   []string{"./testdata", "-count=1", "-run=TestNormal"},
			total:  2,
			logger: l,
		})
		require.NoError(t, err)

		expected := []string{
			"INFO	Running go test -json ./testdata -count=1 -run=TestNormal",
			"INFO	PASS TestNormal1 1/2",
			"INFO	PASS TestNormal2 2/2",
			"INFO	PASS",
			"INFO	ok  	github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata	<SEC>s",
			"INFO	PASS github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata",
		}

		actual := toLogLines(buf)
		assert.Equal(t, expected, actual, "actual:\n%s", actual)
	})

	t.Run("SubtestsPartial", func(t *testing.T) {
		t.Parallel()

		l, buf := bufLogger()

		err := runGoTest(ctx, &runGoTestOpts{
			args:   []string{"./testdata", "-count=1", "-run=TestWithSubtest/Third"},
			total:  1,
			logger: l,
		})
		require.NoError(t, err)

		expected := []string{
			"INFO	Running go test -json ./testdata -count=1 -run=TestWithSubtest/Third",
			"INFO	PASS TestWithSubtest 1/1",
			"INFO	PASS",
			"INFO	ok  	github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata	<SEC>s",
			"INFO	PASS github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata",
		}

		actual := toLogLines(buf)
		assert.Equal(t, expected, actual, "actual:\n%s", strings.Join(actual, "\n"))
	})

	t.Run("SubtestsNotFound", func(t *testing.T) {
		t.Parallel()

		l, buf := bufLogger()

		err := runGoTest(ctx, &runGoTestOpts{
			args:   []string{"./testdata", "-count=1", "-run=TestWithSubtest/None"},
			total:  1,
			logger: l,
		})
		require.NoError(t, err)

		expected := []string{
			"INFO	Running go test -json ./testdata -count=1 -run=TestWithSubtest/None",
			"INFO	PASS TestWithSubtest 1/1",
			"INFO	testing: warning: no tests to run",
			"INFO	PASS",
			"INFO	ok  	github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata	<SEC>s [no tests to run]",
			"INFO	PASS github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata",
		}

		actual := toLogLines(buf)
		assert.Equal(t, expected, actual, "actual:\n%s", strings.Join(actual, "\n"))
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		l, buf := bufLogger()

		err := runGoTest(ctx, &runGoTestOpts{
			args:   []string{"./testdata", "-count=1", "-run=TestError"},
			total:  2,
			logger: l,
		})

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 1, exitErr.ExitCode())

		expected := []string{
			"INFO	Running go test -json ./testdata -count=1 -run=TestError",
			"WARN	FAIL TestError1 1/2:",
			"WARN	=== RUN   TestError1",
			"WARN	    error_test.go:20: not hidden 1",
			"WARN	    error_test.go:22: Error 1",
			"WARN	    error_test.go:24: not hidden 2",
			"WARN	--- FAIL: TestError1 (<SEC>s)",
			"WARN	",
			"WARN	FAIL TestError2/Parallel:",
			"WARN	=== RUN   TestError2/Parallel",
			"WARN	    error_test.go:35: not hidden 5",
			"WARN	=== PAUSE TestError2/Parallel",
			"WARN	=== CONT  TestError2/Parallel",
			"WARN	    error_test.go:39: not hidden 6",
			"WARN	    error_test.go:41: Error 2",
			"WARN	    error_test.go:43: not hidden 7",
			"WARN	--- FAIL: TestError2/Parallel (<SEC>s)",
			"WARN	",
			"WARN	FAIL TestError2 2/2:",
			"WARN	=== RUN   TestError2",
			"WARN	    error_test.go:28: not hidden 3",
			"WARN	=== PAUSE TestError2",
			"WARN	=== CONT  TestError2",
			"WARN	    error_test.go:32: not hidden 4",
			"WARN	--- FAIL: TestError2 (<SEC>s)",
			"WARN	",
			"INFO	FAIL",
			"INFO	FAIL	github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata	<SEC>s",
			"INFO	FAIL github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata",
		}

		actual := toLogLines(buf)
		assert.Equal(t, expected, actual, "actual:\n%s", strings.Join(actual, "\n"))
	})

	t.Run("Skip", func(t *testing.T) {
		t.Parallel()

		l, buf := bufLogger()

		err := runGoTest(ctx, &runGoTestOpts{
			args:   []string{"./testdata", "-count=1", "-run=TestSkip"},
			total:  1,
			logger: l,
		})
		require.NoError(t, err)

		expected := []string{
			"INFO	Running go test -json ./testdata -count=1 -run=TestSkip",
			"WARN	SKIP TestSkip1 1/1:",
			"WARN	=== RUN   TestSkip1",
			"WARN	    skip_test.go:20: not hidden 1",
			"WARN	    skip_test.go:22: Skip 1",
			"WARN	--- SKIP: TestSkip1 (<SEC>s)",
			"WARN	",
			"INFO	PASS",
			"INFO	ok  	github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata	<SEC>s",
			"INFO	PASS github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata",
		}

		actual := toLogLines(buf)
		assert.Equal(t, expected, actual, "actual:\n%s", strings.Join(actual, "\n"))
	})

	t.Run("Panic", func(t *testing.T) {
		t.Parallel()

		l, buf := bufLogger()

		err := runGoTest(ctx, &runGoTestOpts{
			args:   []string{"./testdata", "-count=1", "-run=TestPanic"},
			total:  1,
			logger: l,
		})
		require.Error(t, err)

		expected := []string{
			"INFO	Running go test -json ./testdata -count=1 -run=TestPanic",
			"INFO	FAIL	github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata	<SEC>s",
			"INFO	FAIL github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata",
			"ERROR	",
			"ERROR	Some tests did not finish:",
			"ERROR	  github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata.TestPanic1",
			"ERROR	",
			"ERROR	github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata.TestPanic1:",
			"ERROR	=== RUN   TestPanic1",
			"ERROR	panic: Panic 1",
			"ERROR	",
			"ERROR	goroutine <ID> [running]:",
			"ERROR	github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata.TestPanic1.func1()",
			"ERROR	<DIR>/cmd/envtool/testdata/panic_test.go:25 <ADDR>",
			"ERROR	created by github.com/FerretDB/FerretDB/v2/cmd/envtool/testdata.TestPanic1 in goroutine <ID>",
			"ERROR	<DIR>/cmd/envtool/testdata/panic_test.go:23 <ADDR>",
			"ERROR",
		}

		actual := toLogLines(buf)
		assert.Equal(t, expected, actual, "actual:\n%s", strings.Join(actual, "\n"))
	})
}

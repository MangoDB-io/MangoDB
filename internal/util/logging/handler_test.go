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

package logging

import (
	"bytes"
	"context"
	"encoding/hex"
	"log/slog"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pc, _, _, _ := runtime.Caller(0)
	r := slog.NewRecord(time.Date(2024, 5, 31, 9, 26, 42, 0, time.UTC), slog.LevelInfo, "multi\nline\nmessage", pc)

	r.AddAttrs(slog.Group("g1", slog.Int("k1", 42), slog.Duration("k2", 7*time.Second)))
	r.AddAttrs(slog.Group("g4", slog.Group("g5")))
	r.AddAttrs(slog.String("k3", "s"))
	r.AddAttrs(slog.String("k3", "dup"))

	for base, expected := range map[string]string{
		"console": `2024-05-31T09:26:42.000Z	INFO	logging/handler_test.go:34	` +
			"multi\nline\nmessage\t" +
			`{"g2":{"g1":{"k1":42,"k2":7000000000},"g3":{"s":"a"},"i":1,"k3":"dup","name":"test.logger"}}` + "\n",
		// TODO
		"mongo": `{"t":{"$date":"2024-05-31T09:26:42Z"},"s":"I"}` + "\n",
		"text": `time=2024-05-31T09:26:42.000Z level=INFO source=logging/handler_test.go:34 ` +
			`msg="multi\nline\nmessage" ` +
			`g2.i=1 g2.g3.s=a g2.name=test.logger g2.g1.k1=42 g2.g1.k2=7s g2.k3=s g2.k3=dup` + "\n",
		"json": `{"time":"2024-05-31T09:26:42Z","level":"INFO","source":` +
			`{"function":"github.com/FerretDB/FerretDB/v2/internal/util/logging.TestHandler",` +
			`"file":"logging/handler_test.go","line":34},"msg":"multi\nline\nmessage"` +
			`,"g2":{"i":1,"g3":{"s":"a"},"name":"test.logger","g1":{"k1":42,"k2":7000000000},"k3":"s","k3":"dup"}}` + "\n",
	} {
		t.Run(base, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			var h slog.Handler = NewHandler(&buf, &NewHandlerOpts{
				Base:          base,
				Level:         slog.LevelInfo,
				CheckMessages: true,
			})

			h = h.WithGroup("g2")
			h = h.WithAttrs([]slog.Attr{
				slog.Int("i", 1),
			})
			h = h.WithAttrs([]slog.Attr{
				slog.Group("g3", slog.String("s", "a")),
			})

			l := WithName(slog.New(h), "test.logger")
			require.NoError(t, l.Handler().Handle(ctx, r))

			assert.Equal(t, expected, buf.String(), "actual:\n%s", hex.Dump(buf.Bytes()))
		})
	}
}

func TestShortPath(t *testing.T) {
	t.Parallel()

	// without -trimpath
	assert.Equal(t, "file.go", shortPath("/file.go"))
	assert.Equal(t, "dir1/file.go", shortPath("/dir1/file.go"))
	assert.Equal(t, "dir2/file.go", shortPath("/dir1/dir2/file.go"))
	assert.Equal(t, "dir3/file.go", shortPath("/dir1/dir2/dir3/file.go"))

	// with -trimpath
	assert.Equal(t, "debug/debug.go", shortPath("github.com/FerretDB/FerretDB/internal/util/debug/debug.go"))
	assert.Equal(t, "util/debug.go", shortPath("github.com/FerretDB/FerretDB/internal/util/debug.go"))
	assert.Equal(t, "internal/debug.go", shortPath("github.com/FerretDB/FerretDB/internal/debug.go"))
}

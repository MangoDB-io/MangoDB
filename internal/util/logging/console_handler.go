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
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"sync"

	"github.com/FerretDB/FerretDB/v2/internal/util/devbuild"
	"github.com/FerretDB/FerretDB/v2/internal/util/must"
)

// timeLayout is the format of date time used by the console handler.
const timeLayout = "2006-01-02T15:04:05.000Z0700"

// consoleHandler is a [slog.Handler] that writes logs to the console.
// The format is intended to be more human-readable than [slog.TextHandler]'s logfmt.
// The format is not stable.
//
// See https://golang.org/s/slog-handler-guide.
//
// TODO https://github.com/FerretDB/FerretDB/issues/4438
//
//nolint:vet // for readability
type consoleHandler struct {
	opts *NewHandlerOpts

	ga []groupOrAttrs

	testAttrs map[string]any

	m   *sync.Mutex
	out io.Writer
}

// groupOrAttrs contains group name or attributes.
type groupOrAttrs struct {
	group string
	attrs []slog.Attr
}

// newConsoleHandler creates a new console handler.
func newConsoleHandler(out io.Writer, opts *NewHandlerOpts, testAttrs map[string]any) *consoleHandler {
	must.NotBeZero(opts)

	return &consoleHandler{
		opts:      opts,
		testAttrs: testAttrs,
		m:         new(sync.Mutex),
		out:       out,
	}
}

// Enabled implements [slog.Handler].
func (ch *consoleHandler) Enabled(_ context.Context, l slog.Level) bool {
	minLevel := slog.LevelInfo
	if ch.opts.Level != nil {
		minLevel = ch.opts.Level.Level()
	}

	return l >= minLevel
}

// Handle implements [slog.Handler].
func (ch *consoleHandler) Handle(ctx context.Context, r slog.Record) error {
	var buf bytes.Buffer

	if !ch.opts.RemoveTime && !r.Time.IsZero() {
		t := r.Time.Format(timeLayout)
		buf.WriteString(t)
		buf.WriteRune('\t')

		if ch.testAttrs != nil {
			ch.testAttrs[slog.TimeKey] = t
		}
	}

	if !ch.opts.RemoveLevel {
		l := r.Level.String()
		buf.WriteString(l)
		buf.WriteRune('\t')

		if ch.testAttrs != nil {
			ch.testAttrs[slog.LevelKey] = l
		}
	}

	if !ch.opts.RemoveSource {
		f, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		if f.File != "" {
			s := shortPath(f.File) + ":" + strconv.Itoa(f.Line)
			buf.WriteString(s)
			buf.WriteRune('\t')

			if ch.testAttrs != nil {
				ch.testAttrs[slog.SourceKey] = s
			}
		}
	}

	if r.Message != "" {
		buf.WriteString(r.Message)

		if ch.testAttrs != nil {
			ch.testAttrs[slog.MessageKey] = r.Message
		}
	}

	if m := ch.toMap(r); len(m) > 0 {
		buf.WriteRune('\t')

		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		encoder.SetEscapeHTML(false)

		err := encoder.Encode(m)
		if devbuild.Enabled || ch.opts.CheckMessages {
			must.NoError(err)
		}

		if err == nil {
			buf.Write(bytes.TrimSuffix(b.Bytes(), []byte{'\n'}))
		} else {
			// last resort
			buf.WriteString(fmt.Sprintf("%#v", m))
		}

		if ch.testAttrs != nil {
			maps.Copy(ch.testAttrs, m)
		}
	}

	buf.WriteRune('\n')

	ch.m.Lock()
	defer ch.m.Unlock()

	_, err := buf.WriteTo(ch.out)

	return err
}

// WithAttrs implements [slog.Handler].
func (ch *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return ch
	}

	return &consoleHandler{
		opts:      ch.opts,
		ga:        append(slices.Clone(ch.ga), groupOrAttrs{attrs: attrs}),
		m:         ch.m,
		out:       ch.out,
		testAttrs: ch.testAttrs,
	}
}

// WithGroup implements [slog.Handler].
func (ch *consoleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return ch
	}

	return &consoleHandler{
		opts:      ch.opts,
		ga:        append(slices.Clone(ch.ga), groupOrAttrs{group: name}),
		m:         ch.m,
		out:       ch.out,
		testAttrs: ch.testAttrs,
	}
}

// toMap converts attributes to a map.
// Attributes with duplicate keys are overwritten, and the order of keys is ignored.
func (ch *consoleHandler) toMap(r slog.Record) map[string]any {
	m := make(map[string]any, r.NumAttrs())

	r.Attrs(func(attr slog.Attr) bool {
		if attr.Key != "" {
			m[attr.Key] = resolve(attr.Value)

			return true
		}

		if attr.Value.Kind() == slog.KindGroup {
			for _, gAttr := range attr.Value.Group() {
				m[gAttr.Key] = resolve(gAttr.Value)
			}
		}

		return true
	})

	for i := len(ch.ga) - 1; i >= 0; i-- {
		if ch.ga[i].group != "" && len(m) > 0 {
			m = map[string]any{ch.ga[i].group: m}

			continue
		}

		for _, attr := range ch.ga[i].attrs {
			m[attr.Key] = resolve(attr.Value)
		}
	}

	return m
}

// resolve returns underlying attribute value, or a map for [slog.KindGroup] type.
func resolve(v slog.Value) any {
	v = v.Resolve()

	if v.Kind() != slog.KindGroup {
		return v.Any()
	}

	g := v.Group()
	m := make(map[string]any, len(g))

	for _, attr := range g {
		m[attr.Key] = resolve(attr.Value)
	}

	return m
}

// check interfaces
var (
	_ slog.Handler = (*consoleHandler)(nil)
)

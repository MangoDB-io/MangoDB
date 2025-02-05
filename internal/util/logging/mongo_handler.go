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
	"io"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"sync"

	"github.com/FerretDB/FerretDB/v2/internal/util/must"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// mongoHandler is a [slog.Handler] that writes logs by using mongo structured JSON format.
// The format returns log entries with Relaxed Extended JSON specification.
type mongoHandler struct {
	opts *NewHandlerOpts

	goas      []groupOrAttrs
	testAttrs map[string]any

	m   *sync.Mutex
	out io.Writer
}

// mongoLog represents a single log message in mongo structured JSON format.
type mongoLog struct {
	Timestamp  primitive.DateTime `bson:"t"`
	Severity   string             `bson:"s"`
	Components string             `bson:"c"`   //TODO
	ID         int                `bson:"id"`  //TODO
	Ctx        string             `bson:"ctx"` // TODO
	Svc        string             `bson:"svc,omitempty"`
	Msg        string             `bson:"msg"`
	Attr       map[string]any     `bson:"attr,omitempty"`
	Tags       []string           `bson:"tags,omitempty"`
	Truncated  bson.D             `bson:"truncated,omitempty"`
	Size       bson.D             `bson:"size,omitempty"`
}

// newMongoHandler creates a new mongo handler.
func newMongoHandler(out io.Writer, opts *NewHandlerOpts, testAttrs map[string]any) *mongoHandler {
	must.NotBeZero(opts)

	h := mongoHandler{
		opts:      opts,
		testAttrs: testAttrs,
		m:         new(sync.Mutex),
		out:       out,
	}

	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}

	return &h
}

// Enabled implements [slog.Handler].
func (h *mongoHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.opts.Level.Level()
}

// Handle implements [slog.Handler].
func (h *mongoHandler) Handle(ctx context.Context, r slog.Record) error {
	var buf bytes.Buffer

	logRecord := mongoLog{
		Msg: r.Message,
	}

	if h.testAttrs != nil {
		h.testAttrs[slog.MessageKey] = logRecord.Msg
	}

	if !h.opts.RemoveLevel {
		logRecord.Severity = h.getSeverity(r.Level)

		if h.testAttrs != nil {
			h.testAttrs[slog.LevelKey] = logRecord.Severity
		}
	}

	if !h.opts.RemoveTime && !r.Time.IsZero() {
		logRecord.Timestamp = primitive.NewDateTimeFromTime(r.Time)

		if h.testAttrs != nil {
			h.testAttrs[slog.TimeKey] = logRecord.Timestamp
		}
	}

	if !h.opts.RemoveSource && r.PC != 0 {
		f, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		if f.File != "" {
			logRecord.Ctx = shortPath(f.File) + ":" + strconv.Itoa(f.Line)

			if h.testAttrs != nil {
				h.testAttrs[slog.SourceKey] = logRecord.Ctx
			}
		}
	}

	logRecord.Attr = h.attrs(r)

	if h.testAttrs != nil && len(logRecord.Attr) > 0 {
		maps.Copy(h.testAttrs, logRecord.Attr)
	}

	extJSON, err := bson.MarshalExtJSON(&logRecord, false, false)
	if err != nil {
		return err
	}

	_, err = buf.Write(extJSON)
	if err != nil {
		return err
	}

	buf.WriteRune('\n')

	h.m.Lock()
	defer h.m.Unlock()

	_, err = buf.WriteTo(h.out)
	return err
}

// attrs returns record attributes, as well as handler attributes in map.
// Attributes with duplicate keys are overwritten, and the order of keys is ignored.
func (h *mongoHandler) attrs(r slog.Record) map[string]any {
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

	for _, goa := range slices.Backward(h.goas) {
		if goa.group != "" && len(m) > 0 {
			m = map[string]any{goa.group: m}
			continue
		}

		for _, attr := range goa.attrs {
			m[attr.Key] = resolve(attr.Value)
		}
	}

	return m
}

// WithAttrs implements [slog.Handler].
func (h *mongoHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return &mongoHandler{
		opts:      h.opts,
		goas:      append(slices.Clone(h.goas), groupOrAttrs{attrs: attrs}),
		testAttrs: h.testAttrs,
		m:         h.m,
		out:       h.out,
	}
}

// WithGroup implements [slog.Handler].
func (h *mongoHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &mongoHandler{
		opts:      h.opts,
		goas:      append(slices.Clone(h.goas), groupOrAttrs{group: name}),
		testAttrs: h.testAttrs,
		m:         h.m,
		out:       h.out,
	}
}

// getSeverity maps logging levels to their mongo format counterparts.
// If provided level is not mapped, its standard format is returned.
func (h *mongoHandler) getSeverity(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "D"
	case slog.LevelInfo:
		return "I"
	case slog.LevelWarn:
		return "W"
	case slog.LevelError:
		return "E"
	default:
		return level.String()
	}
}

// check interfaces
var (
	_ slog.Handler = (*mongoHandler)(nil)
)

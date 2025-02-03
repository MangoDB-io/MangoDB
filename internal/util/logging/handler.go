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
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/FerretDB/wire/wirebson"

	"github.com/FerretDB/FerretDB/v2/internal/util/devbuild"
	"github.com/FerretDB/FerretDB/v2/internal/util/must"
)

// Handler is a [slog.Handler] that wraps another handler with support for:
//   - additional log levels
//     (DPanic/ERROR+1 panics in development builds, Panic/ERROR+2 always panics, Fatal/ERROR+3 exits with a non-zero status);
//   - shorter source locations;
//   - removal of time, level, and source attributes;
//   - message checks for leading/trailing spaces and ending punctuation;
//   - collecting recent log entries for `getLog` command.
type Handler struct {
	base          slog.Handler
	out           io.Writer
	checkMessages bool
	recentEntries *circularBuffer
}

// NewHandlerOpts represents [NewHandler] options.
//
//nolint:vet // for readability
type NewHandlerOpts struct {
	Base         string // base handler to create: "console", "text", or "json"
	Level        slog.Leveler
	RemoveTime   bool
	RemoveLevel  bool
	RemoveSource bool

	// When set, causes handler to panic on messages with leading/trailing spaces or ending punctuation.
	// It must not be set unconditionally because we don't control messages from third-party packages.
	// But we can enable it in our tests and when [devbuild.Enabled] is true.
	// TODO https://github.com/FerretDB/FerretDB/issues/4511
	CheckMessages bool

	// for testing only
	recentEntriesSize int
}

// shortPath returns shorter path for the given path.
func shortPath(path string) string {
	must.NotBeZero(path)

	dir := filepath.Base(filepath.Dir(path))
	if dir == "/" {
		dir = ""
	} else {
		dir += "/"
	}

	return dir + filepath.Base(path)
}

// replaceAttrFunc returns [slog.HandlerOptions]'s ReplaceAttr function
// that modified attributes according to options.
func replaceAttrFunc(opts *NewHandlerOpts) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if groups != nil {
			return a
		}

		switch a.Key {
		case slog.TimeKey:
			if opts.RemoveTime {
				return slog.Attr{}
			}

		case slog.LevelKey:
			if opts.RemoveLevel {
				return slog.Attr{}
			}

		case slog.MessageKey:
			if a.Value.String() == "" {
				return slog.Attr{}
			}

		case slog.SourceKey:
			s, ok := a.Value.Any().(*slog.Source)
			if !ok {
				return a
			}

			if opts.RemoveSource {
				return slog.Attr{}
			}

			if s.File != "" {
				s.File = shortPath(s.File)
			}
		}

		return a
	}
}

// NewHandler creates a new handler with the given options.
func NewHandler(out io.Writer, opts *NewHandlerOpts) *Handler {
	must.NotBeZero(opts)

	var h slog.Handler

	stdOpts := &slog.HandlerOptions{
		AddSource:   !opts.RemoveSource,
		Level:       opts.Level,
		ReplaceAttr: replaceAttrFunc(opts),
	}

	switch opts.Base {
	case "console":
		h = newConsoleHandler(out, opts, nil)
	case "text":
		h = slog.NewTextHandler(out, stdOpts)
	case "json":
		h = slog.NewJSONHandler(out, stdOpts)
	default:
		panic(fmt.Sprintf("invalid base handler %q", opts.Base))
	}

	if opts.recentEntriesSize == 0 {
		opts.recentEntriesSize = 1024
	}

	return &Handler{
		base:          h,
		out:           out,
		checkMessages: opts.CheckMessages,
		recentEntries: newCircularBuffer(opts.recentEntriesSize),
	}
}

// Enabled implements [slog.Handler].
func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.base.Enabled(ctx, l)
}

// Handle implements [slog.Handler].
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	err := h.base.Handle(ctx, r)

	if devbuild.Enabled {
		must.NoError(err)
	}

	if h.checkMessages {
		if strings.TrimSpace(r.Message) != r.Message {
			panic(fmt.Sprintf("message %q has leading/trailing spaces", r.Message))
		}

		if strings.TrimRight(r.Message, ".?!") != r.Message {
			panic(fmt.Sprintf("message %q ends with punctuation", r.Message))
		}
	}

	h.recentEntries.add(&r)

	if r.Level < LevelDPanic {
		return err
	}

	// Try to flush the full record before panicking with a message only.
	// It may fail on stderr without TTY, so the error is ignored.
	if f, ok := h.out.(*os.File); ok {
		_ = f.Sync()
	}

	//nolint:exhaustive // levels smaller than LevelDPanic are handled above
	switch r.Level {
	case LevelDPanic:
		if devbuild.Enabled {
			panic(r.Message)
		}

		return err

	case LevelPanic:
		panic(r.Message)

	default: // LevelFatal or larger
		os.Exit(1)
		panic("not reached")
	}
}

// WithAttrs implements [slog.Handler].
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		base:          h.base.WithAttrs(attrs),
		out:           h.out,
		checkMessages: h.checkMessages,
		recentEntries: h.recentEntries,
	}
}

// WithGroup implements [slog.Handler].
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		base:          h.base.WithGroup(name),
		out:           h.out,
		checkMessages: h.checkMessages,
		recentEntries: h.recentEntries,
	}
}

// RecentEntries returns recent log entries.
func (h *Handler) RecentEntries() (*wirebson.Array, error) {
	return h.recentEntries.getArray()
}

// check interfaces
var (
	_ slog.Handler = (*Handler)(nil)
)

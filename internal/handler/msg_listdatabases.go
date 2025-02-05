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

package handler

import (
	"context"
	"maps"
	"slices"

	"github.com/FerretDB/wire"
	"github.com/FerretDB/wire/wirebson"

	"github.com/FerretDB/FerretDB/v2/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/v2/internal/util/must"
)

// MsgListDatabases implements `listDatabases` command.
//
// The passed context is canceled when the client connection is closed.
//
// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/26
func (h *Handler) MsgListDatabases(connCtx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	spec, err := msg.RawDocument()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	if _, _, err = h.s.CreateOrUpdateByLSID(connCtx, spec); err != nil {
		return nil, err
	}

	list, err := h.Pool.ListDatabases(connCtx)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	names := slices.Sorted(maps.Keys(list))
	databases := wirebson.MakeArray(len(names))

	for _, name := range names {
		d := must.NotFail(wirebson.NewDocument(
			"name", name,
		))

		must.NoError(databases.Add(d))
	}

	res := must.NotFail(wirebson.NewDocument(
		"databases", databases,
		"ok", float64(1),
	))

	return wire.NewOpMsg(must.NotFail(res.Encode()))
}

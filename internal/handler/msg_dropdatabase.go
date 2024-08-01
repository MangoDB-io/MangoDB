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

	"github.com/FerretDB/wire"

	"github.com/FerretDB/FerretDB/internal/backends"
	"github.com/FerretDB/FerretDB/internal/handler/common"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
)

// MsgDropDatabase implements `dropDatabase` command.
//
// The passed context is canceled when the client connection is closed.
func (h *Handler) MsgDropDatabase(connCtx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	document, err := OpMsgDocument(msg)
	if err != nil {
		return nil, err
	}

	common.Ignored(document, h.L, "writeConcern", "comment")

	dbName, err := common.GetRequiredParam[string](document, "$db")
	if err != nil {
		return nil, err
	}

	// Most backends would block on `DropDatabase` below otherwise.
	//
	// There is a race condition: another client could create a new cursor for that database
	// after we closed all of them, but before we drop the database itself.
	// In that case, we expect the client to wait or to retry the operation.
	for _, c := range h.cursors.All() {
		if c.DB == dbName {
			h.cursors.CloseAndRemove(c)
		}
	}

	err = h.b.DropDatabase(connCtx, &backends.DropDatabaseParams{
		Name: dbName,
	})

	res := must.NotFail(types.NewDocument())

	switch {
	case err == nil:
		res.Set("dropped", dbName)
	case backends.ErrorCodeIs(err, backends.ErrorCodeDatabaseNameIsInvalid):
		// nothing?
	case backends.ErrorCodeIs(err, backends.ErrorCodeDatabaseDoesNotExist):
		// nothing
	default:
		return nil, lazyerrors.Error(err)
	}

	res.Set("ok", float64(1))

	return NewOpMsg(
		res,
	)
}

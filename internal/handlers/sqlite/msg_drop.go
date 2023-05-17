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

package sqlite

import (
	"context"

	"github.com/FerretDB/FerretDB/internal/backends"
	"github.com/FerretDB/FerretDB/internal/handlers/common"
	"github.com/FerretDB/FerretDB/internal/handlers/commonerrors"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
	"github.com/FerretDB/FerretDB/internal/wire"
)

// MsgDrop implements HandlerInterface.
func (h *Handler) MsgDrop(ctx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	document, err := msg.Document()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	common.Ignored(document, h.L, "writeConcern", "comment")

	command := document.Command()

	db, err := common.GetRequiredParam[string](document, "$db")
	if err != nil {
		return nil, err
	}

	collection, err := common.GetRequiredParam[string](document, command)
	if err != nil {
		return nil, err
	}

	err = h.b.Database(db).DropCollection(ctx, &backends.DropCollectionParams{
		Name: collection,
	})
	switch {
	case err == nil:
		var reply wire.OpMsg
		must.NoError(reply.SetSections(wire.OpMsgSection{
			Documents: []*types.Document{must.NotFail(types.NewDocument(
				"nIndexesWas", int32(1), // TODO
				"ns", db+"."+collection,
				"ok", float64(1),
			))},
		}))

		return &reply, nil

	case backends.ErrorCodeIs(err, backends.ErrorCodeCollectionDoesNotExist):
		return nil, commonerrors.NewCommandErrorMsg(commonerrors.ErrNamespaceNotFound, "ns not found")

	default:
		return nil, lazyerrors.Error(err)
	}
}

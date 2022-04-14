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

package tigris

import (
	"context"

	"github.com/tigrisdata/tigrisdb-client-go/driver"

	"github.com/FerretDB/FerretDB/internal/fjson"
	"github.com/FerretDB/FerretDB/internal/handlers/common"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/wire"
)

// MsgInsert inserts a document or documents into a collection.
func (h *Handler) MsgInsert(ctx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	document, err := msg.Document()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	common.Ignored(document, h.l, "ordered", "writeConcern", "bypassDocumentValidation", "comment")

	command := document.Command()

	var db, collection string
	if db, err = common.GetRequiredParam[string](document, "$db"); err != nil {
		return nil, err
	}
	if collection, err = common.GetRequiredParam[string](document, command); err != nil {
		return nil, err
	}

	var docs *types.Array
	if docs, err = common.GetOptionalParam(document, "documents", docs); err != nil {
		return nil, err
	}

	// on tigris - cannot create schema if it not exists, since it's strictly specified

	var inserted int32
	for i := 0; i < docs.Len(); i++ {
		doc, err := docs.Get(i)
		if err != nil {
			return nil, lazyerrors.Error(err)
		}

		tigrisDoc, err := fjson.Marshal(doc)
		if err != nil {
			return nil, lazyerrors.Error(err)
		}
		if _, err = h.client.conn.Insert(ctx, db, collection, []driver.Document{tigrisDoc}); err != nil {
			return nil, err
		}

		inserted++
	}

	var reply wire.OpMsg
	err = reply.SetSections(wire.OpMsgSection{
		Documents: []*types.Document{types.MustNewDocument(
			"n", inserted,
			"ok", float64(1),
		)},
	})
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return &reply, nil
}

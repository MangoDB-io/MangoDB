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

package common

import (
	"context"
	"errors"
	"fmt"

	"github.com/FerretDB/FerretDB/internal/clientconn/conninfo"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/iterator"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
	"github.com/FerretDB/FerretDB/internal/wire"
)

// MsgGetMore is a common implementation of the `getMore` command.
func MsgGetMore(ctx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	document, err := msg.Document()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	if err = Unimplemented(document, "comment", "maxTimeMS"); err != nil {
		return nil, err
	}

	db, err := GetRequiredParam[string](document, "$db")
	if err != nil {
		return nil, err
	}

	collection, err := GetRequiredParam[string](document, "collection")
	if err != nil {
		return nil, NewCommandErrorMsg(ErrBadValue, `required parameter "collection" is missing`)
	}

	cursorIDValue, err := document.Get("getMore")
	if err != nil {
		return nil, NewCommandErrorMsg(ErrBadValue, `required parameter "getMore" is missing`)
	}

	var cursorID int64
	var ok bool

	if cursorID, ok = cursorIDValue.(int64); !ok {
		return nil, NewCommandErrorMsg(
			ErrTypeMismatch,
			fmt.Sprintf(
				`BSON field 'getMore.getMore' is the wrong type '%s', expected type 'long'`,
				AliasFromType(cursorIDValue),
			),
		)
	}

	if cursorID != 1 {
		return nil, NewCommandErrorMsg(ErrCursorNotFound, fmt.Sprintf("cursor id %d not found", cursorID))
	}

	batchSize, err := getBatchSize(document)
	if err != nil {
		return nil, err
	}

	connInfo := conninfo.Get(ctx)

	cur := connInfo.Cursor(1)
	if cur == nil {
		return nil, lazyerrors.Errorf("cursor for collection %s not found", collection)
	}

	resDocs := types.MakeArray(0)

	var done bool

	for i := 0; i < int(batchSize); i++ {
		var doc any

		_, doc, err = cur.Next()
		if err != nil {
			if errors.Is(err, iterator.ErrIteratorDone) {
				done = true
				break
			}

			return nil, lazyerrors.Error(err)
		}

		resDocs.Append(doc)
	}

	// TODO: https://github.com/FerretDB/FerretDB/issues/1811
	id := int64(1)

	if done {
		id = 0
	}

	var reply wire.OpMsg

	err = reply.SetSections(wire.OpMsgSection{
		Documents: []*types.Document{must.NotFail(types.NewDocument(
			"cursor", must.NotFail(types.NewDocument(
				"nextBatch", resDocs,
				"id", id,
				"ns", db+"."+collection,
			)),
			"ok", float64(1),
		))},
	})
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return &reply, nil
}

// getBatchSize returns the batch size from the document.
func getBatchSize(doc *types.Document) (int64, error) {
	batchSizeValue, err := doc.Get("batchSize")
	if err != nil {
		return 0, nil
	}

	batchSize, err := GetWholeNumberParam(batchSizeValue)
	if err != nil {
		if errors.Is(err, errUnexpectedType) {
			return 0, NewCommandErrorMsg(
				ErrTypeMismatch,
				fmt.Sprintf(
					"BSON field 'batchSize' is the wrong type '%s', expected type 'long'",
					AliasFromType(batchSizeValue),
				),
			)
		}
	}

	if batchSize < 0 {
		return 0, NewCommandErrorMsg(
			ErrBatchSizeNegative,
			"BSON field 'batchSize' value must be >= 0, actual value '-1'",
		)
	}

	return batchSize, nil
}

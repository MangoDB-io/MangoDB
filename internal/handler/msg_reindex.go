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
	"fmt"
	"log/slog"

	"github.com/FerretDB/wire"
	"github.com/FerretDB/wire/wirebson"

	"github.com/FerretDB/FerretDB/v2/internal/documentdb/documentdb_api"
	"github.com/FerretDB/FerretDB/v2/internal/mongoerrors"
	"github.com/FerretDB/FerretDB/v2/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/v2/internal/util/logging"
	"github.com/FerretDB/FerretDB/v2/internal/util/must"
)

// MsgReIndex implements `reIndex` command.
//
// The passed context is canceled when the client connection is closed.
func (h *Handler) MsgReIndex(connCtx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	spec, err := msg.RawDocument()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	if _, _, err = h.s.CreateOrUpdateByLSID(connCtx, spec); err != nil {
		return nil, err
	}

	// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/78
	doc, err := spec.Decode()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	dbName, err := getRequiredParam[string](doc, "$db")
	if err != nil {
		return nil, err
	}

	command := doc.Command()

	v := doc.Get(command)

	collection, ok := v.(string)
	if !ok {
		return nil, mongoerrors.NewWithArgument(
			mongoerrors.ErrInvalidNamespace,
			fmt.Sprintf("collection name has invalid type %T", v),
			"reIndex",
		)
	}

	if collection == "" {
		return nil, mongoerrors.NewWithArgument(
			mongoerrors.ErrInvalidNamespace,
			fmt.Sprintf("Invalid namespace specified '%s.'", dbName),
			"reIndex",
		)
	}

	conn, err := h.Pool.Acquire()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}
	defer conn.Release()

	listIndexesSpec := must.NotFail(wirebson.MustDocument(
		"listIndexes", collection,
		// use large batchSize to get all results in one batch
		"cursor", wirebson.MustDocument("batchSize", int32(10000)),
	).Encode())

	listRes, cursorID, err := h.Pool.ListIndexes(connCtx, dbName, listIndexesSpec)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	if cursorID != 0 {
		return nil, lazyerrors.New("too many indexes for re-indexing")
	}

	listDoc, err := listRes.DecodeDeep()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	lazyRes := slog.Any("res", logging.LazyString(listDoc.LogMessage))
	h.L.DebugContext(connCtx, "MsgReIndex: ListIndexes response", lazyRes)

	indexesBefore := listDoc.Get("cursor").(*wirebson.Document).Get("firstBatch").(*wirebson.Array)

	dropSpec := must.NotFail(wirebson.MustDocument(
		"dropIndexes", collection,
		"index", "*", // drops all but default _id index
	).Encode())

	dropRes, err := documentdb_api.DropIndexes(connCtx, conn.Conn(), h.L, dbName, dropSpec, nil)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	// this currently fails due to
	// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/306
	// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/643
	dropDoc, err := dropRes.DecodeDeep()
	if err != nil {
		h.L.DebugContext(connCtx, "MsgReIndex: failed to decode DropIndexes response", logging.Error(err))
	}

	lazyRes = slog.Any("res", logging.LazyString(dropDoc.LogMessage))
	h.L.DebugContext(connCtx, "MsgReIndex: DropIndexes response", lazyRes)

	createSpec := must.NotFail(wirebson.MustDocument(
		"createIndexes", collection,
		"indexes", indexesBefore,
	).Encode())

	createRes, err := h.createIndexes(connCtx, conn, command, dbName, createSpec)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	createDoc, err := createRes.Decode()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	listRes, cursorID, err = h.Pool.ListIndexes(connCtx, dbName, listIndexesSpec)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	if cursorID != 0 {
		return nil, lazyerrors.New("too many indexes after re-indexing")
	}

	listDoc, err = listRes.DecodeDeep()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	indexesAfter := listDoc.Get("cursor").(*wirebson.Document).Get("firstBatch").(*wirebson.Array)

	res, err := wirebson.NewDocument(
		"nIndexesWas", int32(indexesBefore.Len()),
		"nIndexes", createDoc.Get("numIndexesAfter"),
		"indexes", indexesAfter,
		"ok", float64(1),
	)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return wire.NewOpMsg(res)
}

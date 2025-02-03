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
	"strings"
	"unicode/utf8"

	"github.com/FerretDB/wire"
	"github.com/FerretDB/wire/wirebson"

	"github.com/FerretDB/FerretDB/v2/internal/documentdb/documentdb_api"
	"github.com/FerretDB/FerretDB/v2/internal/mongoerrors"
	"github.com/FerretDB/FerretDB/v2/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/v2/internal/util/must"
)

// MsgRenameCollection implements `renameCollection` command.
//
// The passed context is canceled when the client connection is closed.
func (h *Handler) MsgRenameCollection(connCtx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	spec, err := msg.RawDocument()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	if _, _, err = h.s.CreateOrUpdateByLSID(connCtx, spec); err != nil {
		return nil, err
	}

	document, err := spec.Decode()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	command := "renameCollection"

	oldName, err := getRequiredParam[string](document, command)
	if err != nil {
		from := document.Get(command)
		if from == nil || from == wirebson.Null {
			return nil, mongoerrors.NewWithArgument(
				mongoerrors.ErrLocation40414,
				"BSON field 'renameCollection.from' is missing but a required field",
				command,
			)
		}

		return nil, mongoerrors.NewWithArgument(
			mongoerrors.ErrTypeMismatch,
			fmt.Sprintf("collection name has invalid type %s", aliasFromType(from)),
			command,
		)
	}

	newName, err := getRequiredParam[string](document, "to")
	if err != nil {
		if to := document.Get("to"); to == nil || to == wirebson.Null {
			return nil, mongoerrors.NewWithArgument(
				mongoerrors.ErrLocation40414,
				"BSON field 'renameCollection.to' is missing but a required field",
				command,
			)
		}

		return nil, mongoerrors.NewWithArgument(
			mongoerrors.ErrTypeMismatch,
			"'to' must be of type String",
			command,
		)
	}

	dropTarget, err := getOptionalParam[bool](document, "dropTarget", false)
	if err != nil {
		return nil, err
	}

	oldDBName, oldCName, err := splitNamespace(oldName, command)
	if err != nil {
		return nil, err
	}

	newDBName, newCName, err := splitNamespace(newName, command)
	if err != nil {
		return nil, mongoerrors.NewWithArgument(
			mongoerrors.ErrInvalidNamespace,
			fmt.Sprintf("Invalid target namespace: %s", newName),
			command,
		)
	}

	if !collectionNameRe.MatchString(oldCName) ||
		!utf8.ValidString(oldCName) {
		msg := fmt.Sprintf("Invalid collection name: %s", oldCName)
		return nil, mongoerrors.NewWithArgument(mongoerrors.ErrInvalidNamespace, msg, "renameCollection")
	}

	if !collectionNameRe.MatchString(newCName) ||
		!utf8.ValidString(newCName) {
		msg := fmt.Sprintf("Invalid collection name: %s", newCName)
		return nil, mongoerrors.NewWithArgument(mongoerrors.ErrInvalidNamespace, msg, "renameCollection")
	}

	// support cross-database rename
	// TODO https://github.com/FerretDB/FerretDB/issues/2563
	if oldDBName != newDBName {
		return nil, mongoerrors.NewWithArgument(
			mongoerrors.ErrNotImplemented,
			"Command renameCollection does not support cross-database rename",
			command,
		)
	}

	if oldCName == newCName {
		return nil, mongoerrors.NewWithArgument(
			mongoerrors.ErrIllegalOperation,
			"Can't rename a collection to itself",
			command,
		)
	}

	conn, err := h.Pool.Acquire()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	defer conn.Release()

	err = documentdb_api.RenameCollection(connCtx, conn.Conn(), h.L, oldDBName, oldCName, newCName, dropTarget)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	res := must.NotFail(wirebson.NewDocument(
		"ok", float64(1),
	))

	return wire.NewOpMsg(must.NotFail(res.Encode()))
}

// splitNamespace returns the database and collection name from a given namespace in format "database.collection".
func splitNamespace(ns, argument string) (string, string, error) {
	parts := strings.Split(ns, ".")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", mongoerrors.NewWithArgument(
			mongoerrors.ErrInvalidNamespace,
			fmt.Sprintf("Invalid namespace specified '%s'", ns),
			argument,
		)
	}

	return parts[0], parts[1], nil
}

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
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/FerretDB/FerretDB/internal/backends"
	"github.com/FerretDB/FerretDB/internal/handlers/common"
	"github.com/FerretDB/FerretDB/internal/handlers/commonerrors"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
	"github.com/FerretDB/FerretDB/internal/wire"
)

// Reserved prefix for database and collection names.
const reservedPrefix = "_ferretdb_"

// validateCollectionNameRe validates collection names.
// Empty collection name, names with `$` and `\x00`,
// or exceeding the 255 bytes limit are not allowed.
// Collection names that start with `.` are also not allowed.
var validateCollectionNameRe = regexp.MustCompile("^[^.$\x00][^$\x00]{0,234}$")

// MsgCreate implements HandlerInterface.
func (h *Handler) MsgCreate(ctx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	document, err := msg.Document()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	unimplementedFields := []string{
		"timeseries",
		"expireAfterSeconds",
		"size",
		"max",
		"validator",
		"validationLevel",
		"validationAction",
		"viewOn",
		"pipeline",
		"collation",
	}
	if err = common.Unimplemented(document, unimplementedFields...); err != nil {
		return nil, err
	}

	if err = common.UnimplementedNonDefault(document, "capped", func(v any) bool {
		b, ok := v.(bool)
		return ok && !b
	}); err != nil {
		return nil, err
	}

	ignoredFields := []string{
		"autoIndexId",
		"storageEngine",
		"indexOptionDefaults",
		"writeConcern",
		"comment",
	}
	common.Ignored(document, h.L, ignoredFields...)

	command := document.Command()

	dbName, err := common.GetRequiredParam[string](document, "$db")
	if err != nil {
		return nil, err
	}

	collectionName, err := common.GetRequiredParam[string](document, command)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(collectionName, ".") {
		msg := fmt.Sprintf("Collection names cannot start with '.': %s", collectionName)
		return nil, commonerrors.NewCommandErrorMsgWithArgument(commonerrors.ErrInvalidNamespace, msg, "create")
	}

	if !validateCollectionNameRe.MatchString(collectionName) ||
		!utf8.ValidString(collectionName) ||
		strings.HasPrefix(collectionName, reservedPrefix) {
		msg := fmt.Sprintf("Invalid collection name: '%s.%s'", dbName, collectionName)
		return nil, commonerrors.NewCommandErrorMsgWithArgument(commonerrors.ErrInvalidNamespace, msg, "create")
	}

	db := h.b.Database(dbName)
	defer db.Close()

	err = db.CreateCollection(ctx, &backends.CreateCollectionParams{
		Name: collectionName,
	})

	switch {
	case err == nil:
		var reply wire.OpMsg
		must.NoError(reply.SetSections(wire.OpMsgSection{
			Documents: []*types.Document{must.NotFail(types.NewDocument(
				"ok", float64(1),
			))},
		}))

		return &reply, nil

	case backends.ErrorCodeIs(err, backends.ErrorCodeCollectionAlreadyExists):
		msg := fmt.Sprintf("Collection %s.%s already exists.", dbName, collectionName)
		return nil, commonerrors.NewCommandErrorMsgWithArgument(commonerrors.ErrNamespaceExists, msg, "create")

	default:
		return nil, lazyerrors.Error(err)
	}
}

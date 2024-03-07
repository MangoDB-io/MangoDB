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

	"github.com/FerretDB/FerretDB/internal/clientconn/conninfo"
	"github.com/FerretDB/FerretDB/internal/handler/common"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
	"github.com/FerretDB/FerretDB/internal/wire"
)

// MsgConnectionStatus implements `connectionStatus` command.
func (h *Handler) MsgConnectionStatus(ctx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	document, err := msg.Document()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	dbName, err := common.GetRequiredParam[string](document, "$db")
	if err != nil {
		return nil, err
	}

	users := types.MakeArray(1)
	if username := conninfo.Get(ctx).Username(); username != "" {
		users.Append(must.NotFail(types.NewDocument(
			"user", username,
			"db", dbName,
		)))
	}

	var reply wire.OpMsg
	must.NoError(reply.SetSections(wire.MakeOpMsgSection(
		must.NotFail(types.NewDocument(
			"authInfo", must.NotFail(types.NewDocument(
				"authenticatedUsers", users,
				"authenticatedUserRoles", must.NotFail(types.NewArray()),
				"authenticatedUserPrivileges", must.NotFail(types.NewArray()),
			)),
			"ok", float64(1),
		)),
	)))

	return &reply, nil
}

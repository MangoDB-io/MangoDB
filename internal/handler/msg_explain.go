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
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"os"
	"strconv"

	"github.com/FerretDB/wire"
	"github.com/FerretDB/wire/wirebson"

	"github.com/FerretDB/FerretDB/v2/build/version"
	"github.com/FerretDB/FerretDB/v2/internal/mongoerrors"
	"github.com/FerretDB/FerretDB/v2/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/v2/internal/util/must"
)

// MsgExplain implements `explain` command.
//
// The passed context is canceled when the client connection is closed.
func (h *Handler) MsgExplain(connCtx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	opID := h.operations.Start("command")
	defer h.operations.Stop(opID)

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
		return nil, lazyerrors.Error(err)
	}

	explainV, err := getRequiredParamAny(doc, "explain")
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	explainSpec, ok := explainV.(wirebson.RawDocument)
	if !ok {
		msg := fmt.Sprintf(`required parameter "explain" has type %T (expected document)`, explainV)
		return nil, lazyerrors.Error(mongoerrors.NewWithArgument(mongoerrors.ErrBadValue, msg, "explain"))
	}

	explainDoc, err := explainSpec.Decode()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	err = explainDoc.Add("$db", dbName)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	cmd := explainDoc.Command()

	collection, ok := explainDoc.Get(cmd).(string)
	if !ok {
		return nil, mongoerrors.NewWithArgument(mongoerrors.ErrInvalidNamespace, "Failed to parse namespace element", "explain")
	}

	h.operations.Update(opID, dbName, collection, doc)

	var f string

	switch cmd {
	case "aggregate":
		f = "documentdb_api_catalog.bson_aggregation_pipeline"
	case "count":
		f = "documentdb_api_catalog.bson_aggregation_count"
	case "find":
		f = "documentdb_api_catalog.bson_aggregation_find"
	default:
		return nil, mongoerrors.NewWithArgument(
			mongoerrors.ErrNotImplemented,
			fmt.Sprintf("explain for %s command is not supported", cmd),
			"explain",
		)
	}

	q := fmt.Sprintf(`
		EXPLAIN (FORMAT JSON)
			SELECT document
		FROM %s($1, $2::bytea)`,
		f,
	)

	conn, err := h.Pool.Acquire()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}
	defer conn.Release()

	var dest []byte
	if err = conn.Conn().QueryRow(connCtx, q, dbName, explainSpec).Scan(&dest); err != nil {
		return nil, lazyerrors.Error(mongoerrors.Make(connCtx, err, "", h.L))
	}

	queryPlan, err := unmarshalExplain(dest)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	_, portV, _ := net.SplitHostPort(h.TCPHost)

	var port int

	if portV != "" {
		port, err = strconv.Atoi(portV)
		if err != nil {
			return nil, lazyerrors.Error(err)
		}
	}

	serverInfo := must.NotFail(wirebson.NewDocument(
		"host", hostname,
		"port", int32(port),
		"version", version.Get().MongoDBVersion,
		"gitVersion", version.Get().Commit,

		// our extensions
		"ferretdb", must.NotFail(wirebson.NewDocument(
			"version", version.Get().Version,
		)),
	))

	res, err := wirebson.NewDocument(
		"queryPlanner", queryPlan,
		"explainVersion", "1",
		"command", must.NotFail(explainDoc.Encode()),
		"serverInfo", serverInfo,
		"ok", float64(1),
	)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	reply, err := res.Encode()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return wire.NewOpMsg(reply)
}

// unmarshalExplain unmarshalls the plan from EXPLAIN postgreSQL command.
func unmarshalExplain(b []byte) (*wirebson.Document, error) {
	var plans []map[string]any

	if err := json.Unmarshal(b, &plans); err != nil {
		return nil, lazyerrors.Error(err)
	}

	if len(plans) == 0 {
		return nil, lazyerrors.Error(errors.New("no execution plan returned"))
	}

	return convertJSON(plans[0]).(*wirebson.Document), nil
}

// convertJSON transforms decoded JSON map[string]any value into bson.Document.
func convertJSON(value any) any {
	switch value := value.(type) {
	case map[string]any:
		d := wirebson.MakeDocument(len(value))

		for k := range maps.Keys(value) {
			v := value[k]
			must.NoError(d.Add(k, convertJSON(v)))
		}

		return d

	case []any:
		a := wirebson.MakeArray(len(value))
		for _, v := range value {
			must.NoError(a.Add(convertJSON(v)))
		}

		return a

	case nil:
		return wirebson.Null

	case float64, string, bool:
		return value

	default:
		panic(fmt.Sprintf("unsupported type: %[1]T (%[1]v)", value))
	}
}

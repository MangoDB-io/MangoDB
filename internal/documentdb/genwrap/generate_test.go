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

package main

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateGoFunction(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct { //nolint:vet // use only for testing
		data templateData

		res string
	}{
		"DropIndexes": {
			data: templateData{
				FuncName:    "DropIndexes",
				SQLFuncName: "documentdb_api.drop_indexes",
				IsProcedure: true,
				Comment: `documentdb_api.drop_indexes(p_database_name text, p_arg documentdb_core.bson, ` +
					`INOUT retval documentdb_core.bson DEFAULT NULL)`,
				SQLArgs:      "$1, $2::bytea, $3::bytea",
				SQLReturns:   "retval::bytea",
				Params:       "databaseName string, arg wirebson.RawDocument, retValue wirebson.RawDocument",
				Returns:      "outRetValue wirebson.RawDocument",
				QueryRowArgs: "databaseName, arg, retValue",
				ScanArgs:     "&outRetValue",
			},
			//nolint:lll // generated function is too long
			res: `
// DropIndexes is a wrapper for
//
//	documentdb_api.drop_indexes(p_database_name text, p_arg documentdb_core.bson, INOUT retval documentdb_core.bson DEFAULT NULL).
func DropIndexes(ctx context.Context, conn *pgx.Conn, l *slog.Logger, databaseName string, arg wirebson.RawDocument, retValue wirebson.RawDocument) (outRetValue wirebson.RawDocument, err error) {
	ctx, span := otel.Tracer("").Start(ctx, "documentdb_api.drop_indexes", oteltrace.WithSpanKind(oteltrace.SpanKindClient))
	defer span.End()

	row := conn.QueryRow(ctx, "CALL documentdb_api.drop_indexes($1, $2::bytea, $3::bytea)", databaseName, arg, retValue)
	if err = row.Scan(&outRetValue); err != nil {
		err = mongoerrors.Make(ctx, err, "documentdb_api.drop_indexes", l)
	}
	return
}
`,
		},
	}

	for name, tc := range testCases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var b bytes.Buffer
			w := bufio.NewWriter(&b)
			err := generateGoFunction(w, &tc.data)
			require.NoError(t, err)

			err = w.Flush()
			require.NoError(t, err)
			require.Equal(t, tc.res, b.String())
		})
	}
}

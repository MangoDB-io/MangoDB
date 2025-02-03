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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FerretDB/FerretDB/v2/internal/util/testutil"
)

func TestExtract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}

	t.Parallel()

	ctx := testutil.Ctx(t)
	uri := "postgres://username:password@127.0.0.1:5432/postgres"

	rows, err := Extract(ctx, uri, []string{"documentdb_core", "documentdb_api"})
	require.NoError(t, err)

	expected := map[string]any{
		"specific_schema":    "documentdb_api",
		"specific_name":      "aggregate_cursor_first_page_19111",
		"routine_name":       "aggregate_cursor_first_page",
		"routine_type":       "FUNCTION",
		"routine_data_type":  "record",
		"routine_udt_schema": "pg_catalog",
		"routine_udt_name":   "record",
		"parameter_name":     "database",
		"parameter_mode":     "IN",
		"parameter_default":  nil,
		"data_type":          "text",
		"udt_schema":         "pg_catalog",
		"udt_name":           "text",
	}
	require.Equal(t, expected, rows["documentdb_api.aggregate_cursor_first_page_19111"][0])

	// TODO https://github.com/microsoft/documentdb/issues/49
	expected = map[string]any{
		"specific_schema":    "documentdb_api",
		"specific_name":      "drop_indexes_19097",
		"routine_name":       "drop_indexes",
		"routine_type":       "PROCEDURE",
		"routine_data_type":  nil,
		"routine_udt_schema": nil,
		"routine_udt_name":   nil,
		"parameter_name":     "retval",
		"parameter_mode":     "INOUT",
		"parameter_default":  "NULL::documentdb_core.bson",
		"data_type":          "USER-DEFINED",
		"udt_schema":         "documentdb_core",
		"udt_name":           "bson",
	}
	require.Equal(t, expected, rows["documentdb_api.drop_indexes_19097"][2])
}

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FerretDB/FerretDB/v2/internal/util/testutil"
)

func TestCase(t *testing.T) {
	t.Parallel()

	c := new(converter)

	assert.Equal(t, "cursorGetMore", c.camelCase("cursor_get_more"))
	assert.Equal(t, "CursorGetMore", c.pascalCase("cursor_get_more"))
}

func TestParameterName(t *testing.T) {
	t.Parallel()

	c := new(converter)

	assert.Equal(t, "validateSpec", c.parameterName("validatespec"))
}

func TestConvert(t *testing.T) {
	t.Parallel()

	l := testutil.Logger(t)

	rows := []map[string]any{
		{
			"specific_schema":    "documentdb_api",
			"specific_name":      "binary_extended_version_19132",
			"routine_name":       "binary_extended_version",
			"routine_type":       "FUNCTION",
			"parameter_name":     nil,
			"parameter_mode":     nil,
			"parameter_default":  nil,
			"data_type":          nil,
			"udt_schema":         nil,
			"udt_name":           nil,
			"routine_data_type":  "text",
			"routine_udt_schema": "pg_catalog",
			"routine_udt_name":   "text",
		},
		{
			"specific_schema":    "documentdb_api",
			"specific_name":      "count_query_19116",
			"routine_name":       "count_query",
			"routine_type":       "FUNCTION",
			"parameter_name":     "database",
			"parameter_mode":     "IN",
			"parameter_default":  nil,
			"data_type":          "text",
			"udt_schema":         "pg_catalog",
			"udt_name":           "text",
			"routine_data_type":  "USER-DEFINED",
			"routine_udt_schema": "documentdb_core",
			"routine_udt_name":   "bson",
		},
		{
			"specific_schema":    "documentdb_api",
			"specific_name":      "count_query_19116",
			"routine_name":       "count_query",
			"routine_type":       "FUNCTION",
			"parameter_name":     "countspec",
			"parameter_mode":     "IN",
			"parameter_default":  nil,
			"data_type":          "USER-DEFINED",
			"udt_schema":         "documentdb_core",
			"udt_name":           "bson",
			"routine_data_type":  "USER-DEFINED",
			"routine_udt_schema": "documentdb_core",
			"routine_udt_name":   "bson",
		},
		{
			"specific_schema":    "documentdb_api",
			"specific_name":      "count_query_19116",
			"routine_name":       "count_query",
			"routine_type":       "FUNCTION",
			"parameter_name":     "document",
			"parameter_mode":     "OUT",
			"parameter_default":  nil,
			"data_type":          "USER-DEFINED",
			"udt_schema":         "documentdb_core",
			"udt_name":           "bson",
			"routine_data_type":  "USER-DEFINED",
			"routine_udt_schema": "documentdb_core",
			"routine_udt_name":   "bson",
		},
		{
			"specific_schema":    "documentdb_api",
			"specific_name":      "drop_indexes_19097",
			"routine_name":       "drop_indexes",
			"routine_type":       "PROCEDURE",
			"parameter_name":     "p_database_name",
			"parameter_mode":     "IN",
			"parameter_default":  nil,
			"data_type":          "text",
			"udt_schema":         "pg_catalog",
			"udt_name":           "text",
			"routine_data_type":  nil,
			"routine_udt_schema": nil,
			"routine_udt_name":   nil,
		},
		{
			"specific_schema":    "documentdb_api",
			"specific_name":      "drop_indexes_19097",
			"routine_name":       "drop_indexes",
			"routine_type":       "PROCEDURE",
			"parameter_name":     "p_arg",
			"parameter_mode":     "IN",
			"parameter_default":  nil,
			"data_type":          "USER-DEFINED",
			"udt_schema":         "documentdb_core",
			"udt_name":           "bson",
			"routine_data_type":  nil,
			"routine_udt_schema": nil,
			"routine_udt_name":   nil,
		},
		{
			"specific_schema":    "documentdb_api",
			"specific_name":      "drop_indexes_19097",
			"routine_name":       "drop_indexes",
			"routine_type":       "PROCEDURE",
			"parameter_name":     "retval",
			"parameter_mode":     "INOUT",
			"parameter_default":  "NULL::documentdb_core.bson",
			"data_type":          "USER-DEFINED",
			"udt_schema":         "documentdb_core",
			"udt_name":           "bson",
			"routine_data_type":  nil,
			"routine_udt_schema": nil,
			"routine_udt_name":   nil,
		},
		{
			"specific_schema":    "documentdb_core",
			"specific_name":      "bsonquery_compare_16444",
			"routine_name":       "bsonquery_compare",
			"routine_type":       "FUNCTION",
			"parameter_name":     nil,
			"parameter_mode":     "IN",
			"parameter_default":  nil,
			"data_type":          "USER-DEFINED",
			"udt_schema":         "documentdb_core",
			"udt_name":           "bsonquery",
			"routine_data_type":  "integer",
			"routine_udt_schema": "pg_catalog",
			"routine_udt_name":   "int4",
		},
		{
			"specific_schema":    "documentdb_core",
			"specific_name":      "bsonquery_compare_16444",
			"routine_name":       "bsonquery_compare",
			"routine_type":       "FUNCTION",
			"parameter_name":     nil,
			"parameter_mode":     "IN",
			"parameter_default":  nil,
			"data_type":          "USER-DEFINED",
			"udt_schema":         "documentdb_core",
			"udt_name":           "bsonquery",
			"routine_data_type":  "integer",
			"routine_udt_schema": "pg_catalog",
			"routine_udt_name":   "int4",
		},
		{
			"specific_schema":    "documentdb_core",
			"specific_name":      "bsonquery_compare_16445",
			"routine_name":       "bsonquery_compare",
			"routine_type":       "FUNCTION",
			"parameter_name":     nil,
			"parameter_mode":     "IN",
			"parameter_default":  nil,
			"data_type":          "USER-DEFINED",
			"udt_schema":         "documentdb_core",
			"udt_name":           "bson",
			"routine_data_type":  "integer",
			"routine_udt_schema": "pg_catalog",
			"routine_udt_name":   "int4",
		},
		{
			"specific_schema":    "documentdb_core",
			"specific_name":      "bsonquery_compare_16445",
			"routine_name":       "bsonquery_compare",
			"routine_type":       "FUNCTION",
			"parameter_name":     nil,
			"parameter_mode":     "IN",
			"parameter_default":  nil,
			"data_type":          "USER-DEFINED",
			"udt_schema":         "documentdb_core",
			"udt_name":           "bsonquery",
			"routine_data_type":  "integer",
			"routine_udt_schema": "pg_catalog",
			"routine_udt_name":   "int4",
		},
	}

	expected := map[string]map[string]templateData{
		"documentdb_api": {
			"binary_extended_version": {
				FuncName:    "BinaryExtendedVersion",
				SQLFuncName: "documentdb_api.binary_extended_version",
				SQLArgs:     "",
				SQLReturns:  "binary_extended_version",
				Comment:     `documentdb_api.binary_extended_version(OUT binary_extended_version text)`,
				Returns:     "outBinaryExtendedVersion string",
				ScanArgs:    "&outBinaryExtendedVersion",
			},
			"count_query": {
				FuncName:    "CountQuery",
				SQLFuncName: "documentdb_api.count_query",
				SQLArgs:     "$1, $2::bytea",
				SQLReturns:  "document::bytea",
				Comment: `documentdb_api.count_query(database text, countspec documentdb_core.bson, ` +
					`OUT document documentdb_core.bson)`,
				Params:       "database string, countSpec wirebson.RawDocument",
				Returns:      "outDocument wirebson.RawDocument",
				QueryRowArgs: "database, countSpec",
				ScanArgs:     "&outDocument",
			},
			"drop_indexes": {
				FuncName:    "DropIndexes",
				SQLFuncName: "documentdb_api.drop_indexes",
				IsProcedure: true,
				SQLArgs:     "$1, $2::bytea, $3::bytea",
				SQLReturns:  "retval::bytea",
				Comment: `documentdb_api.drop_indexes(p_database_name text, p_arg documentdb_core.bson, ` +
					`INOUT retval documentdb_core.bson DEFAULT NULL)`,
				Params:       "databaseName string, arg wirebson.RawDocument, retVal wirebson.RawDocument",
				Returns:      "outRetVal wirebson.RawDocument",
				ScanArgs:     "&outRetVal",
				QueryRowArgs: "databaseName, arg, retVal",
			},
		},
		"documentdb_core": {
			"bsonquery_compare": {
				FuncName:    "BsonqueryCompare",
				SQLFuncName: "documentdb_core.bsonquery_compare",
				IsProcedure: false,
				SQLArgs:     "$1, $2",
				SQLReturns:  "bsonquery_compare",
				Comment: `documentdb_core.bsonquery_compare(anonymous documentdb_core.bsonquery, ` +
					`anonymous1 documentdb_core.bsonquery, OUT bsonquery_compare integer)`,
				Params:       "anonymous struct{}, anonymous1 struct{}",
				Returns:      "outBsonqueryCompare int32",
				ScanArgs:     "&outBsonqueryCompare",
				QueryRowArgs: "anonymous, anonymous1",
			},
			"bsonquery_compare1": {
				FuncName:    "BsonqueryCompare1",
				SQLFuncName: "documentdb_core.bsonquery_compare",
				IsProcedure: false,
				SQLArgs:     "$1::bytea, $2",
				SQLReturns:  "bsonquery_compare",
				Comment: `documentdb_core.bsonquery_compare(anonymous documentdb_core.bson, ` +
					`anonymous1 documentdb_core.bsonquery, OUT bsonquery_compare integer)`,
				Params:       "anonymous wirebson.RawDocument, anonymous1 struct{}",
				Returns:      "outBsonqueryCompare int32",
				ScanArgs:     "&outBsonqueryCompare",
				QueryRowArgs: "anonymous, anonymous1",
			},
		},
	}

	res := Convert(rows, l)
	require.Equal(t, expected, res)
}

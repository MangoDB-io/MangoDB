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

// import (
// 	"testing"

// 	"github.com/FerretDB/FerretDB/internal/types"
// 	"github.com/FerretDB/FerretDB/internal/util/must"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestMakeSimpleQuery(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument("a", int32(1), "b", int32(2)))
// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"1", "2"}, values)
// 	assert.Equal(t, "(_jsonb->'a' = $1 AND _jsonb->'b' = $2)", *sql)
// }

// func TestMakeNestedQuery(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument(
// 		"a", int32(1),
// 		"b", must.NotFail(types.NewDocument(
// 			"c", int32(2),
// 			"d", int32(3),
// 		)),
// 	))
// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"1", "2", "3"}, values)
// 	assert.Equal(t, "(_jsonb->'a' = $1 AND (_jsonb->'b'->>'c' = $2 AND _jsonb->'b'->>'d' = $3))", *sql)
// }

// func TestOrMatch(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument("$or",
// 		must.NotFail(types.NewArray(
// 			must.NotFail(types.NewDocument("a", int32(1), "b", int32(2))),
// 			must.NotFail(types.NewDocument("c", "ONE")),
// 		)),
// 	))
// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"1", "2", "ONE"}, values)
// 	assert.Equal(t, "(((_jsonb->'a' = $1 AND _jsonb->'b' = $2) OR (_jsonb->'c' = $3)))", *sql)
// }

// func TestAndMatch(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument("$and",
// 		must.NotFail(types.NewArray(
// 			must.NotFail(types.NewDocument("a", int32(1), "b", int32(2))),
// 			must.NotFail(types.NewDocument("c", "ONE")),
// 		)),
// 	))
// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"1", "2", "ONE"}, values)
// 	assert.Equal(t, "(((_jsonb->'a' = $1 AND _jsonb->'b' = $2) AND (_jsonb->'c' = $3)))", *sql)
// }

// func TestSimpleNotMatch(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument("$not",
// 		must.NotFail(types.NewDocument("color", "blue")),
// 	))
// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"blue"}, values)
// 	assert.Equal(t, "(NOT ((_jsonb->'color' = $1)))", *sql)
// }

// func TestComplexNotMatch(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument("$not",
// 		must.NotFail(types.NewDocument(
// 			"b", must.NotFail(types.NewDocument(
// 				"$gt", int32(1),
// 			)),
// 		)),
// 	))
// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"1"}, values)
// 	assert.Equal(t, "(NOT (((_jsonb->'b' > $1))))", *sql)
// }

// func TestAllMatch(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument("color",
// 		must.NotFail(types.NewDocument("$all",
// 			must.NotFail(types.NewArray("black", "white")),
// 		)),
// 	))
// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{[]string{"black", "white"}}, values)
// 	assert.Equal(t, "((_jsonb->'color' @> ($1)))", *sql)
// }

// func TestNestedWithOr(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument(
// 		"a", int32(1),
// 		"b", must.NotFail(types.NewDocument(
// 			"c", int32(2),
// 			"d", int32(3),
// 		)),
// 		"e", must.NotFail(types.NewDocument("$or",
// 			must.NotFail(types.NewArray(must.NotFail(types.NewDocument("a", "ONE")), must.NotFail(types.NewDocument("b", "TWO"))))),
// 		),
// 	))
// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"1", "2", "3", "ONE", "TWO"}, values)
// 	assert.Equal(t, "(_jsonb->'a' = $1 AND (_jsonb->'b'->>'c' = $2 AND _jsonb->'b'->>'d' = $3) AND (((_jsonb->'b'->>'e'->>'a' = $4) OR (_jsonb->'b'->>'e'->>'b' = $5))))", *sql)
// }

// func TestGtLteComparatorsOp(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument(
// 		"a", int32(1),
// 		"b", must.NotFail(types.NewDocument(
// 			"$gt", int32(1),
// 			"$lte", int32(10),
// 		)),
// 	))

// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"1", "1", "10"}, values)
// 	assert.Equal(t, "(_jsonb->'a' = $1 AND (_jsonb->'b' > $2 AND _jsonb->'b' <= $3))", *sql)
// }

// func TestNeComparatorOp(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument(
// 		"b", must.NotFail(types.NewDocument(
// 			"$ne", int32(1),
// 		)),
// 	))

// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"1"}, values)
// 	assert.Equal(t, "((_jsonb->'b' <> $1))", *sql)
// }

// func TestInComparatorOp(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument(
// 		"b", must.NotFail(types.NewDocument(
// 			"$in",
// 			must.NotFail(types.NewArray(int32(1), int32(2))),
// 		)),
// 	))

// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{[]string{"1", "2"}}, values)
// 	assert.Equal(t, "((_jsonb->'b' = ANY($1)))", *sql)
// }

// func TestNinComparatorOp(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument(
// 		"b", must.NotFail(types.NewDocument(
// 			"$nin",
// 			must.NotFail(types.NewArray(int32(1), int32(2))),
// 		)),
// 	))

// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{[]string{"1", "2"}}, values)
// 	assert.Equal(t, "((_jsonb->'b' <> ALL($1)))", *sql)
// }

// func TestSimpleExistsComparatorOp(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument(
// 		"b", must.NotFail(types.NewDocument(
// 			"$exists", true,
// 		)),
// 	))

// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"b"}, values)
// 	assert.Equal(t, "((_jsonb::jsonb ? $1))", *sql)
// }

// func TestNestedSimpleExistsComparatorOp(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument(
// 		"b", must.NotFail(types.NewDocument(
// 			"c", must.NotFail(types.NewDocument(
// 				"$exists", true,
// 			)),
// 		)),
// 	))

// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"c"}, values)
// 	assert.Equal(t, "(((_jsonb::jsonb->'b' ? $1)))", *sql)
// }

// func TestRegexMatch(t *testing.T) {
// 	t.Parallel()

// 	doc := must.NotFail(types.NewDocument("color",
// 		must.NotFail(types.NewDocument("$regex", "^red$")),
// 	))
// 	sql, values, err := AggregateMatch(doc, "_jsonb")
// 	require.NoError(t, err)

// 	assert.Equal(t, []interface{}{"^red$"}, values)
// 	assert.Equal(t, "((_jsonb->>'color' ~ $1))", *sql)
// }

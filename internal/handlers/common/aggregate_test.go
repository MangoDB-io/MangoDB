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
	"testing"

	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/must"
	"github.com/stretchr/testify/assert"
)

func TestMakeSimpleQuery(t *testing.T) {
	t.Parallel()

	doc := must.NotFail(types.NewDocument("a", int32(1), "b", int32(2)))
	sql, values := AggregateMatch(doc)
	assert.Equal(t, []interface{}{"1", "2"}, values)
	assert.Equal(t, "_jsonb->'a' = $1 AND _jsonb->'b' = $2", sql)
}

func TestMakeNestedQuery(t *testing.T) {
	t.Parallel()

	doc := must.NotFail(types.NewDocument(
		"a", int32(1),
		"b", must.NotFail(types.NewDocument(
			"c", int32(2),
			"d", int32(3),
		)),
	))
	sql, values := AggregateMatch(doc)
	assert.Equal(t, []interface{}{"1", "2", "3"}, values)
	assert.Equal(t, "_jsonb->'a' = $1 AND _jsonb->'b'->>'c' = $2 AND _jsonb->'b'->>'d' = $3", sql)
}

func TestGetValueWithOr(t *testing.T) {
	t.Parallel()

	doc := must.NotFail(types.NewDocument("$or",
		must.NotFail(types.NewArray(must.NotFail(types.NewDocument("a", int32(1), "b", int32(2)))))))
	sql, values := AggregateMatch(doc)
	assert.Equal(t, []interface{}{"1", "2"}, values)
	assert.Equal(t, "(_jsonb->'a' = $1 OR _jsonb->'b' = $2)", sql)
}

func TestNestedWithOr(t *testing.T) {
	t.Parallel()
	doc := must.NotFail(types.NewDocument(
		"a", int32(1),
		"b", must.NotFail(types.NewDocument(
			"c", int32(2),
			"d", int32(3),
		)),
		"e", must.NotFail(types.NewDocument("$or",
			must.NotFail(types.NewArray(must.NotFail(types.NewDocument("a", "ONE", "b", "TWO"))))),
		),
	))
	sql, values := AggregateMatch(doc)

	assert.Equal(t, []interface{}{"1", "2", "3", "ONE", "TWO"}, values)
	assert.Equal(t, "_jsonb->'a' = $1 AND _jsonb->'b'->>'c' = $2 AND _jsonb->'b'->>'d' = $3 AND (_jsonb->'e'->>'a' = $4 OR _jsonb->'e'->>'b' = $5)", sql)
}

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

// Package integration provides FerretDB integration tests.
package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/testutil"
)

// Convert converts given driver value (bson.D, bson.A, etc) to FerretDB types package value.
//
// It then can be used with all types helpers such as testutil.AssertEqual.
func Convert(t testing.TB, v any) any {
	t.Helper()

	switch v := v.(type) {
	// composite types
	case primitive.D:
		doc := types.MustNewDocument()
		for _, e := range v {
			doc.Set(e.Key, Convert(t, e.Value))
		}
		return doc
	case primitive.A:
		arr := types.MakeArray(len(v))
		for _, e := range v {
			arr.Append(Convert(t, e))
		}
		return arr

	// scalar types (in the same order as in types package)
	case float64:
		return v
	case string:
		return v
	case primitive.Binary:
		return types.Binary{
			Subtype: types.BinarySubtype(v.Subtype),
			B:       v.Data,
		}
	case primitive.ObjectID:
		return types.ObjectID(v)
	case bool:
		return v
	case primitive.DateTime:
		return v.Time()
	case nil:
		return v
	case primitive.Regex:
		return types.Regex{
			Pattern: v.Pattern,
			Options: v.Options,
		}
	case int32:
		return v
	case primitive.Timestamp:
		return types.Timestamp(uint64(v.I)<<32 + uint64(v.T))
	case int64:
		return v
	default:
		t.Fatalf("unexpected type %T", v)
		panic("not reached")
	}
}

// ConvertDocument converts given driver's document to FerretDB's *types.Document.
func ConvertDocument(t testing.TB, doc bson.D) *types.Document {
	t.Helper()

	v := Convert(t, doc)

	var res *types.Document
	require.IsType(t, res, v)
	res = v.(*types.Document)
	return res
}

// AssertEqualDocuments asserts that two documents are equal in a way that is useful for tests
// (NaNs are equal, etc).
//
// See testutil.AssertEqual for details.
func AssertEqualDocuments(t testing.TB, expected, actual bson.D) bool {
	t.Helper()

	expectedDoc := ConvertDocument(t, expected)
	actualDoc := ConvertDocument(t, actual)
	return testutil.AssertEqual(t, expectedDoc, actualDoc)
}

// AssertEqualError asserts that expected error is the same as actual, ignoring the Raw part.
func AssertEqualError(t testing.TB, expected mongo.CommandError, altMessage string, actual error) bool {
	t.Helper()

	a, ok := actual.(mongo.CommandError)
	if !ok {
		return assert.Equal(t, expected, actual)
	}

	// raw part might be useful if assertion fails
	require.Nil(t, expected.Raw)
	expected.Raw = a.Raw

	message := expected.Message
	if altMessage != "" {
		message = altMessage
	}

	return assert.Equal(t, expected.Code, a.Code) &&
		assert.Equal(t, message, a.Message) &&
		assert.Equal(t, expected.Name, a.Name)
}

// CollectIDs returns all _id values from given documents.
//
// The order is preserved.
func CollectIDs(t testing.TB, docs []bson.D) []any {
	t.Helper()

	ids := make([]any, len(docs))
	for i, doc := range docs {
		id, ok := doc.Map()["_id"]
		require.True(t, ok)
		ids[i] = id
	}

	return ids
}

// CollectKeys returns document keys.
//
// The order is preserved.
func CollectKeys(t testing.TB, doc bson.D) []string {
	t.Helper()

	res := make([]string, len(doc))
	for i, e := range doc {
		res[i] = e.Key
	}

	return res
}

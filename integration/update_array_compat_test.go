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

package integration

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/integration/shareddata"
)

func TestUpdateArrayCompatPop(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"DuplicateKeys": {
			update:     bson.D{{"$pop", bson.D{{"v", 1}, {"v", 1}}}},
			resultType: emptyResult,
		},
		"Pop": {
			update: bson.D{{"$pop", bson.D{{"v", 1}}}},
		},
		"PopFirst": {
			update: bson.D{{"$pop", bson.D{{"v", -1}}}},
		},
		"NonExistentField": {
			update:     bson.D{{"$pop", bson.D{{"non-existent-field", 1}}}},
			resultType: emptyResult,
		},
		"DotNotation": {
			filter: bson.D{{"_id", "array-documents-nested"}},
			update: bson.D{{"$pop", bson.D{{"v.0.foo", 1}}}},
		},
		"DotNotationPopFirst": {
			filter: bson.D{{"_id", "array-documents-nested"}},
			update: bson.D{{"$pop", bson.D{{"v.0.foo", -1}}}},
		},
		"DotNotationNonArray": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$pop", bson.D{{"v.0.foo.0.bar", 1}}}},
			resultType: emptyResult,
		},
		"DotNotationNonExistentPath": {
			update:     bson.D{{"$pop", bson.D{{"non.existent.path", 1}}}},
			resultType: emptyResult,
		},
		"PathNonExistentIndex": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$pop", bson.D{{"v.0.foo.2.bar", 1}}}},
			providers:  []shareddata.Provider{shareddata.ArrayDocuments},
			resultType: emptyResult,
		},
		"PathInvalidIndex": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$pop", bson.D{{"v.-1.foo", 1}}}},
			providers:  []shareddata.Provider{shareddata.ArrayDocuments},
			resultType: emptyResult,
		},
		"PopEmptyValue": {
			update:     bson.D{{"$pop", bson.D{}}},
			resultType: emptyResult,
		},
		"PopNotValidValueString": {
			update:     bson.D{{"$pop", bson.D{{"v", "foo"}}}},
			resultType: emptyResult,
		},
		"PopNotValidValueInt": {
			update:     bson.D{{"$pop", bson.D{{"v", int32(42)}}}},
			resultType: emptyResult,
		},
		"DotNotationObjectInArray": {
			update:     bson.D{{"$pop", bson.D{{"v.array.foo.array", 1}}}},
			resultType: emptyResult,
		},
		"DotNotationObject": {
			update:     bson.D{{"$pop", bson.D{{"v.foo", 1}}}},
			resultType: emptyResult,
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateArrayCompatPush(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"DuplicateKeys": {
			update:     bson.D{{"$push", bson.D{{"v", "foo"}, {"v", "bar"}}}},
			resultType: emptyResult, // conflict because of duplicate keys "v" set in $push
		},
		"String": {
			update: bson.D{{"$push", bson.D{{"v", "foo"}}}},
		},
		"Int32": {
			update: bson.D{{"$push", bson.D{{"v", int32(42)}}}},
		},
		"NonExistentField": {
			update: bson.D{{"$push", bson.D{{"non-existent-field", int32(42)}}}},
		},
		"DotNotation": {
			filter: bson.D{{"_id", "array-documents-nested"}},
			update: bson.D{{"$push", bson.D{{"v.0.foo", bson.D{{"bar", "zoo"}}}}}},
		},
		"DotNotationNonArray": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$push", bson.D{{"v.0.foo.0.bar", "boo"}}}},
			resultType: emptyResult, // attempt to push to non-array
		},
		"DotNotationNonExistentPath": {
			update: bson.D{{"$push", bson.D{{"non.existent.path", int32(42)}}}},
		},
		"TwoElements": {
			update: bson.D{{"$push", bson.D{{"non.existent.path", int32(42)}, {"v", int32(42)}}}},
		},
	}

	testUpdateCompat(t, testCases)
}

// TestUpdateArrayCompatAddToSet tests the $addToSet update operator.
// Test case "String" will cover the case where the value is already in set when ran against "array-two" document.
func TestUpdateArrayCompatAddToSet(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"DuplicateKeys": {
			update:     bson.D{{"$addToSet", bson.D{{"v", int32(1)}, {"v", int32(1)}}}},
			resultType: emptyResult,
		},
		"String": {
			update: bson.D{{"$addToSet", bson.D{{"v", "foo"}}}},
		},
		"Document": {
			update: bson.D{{"$addToSet", bson.D{{"v", bson.D{{"foo", "bar"}}}}}},
		},
		"Int32": {
			update: bson.D{{"$addToSet", bson.D{{"v", int32(42)}}}},
		},
		"Int64": {
			update: bson.D{{"$addToSet", bson.D{{"v", int64(42)}}}},
		},
		"Float64": {
			update: bson.D{{"$addToSet", bson.D{{"v", float64(42)}}}},
		},
		"NonExistentField": {
			update: bson.D{{"$addToSet", bson.D{{"non-existent-field", int32(42)}}}},
		},
		"DotNotation": {
			filter: bson.D{{"_id", "array-documents-nested"}},
			update: bson.D{{"$addToSet", bson.D{{"v.0.foo", bson.D{{"bar", "zoo"}}}}}},
		},
		"DotNotationNonArray": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$addToSet", bson.D{{"v.0.foo.0.bar", int32(1)}}}},
			resultType: emptyResult,
		},
		"DotNotationNonExistentPath": {
			update: bson.D{{"$addToSet", bson.D{{"non.existent.path", int32(1)}}}},
		},
		"EmptyValue": {
			update:     bson.D{{"$addToSet", bson.D{}}},
			resultType: emptyResult,
		},
	}

	testUpdateCompat(t, testCases)
}

// TestUpdateArrayCompatPullAll tests the $pullAll update operator.
func TestUpdateArrayCompatPullAll(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"DuplicateKeys": {
			update:     bson.D{{"$pullAll", bson.D{{"v", bson.A{int32(1)}}, {"v", bson.A{int32(1)}}}}},
			resultType: emptyResult,
		},
		"StringValue": {
			update:     bson.D{{"$pullAll", bson.D{{"v", "foo"}}}},
			resultType: emptyResult,
		},
		"String": {
			update: bson.D{{"$pullAll", bson.D{{"v", bson.A{"foo"}}}}},
		},
		"Document": {
			update: bson.D{{"$pullAll", bson.D{{"v", bson.A{bson.D{{"field", int32(42)}}}}}}},
		},
		"Int32": {
			update: bson.D{{"$pullAll", bson.D{{"v", bson.A{int32(42)}}}}},
		},
		"Int32-Six-Elements": {
			update: bson.D{{"$pullAll", bson.D{{"v", bson.A{int32(42), int32(43)}}}}},
		},
		"Int64": {
			update: bson.D{{"$pullAll", bson.D{{"v", bson.A{int64(42)}}}}},
		},
		"Float64": {
			update: bson.D{{"$pullAll", bson.D{{"v", bson.A{float64(42)}}}}},
		},
		"NonExistentField": {
			update:     bson.D{{"$pullAll", bson.D{{"non-existent-field", bson.A{int32(42)}}}}},
			resultType: emptyResult,
		},
		"NonExistentFieldUpsert": {
			filter:     bson.D{{"_id", "non-existent"}},
			update:     bson.D{{"$pullAll", bson.D{{"non-existent-field", bson.A{int32(42)}}}}},
			updateOpts: options.Update().SetUpsert(true),
			providers:  []shareddata.Provider{shareddata.Int32s},
		},
		"NotSuitableField": {
			filter:     bson.D{{"_id", "int32"}},
			update:     bson.D{{"$pullAll", bson.D{{"v.foo", bson.A{int32(42)}}}}},
			providers:  []shareddata.Provider{shareddata.Int32s},
			resultType: emptyResult,
		},
		"DotNotation": {
			filter:    bson.D{{"_id", "array-documents-nested"}},
			update:    bson.D{{"$pullAll", bson.D{{"v.0.foo", bson.A{bson.D{{"bar", "hello"}}}}}}},
			providers: []shareddata.Provider{shareddata.ArrayDocuments},
		},
		"DotNotationNonArray": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$pullAll", bson.D{{"v.0.foo.0.bar", bson.A{int32(42)}}}}},
			providers:  []shareddata.Provider{shareddata.ArrayDocuments},
			resultType: emptyResult,
		},
		"DotNotationNonExistentPath": {
			update:     bson.D{{"$pullAll", bson.D{{"non.existent.path", bson.A{int32(42)}}}}},
			resultType: emptyResult,
		},
		"PathNonExistentIndex": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$pullAll", bson.D{{"v.0.foo.2.bar", bson.A{int32(42)}}}}},
			providers:  []shareddata.Provider{shareddata.ArrayDocuments},
			resultType: emptyResult,
		},
		"PathInvalidIndex": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$pullAll", bson.D{{"v.-1.foo", bson.A{int32(42)}}}}},
			providers:  []shareddata.Provider{shareddata.ArrayDocuments},
			resultType: emptyResult,
		},
		"EmptyValue": {
			update:     bson.D{{"$pullAll", bson.D{}}},
			resultType: emptyResult,
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateArrayCompatAddToSetEach(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"Document": {
			update: bson.D{{"$addToSet", bson.D{{"v", bson.D{
				{"$each", bson.A{bson.D{{"field", int32(42)}}}},
			}}}}},
		},
		"String": {
			update: bson.D{{"$addToSet", bson.D{{"v", bson.D{{"$each", bson.A{"foo"}}}}}}},
		},
		"Int32": {
			update: bson.D{{"$addToSet", bson.D{{"v", bson.D{
				{"$each", bson.A{int32(1), int32(42), int32(2)}},
			}}}}},
		},
		"NotArray": {
			update:     bson.D{{"$addToSet", bson.D{{"v", bson.D{{"$each", int32(1)}}}}}},
			resultType: emptyResult,
		},
		"EmptyArray": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$addToSet", bson.D{{"v", bson.D{{"$each", bson.A{}}}}}}},
			resultType: emptyResult,
		},
		"ArrayMixedValuesExists": {
			update: bson.D{{"$addToSet", bson.D{{"v", bson.D{{"$each", bson.A{int32(42), "foo"}}}}}}},
		},
		"NonExistentField": {
			update: bson.D{{"$addToSet", bson.D{{"non-existent-field", bson.D{{"$each", bson.A{int32(42)}}}}}}},
		},
		"DotNotation": {
			update: bson.D{{"$addToSet", bson.D{{"v.0.foo", bson.D{{"$each", bson.A{int32(42)}}}}}}},
		},
		"DotNotationNonArray": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$addToSet", bson.D{{"v.0.foo.0.bar", bson.D{{"$each", bson.A{int32(42)}}}}}}},
			resultType: emptyResult,
		},
		"DotNotatPathNotExist": {
			update: bson.D{{"$addToSet", bson.D{{"non.existent.path", bson.D{{"$each", bson.A{int32(42)}}}}}}},
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateArrayCompatPushEach(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"Document": {
			update: bson.D{{"$push", bson.D{{"v", bson.D{
				{"$each", bson.A{bson.D{{"field", int32(42)}}}},
			}}}}},
		},
		"String": {
			update: bson.D{{"$push", bson.D{{"v", bson.D{{"$each", bson.A{"foo"}}}}}}},
		},
		"Int32": {
			update: bson.D{{"$push", bson.D{{"v", bson.D{
				{"$each", bson.A{int32(1), int32(42), int32(2)}},
			}}}}},
		},
		"NotArray": {
			update:     bson.D{{"$push", bson.D{{"v", bson.D{{"$each", int32(1)}}}}}},
			resultType: emptyResult,
		},
		"EmptyArray": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$push", bson.D{{"v", bson.D{{"$each", bson.A{}}}}}}},
			resultType: emptyResult,
		},
		"MixedValuesExists": {
			update: bson.D{{"$push", bson.D{{"v", bson.D{{"$each", bson.A{int32(42), "foo"}}}}}}},
		},
		"NonExistentField": {
			update: bson.D{{"$push", bson.D{{"non-existent-field", bson.D{{"$each", bson.A{int32(42)}}}}}}},
		},
		"DotNotation": {
			update: bson.D{{"$push", bson.D{{"v.0.foo", bson.D{{"$each", bson.A{int32(42)}}}}}}},
		},
		"DotNotationNonArray": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$push", bson.D{{"v.0.foo.0.bar", bson.D{{"$each", bson.A{int32(42)}}}}}}},
			resultType: emptyResult,
		},
		"DotNotationPathNotExist": {
			update: bson.D{{"$push", bson.D{{"non.existent.path", bson.D{{"$each", bson.A{int32(42)}}}}}}},
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateArrayCompatPull(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"Int32": {
			update: bson.D{{"$pull", bson.D{{"v", int32(42)}}}},
		},
		"String": {
			update: bson.D{{"$pull", bson.D{{"v", "foo"}}}},
		},
		"StringDuplicates": {
			update: bson.D{{"$pull", bson.D{{"v", "b"}}}},
		},
		"FieldNotExist": {
			update:     bson.D{{"$pull", bson.D{{"non-existent-field", int32(42)}}}},
			resultType: emptyResult,
		},
		"FieldNotExistUpsert": {
			filter:     bson.D{{"_id", "non-existent"}},
			update:     bson.D{{"$pull", bson.D{{"non-existent-field", int32(42)}}}},
			updateOpts: options.Update().SetUpsert(true),
			providers:  []shareddata.Provider{shareddata.Int32s},
		},
		"Array": {
			update:     bson.D{{"$pull", bson.D{{"v", bson.A{int32(42)}}}}},
			resultType: emptyResult,
		},
		"Null": {
			update: bson.D{{"$pull", bson.D{{"v", nil}}}},
		},
		"DotNotation": {
			update: bson.D{{"$pull", bson.D{{"v.0.foo", bson.D{{"bar", "hello"}}}}}},
		},
		"DotNotationPathNotExist": {
			update:     bson.D{{"$pull", bson.D{{"non.existent.path", int32(42)}}}},
			resultType: emptyResult,
		},
		"DotNotationNotArray": {
			update:     bson.D{{"$pull", bson.D{{"v.0.foo.0.bar", int32(42)}}}},
			resultType: emptyResult,
		},
		"PathNonExistentIndex": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$pull", bson.D{{"v.0.foo.2.bar", int32(42)}}}},
			providers:  []shareddata.Provider{shareddata.ArrayDocuments},
			resultType: emptyResult,
		},
		"PathInvalidIndex": {
			filter:     bson.D{{"_id", "array-documents-nested"}},
			update:     bson.D{{"$pull", bson.D{{"v.-1.foo", int32(42)}}}},
			providers:  []shareddata.Provider{shareddata.ArrayDocuments},
			resultType: emptyResult,
		},
	}

	testUpdateCompat(t, testCases)
}

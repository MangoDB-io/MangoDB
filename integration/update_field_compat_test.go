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
	"math"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestUpdateFieldCompatCurrentDate(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"DuplicateKeys": {
			update: bson.D{{"$currentDate", bson.D{
				{"v", bson.D{{"$type", "timestamp"}}},
				{"v", bson.D{{"$type", "timestamp"}}},
			}}},
			resultType: emptyResult,
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateFieldCompatInc(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"Int32": {
			update: bson.D{{"$inc", bson.D{{"v", int32(42)}}}},
		},
		"Int32Negative": {
			update: bson.D{{"$inc", bson.D{{"v", int32(-42)}}}},
		},
		"Int64Max": {
			update: bson.D{{"$inc", bson.D{{"v", math.MaxInt64}}}},
		},
		"Int64Min": {
			update: bson.D{{"$inc", bson.D{{"v", math.MinInt64}}}},
		},
		"EmptyUpdatePath": {
			update: bson.D{{"$inc", bson.D{{}}}},
			skip:   "https://github.com/FerretDB/FerretDB/issues/673",
		},
		"DotNotationFieldExist": {
			update:        bson.D{{"$inc", bson.D{{"v.foo", int32(1)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1088",
		},
		"DotNotationFieldNotExist": {
			update:        bson.D{{"$inc", bson.D{{"foo.bar", int32(1)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1088",
		},
		"DuplicateKeys": {
			update:     bson.D{{"$inc", bson.D{{"v", int32(42)}, {"v", int32(43)}}}},
			resultType: emptyResult,
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateFieldCompatMax(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"Int32Lower": {
			update:        bson.D{{"$max", bson.D{{"v", int32(30)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"Int32Higher": {
			update:        bson.D{{"$max", bson.D{{"v", int32(60)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"Int32Negative": {
			update:        bson.D{{"$max", bson.D{{"v", int32(-22)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"Document": {
			update: bson.D{{"$max", bson.D{{"v", bson.D{{"foo", "bar"}}}}}},
			skip:   "https://github.com/FerretDB/FerretDB/issues/457",
		},
		"EmptyDocument": {
			update: bson.D{{"$max", bson.D{{"v", bson.D{{}}}}}},
			skip:   "https://github.com/FerretDB/FerretDB/issues/457",
		},
		"Double": {
			update: bson.D{{"$max", bson.D{{"v", 54.32}}}},
		},
		"DoubleNegative": {
			update:        bson.D{{"$max", bson.D{{"v", -54.32}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"NotExisting": {
			update:        bson.D{{"$max", bson.D{{"v", int32(60)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},

		"MultipleQueries": {
			update:        bson.D{{"$max", bson.D{{"a", int32(30)}, {"v", int32(39)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"MultipleQueriesSorted": {
			update:        bson.D{{"$max", bson.D{{"v", int32(39)}, {"a", int32(30)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"DuplicateKeys": {
			update:     bson.D{{"$max", bson.D{{"v", int32(39)}, {"v", int32(30)}}}},
			resultType: emptyResult,
		},

		// Strings are not converted to numbers (except for Tigris with int64 fields)
		"StringIntegerHigher": {
			update:        bson.D{{"$max", bson.D{{"v", "60"}}}},
			skipForTigris: "In compat collection `v` will be a string, in Tigris - a number.",
		},
		"StringIntegerLower": {
			update:        bson.D{{"$max", bson.D{{"v", "30"}}}},
			skipForTigris: "In compat collection `v` will be a string, in Tigris - a number.",
		},
		"StringDouble": {
			update: bson.D{{"$max", bson.D{{"v", "54.32"}}}},
		},
		"StringDoubleNegative": {
			update: bson.D{{"$max", bson.D{{"v", "-54.32"}}}},
		},
		"StringLexicographicHigher": {
			update: bson.D{{"$max", bson.D{{"v", "goo"}}}},
		},
		"StringLexicographicLower": {
			update: bson.D{{"$max", bson.D{{"v", "eoo"}}}},
		},
		"StringLexicographicUpperCase": {
			update: bson.D{{"$max", bson.D{{"v", "Foo"}}}},
		},
		"BoolTrue": {
			update: bson.D{{"$max", bson.D{{"v", true}}}},
		},
		"BoolFalse": {
			update:        bson.D{{"$max", bson.D{{"v", false}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"EmptyOperand": {
			update:     bson.D{{"$max", bson.D{}}},
			resultType: emptyResult,
		},
		"DateTime": {
			update:        bson.D{{"$max", bson.D{{"v", primitive.NewDateTimeFromTime(time.Date(2021, 11, 1, 12, 18, 42, 123000000, time.UTC))}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"DateTimeLower": {
			update:        bson.D{{"$max", bson.D{{"v", primitive.NewDateTimeFromTime(time.Date(2021, 11, 1, 3, 18, 42, 123000000, time.UTC))}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayEmpty": {
			update:        bson.D{{"$max", bson.D{{"v", bson.A{}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayOne": {
			update:        bson.D{{"$max", bson.D{{"v", bson.A{int32(42)}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"Array": {
			update:        bson.D{{"$max", bson.D{{"v", bson.A{int32(42), "foo", nil}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayReverse": {
			update:        bson.D{{"$max", bson.D{{"v", bson.A{nil, "foo", int32(42)}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayNull": {
			update:        bson.D{{"$max", bson.D{{"v", bson.A{nil}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArraySlice": {
			update:        bson.D{{"$max", bson.D{{"v", bson.A{int32(42), "foo"}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayShuffledValues": {
			update:        bson.D{{"$max", bson.D{{"v", bson.A{"foo", nil, int32(42)}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayDocuments": {
			update:        bson.D{{"$max", bson.D{{"v", bson.A{bson.D{{"foo", int32(42)}}, bson.D{{"foo", nil}}}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateFieldCompatMin(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"Int32Lower": {
			update:        bson.D{{"$min", bson.D{{"v", int32(30)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"Int32Higher": {
			update:        bson.D{{"$min", bson.D{{"v", int32(60)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"Int32Negative": {
			update:        bson.D{{"$min", bson.D{{"v", int32(-22)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"Document": {
			update: bson.D{{"$min", bson.D{{"v", bson.D{{"foo", "bar"}}}}}},
			skip:   "https://github.com/FerretDB/FerretDB/issues/457",
		},
		"EmptyDocument": {
			update: bson.D{{"$min", bson.D{{"v", bson.D{{}}}}}},
			skip:   "https://github.com/FerretDB/FerretDB/issues/457",
		},
		"Double": {
			update: bson.D{{"$min", bson.D{{"v", 54.32}}}},
		},
		"DoubleNegative": {
			update:        bson.D{{"$min", bson.D{{"v", -54.32}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"NotExisting": {
			update:        bson.D{{"$min", bson.D{{"v", int32(60)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"MultipleQueries": {
			update:        bson.D{{"$min", bson.D{{"a", int32(30)}, {"v", int32(39)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"MultipleQueriesSorted": {
			update:        bson.D{{"$min", bson.D{{"v", int32(39)}, {"a", int32(30)}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"DuplicateKeys": {
			update:     bson.D{{"$min", bson.D{{"v", int32(39)}, {"v", int32(30)}}}},
			resultType: emptyResult,
		},
		"StringIntegerHigher": {
			update:        bson.D{{"$min", bson.D{{"v", "60"}}}},
			skipForTigris: "In compat collection `v` will be a string, in Tigris - a number.",
		},
		"StringIntegerLower": {
			update:        bson.D{{"$min", bson.D{{"v", "30"}}}},
			skipForTigris: "In compat collection `v` will be a string, in Tigris - a number.",
		},
		"StringDouble": {
			update: bson.D{{"$min", bson.D{{"v", "54.32"}}}},
		},
		"StringDoubleNegative": {
			update: bson.D{{"$min", bson.D{{"v", "-54.32"}}}},
		},
		"StringLexicographicHigher": {
			update: bson.D{{"$min", bson.D{{"v", "goo"}}}},
		},
		"StringLexicographicLower": {
			update: bson.D{{"$min", bson.D{{"v", "eoo"}}}},
		},
		"StringLexicographicUpperCase": {
			update: bson.D{{"$min", bson.D{{"v", "Foo"}}}},
		},
		"BoolTrue": {
			update:        bson.D{{"$min", bson.D{{"v", true}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"BoolFalse": {
			update:        bson.D{{"$min", bson.D{{"v", false}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"EmptyOperand": {
			update:     bson.D{{"$min", bson.D{}}},
			resultType: emptyResult,
		},
		"DateTime": {
			update:        bson.D{{"$min", bson.D{{"v", primitive.NewDateTimeFromTime(time.Date(2021, 11, 1, 12, 18, 42, 123000000, time.UTC))}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"DateTimeLower": {
			update:        bson.D{{"$min", bson.D{{"v", primitive.NewDateTimeFromTime(time.Date(2021, 11, 1, 3, 18, 42, 123000000, time.UTC))}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayEmpty": {
			update:        bson.D{{"$min", bson.D{{"v", bson.A{}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayOne": {
			update:        bson.D{{"$min", bson.D{{"v", bson.A{int32(42)}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"Array": {
			update:        bson.D{{"$min", bson.D{{"v", bson.A{int32(42), "foo", nil}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayReverse": {
			update:        bson.D{{"$min", bson.D{{"v", bson.A{nil, "foo", int32(42)}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayNull": {
			update:        bson.D{{"$min", bson.D{{"v", bson.A{nil}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArraySlice": {
			update:        bson.D{{"$min", bson.D{{"v", bson.A{int32(42), "foo"}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayShuffledValues": {
			update:        bson.D{{"$min", bson.D{{"v", bson.A{"foo", nil, int32(42)}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
		"ArrayDocuments": {
			update:        bson.D{{"$min", bson.D{{"v", bson.A{bson.D{{"foo", int32(42)}}, bson.D{{"foo", nil}}}}}}},
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/1061",
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateFieldCompatUnset(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"Simple": {
			update: bson.D{{"$unset", bson.D{{"v", ""}}}},
		},
		"NonExisting": {
			update:     bson.D{{"$unset", bson.D{{"foo", ""}}}},
			resultType: emptyResult,
		},
		"Nested": {
			update: bson.D{{"$unset", bson.D{{"v", bson.D{{"array", ""}}}}}},
		},
		"DotDocument": {
			update: bson.D{{"$unset", bson.D{{"v.foo", ""}}}},
		},
		"DotDocumentNonExisting": {
			update:     bson.D{{"$unset", bson.D{{"foo.bar", ""}}}},
			resultType: emptyResult,
		},
		"DotArrayField": {
			update:        bson.D{{"$unset", bson.D{{"v.array.0", ""}}}},
			skip:          "https://github.com/FerretDB/FerretDB/issues/1242",
			skipForTigris: "https://github.com/FerretDB/FerretDB/issues/908",
		},
		"DotArrayNonExisting": {
			update:     bson.D{{"$unset", bson.D{{"foo.0.baz", int32(1)}}}},
			resultType: emptyResult,
		},
		"DuplicateKeys": {
			update:     bson.D{{"$unset", bson.D{{"v", ""}, {"v", ""}}}},
			resultType: emptyResult,
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateFieldCompatSet(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"SetNullInExisingField": {
			update: bson.D{{"$set", bson.D{{"v", nil}}}},
		},
		"DuplicateKeys": {
			update:     bson.D{{"$set", bson.D{{"v", 42}, {"v", "hello"}}}},
			resultType: emptyResult,
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateFieldCompatSetOnInsert(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"DuplicateKeys": {
			update:     bson.D{{"$setOnInsert", bson.D{{"v", 1}, {"v", 2}}}},
			resultType: emptyResult,
		},
	}

	testUpdateCompat(t, testCases)
}

func TestUpdateFieldCompatPop(t *testing.T) {
	t.Parallel()

	testCases := map[string]updateCompatTestCase{
		"DuplicateKeys": {
			update:     bson.D{{"$pop", bson.D{{"v", 1}, {"v", 1}}}},
			resultType: emptyResult,
		},
	}

	testUpdateCompat(t, testCases)
}

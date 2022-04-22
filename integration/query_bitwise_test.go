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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/integration/shareddata"
)

func TestQueryBitwiseAllClear(t *testing.T) {
	t.Parallel()
	ctx, collection := setup(t, shareddata.Scalars)

	// TODO: return this after fixing filtering against types.Binary.
	//_, err := collection.InsertMany(ctx, []any{
	//	bson.D{{"_id", "binary-big"}, {"value", primitive.Binary{Data: []byte{0, 0, 128}}}},
	//	bson.D{{"_id", "binary-user-1"}, {"value", primitive.Binary{Subtype: 0x80, Data: []byte{0, 0, 30}}}},
	//	bson.D{{"_id", "binary-user-2"}, {"value", primitive.Binary{Subtype: 0x80, Data: []byte{15, 0, 0, 0}}}},
	//	bson.D{{"_id", "binary-user-3"}, {"value", primitive.Binary{Data: []byte{15, 0, 0, 0}}}},
	//	bson.D{{"_id", "binary-user-4"}, {"value", primitive.Binary{Data: []byte{0, 0, 30}}}},
	//})
	//require.NoError(t, err)

	for name, tc := range map[string]struct {
		value       any
		expectedIDs []any
		err         mongo.CommandError
	}{
		"Double": {
			value: 1.2,
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected an integer: $bitsAllClear: 1.2",
			},
		},
		"DoubleWhole": {
			value: 2.0,
			expectedIDs: []any{
				"binary-empty",
				"double-negative-zero", "double-zero",
				"int32-min", "int32-zero",
				"int64-min", "int64-zero",
			},
		},
		"DoubleNegativeValue": {
			value: float64(-1),
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAllClear: -1.0",
			},
		},
		// TODO: return this after types.Binary filtering fixed.
		//"BinaryOneByte": {
		//	value: primitive.Binary{Data: []byte{2}},
		//	expectedIDs: []any{
		//		"binary-empty",
		//		"double-negative-zero", "double-zero",
		//		"int32-min", "int32-zero",
		//		"int64-min", "int64-zero",
		//	},
		//},
		//"BinaryTwoBytes": {
		//	value: primitive.Binary{Data: []byte{2, 2}},
		//	expectedIDs: []any{
		//		"binary-empty",
		//		"double-negative-zero", "double-zero",
		//		"int32-min", "int32-zero",
		//		"int64-min", "int64-zero",
		//	},
		//},
		//"BinaryBig": {
		//	value: primitive.Binary{Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		//	expectedIDs: []any{
		//		"binary-empty",
		//		"double-negative-zero", "double-whole", "double-zero",
		//		"int32", "int32-zero",
		//		"int64", "int64-zero",
		//	},
		//},
		"String": {
			value: "123",
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: "value takes an Array, a number, or a BinData but received: $bitsAllClear: \"123\"",
			},
		},

		"Int32": {
			value: int32(2),
			expectedIDs: []any{
				"binary-empty",
				"double-negative-zero", "double-zero",
				"int32-min", "int32-zero",
				"int64-min", "int64-zero",
			},
		},
		"Int32NegativeValue": {
			value: int32(-1),
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAllClear: -1",
			},
		},

		"Int64": {
			value: math.MaxInt64,
			expectedIDs: []any{
				"binary-empty",
				"double-negative-zero", "double-zero",
				"int32-zero",
				"int64-min", "int64-zero",
			},
		},
		"Int64NegativeValue": {
			value: int64(-1),
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAllClear: -1",
			},
		},

		"Array": {
			value: primitive.A{1, 5},
			expectedIDs: []any{
				"binary-empty",
				"double-negative-zero", "double-zero",
				"int32-min", "int32-zero",
				"int64-min", "int64-zero",
			},
		},
		"ArrayNegativeBitPositionValue": {
			value: primitive.A{-1},
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: "bit positions must be >= 0 but got: 0: -1",
			},
		},
		"ArrayBadValue": {
			value: primitive.A{"123"},
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: `bit positions must be an integer but got: 0: "123"`,
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			filter := bson.D{{"value", bson.D{{"$bitsAllClear", tc.value}}}}
			cursor, err := collection.Find(ctx, filter, options.Find().SetSort(bson.D{{"_id", 1}}))
			if tc.err.Code != 0 {
				require.Nil(t, tc.expectedIDs)
				AssertEqualError(t, tc.err, err)
				return
			}
			require.NoError(t, err)

			var actual []bson.D
			err = cursor.All(ctx, &actual)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedIDs, CollectIDs(t, actual))
		})
	}
}

func TestQueryBitwiseAllSet(t *testing.T) {
	t.Parallel()
	ctx, collection := setup(t, shareddata.Scalars)

	for name, tc := range map[string]struct {
		value       any
		expectedIDs []any
		err         mongo.CommandError
	}{
		"Double": {
			value: 1.2,
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected an integer: $bitsAllSet: 1.2",
			},
		},
		"DoubleWhole": {
			value:       2.0,
			expectedIDs: []any{"binary", "double-whole", "int32", "int32-max", "int64", "int64-max"},
		},
		"DoubleNegativeValue": {
			value: -1.0,
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAllSet: -1.0",
			},
		},

		"String": {
			value: "123",
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: "value takes an Array, a number, or a BinData but received: $bitsAllSet: \"123\"",
			},
		},

		"Int32": {
			value:       int32(2),
			expectedIDs: []any{"binary", "double-whole", "int32", "int32-max", "int64", "int64-max"},
		},
		"Int32NegativeValue": {
			value: int32(-1),
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAllSet: -1",
			},
		},

		"MaxInt64": {
			value:       math.MaxInt64,
			expectedIDs: []any{"int64-max"},
		},
		"Int64NegativeValue": {
			value: int64(-1),
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAllSet: -1",
			},
		},

		"Array": {
			value:       primitive.A{1, 5},
			expectedIDs: []any{"binary", "double-whole", "int32", "int32-max", "int64", "int64-max"},
		},
		"ArrayNegativeBitPositionValue": {
			value: primitive.A{-1},
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: "bit positions must be >= 0 but got: 0: -1",
			},
		},
		"ArrayBadValue": {
			value: primitive.A{"123"},
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: `bit positions must be an integer but got: 0: "123"`,
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			filter := bson.D{{"value", bson.D{{"$bitsAllSet", tc.value}}}}
			cursor, err := collection.Find(ctx, filter, options.Find().SetSort(bson.D{{"_id", 1}}))
			if tc.err.Code != 0 {
				require.Nil(t, tc.expectedIDs)
				AssertEqualError(t, tc.err, err)
				return
			}
			require.NoError(t, err)

			var actual []bson.D
			err = cursor.All(ctx, &actual)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedIDs, CollectIDs(t, actual))
		})
	}
}

func TestQueryBitwiseAnyClear(t *testing.T) {
	t.Parallel()
	ctx, collection := setup(t, shareddata.Scalars)

	for name, tc := range map[string]struct {
		value       any
		expectedIDs []any
		err         mongo.CommandError
	}{
		"Double": {
			value: 1.2,
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected an integer: $bitsAnyClear: 1.2",
			},
		},
		"DoubleWhole": {
			value: 2.0,
			expectedIDs: []any{
				"binary-empty",
				"double-negative-zero", "double-zero",
				"int32-min", "int32-zero",
				"int64-min", "int64-zero",
			},
		},
		"DoubleNegativeValue": {
			value: -1.0,
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAnyClear: -1.0",
			},
		},

		"String": {
			value: "123",
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: "value takes an Array, a number, or a BinData but received: $bitsAnyClear: \"123\"",
			},
		},

		"Int32": {
			value: int32(2),
			expectedIDs: []any{
				"binary-empty",
				"double-negative-zero", "double-zero",
				"int32-min", "int32-zero",
				"int64-min", "int64-zero",
			},
		},
		"Int32NegativeValue": {
			value: int32(-1),
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAnyClear: -1",
			},
		},

		"Int64": {
			value: math.MaxInt64,
			expectedIDs: []any{
				"binary", "binary-empty",
				"double-negative-zero", "double-whole", "double-zero",
				"int32", "int32-max", "int32-min", "int32-zero",
				"int64", "int64-min", "int64-zero",
			},
		},
		"Int64NegativeValue": {
			value: int64(-1),
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAnyClear: -1",
			},
		},

		"Array": {
			value: primitive.A{1, 5},
			expectedIDs: []any{
				"binary-empty",
				"double-negative-zero", "double-zero",
				"int32-min", "int32-zero",
				"int64-min", "int64-zero",
			},
		},
		"ArrayNegativeBitPositionValue": {
			value: primitive.A{-1},
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: "bit positions must be >= 0 but got: 0: -1",
			},
		},
		"ArrayBadValue": {
			value: primitive.A{"123"},
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: `bit positions must be an integer but got: 0: "123"`,
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			filter := bson.D{{"value", bson.D{{"$bitsAnyClear", tc.value}}}}
			cursor, err := collection.Find(ctx, filter, options.Find().SetSort(bson.D{{"_id", 1}}))
			if tc.err.Code != 0 {
				require.Nil(t, tc.expectedIDs)
				AssertEqualError(t, tc.err, err)
				return
			}
			require.NoError(t, err)

			var actual []bson.D
			err = cursor.All(ctx, &actual)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedIDs, CollectIDs(t, actual))
		})
	}
}

func TestQueryBitwiseAnySet(t *testing.T) {
	t.Parallel()
	ctx, collection := setup(t, shareddata.Scalars)

	for name, tc := range map[string]struct {
		value       any
		expectedIDs []any
		err         mongo.CommandError
	}{
		"Double": {
			value: 1.2,
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected an integer: $bitsAnySet: 1.2",
			},
		},
		"DoubleWhole": {
			value: 2.0,
			expectedIDs: []any{
				"binary",
				"double-whole",
				"int32", "int32-max",
				"int64", "int64-max",
			},
		},
		"DoubleNegativeValue": {
			value: -1.0,
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAnySet: -1.0",
			},
		},

		"String": {
			value: "123",
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: "value takes an Array, a number, or a BinData but received: $bitsAnySet: \"123\"",
			},
		},

		"Int32": {
			value: int32(2),
			expectedIDs: []any{
				"binary",
				"double-whole",
				"int32", "int32-max",
				"int64", "int64-max",
			},
		},
		"Int32NegativeValue": {
			value: int32(-1),
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAnySet: -1",
			},
		},

		"Int64": {
			value: math.MaxInt64,
			expectedIDs: []any{
				"binary",
				"double-whole",
				"int32", "int32-max", "int32-min",
				"int64", "int64-max",
			},
		},
		"Int64NegativeValue": {
			value: int64(-1),
			err: mongo.CommandError{
				Code:    9,
				Name:    "FailedToParse",
				Message: "Expected a positive number in: $bitsAnySet: -1",
			},
		},

		"Array": {
			value: primitive.A{1, 5},
			expectedIDs: []any{
				"binary",
				"double-whole",
				"int32", "int32-max",
				"int64", "int64-max",
			},
		},
		"ArrayNegativeBitPositionValue": {
			value: primitive.A{-1},
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: "bit positions must be >= 0 but got: 0: -1",
			},
		},
		"ArrayBadValue": {
			value: primitive.A{"123"},
			err: mongo.CommandError{
				Code:    2,
				Name:    "BadValue",
				Message: `bit positions must be an integer but got: 0: "123"`,
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			filter := bson.D{{"value", bson.D{{"$bitsAnySet", tc.value}}}}
			cursor, err := collection.Find(ctx, filter, options.Find().SetSort(bson.D{{"_id", 1}}))
			if tc.err.Code != 0 {
				require.Nil(t, tc.expectedIDs)
				AssertEqualError(t, tc.err, err)
				return
			}
			require.NoError(t, err)

			var actual []bson.D
			err = cursor.All(ctx, &actual)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedIDs, CollectIDs(t, actual))
		})
	}
}

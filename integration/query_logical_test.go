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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/FerretDB/FerretDB/integration/shareddata"
)

func TestQueryLogicalAnd(t *testing.T) {
	t.Parallel()
	ctx, collection := setupWithOpts(t, &setupOpts{
		providers: []shareddata.Provider{shareddata.Scalars},
	})

	for name, tc := range map[string]struct {
		q           bson.D
		expectedIDs []any
		err         error
	}{
		"And": {
			q: bson.D{{
				"$and",
				bson.A{
					bson.D{{"value", bson.D{{"$gt", 0}}}},
					bson.D{{"value", bson.D{{"$lte", 42}}}},
				},
			}},
			expectedIDs: []any{"double-smallest", "int32", "int64"},
		},
		"BadInput": {
			q: bson.D{{"$and", nil}},
			err: mongo.CommandError{
				Code:    2,
				Message: "$and must be an array",
				Name:    "BadValue",
			},
		},
		"BadExpressionValue": {
			q: bson.D{{
				"$and",
				bson.A{
					bson.D{{"value", bson.D{{"$gt", 0}}}},
					nil,
				},
			}},
			err: mongo.CommandError{
				Code:    2,
				Message: "$or/$and/$nor entries need to be full objects",
				Name:    "BadValue",
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var actual []bson.D
			cursor, err := collection.Find(ctx, tc.q)
			if tc.err != nil {
				require.Nil(t, tc.expectedIDs)
				assertEqualError(t, tc.err.(mongo.CommandError), err)
				return
			}
			require.NoError(t, err)
			err = cursor.All(ctx, &actual)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedIDs, collectIDs(t, actual))
		})
	}
}

func TestQueryLogicalOr(t *testing.T) {
	t.Parallel()
	ctx, collection := setupWithOpts(t, &setupOpts{
		providers: []shareddata.Provider{shareddata.Scalars},
	})

	for name, tc := range map[string]struct {
		q           bson.D
		expectedIDs []any
		err         error
	}{
		"Or": {
			q: bson.D{{
				"$or",
				bson.A{
					bson.D{{"value", bson.D{{"$lt", 0}}}},
					bson.D{{"value", bson.D{{"$lt", 42}}}},
				},
			}},
			expectedIDs: []any{
				"double-negative-infinity", "double-negative-zero",
				"double-smallest", "double-zero",
				"int32-min", "int32-zero", "int64-min", "int64-zero"},
		},
		"BadInput": {
			q: bson.D{{"$or", nil}},
			err: mongo.CommandError{
				Code:    2,
				Message: "$or must be an array",
				Name:    "BadValue",
			},
		},
		"BadExpressionValue": {
			q: bson.D{{
				"$or",
				bson.A{
					bson.D{{"value", bson.D{{"$gt", 0}}}},
					nil,
				},
			}},
			err: mongo.CommandError{
				Code:    2,
				Message: "$or/$and/$nor entries need to be full objects",
				Name:    "BadValue",
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var actual []bson.D
			cursor, err := collection.Find(ctx, tc.q)
			if tc.err != nil {
				require.Nil(t, tc.expectedIDs)
				assertEqualError(t, tc.err.(mongo.CommandError), err)
				return
			}
			require.NoError(t, err)
			err = cursor.All(ctx, &actual)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedIDs, collectIDs(t, actual))
		})
	}
}

func TestQueryLogicalNor(t *testing.T) {
	t.Parallel()
	ctx, collection := setupWithOpts(t, &setupOpts{
		providers: []shareddata.Provider{shareddata.Scalars},
	})

	for name, tc := range map[string]struct {
		q           bson.D
		expectedIDs []any
		err         error
	}{
		"Nor": {
			q: bson.D{{
				"$nor",
				bson.A{
					bson.D{{"value", bson.D{{"$gt", 0}}}},
					bson.D{{"value", bson.D{{"$gt", 42}}}},
				},
			}},
			expectedIDs: []any{
				"binary", "binary-empty", "bool-false", "bool-true",
				"datetime", "datetime-epoch", "datetime-year-max", "datetime-year-min",
				"double-nan", "double-negative-infinity", "double-negative-zero", "double-zero",
				"int32-min", "int32-zero", "int64-min", "int64-zero",
				"null", "regex", "regex-empty", "string", "string-empty", "timestamp", "timestamp-i",
			},
		},
		"BadInput": {
			q: bson.D{{"$nor", nil}},
			err: mongo.CommandError{
				Code:    2,
				Message: "$nor must be an array",
				Name:    "BadValue",
			},
		},
		"BadExpressionValue": {
			q: bson.D{{
				"$nor",
				bson.A{
					bson.D{{"value", bson.D{{"$gt", 0}}}},
					nil,
				},
			}},
			err: mongo.CommandError{
				Code:    2,
				Message: "$or/$and/$nor entries need to be full objects",
				Name:    "BadValue",
			},
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var actual []bson.D
			cursor, err := collection.Find(ctx, tc.q)
			if tc.err != nil {
				require.Nil(t, tc.expectedIDs)
				assertEqualError(t, tc.err.(mongo.CommandError), err)
				return
			}
			require.NoError(t, err)
			err = cursor.All(ctx, &actual)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedIDs, collectIDs(t, actual))
		})
	}
}

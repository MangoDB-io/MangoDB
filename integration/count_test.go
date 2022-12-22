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
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/FerretDB/FerretDB/integration/setup"
	"github.com/FerretDB/FerretDB/integration/shareddata"
)

func TestQueryBadCountType(t *testing.T) {
	t.Parallel()
	s := setup.SetupWithOpts(t, &setup.SetupOpts{
		Providers: []shareddata.Provider{shareddata.Scalars, shareddata.Composites},
	})

	ctx, collection := s.Ctx, s.Collection

	for name, tc := range map[string]struct {
		value any
		err   string
	}{
		"Document": {
			value: bson.D{},
			err:   "object",
		},
		"Array": {
			value: primitive.A{},
			err:   "array",
		},
		"Double": {
			value: 3.14,

			err: "double",
		},
		"Binary": {
			value: primitive.Binary{},
			err:   "binData",
		},
		"ObjectID": {
			value: primitive.ObjectID{},

			err: "objectId",
		},
		"Bool": {
			value: true,
			err:   "bool",
		},
		"Date": {
			value: time.Now(),

			err: "date",
		},
		"Null": {
			value: nil,

			err: "null",
		},
		"Regex": {
			value: primitive.Regex{Pattern: "/foo/"},

			err: "regex",
		},
		"Int": {
			value: int32(42),
			err:   "int",
		},
		"Timestamp": {
			value: primitive.Timestamp{},
			err:   "timestamp",
		},
		"Long": {
			value: int64(42),
			err:   "long",
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var actual bson.D
			err := collection.Database().
				RunCommand(ctx, bson.D{
					{"count", tc.value},
					{"query", bson.D{{"v", "some"}}},
				}).
				Decode(&actual)
			require.Error(t, err)

			var expectedErr mongo.CommandError
			if s.IsAuthEnabled(t) {
				expectedErr = mongo.CommandError{
					Code:    73,
					Name:    "InvalidNamespace",
					Message: "Failed to parse namespace element",
				}
			} else {
				expectedErr = mongo.CommandError{
					Code:    2,
					Name:    "BadValue",
					Message: "collection name has invalid type: " + tc.err,
				}
			}

			AssertEqualError(t, expectedErr, err)
		})
	}
}

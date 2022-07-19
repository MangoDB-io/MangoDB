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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestQueryArrayCompatAll(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"All": {
			filter: bson.D{{
				"$and",
				bson.A{
					bson.D{{"_id", bson.D{{"$not", bson.D{{"$regex", primitive.Regex{Pattern: "array"}}}}}}},
					bson.D{{"_id", bson.D{{"$not", bson.D{{"$regex", primitive.Regex{Pattern: "document"}}}}}}},
					bson.D{{"value", bson.D{{"$all", bson.A{42}}}}},
				},
			}},
		},
	}

	// e := bson.D{{"_id", bson.D{{"$not", bson.D{{"$regex", primitive.Regex{Pattern: "array"}}}}}}}

	testQueryCompat(t, testCases)
}

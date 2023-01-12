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
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/integration/setup"
	"github.com/FerretDB/FerretDB/integration/shareddata"
)

type queryGetMoreCompatTestCase struct {
	batchSize int
	limit     int
	sort      bson.D
}

func testGetMoreCompat(t *testing.T, testCases map[string]queryGetMoreCompatTestCase) {
	t.Helper()

	res := setup.SetupCompatWithOpts(t, &setup.SetupCompatOpts{
		Providers: []shareddata.Provider{
			shareddata.Int32BigAmounts,
		},
	})

	ctx, targetCollections, compatCollections := res.Ctx, res.TargetCollections, res.CompatCollections

	for name, tc := range testCases {
		name, _ := name, tc
		t.Run(name, func(t *testing.T) {
			t.Helper()

			t.Parallel()

			for i := range targetCollections {
				targetCollection := targetCollections[i]
				compatCollection := compatCollections[i]
				t.Run(targetCollection.Name(), func(t *testing.T) {
					t.Helper()

					sort := tc.sort
					if sort == nil {
						sort = bson.D{{"_id", 1}}
					}
					opts := options.Find().SetSort(sort)

					targetResult, targetErr := targetCollection.Find(ctx, bson.D{}, opts)
					compatResult, compatErr := compatCollection.Find(ctx, bson.D{}, opts)

					if targetErr != nil {
						t.Logf("Target error: %v", targetErr)
						AssertMatchesCommandError(t, compatErr, targetErr)

						return
					}
					require.NoError(t, compatResult.Err(), "compat error; target returned no error")

					var targetRes, compatRes []bson.D
					require.NoError(t, targetResult.All(ctx, &targetRes))
					require.NoError(t, compatResult.All(ctx, &compatRes))

					assert.Equal(t, len(compatRes), len(targetRes), "result length mismatch")
				})
			}
		})
	}
}

func TestGetMore(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryGetMoreCompatTestCase{
		"getMore": {},
	}

	testGetMoreCompat(t, testCases)
}

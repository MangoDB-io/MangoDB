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

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/integration/setup"
	"github.com/FerretDB/FerretDB/integration/shareddata"
)

// queryCompatTestCase describes query compatibility test case.
type queryCompatTestCase struct {
	filter         bson.D                   // required
	sort           bson.D                   // defaults to `bson.D{{"_id", 1}}`
	optSkip        *int64                   // defaults to nil to leave unset
	limit          *int64                   // defaults to nil to leave unset
	batchSize      *int32                   // defaults to nil to leave unset
	projection     bson.D                   // nil for leaving projection unset
	resultType     compatTestCaseResultType // defaults to nonEmptyResult
	resultPushdown ResultPushdown           // defaults to false

	skipIDCheck bool   // skip check collected IDs, use it when no ids returned from query
	skip        string // skip test for all handlers, must have issue number mentioned
}

func testQueryCompatWithProviders(t *testing.T, providers shareddata.Providers, testCases map[string]queryCompatTestCase) {
	t.Helper()

	require.NotEmpty(t, providers)

	// Use shared setup because find queries can't modify data.
	//
	// Use read-only user.
	// TODO https://github.com/FerretDB/FerretDB/issues/1025
	s := setup.SetupCompatWithOpts(t, &setup.SetupCompatOpts{
		Providers: providers,
	})

	ctx, targetCollections, compatCollections := s.Ctx, s.TargetCollections, s.CompatCollections

	for name, tc := range testCases {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Helper()

			if tc.skip != "" {
				t.Skip(tc.skip)
			}

			t.Parallel()

			filter := tc.filter
			require.NotNil(t, filter, "filter should be set")

			opts := options.Find()

			opts.SetSort(tc.sort)
			if tc.sort == nil {
				opts.SetSort(bson.D{{"_id", 1}})
			}

			if tc.optSkip != nil {
				opts.SetSkip(*tc.optSkip)
			}

			if tc.limit != nil {
				opts.SetLimit(*tc.limit)
			}

			if tc.batchSize != nil {
				opts.SetBatchSize(*tc.batchSize)
			}

			if tc.projection != nil {
				opts.SetProjection(tc.projection)
			}

			var nonEmptyResults bool
			for i := range targetCollections {
				targetCollection := targetCollections[i]
				compatCollection := compatCollections[i]
				t.Run(targetCollection.Name(), func(t *testing.T) {
					t.Helper()

					targetIdx, tagetErr := targetCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
						Keys: bson.D{{"v", 1}},
					})
					compatIdx, compatErr := compatCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
						Keys: bson.D{{"v", 1}},
					})

					require.NoError(t, tagetErr)
					require.NoError(t, compatErr)
					require.Equal(t, compatIdx, targetIdx)

					// don't add sort, limit, skip, and projection because we don't pushdown them yet
					explainQuery := bson.D{{"explain", bson.D{
						{"find", targetCollection.Name()},
						{"filter", filter},
					}}}

					var explainRes bson.D
					require.NoError(t, targetCollection.Database().RunCommand(ctx, explainQuery).Decode(&explainRes))

					resultPushdown := tc.resultPushdown

					var msg string
					if setup.IsPushdownDisabled() {
						resultPushdown = noPushdown
						msg = "Query pushdown is disabled, but target resulted with pushdown"
					}

					doc := ConvertDocument(t, explainRes)
					pushdown, _ := doc.Get("pushdown")
					assert.Equal(t, resultPushdown.PushdownExpected(t), pushdown, msg)

					targetCursor, targetErr := targetCollection.Find(ctx, filter, opts)
					compatCursor, compatErr := compatCollection.Find(ctx, filter, opts)

					if targetCursor != nil {
						defer targetCursor.Close(ctx)
					}
					if compatCursor != nil {
						defer compatCursor.Close(ctx)
					}

					if targetErr != nil {
						t.Logf("Target error: %v", targetErr)
						t.Logf("Compat error: %v", compatErr)

						// error messages are intentionally not compared
						AssertMatchesCommandError(t, compatErr, targetErr)

						return
					}
					require.NoError(t, compatErr, "compat error; target returned no error")

					targetRes := FetchAll(t, ctx, targetCursor)
					compatRes := FetchAll(t, ctx, compatCursor)

					if !tc.skipIDCheck {
						t.Logf("Compat (expected) IDs: %v", CollectIDs(t, compatRes))
						t.Logf("Target (actual)   IDs: %v", CollectIDs(t, targetRes))
					}

					AssertEqualDocumentsSlice(t, compatRes, targetRes)

					if len(targetRes) > 0 || len(compatRes) > 0 {
						nonEmptyResults = true
					}
				})
			}

			switch tc.resultType {
			case nonEmptyResult:
				assert.True(t, nonEmptyResults, "expected non-empty results")
			case emptyResult:
				assert.False(t, nonEmptyResults, "expected empty results")
			default:
				t.Fatalf("unknown result type %v", tc.resultType)
			}
		})
	}
}

// testQueryCompat tests query compatibility test cases.
func testQueryCompat(t *testing.T, testCases map[string]queryCompatTestCase) {
	t.Helper()

	testQueryCompatWithProviders(t, shareddata.AllProviders(), testCases)
}

func TestQueryCompatFilter(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"Empty": {
			filter: bson.D{},
		},
		"String": {
			filter:         bson.D{{"v", "foo"}},
			resultPushdown: allPushdown,
		},
		"Int32": {
			filter:         bson.D{{"v", int32(42)}},
			resultPushdown: allPushdown,
		},
		"IDString": {
			filter:         bson.D{{"_id", "string"}},
			resultPushdown: allPushdown,
		},
		"IDNilObjectID": {
			filter:         bson.D{{"_id", primitive.NilObjectID}},
			resultPushdown: allPushdown,
		},
		"IDObjectID": {
			filter:         bson.D{{"_id", primitive.ObjectID{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x10, 0x11}}},
			resultPushdown: allPushdown,
		},
		"ObjectID": {
			filter:         bson.D{{"v", primitive.NilObjectID}},
			resultPushdown: allPushdown,
		},
		"UnknownFilterOperator": {
			filter:     bson.D{{"v", bson.D{{"$someUnknownOperator", 42}}}},
			resultType: emptyResult,
		},
	}

	testQueryCompat(t, testCases)
}

func TestQueryCompatSort(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"Asc": {
			filter: bson.D{},
			sort:   bson.D{{"v", 1}, {"_id", 1}},
		},
		"Desc": {
			filter: bson.D{},
			sort:   bson.D{{"v", -1}, {"_id", 1}},
		},
		"AscDesc": {
			filter: bson.D{},
			sort:   bson.D{{"v", 1}, {"_id", -1}},
		},
		"DescDesc": {
			filter: bson.D{},
			sort:   bson.D{{"v", -1}, {"_id", -1}},
		},
		"AscSingle": {
			filter: bson.D{},
			sort:   bson.D{{"_id", 1}},
		},
		"DescSingle": {
			filter: bson.D{},
			sort:   bson.D{{"_id", -1}},
		},

		"Bad": {
			filter:     bson.D{},
			sort:       bson.D{{"v", 13}},
			resultType: emptyResult,
		},
		"BadZero": {
			filter:     bson.D{},
			sort:       bson.D{{"v", 0}},
			resultType: emptyResult,
		},
		"BadNull": {
			filter:     bson.D{},
			sort:       bson.D{{"v", nil}},
			resultType: emptyResult,
		},

		"DotNotationIndex": {
			filter: bson.D{},
			sort:   bson.D{{"v.0", 1}, {"_id", 1}},
		},
		"DotNotationNonExistent": {
			filter: bson.D{},
			sort:   bson.D{{"invalid.foo", 1}, {"_id", 1}},
		},
		"DotNotationMissingField": {
			filter:     bson.D{},
			sort:       bson.D{{"v..foo", 1}, {"_id", 1}},
			resultType: emptyResult,
		},

		"BadDollarStart": {
			filter:     bson.D{},
			sort:       bson.D{{"$v.foo", 1}},
			resultType: emptyResult,
		},
		"BadDollarMid": {
			filter:     bson.D{},
			sort:       bson.D{{"v.$foo.bar", 1}, {"_id", 1}},
			resultType: emptyResult,
		},
		"BadDollarEnd": {
			filter:     bson.D{},
			sort:       bson.D{{"_id", 1}, {"v.$foo", 1}},
			resultType: emptyResult,
		},
		"DollarPossible": {
			filter: bson.D{},
			sort:   bson.D{{"v.f$oo.bar", 1}, {"_id", 1}},
		},
	}

	testQueryCompat(t, testCases)
}

func TestQueryCompatSortDotNotation(t *testing.T) {
	t.Parallel()

	providers := shareddata.AllProviders().
		// TODO https://github.com/FerretDB/FerretDB/issues/2618
		Remove(shareddata.ArrayDocuments)

	testCases := map[string]queryCompatTestCase{
		"DotNotation": {
			filter: bson.D{},
			sort:   bson.D{{"v.foo", 1}, {"_id", 1}},
		},
	}
	testQueryCompatWithProviders(t, providers, testCases)
}

func TestQueryCompatSkip(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"Simple": {
			filter:  bson.D{},
			optSkip: pointer.ToInt64(1),
		},
		"SimpleWithLimit": {
			filter:  bson.D{},
			optSkip: pointer.ToInt64(1),
			limit:   pointer.ToInt64(1),
		},
		"AlmostAll": {
			filter:  bson.D{},
			optSkip: pointer.ToInt64(int64(len(shareddata.Strings.Docs()) - 1)),
		},
		"All": {
			filter:  bson.D{},
			optSkip: pointer.ToInt64(int64(len(shareddata.Strings.Docs()))),
		},
		"More": {
			filter:  bson.D{},
			optSkip: pointer.ToInt64(int64(len(shareddata.Strings.Docs()) + 1)),
		},
		"Big": {
			filter:     bson.D{},
			optSkip:    pointer.ToInt64(1000),
			resultType: emptyResult,
		},
		"Zero": {
			filter:  bson.D{},
			optSkip: pointer.ToInt64(0),
		},
		"Bad": {
			filter:     bson.D{},
			optSkip:    pointer.ToInt64(-1),
			resultType: emptyResult,
		},
		"MaxInt64": {
			filter:     bson.D{},
			optSkip:    pointer.ToInt64(math.MaxInt64),
			resultType: emptyResult,
		},
		"MinInt64": {
			filter:     bson.D{},
			optSkip:    pointer.ToInt64(math.MinInt64),
			resultType: emptyResult,
		},
	}

	testQueryCompat(t, testCases)
}

func TestQueryCompatLimit(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"Simple": {
			filter: bson.D{},
			limit:  pointer.ToInt64(1),
		},
		"AlmostAll": {
			filter: bson.D{},
			limit:  pointer.ToInt64(int64(len(shareddata.Strings.Docs()) - 1)),
		},
		"All": {
			filter: bson.D{},
			limit:  pointer.ToInt64(int64(len(shareddata.Strings.Docs()))),
		},
		"More": {
			filter: bson.D{},
			limit:  pointer.ToInt64(int64(len(shareddata.Strings.Docs()) + 1)),
		},
		"Big": {
			filter: bson.D{},
			limit:  pointer.ToInt64(1000),
		},
		"Zero": {
			filter: bson.D{},
			limit:  pointer.ToInt64(0),
		},
		"SingleBatch": {
			// The meaning of negative limits is redefined by the Go driver:
			// > A negative limit specifies that the resulting documents should be returned in a single batch.
			// On the wire, "limit" can't be negative.
			// TODO https://github.com/FerretDB/FerretDB/issues/2255
			filter: bson.D{},
			limit:  pointer.ToInt64(-1),
		},
	}

	testQueryCompat(t, testCases)
}

func TestQueryCompatBatchSize(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"Simple": {
			filter:    bson.D{},
			batchSize: pointer.ToInt32(1),
		},
		"AlmostAll": {
			filter:    bson.D{},
			batchSize: pointer.ToInt32(int32(len(shareddata.Strings.Docs()) - 1)),
		},
		"All": {
			filter:    bson.D{},
			batchSize: pointer.ToInt32(int32(len(shareddata.Strings.Docs()))),
		},
		"More": {
			filter:    bson.D{},
			batchSize: pointer.ToInt32(int32(len(shareddata.Strings.Docs()) + 1)),
		},
		"Big": {
			filter:    bson.D{},
			batchSize: pointer.ToInt32(1000),
		},
		"Zero": {
			filter:     bson.D{},
			batchSize:  pointer.ToInt32(0),
			resultType: emptyResult,
		},
		"Bad": {
			filter:     bson.D{},
			batchSize:  pointer.ToInt32(-1),
			resultType: emptyResult,
		},
	}

	testQueryCompat(t, testCases)
}

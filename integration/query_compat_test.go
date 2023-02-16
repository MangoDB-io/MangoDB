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
	"errors"
	"math"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/integration/setup"
)

// queryCompatTestCase describes query compatibility test case.
type queryCompatTestCase struct {
	filter         bson.D                   // required
	sort           bson.D                   // defaults to `bson.D{{"_id", 1}}` for stable results
	projection     bson.D                   // defaults to nil for leaving projection unset
	resultType     compatTestCaseResultType // defaults to nonEmptyResult
	resultPushdown bool                     // defaults to false
	skipForTigris  string                   // skip test for Tigris
	skip           string                   // skip test for all handlers, must have issue number mentioned
	batchSize      int32                    // defaults to 0
}

// testQueryCompat tests query compatibility test cases.
func testQueryCompat(t *testing.T, testCases map[string]queryCompatTestCase) {
	t.Helper()

	// Use shared setup because find queries can't modify data.
	// TODO Use read-only user. https://github.com/FerretDB/FerretDB/issues/1025
	ctx, targetCollections, compatCollections := setup.SetupCompat(t)

	for name, tc := range testCases {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Helper()

			if tc.skipForTigris != "" {
				setup.SkipForTigrisWithReason(t, tc.skipForTigris)
			}

			if tc.skip != "" {
				t.Skip(tc.skip)
			}

			t.Parallel()

			filter := tc.filter
			require.NotNil(t, filter, "filter should be set")

			sort := tc.sort
			if sort == nil {
				sort = bson.D{{"_id", 1}}
			}
			opts := options.Find().SetSort(sort)

			if tc.projection != nil {
				opts = opts.SetProjection(tc.projection)
			}

			if tc.batchSize != 0 {
				opts = opts.SetBatchSize(tc.batchSize)
			}

			var nonEmptyResults bool
			for i := range targetCollections {
				targetCollection := targetCollections[i]
				compatCollection := compatCollections[i]
				t.Run(targetCollection.Name(), func(t *testing.T) {
					t.Helper()

					explainQuery := bson.D{{"explain", bson.D{
						{"find", targetCollection.Name()},
						{"filter", filter},
						{"sort", opts.Sort},
						{"projection", opts.Projection},
					}}}

					var explainRes bson.D
					require.NoError(t, targetCollection.Database().RunCommand(ctx, explainQuery).Decode(&explainRes))

					var msg string
					if setup.IsPushdownDisabled() {
						tc.resultPushdown = false
						msg = "Query pushdown is disabled, but target resulted with pushdown"
					}

					assert.Equal(t, tc.resultPushdown, explainRes.Map()["pushdown"], msg)

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
						AssertMatchesCommandError(t, compatErr, targetErr)

						return
					}
					require.NoError(t, compatErr, "compat error; target returned no error")

					var targetRes, compatRes []bson.D
					require.NoError(t, targetCursor.All(ctx, &targetRes))
					require.NoError(t, compatCursor.All(ctx, &compatRes))

					t.Logf("Compat (expected) IDs: %v", CollectIDs(t, compatRes))
					t.Logf("Target (actual)   IDs: %v", CollectIDs(t, targetRes))
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

// TestQueryCompatRunner is temporary runner to address
// slowness of compat setup by only setting it up once
// for all query tests.
func TestQueryCompatRunner(t *testing.T) {
	t.Parallel()

	testcases := map[string]map[string]queryCompatTestCase{
		"Basic":         testQueryCompatBasic(),
		"Sort":          testQueryCompatSort(),
		"BatchSize":     testQueryCompatBatchSize(),
		"Size":          testQueryArrayCompatSize(),
		"DotNotation":   testQueryArrayCompatDotNotation(),
		"ElemMatch":     testQueryArrayCompatElemMatch(),
		"ArrayEquality": testQueryArrayCompatEquality(),
		"ArrayAll":      testQueryArrayCompatAll(),
		"ImplicitEq":    testQueryComparisonCompatImplicit(),
		"Eq":            testQueryComparisonCompatEq(),
		"Gt":            testQueryComparisonCompatGt(),
		"Gte":           testQueryComparisonCompatGte(),
		"Lt":            testQueryComparisonCompatLt(),
		"Lte":           testQueryComparisonCompatLte(),
		"Nin":           testQueryComparisonCompatNin(),
		"In":            testQueryComparisonCompatIn(),
		"Ne":            testQueryComparisonCompatNe(),
		"MultipleOp":    testQueryComparisonCompatMultipleOperators(),
		"Exists":        testQueryElementCompatExists(),
		"Type":          testQueryElementCompatElementType(),
		"Regex":         testQueryEvaluationCompatRegexErrors(),
		"And":           testQueryLogicalCompatAnd(),
		"Or":            testQueryLogicalCompatOr(),
		"Nor":           testQueryLogicalCompatNor(),
		"Not":           testQueryLogicalCompatNot(),
		"Projection":    testQueryProjectionCompat(),
	}

	if runtime.GOARCH != "arm64" {
		// https://github.com/FerretDB/FerretDB/issues/491
		testcases["Mod"] = testQueryEvaluationCompatMod()
	}

	allTestcases := make(map[string]queryCompatTestCase, 0)

	for op, tcs := range testcases {
		for name, tc := range tcs {
			allTestcases[op+name] = tc
		}
	}

	testQueryCompat(t, allTestcases)
}

func testQueryCompatBasic() map[string]queryCompatTestCase {
	testCases := map[string]queryCompatTestCase{
		"BadSortValue": {
			filter:     bson.D{},
			sort:       bson.D{{"v", 11}},
			resultType: emptyResult,
		},
		"BadSortZeroValue": {
			filter:     bson.D{},
			sort:       bson.D{{"v", 0}},
			resultType: emptyResult,
		},
		"BadSortNullValue": {
			filter:     bson.D{},
			sort:       bson.D{{"v", nil}},
			resultType: emptyResult,
		},
		"Empty": {
			filter: bson.D{},
		},
		"IDString": {
			filter:         bson.D{{"_id", "string"}},
			resultPushdown: true,
		},
		"IDObjectID": {
			filter:         bson.D{{"_id", primitive.NilObjectID}},
			resultPushdown: true,
		},
		"ObjectID": {
			filter:         bson.D{{"v", primitive.NilObjectID}},
			resultPushdown: true,
		},
		"UnknownFilterOperator": {
			filter:     bson.D{{"v", bson.D{{"$someUnknownOperator", 42}}}},
			resultType: emptyResult,
		},
	}

	return testCases
}

func testQueryCompatSort() map[string]queryCompatTestCase {
	testCases := map[string]queryCompatTestCase{
		"Asc": {
			filter: bson.D{},
			sort:   bson.D{{"v", 1}, {"_id", 1}},
		},
		"Desc": {
			filter: bson.D{},
			sort:   bson.D{{"v", -1}, {"_id", 1}},
		},
	}

	return testCases
}

func testQueryCompatBatchSize() map[string]queryCompatTestCase {
	testCases := map[string]queryCompatTestCase{
		"BatchSize1": {
			filter:    bson.D{},
			batchSize: 1,
		},
		// Test batch size less than the default batch size of 101.
		"BatchSize100": {
			filter:    bson.D{},
			batchSize: 100,
		},
		// Test batch size greater than the default batch size of 101.
		"BatchSize102": {
			filter:    bson.D{},
			batchSize: 102,
		},
	}

	return testCases
}

type queryCompatBatchSizeErrorsTestCase struct {
	altMessage string
	command    bson.D
	err        bool
}

func testQueryCompatBatchSizeErrors(t *testing.T, testCases map[string]queryCompatBatchSizeErrorsTestCase) {
	t.Helper()

	s := setup.SetupCompatWithOpts(t, &setup.SetupCompatOpts{Providers: nil, AddNonExistentCollection: true})

	// We expect to have only one collection as the result of setup.
	require.Len(t, s.TargetCollections, 1)
	require.Len(t, s.CompatCollections, 1)

	targetCollection := s.TargetCollections[0]
	compatCollection := s.CompatCollections[0]

	ctx := s.Ctx

	for name, tc := range testCases {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			targetCommand := bson.D{{"find", targetCollection.Name()}}
			targetCommand = append(targetCommand, tc.command...)
			compatCommand := bson.D{{"find", compatCollection.Name()}}
			compatCommand = append(compatCommand, tc.command...)

			var targetResult, compatResult bson.D
			targetErr := targetCollection.Database().RunCommand(ctx, targetCommand).Decode(&targetResult)
			compatErr := compatCollection.Database().RunCommand(ctx, compatCommand).Decode(&compatResult)

			if tc.err {
				var compatCommandErr mongo.CommandError
				if !errors.As(compatErr, &compatCommandErr) {
					t.Fatalf("expected error, got %v", compatCommandErr)
				}

				compatCommandErr.Raw = nil

				AssertEqualAltError(t, compatCommandErr, tc.altMessage, targetErr)
				return
			}

			require.NoError(t, targetErr)
			require.NoError(t, compatErr)

			require.Equal(t, compatResult, targetResult)
		})
	}
}

func TestQueryBatchSizeCompatErrors(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatBatchSizeErrorsTestCase{
		"BatchSizeDocument": {
			command: bson.D{
				{"batchSize", bson.D{}},
			},
			err:        true,
			altMessage: "BSON field 'batchSize' is the wrong type 'object', expected type 'int'",
		},
		"BatchSizeArray": {
			command: bson.D{
				{"batchSize", bson.A{}},
			},
			err:        true,
			altMessage: "BSON field 'batchSize' is the wrong type 'array', expected type 'int'",
		},
		"BatchSizeNegative": {
			command: bson.D{
				{"batchSize", int32(-1)},
			},
			err: true,
		},
		"BatchSizeZero": {
			command: bson.D{
				{"batchSize", int32(0)},
			},
		},
		"BatchSizeMaxInt32": {
			command: bson.D{
				{"batchSize", math.MaxInt32},
			},
		},
	}

	testQueryCompatBatchSizeErrors(t, testCases)
}

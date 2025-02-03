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
	"fmt"
	"math"
	"math/rand/v2"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/v2/build/version"
	"github.com/FerretDB/FerretDB/v2/internal/util/ctxutil"

	"github.com/FerretDB/FerretDB/v2/integration/setup"
	"github.com/FerretDB/FerretDB/v2/integration/shareddata"
)

// PostgreSQL version expected by tests.
const expectedPostgreSQLVersion = "PostgreSQL 16.6 (Debian 16.6-1.pgdg120+1) on x86_64-pc-linux-gnu, " +
	"compiled by gcc (Debian 12.2.0-14) 12.2.0, 64-bit"

func TestCreateCollectionDropListCollections(t *testing.T) {
	ctx, collection := setup.Setup(t)

	db := collection.Database()
	name := collection.Name()

	names, err := db.ListCollectionNames(ctx, bson.D{})
	require.NoError(t, err)
	require.Empty(t, names, "setup should not create collection if no providers are given")

	// drop non-existing collection in non-existing database; error is consumed by the driver
	err = collection.Drop(ctx)
	require.NoError(t, err)
	err = db.Collection(name).Drop(ctx)
	require.NoError(t, err)

	err = db.CreateCollection(ctx, name)
	require.NoError(t, err)

	// List collection names
	names, err = db.ListCollectionNames(ctx, bson.D{})
	require.NoError(t, err)
	assert.Contains(t, names, name)

	// drop existing collection
	err = collection.Drop(ctx)
	require.NoError(t, err)

	// And try to drop existing collection again manually to check behavior.
	var actual bson.D
	err = db.RunCommand(ctx, bson.D{{"drop", name}}).Decode(&actual)
	assert.NoError(t, err)

	AssertEqualDocuments(t, bson.D{{"ok", float64(1)}}, actual)
}

func TestDropDatabaseListDatabases(tt *testing.T) {
	tt.Parallel()

	t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB/issues/4722")

	ctx, collection := setup.Setup(tt) // no providers there

	db := collection.Database()
	name := db.Name()

	filter := bson.D{{
		"name", bson.D{{
			"$in", bson.A{name}, // skip admin, other tests databases, etc
		}},
	}}
	names, err := db.Client().ListDatabaseNames(ctx, filter)
	require.NoError(t, err)
	require.Empty(t, names, "setup should not create database if no providers are given")

	// drop non-existing database; error is consumed by the driver
	err = db.Drop(ctx)
	require.NoError(t, err)

	// drop manually to check error
	var res bson.D
	err = db.RunCommand(ctx, bson.D{{"dropDatabase", 1}}).Decode(&res)
	require.NoError(t, err)

	AssertEqualDocuments(t, bson.D{{"ok", float64(1)}}, res)

	// there is no explicit command to create database, so create collection instead
	err = db.Client().Database(name).CreateCollection(ctx, collection.Name())
	require.NoError(t, err)

	names, err = db.Client().ListDatabaseNames(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, []string{name}, names)

	// drop existing database
	err = db.Drop(ctx)
	require.NoError(t, err)

	// drop manually to check error
	err = db.RunCommand(ctx, bson.D{{"dropDatabase", 1}}).Decode(&res)
	require.NoError(t, err)

	AssertEqualDocuments(t, bson.D{{"ok", float64(1)}}, res)
}

func TestListDatabases(t *testing.T) {
	t.Parallel()

	ctx, collection := setup.Setup(t, shareddata.DocumentsStrings)

	db := collection.Database()
	name := db.Name()
	dbClient := collection.Database().Client()

	// Add an extra DB to help verify if ListDatabases returns multiple databases as intended.
	extraDB := dbClient.Database(name + "_extra")
	_, err := extraDB.Collection(collection.Name()+"_extra").InsertOne(ctx, shareddata.DocumentsDoubles)
	assert.NoError(t, err, "failed to insert document on extra collection")
	t.Cleanup(func() {
		assert.NoError(t, extraDB.Drop(ctx), "failed to drop extra DB")
	})

	testCases := map[string]struct { //nolint:vet // for readability
		filter any
		opts   []*options.ListDatabasesOptions

		expectedNameOnly bool
		expected         mongo.ListDatabasesResult
	}{
		"Exists": {
			filter: bson.D{{Key: "name", Value: name}},
			expected: mongo.ListDatabasesResult{
				Databases: []mongo.DatabaseSpecification{{
					Name:  name,
					Empty: false,
				}},
			},
		},
		"ExistsNameOnly": {
			filter: bson.D{{Key: "name", Value: name}},
			opts: []*options.ListDatabasesOptions{
				options.ListDatabases().SetNameOnly(true),
			},
			expectedNameOnly: true,
			expected: mongo.ListDatabasesResult{
				Databases: []mongo.DatabaseSpecification{{
					Name: name,
				}},
			},
		},
		"Regex": {
			filter: bson.D{
				{Key: "name", Value: name},
				{Key: "name", Value: primitive.Regex{Pattern: "^Test", Options: "i"}},
			},
			expected: mongo.ListDatabasesResult{
				Databases: []mongo.DatabaseSpecification{{
					Name: name,
				}},
			},
		},
		"RegexNameOnly": {
			filter: bson.D{
				{Key: "name", Value: name},
				{Key: "name", Value: primitive.Regex{Pattern: "^Test", Options: "i"}},
			},
			opts: []*options.ListDatabasesOptions{
				options.ListDatabases().SetNameOnly(true),
			},
			expectedNameOnly: true,
			expected: mongo.ListDatabasesResult{
				Databases: []mongo.DatabaseSpecification{{
					Name: name,
				}},
			},
		},
		"NotFound": {
			filter: bson.D{{Key: "name", Value: "unknown"}},
			expected: mongo.ListDatabasesResult{
				Databases: []mongo.DatabaseSpecification{},
			},
		},
		"RegexNotFound": {
			filter: bson.D{
				{Key: "name", Value: name},
				{Key: "name", Value: primitive.Regex{Pattern: "^xyz$", Options: "i"}},
			},
			expected: mongo.ListDatabasesResult{
				Databases: []mongo.DatabaseSpecification{},
			},
		},
		"RegexNotFoundNameOnly": {
			filter: bson.D{
				{Key: "name", Value: name},
				{Key: "name", Value: primitive.Regex{Pattern: "^xyz$", Options: "i"}},
			},
			opts: []*options.ListDatabasesOptions{
				options.ListDatabases().SetNameOnly(true),
			},
			expectedNameOnly: true,
			expected: mongo.ListDatabasesResult{
				Databases: []mongo.DatabaseSpecification{},
			},
		},
		"Multiple": {
			filter: bson.D{
				{Key: "name", Value: primitive.Regex{Pattern: "^" + name, Options: "i"}},
			},
			expected: mongo.ListDatabasesResult{
				Databases: []mongo.DatabaseSpecification{
					{
						Name:  name,
						Empty: false,
					},
					{
						Name:  name + "_extra",
						Empty: false,
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(tt *testing.T) {
			tt.Parallel()

			t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB/issues/4722")

			actual, err := db.Client().ListDatabases(ctx, tc.filter, tc.opts...)
			assert.NoError(t, err)
			assert.Len(t, actual.Databases, len(tc.expected.Databases))
			if tc.expectedNameOnly || len(tc.expected.Databases) == 0 {
				assert.Zero(t, actual.TotalSize, "TotalSize should be zero")
			} else {
				assert.NotZero(t, actual.TotalSize, "TotalSize should be non-zero")
			}

			// Reset values of dynamic data received by the server to zero for making comparison viable.
			for index := range actual.Databases {
				actual.Databases[index].SizeOnDisk = 0
			}
			actual.TotalSize = 0

			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestListCollectionNames(t *testing.T) {
	t.Parallel()

	ctx, targetCollections, compatCollections := setup.SetupCompat(t)

	filterNames := make(bson.A, len(targetCollections))
	for i, n := range targetCollections {
		filterNames[i] = n.Name()
	}

	// We should remove shuffle there once it is implemented in the setup.
	// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/825

	rand.Shuffle(len(filterNames), func(i, j int) { filterNames[i], filterNames[j] = filterNames[j], filterNames[i] })
	filterNames = filterNames[:len(filterNames)-1]
	require.NotEmpty(t, filterNames)

	filter := bson.D{{
		"name", bson.D{{
			"$in", filterNames,
		}},
	}}

	compat, err := compatCollections[0].Database().ListCollectionNames(ctx, filter)
	require.NoError(t, err)

	require.True(t, slices.IsSorted(compat), "compat collections are not sorted")

	target, err := targetCollections[0].Database().ListCollectionNames(ctx, filter)
	require.NoError(t, err)

	assert.Len(t, target, len(filterNames))
	assert.True(t, slices.IsSorted(target), "target collections are not sorted")
	assert.Equal(t, compat, target)
}

func TestListCollectionsUUID(t *testing.T) {
	t.Parallel()

	ctx, collection := setup.Setup(t)
	db := collection.Database()
	collName := collection.Name()

	err := db.CreateCollection(ctx, collName)
	require.NoError(t, err)

	cursor, err := db.ListCollections(ctx, bson.D{})
	require.NoError(t, err)

	var res []bson.D
	err = cursor.All(ctx, &res)
	require.NoError(t, err)
	require.Len(t, res, 1)

	var uuid primitive.Binary

	for _, v := range res[0] {
		if v.Key == "info" {
			info := v.Value.(bson.D)
			for _, vv := range info {
				if vv.Key == "uuid" {
					uuid = vv.Value.(primitive.Binary)
					assert.Equal(t, bson.TypeBinaryUUID, uuid.Subtype)
					assert.Len(t, uuid.Data, 16)
				}
			}
		}
	}

	// collection rename should not change the initial UUID

	newName := collName + "_new"
	command := bson.D{
		{"renameCollection", db.Name() + "." + collName},
		{"to", db.Name() + "." + newName},
	}
	err = collection.Database().Client().Database("admin").RunCommand(ctx, command).Err()
	require.NoError(t, err)

	cursor, err = db.ListCollections(ctx, bson.D{})
	require.NoError(t, err)

	err = cursor.All(ctx, &res)
	require.NoError(t, err)
	require.Len(t, res, 1)

	expected := bson.D{
		{"name", newName},
		{"type", "collection"},
		{"options", bson.D{}},
		{"info", bson.D{
			{"readOnly", false},
			{"uuid", uuid},
		}},
		{"idIndex", bson.D{
			{"v", int32(2)},
			{"key", bson.D{{"_id", int32(1)}}},
			{"name", "_id_"},
		}},
	}

	AssertEqualDocuments(t, expected, res[0])
}

func TestGetParameterCommand(t *testing.T) {
	t.Parallel()
	s := setup.SetupWithOpts(t, &setup.SetupOpts{
		DatabaseName: "admin",
	})

	for name, tc := range map[string]struct {
		command    bson.D         // required, command to run
		expected   map[string]any // optional, expected keys of response
		unexpected []string       // optional, unexpected keys of response

		err        *mongo.CommandError // optional, expected error from MongoDB
		altMessage string              // optional, alternative error message for FerretDB, ignored if empty
	}{
		"GetParameter_Asterisk1": {
			command: bson.D{{"getParameter", "*"}},
			expected: map[string]any{
				"authSchemaVersion": int32(5),
				"quiet":             false,
				"ok":                float64(1),
			},
		},
		"GetParameter_Asterisk2": {
			command: bson.D{{"getParameter", "*"}, {"quiet", 1}, {"comment", "getParameter test"}},
			expected: map[string]any{
				"authSchemaVersion": int32(5),
				"quiet":             false,
				"ok":                float64(1),
			},
		},
		"GetParameter_Asterisk3": {
			command: bson.D{{"getParameter", "*"}, {"quiet", 1}, {"quiet_other", 1}, {"comment", "getParameter test"}},
			expected: map[string]any{
				"authSchemaVersion": int32(5),
				"quiet":             false,
				"ok":                float64(1),
			},
		},
		"GetParameter_Asterisk4": {
			command: bson.D{{"getParameter", "*"}, {"quiet_other", 1}, {"comment", "getParameter test"}},
			expected: map[string]any{
				"authSchemaVersion": int32(5),
				"quiet":             false,
				"ok":                float64(1),
			},
		},
		"GetParameter_Int": {
			command: bson.D{{"getParameter", 1}, {"quiet", 1}, {"comment", "getParameter test"}},
			expected: map[string]any{
				"quiet": false,
				"ok":    float64(1),
			},
		},
		"GetParameter_Zero": {
			command: bson.D{{"getParameter", 0}, {"quiet", 1}, {"comment", "getParameter test"}},
			expected: map[string]any{
				"quiet": false,
				"ok":    float64(1),
			},
		},
		"GetParameter_Nil": {
			command: bson.D{{"getParameter", nil}, {"quiet", 1}, {"comment", "getParameter test"}},
			expected: map[string]any{
				"quiet": false,
				"ok":    float64(1),
			},
		},
		"GetParameter_String": {
			command: bson.D{{"getParameter", "1"}, {"quiet", 1}, {"comment", "getParameter test"}},
			expected: map[string]any{
				"quiet": false,
				"ok":    float64(1),
			},
		},
		"NonexistentParameters": {
			command: bson.D{{"getParameter", 1}, {"quiet", 1}, {"quiet_other", 1}, {"comment", "getParameter test"}},
			expected: map[string]any{
				"quiet": false,
				"ok":    float64(1),
			},
			unexpected: []string{"quiet_other"},
		},
		"EmptyParameters": {
			command: bson.D{{"getParameter", 1}, {"comment", "getParameter test"}},
			err:     &mongo.CommandError{Code: 72, Message: `no option found to get`, Name: "InvalidOptions"},
		},
		"OnlyNonexistentParameters": {
			command: bson.D{{"getParameter", 1}, {"quiet_other", 1}, {"comment", "getParameter test"}},
			err:     &mongo.CommandError{Code: 72, Message: `no option found to get`, Name: "InvalidOptions"},
		},
		"ShowDetailsTrue": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
			unexpected: []string{"acceptApiVersion2"},
		},
		"ShowDetailsFalse": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", false}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": false,
				"ok":    float64(1),
			},
			unexpected: []string{"acceptApiVersion2"},
		},
		"ShowDetails_NoParameter_1": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}}}},
			err:     &mongo.CommandError{Code: 72, Message: `no option found to get`, Name: "InvalidOptions"},
		},
		"ShowDetails_NoParameter_2": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", false}}}},
			err:     &mongo.CommandError{Code: 72, Message: `no option found to get`, Name: "InvalidOptions"},
		},
		"AllParametersTrue": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", true}}}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
		},
		"AllParametersFalse_MissingParameter": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", false}}}},
			err:     &mongo.CommandError{Code: 72, Message: `no option found to get`, Name: "InvalidOptions"},
		},
		"AllParametersFalse_PresentParameter": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", false}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
			unexpected: []string{"acceptApiVersion2"},
		},
		"AllParametersFalse_NonexistentParameter": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", false}}}, {"quiet_other", true}},
			err:     &mongo.CommandError{Code: 72, Message: `no option found to get`, Name: "InvalidOptions"},
		},
		"ShowDetailsFalse_AllParametersTrue": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", false}, {"allParameters", true}}}},
			expected: map[string]any{
				"authSchemaVersion": int32(5),
				"quiet":             false,
				"ok":                float64(1),
			},
		},
		"ShowDetailsFalse_AllParametersFalse_1": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", false}, {"allParameters", false}}}},
			err:     &mongo.CommandError{Code: 72, Message: `no option found to get`, Name: "InvalidOptions"},
		},
		"ShowDetailsFalse_AllParametersFalse_2": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", false}, {"allParameters", false}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": false,
				"ok":    float64(1),
			},
			unexpected: []string{"acceptApiVersion2"},
		},
		"ShowDetails_NegativeInt": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", int64(-1)}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
		},
		"ShowDetails_PositiveInt": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", int64(1)}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
		},
		"ShowDetails_ZeroInt": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", int64(0)}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": false,
				"ok":    float64(1),
			},
			unexpected: []string{"acceptApiVersion2"},
		},
		"ShowDetails_ZeroFloat": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", float64(0)}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": false,
				"ok":    float64(1),
			},
			unexpected: []string{"acceptApiVersion2"},
		},
		"ShowDetails_SmallestNonzeroFloat": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", math.SmallestNonzeroFloat64}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
		},
		"ShowDetails_Nil": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", nil}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": false,
				"ok":    float64(1),
			},
		},
		"ShowDetails_String": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", "1"}}}, {"quiet", true}},
			err: &mongo.CommandError{
				Code:    14,
				Name:    "TypeMismatch",
				Message: `BSON field 'getParameter.showDetails' is the wrong type 'string', expected types '[bool, long, int, decimal, double']`,
			},
			altMessage: `BSON field 'showDetails' is the wrong type 'string', expected types '[bool, long, int, decimal, double]'`,
		},
		"AllParameters_NegativeInt": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", int64(-1)}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
		},
		"AllParameters_PositiveInt": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", int64(1)}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
		},
		"AllParameters_ZeroInt": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", int64(0)}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
		},
		"AllParameters_ZeroFloat": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", float64(0)}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
		},
		"AllParameters_SmallestNonzeroFloat": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", math.SmallestNonzeroFloat64}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
		},
		"AllParameters_Nil": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", nil}}}, {"quiet", true}},
			expected: map[string]any{
				"quiet": bson.D{
					{"value", false},
					{"settableAtRuntime", true},
					{"settableAtStartup", true},
				},
				"ok": float64(1),
			},
			unexpected: []string{"acceptApiVersion2"},
		},
		"AllParameters_String": {
			command: bson.D{{"getParameter", bson.D{{"showDetails", true}, {"allParameters", "1"}}}, {"quiet", true}},
			err: &mongo.CommandError{
				Code:    14,
				Name:    "TypeMismatch",
				Message: `BSON field 'getParameter.allParameters' is the wrong type 'string', expected types '[bool, long, int, decimal, double']`,
			},
			altMessage: `BSON field 'allParameters' is the wrong type 'string', expected types '[bool, long, int, decimal, double]'`,
		},
		"FeatureCompatibilityVersion": {
			command: bson.D{
				{"getParameter", bson.D{}},
				{"featureCompatibilityVersion", 1},
			},
			expected: map[string]any{
				"featureCompatibilityVersion": bson.D{{"version", "7.0"}},
				"ok":                          float64(1),
			},
		},
		"FeatureCompatibilityVersionShowDetails": {
			command: bson.D{
				{"getParameter", bson.D{{"showDetails", true}}},
				{"featureCompatibilityVersion", 1},
			},
			expected: map[string]any{
				"featureCompatibilityVersion": bson.D{
					{"value", bson.D{{"version", "7.0"}}},
					{"settableAtRuntime", false},
					{"settableAtStartup", false},
				},
				"ok": float64(1),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, tc.command, "command must not be nil")

			var actual bson.D
			err := s.Collection.Database().RunCommand(s.Ctx, tc.command).Decode(&actual)
			if tc.err != nil {
				AssertEqualAltCommandError(t, *tc.err, tc.altMessage, err)
				return
			}

			require.NoError(t, err)

			m := actual.Map()
			keys := CollectKeys(t, actual)

			for k, item := range tc.expected {
				assert.Contains(t, keys, k)
				assert.IsType(t, item, m[k])
				if it, ok := item.(primitive.D); ok {
					z := m[k].(primitive.D)
					AssertEqualDocuments(t, it, z)
				} else {
					assert.Equal(t, m[k], item)
				}
			}

			for _, k := range tc.unexpected {
				assert.NotContains(t, keys, k)
			}
		})
	}
}

func TestGetParameterCommandAuthenticationMechanisms(t *testing.T) {
	t.Parallel()

	s := setup.SetupWithOpts(t, &setup.SetupOpts{
		DatabaseName: "admin",
	})

	t.Run("ShowDetails", func(t *testing.T) {
		var actual bson.D
		err := s.Collection.Database().RunCommand(s.Ctx, bson.D{
			{"getParameter", bson.D{{"showDetails", true}}},
			{"authenticationMechanisms", 1},
		}).Decode(&actual)
		require.NoError(t, err)

		var actualComparable bson.D

		for _, field := range actual {
			switch field.Key {
			case "authenticationMechanisms":
				var mComparable bson.D

				for _, m := range field.Value.(bson.D) {
					switch m.Key {
					case "value":
						var values bson.A

						for _, v := range m.Value.(bson.A) {
							switch v.(string) {
							case "MONGODB-X509":
								// exclusive to MongoDB
							default:
								values = append(values, v)
							}
						}

						mComparable = append(mComparable, bson.E{Key: m.Key, Value: values})

					default:
						mComparable = append(mComparable, m)
					}
				}

				actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: mComparable})

			default:
				actualComparable = append(actualComparable, field)
			}
		}

		expected := bson.D{
			{
				"authenticationMechanisms",
				bson.D{
					{"value", bson.A{"SCRAM-SHA-1", "SCRAM-SHA-256"}},
					{"settableAtRuntime", false},
					{"settableAtStartup", true},
				},
			},
			{"ok", float64(1)},
		}

		AssertEqualDocuments(t, expected, actualComparable)
	})

	t.Run("authenticationMechanisms", func(t *testing.T) {
		var actual bson.D
		err := s.Collection.Database().RunCommand(s.Ctx, bson.D{
			{"getParameter", bson.D{}},
			{"authenticationMechanisms", 1},
		}).Decode(&actual)
		require.NoError(t, err)

		var actualComparable bson.D

		for _, field := range actual {
			switch field.Key {
			case "authenticationMechanisms":
				var values bson.A

				for _, v := range field.Value.(bson.A) {
					switch v.(string) {
					case "MONGODB-X509":
						// exclusive to MongoDB
					default:
						values = append(values, v)
					}
				}

				actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: values})

			default:
				actualComparable = append(actualComparable, field)
			}
		}

		expected := bson.D{
			{"authenticationMechanisms", bson.A{"SCRAM-SHA-1", "SCRAM-SHA-256"}},
			{"ok", float64(1)},
		}

		AssertEqualDocuments(t, expected, actualComparable)
	})
}

func TestBuildInfoCommand(t *testing.T) {
	t.Parallel()
	ctx, collection := setup.Setup(t)

	var actual bson.D
	command := bson.D{{"buildInfo", int32(1)}}
	err := collection.Database().RunCommand(ctx, command).Decode(&actual)
	require.NoError(t, err)

	var actualComparable bson.D

	for _, field := range actual {
		switch field.Key {
		case "ferretdb":
			value, ok := field.Value.(bson.D)
			require.True(t, ok)
			AssertEqualDocuments(t, bson.D{{"package", "unknown"}, {"version", "unknown"}}, value)

		case "version":
			assert.IsType(t, "", field.Value)
			assert.Regexp(t, `^7\.0\.`, field.Value)
			actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: "7.0.0"})

		case "debug":
			assert.IsType(t, false, field.Value)
			actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: false})

		case "versionArray":
			arr, ok := field.Value.(bson.A)
			require.True(t, ok)

			assert.Equal(t, int32(7), arr[0])
			assert.Equal(t, int32(0), arr[1])

			actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: bson.A{}})

		case "gitVersion":
			assert.IsType(t, "", field.Value)
			assert.NotEmpty(t, field.Value)
			actualComparable = append(actualComparable, bson.E{"gitVersion", ""})

		case "bits":
			assert.Equal(t, int32(strconv.IntSize), field.Value)
			actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: int32(0)})

		case "buildEnvironment":
			assert.IsType(t, bson.D{}, field.Value)
			actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: bson.D{}})

		case "openssl", "storageEngines", "allocator", "javascriptEngine":
		// exclusive to MongoDB

		default:
			actualComparable = append(actualComparable, field)
		}
	}

	expected := bson.D{
		{"version", "7.0.0"},
		{"gitVersion", ""},
		{"modules", bson.A{}},
		{"sysInfo", "deprecated"},
		{"versionArray", bson.A{}},
		{"buildEnvironment", bson.D{}},
		{"bits", int32(0)},
		{"debug", false},
		{"maxBsonObjectSize", int32(16777216)},
		{"ok", float64(1)},
	}

	AssertEqualDocuments(t, expected, actualComparable)
}

func TestCollStatsCommandEmpty(tt *testing.T) {
	tt.Parallel()
	t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/556")
	ctx, collection := setup.Setup(tt)

	var actual bson.D
	command := bson.D{{"collStats", collection.Name()}}
	err := collection.Database().RunCommand(ctx, command).Decode(&actual)
	require.NoError(t, err)

	expected := bson.D{
		{"ns", collection.Database().Name() + "." + collection.Name()},
		{"size", int32(0)},
		{"count", int32(0)},
		{"numOrphanDocs", int32(0)},
		{"storageSize", int32(0)},
		{"totalSize", int32(0)},
		{"nindexes", int32(0)},
		{"totalIndexSize", int32(0)},
		{"indexDetails", bson.D{}},
		{"indexSizes", bson.D{}},
		{"scaleFactor", int32(1)},
		{"ok", float64(1)},
	}

	AssertEqualDocuments(t, expected, actual)
}

func TestCollStatsCommand(tt *testing.T) {
	tt.Parallel()
	t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/555")

	ctx, collection := setup.Setup(tt, shareddata.DocumentsStrings)

	var actual bson.D
	command := bson.D{{"collStats", collection.Name()}}
	err := collection.Database().RunCommand(ctx, command).Decode(&actual)
	require.NoError(t, err)

	var actualComparable, actualFieldNames bson.D

	for _, field := range actual {
		switch field.Key {
		case "size", "avgObjSize", "storageSize", "totalIndexSize", "totalSize":
			assert.GreaterOrEqual(t, field.Value, int32(0))
		case "wiredTiger":
			// exclusive to MongoDB
			continue
		case "indexDetails", "indexSizes":
			v, ok := field.Value.(bson.D)
			require.True(t, ok)

			var indexNames []string
			for _, index := range v {
				indexNames = append(indexNames, index.Key)
			}

			assert.Equal(t, []string{"_id_"}, indexNames)
		default:
			actualComparable = append(actualComparable, field)
		}

		actualFieldNames = append(actualFieldNames, bson.E{Key: field.Key})
	}

	expectedComparable := bson.D{
		{"ns", collection.Database().Name() + "." + collection.Name()},
		{"count", int32(6)},
		{"numOrphanDocs", int32(0)},
		{"freeStorageSize", int32(0)},
		{"capped", false},
		{"nindexes", int32(1)},
		{"indexBuilds", bson.A{}},
		{"scaleFactor", int32(1)},
		{"ok", float64(1)},
	}
	AssertEqualDocuments(t, expectedComparable, actualComparable)

	expectedFieldNames := bson.D{
		{Key: "ns"},
		{Key: "size"},
		{Key: "count"},
		{Key: "avgObjSize"},
		{Key: "numOrphanDocs"},
		{Key: "storageSize"},
		{Key: "freeStorageSize"},
		{Key: "capped"},
		{Key: "nindexes"},
		{Key: "indexDetails"},
		{Key: "indexBuilds"},
		{Key: "totalIndexSize"},
		{Key: "indexSizes"},
		{Key: "totalSize"},
		{Key: "scaleFactor"},
		{Key: "ok"},
	}
	AssertEqualDocuments(t, expectedFieldNames, actualFieldNames)
}

func TestCollStatsCommandScale(t *testing.T) {
	t.Parallel()

	s := setup.SetupWithOpts(t, &setup.SetupOpts{
		Providers: []shareddata.Provider{shareddata.DocumentsStrings},
	})

	ctx, collection := s.Ctx, s.Collection

	for name, tc := range map[string]struct { //nolint:vet // for readability
		scale            any
		scaleFactor      any
		err              *mongo.CommandError
		altMessage       string
		failsForFerretDB string
	}{
		"scaleOne": {
			scale:            int32(1),
			scaleFactor:      int32(1),
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/556",
		},
		"scaleBig": {
			scale:            int64(1000),
			scaleFactor:      int32(1000),
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/556",
		},
		"scaleMaxInt": {
			scale:            math.MaxInt64,
			scaleFactor:      int32(math.MaxInt32),
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/556",
		},
		"scaleZero": {
			scale: int32(0),
			err: &mongo.CommandError{
				Code:    51024,
				Name:    "Location51024",
				Message: `BSON field 'scale' value must be >= 1, actual value '0'`,
			},
		},
		"scaleNegative": {
			scale: int32(-100),
			err: &mongo.CommandError{
				Code:    51024,
				Name:    "Location51024",
				Message: `BSON field 'scale' value must be >= 1, actual value '-100'`,
			},
		},
		"scaleFloat": {
			scale:            2.8,
			scaleFactor:      int32(2),
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/556",
		},
		"scaleFloatNegative": {
			scale: -2.8,
			err: &mongo.CommandError{
				Code:    51024,
				Name:    "Location51024",
				Message: `BSON field 'scale' value must be >= 1, actual value '-2'`,
			},
		},
		"scaleMinFloat": {
			scale: -math.MaxFloat64,
			err: &mongo.CommandError{
				Code:    51024,
				Name:    "Location51024",
				Message: `BSON field 'scale' value must be >= 1, actual value '-2147483648'`,
			},
		},
		"scaleMaxFloat": {
			scale:            math.MaxFloat64,
			scaleFactor:      int32(math.MaxInt32),
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/556",
		},
		"scaleString": {
			scale: "1",
			err: &mongo.CommandError{
				Code:    14,
				Name:    "TypeMismatch",
				Message: `BSON field 'collStats.scale' is the wrong type 'string', expected types '[long, int, decimal, double']`,
			},
			altMessage: `BSON field 'collStats.scale' is the wrong type 'string', expected types '[long, int, decimal, double]'`,
		},
		"scaleObject": {
			scale: bson.D{{"a", 1}},
			err: &mongo.CommandError{
				Code:    14,
				Name:    "TypeMismatch",
				Message: `BSON field 'collStats.scale' is the wrong type 'object', expected types '[long, int, decimal, double']`,
			},
			altMessage: `BSON field 'collStats.scale' is the wrong type 'wirebson.RawDocument', expected types '[long, int, decimal, double]'`,
		},
		"scaleNull": {
			scale:            nil,
			scaleFactor:      int32(1),
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/556",
		},
	} {
		t.Run(name, func(tt *testing.T) {
			tt.Parallel()

			var t testing.TB = tt
			if tc.failsForFerretDB != "" {
				t = setup.FailsForFerretDB(tt, tc.failsForFerretDB)
			}

			var res bson.D
			command := bson.D{{"collStats", collection.Name()}, {"scale", tc.scale}}

			err := collection.Database().RunCommand(ctx, command).Decode(&res)
			if err != nil {
				AssertEqualAltCommandError(t, *tc.err, tc.altMessage, err)

				return
			}

			require.NoError(t, err)

			var actualComparable bson.D

			for _, field := range res {
				switch field.Key {
				case "size", "avgObjSize", "storageSize", "totalIndexSize", "totalSize", "freeStorageSize":
					assert.GreaterOrEqual(t, field.Value, int32(0))
				case "wiredTiger":
					// exclusive to MongoDB
				case "indexDetails", "indexSizes":
					v, ok := field.Value.(bson.D)
					require.True(t, ok)

					var indexNames []string
					for _, index := range v {
						indexNames = append(indexNames, index.Key)
					}

					assert.Equal(t, []string{"_id_"}, indexNames)
				default:
					actualComparable = append(actualComparable, field)
				}
			}

			expectedComparable := bson.D{
				{"ns", collection.Database().Name() + "." + collection.Name()},
				{"count", int32(6)},
				{"numOrphanDocs", int32(0)},
				{"capped", false},
				{"nindexes", int32(1)},
				{"indexBuilds", bson.A{}},
				{"scaleFactor", tc.scaleFactor},
				{"ok", float64(1)},
			}
			AssertEqualDocuments(t, expectedComparable, actualComparable)
		})
	}
}

// TestCollStatsCommandCount adds large number of documents and checks
// approximation used by backends returns the correct count of documents from collStats.
func TestCollStatsCommandCount(tt *testing.T) {
	tt.Parallel()
	t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/555")

	ctx, collection := setup.Setup(tt)

	var n int32 = 1000
	docs := GenerateDocuments(0, n)
	_, err := collection.InsertMany(ctx, docs)
	require.NoError(t, err)

	var actual bson.D
	command := bson.D{{"collStats", collection.Name()}}
	err = collection.Database().RunCommand(ctx, command).Decode(&actual)
	require.NoError(t, err)

	var actualComparable bson.D

	for _, field := range actual {
		switch field.Key {
		case "size", "avgObjSize", "storageSize", "totalIndexSize", "totalSize":
			assert.GreaterOrEqual(t, field.Value, int32(0))
		case "wiredTiger":
			// exclusive to MongoDB
		case "indexDetails", "indexSizes":
			v, ok := field.Value.(bson.D)
			require.True(t, ok)

			var indexNames []string
			for _, index := range v {
				indexNames = append(indexNames, index.Key)
			}

			assert.Equal(t, []string{"_id_"}, indexNames)
		default:
			actualComparable = append(actualComparable, field)
		}
	}

	expectedComparable := bson.D{
		{"ns", collection.Database().Name() + "." + collection.Name()},
		{"count", int32(1000)},
		{"numOrphanDocs", int32(0)},
		{"freeStorageSize", int32(0)},
		{"capped", false},
		{"nindexes", int32(1)},
		{"indexBuilds", bson.A{}},
		{"scaleFactor", int32(1)},
		{"ok", float64(1)},
	}
	AssertEqualDocuments(t, expectedComparable, actualComparable)
}

func TestCollStatsCommandScaleSize(t *testing.T) {
	t.Parallel()

	ctx, collection := setup.Setup(t, shareddata.DocumentsStrings)

	indexName := "custom-name"
	resIndexName, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{"foo", 1}, {"bar", -1}},
		Options: options.Index().SetName(indexName),
	})
	require.NoError(t, err)
	require.Equal(t, indexName, resIndexName)

	var resComparable, resNoScaleComparable bson.D
	var ready bool
	scale := int32(10)
	retries := 10

	for range retries {
		if ready {
			break
		}

		resComparable, resNoScaleComparable = nil, nil

		var resNoScale, res bson.D

		err = collection.Database().RunCommand(ctx, bson.D{{"collStats", collection.Name()}}).Decode(&resNoScale)
		require.NoError(t, err)

		err = collection.Database().RunCommand(ctx, bson.D{{"collStats", collection.Name()}, {"scale", scale}}).Decode(&res)
		require.NoError(t, err)

		var resTotalIndexSize any

		for _, field := range res {
			switch field.Key {
			case "indexDetails":
				v, ok := field.Value.(bson.D)
				require.True(t, ok)

				var indexNames []string
				for _, index := range v {
					indexNames = append(indexNames, index.Key)
				}

				assert.Equal(t, []string{"_id_", indexName}, indexNames)
			case "scaleFactor":
				assert.Equal(t, int32(10), field.Value)
			case "totalIndexSize":
				resTotalIndexSize = field.Value
				resComparable = append(resComparable, field)
			default:
				resComparable = append(resComparable, field)
			}
		}

		for _, field := range resNoScale {
			switch field.Key {
			case "totalIndexSize":
				size, ok := field.Value.(int32)
				require.True(t, ok)

				scaledSize := size / scale

				if scaledSize == resTotalIndexSize {
					ready = true
				}

				resNoScaleComparable = append(resNoScaleComparable, bson.E{Key: field.Key, Value: scaledSize})

			case "size", "storageSize", "totalSize":
				size, ok := field.Value.(int32)
				require.True(t, ok)

				scaledSize := size / scale

				resNoScaleComparable = append(resNoScaleComparable, bson.E{Key: field.Key, Value: scaledSize})
			case "indexDetails":
				v, ok := field.Value.(bson.D)
				require.True(t, ok)

				var indexNames []string
				for _, index := range v {
					indexNames = append(indexNames, index.Key)
				}

				assert.Equal(t, []string{"_id_", indexName}, indexNames)
			case "indexSizes":
				v, ok := field.Value.(bson.D)
				require.True(t, ok)

				var indexSizes bson.D

				for _, fieldName := range v {
					var size int32
					size, ok = fieldName.Value.(int32)
					require.True(t, ok)

					scaledSize := size / scale
					indexSizes = append(indexSizes, bson.E{Key: fieldName.Key, Value: scaledSize})
				}

				resNoScaleComparable = append(resNoScaleComparable, bson.E{Key: field.Key, Value: indexSizes})
			case "scaleFactor":
				assert.Equal(t, int32(1), field.Value)
			default:
				resNoScaleComparable = append(resNoScaleComparable, field)
			}
		}
	}

	AssertEqualDocuments(t, resComparable, resNoScaleComparable)
}

func TestDataSizeCommand(t *testing.T) {
	t.Parallel()

	t.Run("Existing", func(tt *testing.T) {
		tt.Parallel()

		t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/802")

		ctx, collection := setup.Setup(tt, shareddata.DocumentsStrings)

		// call validate to updated statistics
		err := collection.Database().RunCommand(ctx, bson.D{{"validate", collection.Name()}}).Err()
		require.NoError(t, err)

		var actual bson.D
		command := bson.D{{"dataSize", collection.Database().Name() + "." + collection.Name()}}
		err = collection.Database().RunCommand(ctx, command).Decode(&actual)
		require.NoError(t, err)

		var actualComparable, actualFieldNames bson.D

		for _, field := range actual {
			switch field.Key {
			case "size", "millis":
				assert.GreaterOrEqual(t, field.Value, int64(0))
			default:
				actualComparable = append(actualComparable, field)
			}

			actualFieldNames = append(actualFieldNames, bson.E{Key: field.Key})
		}

		expectedComparable := bson.D{
			{"numObjects", int64(len(shareddata.DocumentsStrings.Docs()))},
			{"estimate", false},
			{"ok", float64(1)},
		}
		AssertEqualDocuments(t, expectedComparable, actualComparable)

		expectedFieldNames := bson.D{
			{Key: "size"},
			{Key: "numObjects"},
			{Key: "millis"},
			{Key: "estimate"},
			{Key: "ok"},
		}
		AssertEqualDocuments(t, expectedFieldNames, actualFieldNames)
	})

	t.Run("NonExistent", func(tt *testing.T) {
		tt.Parallel()

		t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/802")

		ctx, collection := setup.Setup(tt)

		var actual bson.D
		command := bson.D{{"dataSize", collection.Database().Name() + "." + collection.Name()}}
		err := collection.Database().RunCommand(ctx, command).Decode(&actual)
		require.NoError(t, err)

		var actualComparable, actualFieldNames bson.D

		for _, field := range actual {
			switch field.Key {
			case "millis":
				assert.GreaterOrEqual(t, field.Value, int64(0))
			default:
				actualComparable = append(actualComparable, field)
			}

			actualFieldNames = append(actualFieldNames, bson.E{Key: field.Key})
		}

		expectedComparable := bson.D{
			{"size", int64(0)},
			{"numObjects", int64(0)},
			{"ok", float64(1)},
		}
		AssertEqualDocuments(t, expectedComparable, actualComparable)

		expectedFieldNames := bson.D{
			{Key: "size"},
			{Key: "numObjects"},
			{Key: "millis"},
			{Key: "ok"},
		}
		AssertEqualDocuments(t, expectedFieldNames, actualFieldNames)
	})
}

func TestDataSizeCommandErrors(t *testing.T) {
	t.Parallel()

	ctx, collection := setup.Setup(t, shareddata.DocumentsStrings)

	for name, tc := range map[string]struct { //nolint:vet // for readability
		command bson.D // required, command to run

		err        *mongo.CommandError // required, expected error from MongoDB
		altMessage string              // optional, alternative error message for FerretDB, ignored if empty
	}{
		"InvalidNamespace": {
			command: bson.D{{"dataSize", "invalid"}},
			err: &mongo.CommandError{
				Code:    73,
				Name:    "InvalidNamespace",
				Message: "Namespace invalid is not a valid collection name",
			},
			altMessage: "Invalid namespace specified 'invalid'",
		},
		"InvalidNamespaceTypeDocument": {
			command: bson.D{{"dataSize", bson.D{}}},
			err: &mongo.CommandError{
				Code:    14,
				Name:    "TypeMismatch",
				Message: "BSON field 'dataSize.dataSize' is the wrong type 'object', expected type 'string'",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, tc.command, "command must not be nil")
			require.NotNil(t, tc.err, "err must not be nil")

			var actual bson.D
			err := collection.Database().RunCommand(ctx, tc.command).Decode(&actual)

			assert.Nil(t, actual)
			AssertEqualAltCommandError(t, *tc.err, tc.altMessage, err)
		})
	}
}

func TestDBStatsCommand(tt *testing.T) {
	tt.Parallel()
	t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/9")

	ctx, collection := setup.Setup(tt, shareddata.DocumentsStrings)

	var actual bson.D
	command := bson.D{{"dbStats", int32(1)}}
	err := collection.Database().RunCommand(ctx, command).Decode(&actual)
	require.NoError(t, err)

	var actualComparalbe bson.D

	for _, field := range actual {
		switch field.Key {
		case "avgObjSize", "dataSize", "storageSize", "indexSize", "totalSize":
			val, ok := field.Value.(float64)
			require.True(t, ok)

			assert.InDelta(t, 37_500, val, 37_460)
			actualComparalbe = append(actualComparalbe, bson.E{Key: field.Key, Value: float64(0)})

		case "fsUsedSize", "fsTotalSize":
			val, ok := field.Value.(float64)
			require.True(t, ok)

			assert.Greater(t, val, float64(0))
			actualComparalbe = append(actualComparalbe, bson.E{Key: field.Key, Value: float64(0)})

		default:
			actualComparalbe = append(actualComparalbe, field)
		}
	}

	expected := bson.D{
		{"db", collection.Database().Name()},
		{"collections", int64(1)},
		{"views", int64(0)},
		{"objects", int64(len(shareddata.DocumentsStrings.Docs()))},
		{"avgObjSize", float64(0)},
		{"dataSize", float64(0)},
		{"storageSize", float64(0)},
		{"indexes", int64(1)},
		{"indexSize", float64(0)},
		{"totalSize", float64(0)},
		{"scaleFactor", int64(1)},
		{"fsUsedSize", float64(0)},
		{"fsTotalSize", float64(0)},
		{"ok", float64(1)},
	}

	AssertEqualDocuments(t, expected, actualComparalbe)
}

func TestDBStatsCommandEmpty(tt *testing.T) {
	tt.Parallel()
	t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/9")

	ctx, collection := setup.Setup(tt)

	var actual bson.D
	command := bson.D{{"dbStats", int32(1)}}
	err := collection.Database().RunCommand(ctx, command).Decode(&actual)
	require.NoError(t, err)

	expected := bson.D{
		{"db", collection.Database().Name()},
		{"collections", int64(0)},
		{"views", int64(0)},
		{"objects", int64(0)},
		{"avgObjSize", float64(0)},
		{"dataSize", float64(0)},
		{"storageSize", float64(0)},
		{"indexes", int64(0)},
		{"indexSize", float64(0)},
		{"totalSize", float64(0)},
		{"scaleFactor", int64(1)},
		{"fsUsedSize", float64(0)},
		{"fsTotalSize", float64(0)},
		{"ok", float64(1)},
	}

	AssertEqualDocuments(t, expected, actual)
}

func TestDBStatsCommandScale(tt *testing.T) {
	tt.Parallel()

	ctx, collection := setup.Setup(tt, shareddata.DocumentsStrings)

	for name, tc := range map[string]struct { //nolint:vet // for readability
		scale               any
		expectedScaleFactor int64
	}{
		"scaleOne":   {scale: int32(1), expectedScaleFactor: int64(1)},
		"scaleBig":   {scale: int64(1_000), expectedScaleFactor: int64(1_000)},
		"scaleFloat": {scale: 2.8, expectedScaleFactor: int64(2)},
		"scaleNull":  {scale: nil, expectedScaleFactor: int64(1)},
	} {
		tt.Run(name, func(tt *testing.T) {
			tt.Helper()
			tt.Parallel()

			t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/9")

			var actual bson.D
			command := bson.D{{"dbStats", int32(1)}, {"scale", tc.scale}}
			err := collection.Database().RunCommand(ctx, command).Decode(&actual)
			require.NoError(t, err)

			var actualComparalbe bson.D

			for _, field := range actual {
				switch field.Key {
				case "avgObjSize", "dataSize", "storageSize", "indexSize", "totalSize":
					val, ok := field.Value.(float64)
					require.True(t, ok)

					assert.InDelta(t, 35_500, val, 35_500)
					actualComparalbe = append(actualComparalbe, bson.E{Key: field.Key, Value: float64(0)})

				case "fsUsedSize", "fsTotalSize":
					val, ok := field.Value.(float64)
					require.True(t, ok)

					assert.Greater(t, val, float64(0))
					actualComparalbe = append(actualComparalbe, bson.E{Key: field.Key, Value: float64(0)})

				default:
					actualComparalbe = append(actualComparalbe, field)
				}
			}

			expected := bson.D{
				{"db", collection.Database().Name()},
				{"collections", int64(1)},
				{"views", int64(0)},
				{"objects", int64(len(shareddata.DocumentsStrings.Docs()))},
				{"avgObjSize", float64(0)},
				{"dataSize", float64(0)},
				{"storageSize", float64(0)},
				{"indexes", int64(1)},
				{"indexSize", float64(0)},
				{"totalSize", float64(0)},
				{"scaleFactor", tc.expectedScaleFactor},
				{"fsUsedSize", float64(0)},
				{"fsTotalSize", float64(0)},
				{"ok", float64(1)},
			}

			AssertEqualDocuments(t, expected, actualComparalbe)
		})
	}
}

func TestDBStatsCommandScaleEmptyDatabase(tt *testing.T) {
	tt.Parallel()
	t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/9")

	ctx, collection := setup.Setup(tt)

	var actual bson.D
	command := bson.D{{"dbStats", int32(1)}, {"scale", float64(1_000)}}
	err := collection.Database().RunCommand(ctx, command).Decode(&actual)
	require.NoError(t, err)

	expected := bson.D{
		{"db", collection.Database().Name()},
		{"collections", int64(0)},
		{"views", int64(0)},
		{"objects", int64(0)},
		{"avgObjSize", float64(0)},
		{"dataSize", float64(0)},
		{"storageSize", float64(0)},
		{"indexes", int64(0)},
		{"indexSize", float64(0)},
		{"totalSize", float64(0)},
		{"scaleFactor", int64(1000)},
		{"fsUsedSize", float64(0)},
		{"fsTotalSize", float64(0)},
		{"ok", float64(1)},
	}

	AssertEqualDocuments(t, expected, actual)
}

func TestDBStatsCommandFreeStorage(tt *testing.T) {
	tt.Parallel()

	ctx, collection := setup.Setup(tt, shareddata.DocumentsStrings)

	for name, tc := range map[string]struct { //nolint:vet // for readability
		command  bson.D // required, command to run
		expected bson.D // required, expected result
	}{
		"Unset": {
			command: bson.D{{"dbStats", int32(1)}},
			expected: bson.D{
				{"db", collection.Database().Name()},
				{"collections", int64(1)},
				{"views", int64(0)},
				{"objects", int64(len(shareddata.DocumentsStrings.Docs()))},
				{"avgObjSize", float64(0)},
				{"dataSize", float64(0)},
				{"storageSize", float64(0)},
				{"indexes", int64(1)},
				{"indexSize", float64(0)},
				{"totalSize", float64(0)},
				{"scaleFactor", int64(1)},
				{"fsUsedSize", float64(0)},
				{"fsTotalSize", float64(0)},
				{"ok", float64(1)},
			},
		},
		"Int32Zero": {
			command: bson.D{{"dbStats", int32(1)}, {"freeStorage", int32(0)}},
			expected: bson.D{
				{"db", collection.Database().Name()},
				{"collections", int64(1)},
				{"views", int64(0)},
				{"objects", int64(len(shareddata.DocumentsStrings.Docs()))},
				{"avgObjSize", float64(0)},
				{"dataSize", float64(0)},
				{"storageSize", float64(0)},
				{"indexes", int64(1)},
				{"indexSize", float64(0)},
				{"totalSize", float64(0)},
				{"scaleFactor", int64(1)},
				{"fsUsedSize", float64(0)},
				{"fsTotalSize", float64(0)},
				{"ok", float64(1)},
			},
		},
		"Int32One": {
			command: bson.D{{"dbStats", int32(1)}, {"freeStorage", int32(1)}},
			expected: bson.D{
				{"db", collection.Database().Name()},
				{"collections", int64(1)},
				{"views", int64(0)},
				{"objects", int64(len(shareddata.DocumentsStrings.Docs()))},
				{"avgObjSize", float64(0)},
				{"dataSize", float64(0)},
				{"storageSize", float64(0)},
				{"indexes", int64(1)},
				{"indexSize", float64(0)},
				{"totalSize", float64(0)},
				{"freeStorageSize", float64(0)},
				{"indexFreeStorageSize", float64(0)},
				{"totalFreeStorageSize", float64(0)},
				{"scaleFactor", int64(1)},
				{"fsUsedSize", float64(0)},
				{"fsTotalSize", float64(0)},
				{"ok", float64(1)},
			},
		},
		"Int32Negative": {
			command: bson.D{{"dbStats", int32(1)}, {"freeStorage", int32(-1)}},
			expected: bson.D{
				{"db", collection.Database().Name()},
				{"collections", int64(1)},
				{"views", int64(0)},
				{"objects", int64(len(shareddata.DocumentsStrings.Docs()))},
				{"avgObjSize", float64(0)},
				{"dataSize", float64(0)},
				{"storageSize", float64(0)},
				{"indexes", int64(1)},
				{"indexSize", float64(0)},
				{"totalSize", float64(0)},
				{"freeStorageSize", float64(0)},
				{"indexFreeStorageSize", float64(0)},
				{"totalFreeStorageSize", float64(0)},
				{"scaleFactor", int64(1)},
				{"fsUsedSize", float64(0)},
				{"fsTotalSize", float64(0)},
				{"ok", float64(1)},
			},
		},
		"True": {
			command: bson.D{{"dbStats", int32(1)}, {"freeStorage", true}},
			expected: bson.D{
				{"db", collection.Database().Name()},
				{"collections", int64(1)},
				{"views", int64(0)},
				{"objects", int64(len(shareddata.DocumentsStrings.Docs()))},
				{"avgObjSize", float64(0)},
				{"dataSize", float64(0)},
				{"storageSize", float64(0)},
				{"indexes", int64(1)},
				{"indexSize", float64(0)},
				{"totalSize", float64(0)},
				{"freeStorageSize", float64(0)},
				{"indexFreeStorageSize", float64(0)},
				{"totalFreeStorageSize", float64(0)},
				{"scaleFactor", int64(1)},
				{"fsUsedSize", float64(0)},
				{"fsTotalSize", float64(0)},
				{"ok", float64(1)},
			},
		},
		"False": {
			command: bson.D{{"dbStats", int32(1)}, {"freeStorage", false}},
			expected: bson.D{
				{"db", collection.Database().Name()},
				{"collections", int64(1)},
				{"views", int64(0)},
				{"objects", int64(len(shareddata.DocumentsStrings.Docs()))},
				{"avgObjSize", float64(0)},
				{"dataSize", float64(0)},
				{"storageSize", float64(0)},
				{"indexes", int64(1)},
				{"indexSize", float64(0)},
				{"totalSize", float64(0)},
				{"scaleFactor", int64(1)},
				{"fsUsedSize", float64(0)},
				{"fsTotalSize", float64(0)},
				{"ok", float64(1)},
			},
		},
		"Nil": {
			command: bson.D{{"dbStats", int32(1)}, {"freeStorage", nil}},
			expected: bson.D{
				{"db", collection.Database().Name()},
				{"collections", int64(1)},
				{"views", int64(0)},
				{"objects", int64(len(shareddata.DocumentsStrings.Docs()))},
				{"avgObjSize", float64(0)},
				{"dataSize", float64(0)},
				{"storageSize", float64(0)},
				{"indexes", int64(1)},
				{"indexSize", float64(0)},
				{"totalSize", float64(0)},
				{"scaleFactor", int64(1)},
				{"fsUsedSize", float64(0)},
				{"fsTotalSize", float64(0)},
				{"ok", float64(1)},
			},
		},
	} {
		tt.Run(name, func(tt *testing.T) {
			tt.Parallel()

			t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/9")

			var actual bson.D
			err := collection.Database().RunCommand(ctx, tc.command).Decode(&actual)
			require.NoError(t, err)

			var actualComparalbe bson.D

			for _, field := range actual {
				switch field.Key {
				case "avgObjSize", "dataSize", "storageSize", "indexSize", "totalSize",
					"freeStorageSize", "indexFreeStorageSize", "totalFreeStorageSize":
					val, ok := field.Value.(float64)
					require.True(t, ok)

					assert.InDelta(t, 35_500, val, 35_500)
					actualComparalbe = append(actualComparalbe, bson.E{Key: field.Key, Value: float64(0)})

				case "fsUsedSize", "fsTotalSize":
					val, ok := field.Value.(float64)
					require.True(t, ok)

					assert.Greater(t, val, float64(0))
					actualComparalbe = append(actualComparalbe, bson.E{Key: field.Key, Value: float64(0)})

				default:
					actualComparalbe = append(actualComparalbe, field)
				}
			}

			AssertEqualDocuments(t, tc.expected, actualComparalbe)
		})
	}
}

//nolint:paralleltest // we test a global server status
func TestServerStatusCommand(t *testing.T) {
	ctx, collection := setup.Setup(t)

	var actual bson.D
	command := bson.D{{"serverStatus", int32(1)}}
	err := collection.Database().RunCommand(ctx, command).Decode(&actual)
	require.NoError(t, err)

	var actualComparable bson.D
	var freeMonitoringComparable bson.D // only for FerretDB

	for _, field := range actual {
		switch field.Key {
		case "ferretdb":
			value, ok := field.Value.(bson.D)
			require.True(t, ok)
			expected := bson.D{
				{"version", "unknown"},
				{"gitVersion", "unknown"},
				{"buildEnvironment", bson.D{}},
				{"debug", true},
				{"package", "unknown"},
				{"postgresql", expectedPostgreSQLVersion},
				{"documentdb", version.DocumentDB},
			}
			AssertEqualDocuments(t, expected, value)

		case "freeMonitoring":
			freeMonitoring, ok := field.Value.(bson.D)
			require.True(t, ok)

			freeMonitoringComparable = freeMonitoring

		case "version":
			assert.IsType(t, "", field.Value)
			assert.Regexp(t, `^7\.0\.`, field.Value)
			actualComparable = append(actualComparable, bson.E{"version", "7.0.0"})

		case "catalogStats":
			catalogStats, ok := field.Value.(bson.D)
			require.True(t, ok)

			var catalogStatsComparable bson.D

			for _, subField := range catalogStats {
				switch subField.Key {
				case "capped", "csfle", "queryableEncryption":
				// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/629
				// Fields are not set in FerretDB

				case "collections", "internalCollections", "internalViews":
					assert.IsType(t, int32(0), subField.Value)
					catalogStatsComparable = append(catalogStatsComparable, bson.E{subField.Key, int32(0)})

				default:
					catalogStatsComparable = append(catalogStatsComparable, subField)
				}
			}

			actualComparable = append(actualComparable, bson.E{field.Key, catalogStatsComparable})

		case "host", "process":
			assert.IsType(t, "", field.Value)
			assert.NotEmpty(t, field.Value)
			actualComparable = append(actualComparable, bson.E{field.Key, ""})

		case "localTime":
			assert.IsType(t, primitive.DateTime(0), field.Value)
			assert.WithinDuration(t, time.Now(), field.Value.(primitive.DateTime).Time(), 2*time.Second)
			actualComparable = append(actualComparable, bson.E{field.Key, primitive.DateTime(0)})

		case "pid", "uptimeMillis":
			assert.IsType(t, int64(0), field.Value)
			assert.Greater(t, field.Value, int64(0), field.Key)
			actualComparable = append(actualComparable, bson.E{field.Key, int64(0)})

		case "uptimeEstimate":
			assert.IsType(t, int64(0), field.Value)
			actualComparable = append(actualComparable, bson.E{field.Key, int64(0)})

		case "uptime":
			assert.IsType(t, float64(0), field.Value)
			actualComparable = append(actualComparable, bson.E{field.Key, float64(0)})

		case "wiredTiger", "query", "metrics", "asserts", "batchedDeletes", "connections", "defaultRWConcern",
			"electionMetrics", "internalTransactions", "extra_info", "logicalSessionRecordCache",
			"featureCompatibilityVersion", "flowControl", "globalLock", "indexBuilds", "indexBulkBuilder",
			"indexStats", "locks", "network", "compression", "serviceExecutors", "opLatencies",
			"opcounters", "opcountersRepl", "oplogTruncation", "queryAnalyzers",
			"readConcernCounters", "readPreferenceCounters", "repl", "scramCache", "security", "shardSplits",
			"storageEngine", "tcmalloc", "tenantMigrations", "trafficRecording", "transactions", "transportSecurity",
			"twoPhaseCommitCoordinator", "mem", "collectionCatalog":
			// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/629
			// Not implemented in FerretDB, we might want to support some of these fields

		default:
			actualComparable = append(actualComparable, field)
		}
	}

	t.Run("FreeMonitoring", func(t *testing.T) {
		tt := setup.FailsForMongoDB(t, "MongoDB decommissioned free monitoring")
		freeMonitoringExpected := bson.D{
			{"state", "undecided"},
		}
		AssertEqualDocuments(tt, freeMonitoringExpected, freeMonitoringComparable)
	})

	expected := bson.D{
		{"host", ""},
		{"version", "7.0.0"},
		{"process", ""},
		{"pid", int64(0)},
		{"uptime", float64(0)},
		{"uptimeMillis", int64(0)},
		{"uptimeEstimate", int64(0)},
		{"localTime", primitive.DateTime(0)},
		{"catalogStats", bson.D{
			{"collections", int32(0)},
			{"clustered", int32(0)},
			{"timeseries", int32(0)},
			{"views", int32(0)},
			{"internalCollections", int32(0)},
			{"internalViews", int32(0)},
		}},
		{"ok", float64(1)},
	}

	AssertEqualDocuments(t, expected, actualComparable)
}

func TestServerStatusCommandMetrics(t *testing.T) {
	setup.SkipForMongoDB(t, "MongoDB decommissioned server status metrics")

	t.Parallel()

	for name, tc := range map[string]struct {
		cmds            []bson.D
		expectedNonZero []string
	}{
		"BasicCmd": {
			cmds: []bson.D{
				{{"ping", int32(1)}},
			},
			expectedNonZero: []string{"total"},
		},
		"UpdateCmd": {
			cmds: []bson.D{
				{{"update", "values"}, {"updates", bson.A{bson.D{{"q", bson.D{{"v", "foo"}}}}}}},
			},
			expectedNonZero: []string{"total"},
		},
		"UpdateCmdFailed": {
			cmds: []bson.D{
				{{"update", int32(1)}},
			},
			expectedNonZero: []string{"failed", "total"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, collection := setup.Setup(t)

			for _, cmd := range tc.cmds {
				collection.Database().RunCommand(ctx, cmd)
			}

			// ensure non-zero uptime
			ctxutil.Sleep(ctx, time.Second)

			command := bson.D{{"serverStatus", int32(1)}}

			var actual bson.D
			err := collection.Database().RunCommand(ctx, command).Decode(&actual)
			require.NoError(t, err)

			var actualComparable bson.D

			for _, field := range actual {
				switch field.Key {
				case "host":
					host, ok := field.Value.(string)
					require.True(t, ok)
					assert.NotEmpty(t, host)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: ""})

				case "localTime":
					localTime, ok := field.Value.(primitive.DateTime)
					require.True(t, ok)
					assert.NotEmpty(t, localTime)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: primitive.DateTime(0)})

				case "pid", "uptimeMillis":
					subField, ok := field.Value.(int64)
					require.True(t, ok)
					assert.Greater(t, subField, int64(0), field.Key)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: int64(0)})

				case "uptimeEstimate":
					require.IsType(t, int64(0), field.Value)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: int64(0)})

				case "process":
					process, ok := field.Value.(string)
					require.True(t, ok)
					assert.NotEmpty(t, process)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: ""})

				case "uptime":
					uptime, ok := field.Value.(float64)
					require.True(t, ok)
					assert.Greater(t, uptime, float64(0))
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: float64(0)})

				case "version":
					version, ok := field.Value.(string)
					require.True(t, ok)
					assert.Regexp(t, `^7\.0\.`, version)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: "7.0.0"})

				case "metrics":
					metrics, ok := field.Value.(bson.D)
					require.True(t, ok)

					var metricsComparable bson.D

					for _, mField := range metrics {
						switch mField.Key {
						case "commands":
							commands, cOk := mField.Value.(bson.D)
							assert.True(t, cOk)

							for _, command := range commands {
								cmd, cmdOk := command.Value.(bson.D)
								assert.True(t, cmdOk)

								var cmdComparable bson.D

								for _, cmdField := range cmd {
									switch cmdField.Key {
									case "total", "failed":
										assert.IsType(t, int64(0), cmdField.Value)
										cmdComparable = append(cmdComparable, bson.E{Key: cmdField.Key, Value: int64(0)})

									default:
										cmdComparable = append(cmdComparable, cmdField)
									}
								}

								cmdExpected := bson.D{
									{"failed", int64(0)},
									{"total", int64(0)},
								}

								AssertEqualDocuments(t, cmdExpected, cmdComparable)
							}

							metricsComparable = append(metricsComparable, bson.E{Key: mField.Key, Value: bson.D{}})

						default:
							metricsComparable = append(metricsComparable, mField)
						}
					}

					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: metricsComparable})

				default:
					actualComparable = append(actualComparable, field)
				}
			}

			expected := bson.D{
				{"catalogStats", bson.D{
					{"clustered", int32(0)},
					{"collections", int32(0)},
					{"internalCollections", int32(0)},
					{"internalViews", int32(0)},
					{"timeseries", int32(0)},
					{"views", int32(0)},
				}},
				{"ferretdb", bson.D{
					{"version", "unknown"},
					{"gitVersion", "unknown"},
					{"buildEnvironment", bson.D{}},
					{"debug", true},
					{"package", "unknown"},
					{"postgresql", expectedPostgreSQLVersion},
					{"documentdb", version.DocumentDB},
				}},
				{"freeMonitoring", bson.D{{"state", "undecided"}}},
				{"host", ""},
				{"localTime", primitive.DateTime(0)},
				{"metrics", bson.D{{"commands", bson.D{}}}},
				{"ok", float64(1)},
				{"pid", int64(0)},
				{"process", ""},
				{"uptime", float64(0)},
				{"uptimeEstimate", int64(0)},
				{"uptimeMillis", int64(0)},
				{"version", "7.0.0"},
			}

			AssertEqualDocuments(t, expected, actualComparable)
		})
	}
}

func TestServerStatusCommandFreeMonitoring(t *testing.T) {
	setup.SkipForMongoDB(t, "MongoDB decommissioned free monitoring")

	// this test shouldn't be run in parallel, because it requires a specific state of the field which would be modified by the other tests.
	s := setup.SetupWithOpts(t, &setup.SetupOpts{
		DatabaseName: "admin",
	})

	for name, tc := range map[string]struct {
		command        bson.D
		expectedStatus string
	}{
		"Enable": {
			command:        bson.D{{"setFreeMonitoring", 1}, {"action", "enable"}},
			expectedStatus: "enabled",
		},
		"Disable": {
			command:        bson.D{{"setFreeMonitoring", 1}, {"action", "disable"}},
			expectedStatus: "disabled",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, tc.command, "command must not be nil")

			res := s.Collection.Database().RunCommand(s.Ctx, tc.command)
			require.NoError(t, res.Err())

			var actual bson.D
			err := s.Collection.Database().RunCommand(s.Ctx, bson.D{{"serverStatus", 1}}).Decode(&actual)
			require.NoError(t, err)

			var status any
			var actualComparable bson.D

			for _, field := range actual {
				switch field.Key {
				case "host":
					host, ok := field.Value.(string)
					require.True(t, ok)
					assert.NotEmpty(t, host)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: ""})

				case "localTime":
					localTime, ok := field.Value.(primitive.DateTime)
					require.True(t, ok)
					assert.NotEmpty(t, localTime)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: primitive.DateTime(0)})

				case "pid", "uptimeMillis":
					subField, ok := field.Value.(int64)
					require.True(t, ok)
					assert.Greater(t, subField, int64(0), field.Key)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: int64(0)})

				case "uptimeEstimate":
					require.IsType(t, int64(0), field.Value)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: int64(0)})

				case "process":
					process, ok := field.Value.(string)
					require.True(t, ok)
					assert.NotEmpty(t, process)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: ""})

				case "uptime":
					uptime, ok := field.Value.(float64)
					require.True(t, ok)
					assert.Greater(t, uptime, float64(0))
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: float64(0)})

				case "version":
					version, ok := field.Value.(string)
					require.True(t, ok)
					assert.Regexp(t, `^7\.0\.`, version)
					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: "7.0.0"})

				case "freeMonitoring":
					freeMonitoring, ok := field.Value.(bson.D)
					require.True(t, ok)

					for _, subField := range freeMonitoring {
						switch subField.Key {
						case "state":
							status = subField.Value
						}
					}

					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: freeMonitoring})

				case "metrics":
					metrics, ok := field.Value.(bson.D)
					require.True(t, ok)

					var metricsComparable bson.D

					for _, mField := range metrics {
						switch mField.Key {
						case "commands":
							commands, cOk := mField.Value.(bson.D)
							assert.True(t, cOk)

							for _, command := range commands {
								cmd, cmdOk := command.Value.(bson.D)
								assert.True(t, cmdOk)

								var cmdComparable bson.D

								for _, cmdField := range cmd {
									switch cmdField.Key {
									case "total", "failed":
										assert.IsType(t, int64(0), cmdField.Value)
										cmdComparable = append(cmdComparable, bson.E{Key: cmdField.Key, Value: int64(0)})

									default:
										cmdComparable = append(cmdComparable, cmdField)
									}
								}

								cmdExpected := bson.D{
									{"failed", int64(0)},
									{"total", int64(0)},
								}

								AssertEqualDocuments(t, cmdExpected, cmdComparable)
							}

							metricsComparable = append(metricsComparable, bson.E{Key: mField.Key, Value: bson.D{}})

						default:
							metricsComparable = append(metricsComparable, mField)
						}
					}

					actualComparable = append(actualComparable, bson.E{Key: field.Key, Value: metricsComparable})

				default:
					actualComparable = append(actualComparable, field)
				}
			}

			expected := bson.D{
				{"catalogStats", bson.D{
					{"clustered", int32(0)},
					{"collections", int32(0)},
					{"internalCollections", int32(0)},
					{"internalViews", int32(0)},
					{"timeseries", int32(0)},
					{"views", int32(0)},
				}},
				{"ferretdb", bson.D{
					{"version", "unknown"},
					{"gitVersion", "unknown"},
					{"buildEnvironment", bson.D{}},
					{"debug", true},
					{"package", "unknown"},
					{"postgresql", expectedPostgreSQLVersion},
					{"documentdb", version.DocumentDB},
				}},
				{"freeMonitoring", bson.D{{"state", tc.expectedStatus}}},
				{"host", ""},
				{"localTime", primitive.DateTime(0)},
				{"metrics", bson.D{{"commands", bson.D{}}}},
				{"ok", float64(1)},
				{"pid", int64(0)},
				{"process", ""},
				{"uptime", float64(0)},
				{"uptimeEstimate", int64(0)},
				{"uptimeMillis", int64(0)},
				{"version", "7.0.0"},
			}

			AssertEqualDocuments(t, expected, actualComparable)

			assert.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestServerStatusCommandStress(t *testing.T) {
	// It should be rewritten to use teststress.Stress.
	ctx, collection := setup.Setup(t) // no providers there, we will create collections concurrently
	client := collection.Database().Client()

	dbNum := runtime.GOMAXPROCS(-1) * 10

	ready := make(chan struct{}, dbNum)
	start := make(chan struct{})

	var wg sync.WaitGroup
	for i := 0; i < dbNum; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			ready <- struct{}{}

			<-start

			dbName := fmt.Sprintf("%s_stress_%d", collection.Database().Name(), i)
			db := client.Database(dbName)
			_ = db.Drop(ctx) // make sure DB doesn't exist (it will be created together with the collection)

			collName := fmt.Sprintf("stress_%d", i)

			err := db.CreateCollection(ctx, collName)
			assert.NoError(t, err)

			err = db.Drop(ctx)
			assert.NoError(t, err)

			command := bson.D{{"serverStatus", int32(1)}}
			var actual bson.D
			err = collection.Database().RunCommand(ctx, command).Decode(&actual)

			assert.NoError(t, err)
		}(i)
	}

	for i := 0; i < dbNum; i++ {
		<-ready
	}

	close(start)

	wg.Wait()
}

func TestCompactCommandForce(t *testing.T) {
	// don't run in parallel as parallel `VACUUM FULL ANALYZE` may result in deadlock
	s := setup.SetupWithOpts(t, &setup.SetupOpts{
		DatabaseName: "admin",
		Providers:    []shareddata.Provider{shareddata.DocumentsStrings},
	})

	for name, tc := range map[string]struct {
		force any // optional, defaults to unset

		err            *mongo.CommandError // optional
		altMessage     string              // optional, alternative error message
		skipForMongoDB string              // optional, skip test for mongoDB with a specific reason
	}{
		"True": {
			force: true,
		},
		"False": {
			force:          false,
			skipForMongoDB: "Only {force:true} can be run on active replica set primary",
		},
		"Int32": {
			force: int32(1),
		},
		"Int32Zero": {
			force:          int32(0),
			skipForMongoDB: "Only {force:true} can be run on active replica set primary",
		},
		"Int64": {
			force: int64(1),
		},
		"Int64Zero": {
			force:          int64(0),
			skipForMongoDB: "Only {force:true} can be run on active replica set primary",
		},
		"Double": {
			force: float64(1),
		},
		"DoubleZero": {
			force:          float64(0),
			skipForMongoDB: "Only {force:true} can be run on active replica set primary",
		},
		"Unset": {
			skipForMongoDB: "Only {force:true} can be run on active replica set primary",
		},
		"String": {
			force: "foo",
			err: &mongo.CommandError{
				Code:    14,
				Name:    "TypeMismatch",
				Message: "BSON field 'force' is the wrong type 'string', expected types '[bool, long, int, decimal, double]'",
			},
			skipForMongoDB: "force is FerretDB specific field",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.skipForMongoDB != "" {
				setup.SkipForMongoDB(t, tc.skipForMongoDB)
			}

			command := bson.D{{"compact", s.Collection.Name()}}
			if tc.force != nil {
				command = append(command, bson.E{Key: "force", Value: tc.force})
			}

			var res bson.D
			err := s.Collection.Database().RunCommand(
				s.Ctx,
				command,
			).Decode(&res)

			if tc.err != nil {
				AssertEqualAltCommandError(t, *tc.err, tc.altMessage, err)
				return
			}

			require.NoError(t, err)

			expected := bson.D{
				{"bytesFreed", int32(0)},
				{"ok", float64(1)},
			}

			AssertEqualDocuments(t, expected, res)
		})
	}
}

func TestCompactCommandErrors(tt *testing.T) {
	tt.Parallel()

	for name, tc := range map[string]struct {
		dbName string

		err              *mongo.CommandError // required
		altMessage       string              // optional, alternative error message
		skipForMongoDB   string              // optional, skip test for MongoDB backend with a specific reason
		failsForFerretDB string
	}{
		"NonExistentDB": {
			dbName: "non-existent",
			err: &mongo.CommandError{
				Code:    26,
				Name:    "NamespaceNotFound",
				Message: "database does not exist",
			},
			altMessage: "Invalid namespace specified 'non-existent.non-existent'",

			skipForMongoDB:   "Only {force:true} can be run on active replica set primary",
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/286",
		},
		"NonExistentCollection": {
			dbName: "admin",
			err: &mongo.CommandError{
				Code:    26,
				Name:    "NamespaceNotFound",
				Message: "collection does not exist",
			},
			altMessage:       "Invalid namespace specified 'admin.non-existent'",
			skipForMongoDB:   "Only {force:true} can be run on active replica set primary",
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/286",
		},
	} {
		tt.Run(name, func(tt *testing.T) {
			tt.Parallel()

			var t testing.TB = tt
			if tc.failsForFerretDB != "" {
				t = setup.FailsForFerretDB(tt, tc.failsForFerretDB)
			}

			if tc.skipForMongoDB != "" {
				setup.SkipForMongoDB(t, tc.skipForMongoDB)
			}

			require.NotNil(t, tc.err, "err must not be nil")

			s := setup.SetupWithOpts(tt, &setup.SetupOpts{
				DatabaseName: tc.dbName,
			})

			var res bson.D
			err := s.Collection.Database().RunCommand(
				s.Ctx,
				bson.D{{"compact", "non-existent"}},
			).Decode(&res)

			AssertEqualAltCommandError(t, *tc.err, tc.altMessage, err)
		})
	}
}

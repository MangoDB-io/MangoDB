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

// Package integration provides FerretDB integration tests.
package integration

import (
	"context"
	"slices"
	"testing"

	"github.com/FerretDB/wire/wirebson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/v2/internal/util/testutil"

	"github.com/FerretDB/FerretDB/v2/integration/setup"
)

//go:generate ../bin/stringer -linecomment -type compatTestCaseResultType

// compatTestCaseResultType represents compatibility test case result type.
//
// It is used to avoid errors with invalid queries making tests pass.
type compatTestCaseResultType int

const (
	// Test case should return non-empty result at least for one collection/provider.
	nonEmptyResult compatTestCaseResultType = iota

	// Test case should return empty result for all collections/providers.
	emptyResult
)

// convert converts given driver value ([bson.D], [bson.A], etc) to FerretDB's bson package value.
func convert(t testing.TB, v any) any {
	t.Helper()

	switch v := v.(type) {
	// composite types
	case primitive.D:
		doc := wirebson.MakeDocument(len(v))
		for _, e := range v {
			err := doc.Add(e.Key, convert(t, e.Value))
			require.NoError(t, err)
		}

		return doc

	case primitive.A:
		arr := wirebson.MakeArray(len(v))
		for _, e := range v {
			err := arr.Add(convert(t, e))
			require.NoError(t, err)
		}

		return arr

	// scalar types (in the same order as in bson package)
	case float64:
		return v
	case string:
		return v
	case primitive.Binary:
		return wirebson.Binary{
			Subtype: wirebson.BinarySubtype(v.Subtype),
			B:       v.Data,
		}
	case primitive.ObjectID:
		return wirebson.ObjectID(v)
	case bool:
		return v
	case primitive.DateTime:
		return v.Time()
	case nil:
		return wirebson.Null
	case primitive.Regex:
		return wirebson.Regex{
			Pattern: v.Pattern,
			Options: v.Options,
		}
	case int32:
		return v
	case primitive.Timestamp:
		return wirebson.NewTimestamp(v.T, v.I)
	case int64:
		return v
	case primitive.Decimal128:
		h, l := v.GetBytes()

		return wirebson.Decimal128{
			H: h,
			L: l,
		}

	default:
		t.Fatalf("unexpected type %T", v)
		panic("not reached")
	}
}

// fixCluster removes document fields that are specific for MongoDB running in a cluster.
func fixCluster(t testing.TB, doc *wirebson.Document) {
	t.Helper()

	doc.Remove("$clusterTime")
	doc.Remove("electionId")
	doc.Remove("operationTime")
	doc.Remove("opTime")
	doc.Remove("commitQuorum")
}

// fixOrder sorts document fields.
//
// It does nothing if the current test is running for MongoDB.
//
// This function should eventually be removed.
// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/410
// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/348
func fixOrder(t testing.TB, doc *wirebson.Document) {
	t.Helper()

	if setup.IsMongoDB(t) {
		return
	}

	fieldNames := doc.FieldNames()
	slices.Sort(fieldNames)
	fieldNames = slices.Compact(fieldNames)
	require.Len(t, fieldNames, len(doc.FieldNames()), "duplicate field names are not handled")

	for _, n := range fieldNames {
		v := doc.Get(n)
		require.NotNil(t, v)

		// no practical need to handle arrays yet
		if ad, ok := v.(wirebson.AnyDocument); ok {
			d, err := ad.Decode()
			require.NoError(t, err)
			fixOrder(t, d)
			v = d
		}

		doc.Remove(n)
		require.NoError(t, doc.Add(n, v))
	}
}

// fixActualUpdateN replaces the int64 `nMatched`, `nModified`, `nUpserted`, `n` fields with a int32 values.
//
// It does nothing if the current test is running for MongoDB.
//
// It should be used only with actual/target document, not with expected/compat document.
//
// This function should eventually be removed.
// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/359
func fixActualUpdateN(t testing.TB, actual *wirebson.Document) {
	t.Helper()

	if setup.IsMongoDB(t) {
		return
	}

	// avoid updating generic responses that just happened to contain `n`
	if actual.Get("nMatched") == nil && actual.Get("nModified") == nil && actual.Get("nUpserted") == nil {
		return
	}

	for _, f := range []string{"nMatched", "nModified", "nUpserted", "n"} {
		switch v := actual.Get(f).(type) {
		case int64:
			require.NoError(t, actual.Replace(f, int32(v)))
		}
	}
}

// fixExpected applies fixes to the expected/compat document.
func fixExpected(t testing.TB, expected *wirebson.Document) {
	t.Helper()

	fixCluster(t, expected)
	fixOrder(t, expected)
}

// fixActual applies fixes to the actual/target document.
func fixActual(t testing.TB, actual *wirebson.Document) {
	t.Helper()

	fixCluster(t, actual)
	fixOrder(t, actual)
	fixActualUpdateN(t, actual)
}

// convertDocument converts given driver's document to FerretDB's *bson.Document.
func convertDocument(t testing.TB, doc bson.D) *wirebson.Document {
	t.Helper()

	v := convert(t, doc)

	var res *wirebson.Document
	require.IsType(t, res, v)

	return v.(*wirebson.Document)
}

// convertDocuments converts given driver's documents slice to FerretDB's []*bson.Document.
func convertDocuments(t testing.TB, docs []bson.D) []*wirebson.Document {
	t.Helper()

	res := make([]*wirebson.Document, len(docs))
	for i, doc := range docs {
		res[i] = convertDocument(t, doc)
	}

	return res
}

// AssertEqualDocuments asserts that two documents are equal in a way that is useful for tests.
func AssertEqualDocuments(t testing.TB, expected, actual bson.D) bool {
	t.Helper()

	expectedDoc := convertDocument(t, expected)
	actualDoc := convertDocument(t, actual)

	fixExpected(t, expectedDoc)
	fixActual(t, actualDoc)

	return testutil.AssertEqual(t, expectedDoc, actualDoc)
}

// AssertEqualDocumentsSlice asserts that two document slices are equal in a way that is useful for tests.
func AssertEqualDocumentsSlice(t testing.TB, expected, actual []bson.D) bool {
	t.Helper()

	expectedDocs := convertDocuments(t, expected)
	actualDocs := convertDocuments(t, actual)

	for _, d := range expectedDocs {
		fixExpected(t, d)
	}

	for _, d := range actualDocs {
		fixActual(t, d)
	}

	return testutil.AssertEqualSlices(t, expectedDocs, actualDocs)
}

// AssertEqualCommandError asserts that the expected error is the same as the actual (ignoring the Raw part).
func AssertEqualCommandError(t testing.TB, expected mongo.CommandError, actual error) bool {
	t.Helper()

	a, ok := actual.(mongo.CommandError)
	if !ok {
		return assert.Equal(t, expected, actual)
	}

	// set expected fields that might be helpful in the test output
	require.Nil(t, expected.Raw)
	expected.Raw = a.Raw

	return assert.Equal(t, expected, a)
}

// AssertEqualWriteError asserts that actual is a WriteException containing exactly one expected error (ignoring the Raw part).
func AssertEqualWriteError(t testing.TB, expected mongo.WriteError, actual error) bool {
	t.Helper()

	we, ok := actual.(mongo.WriteException) //nolint:errorlint // do not inspect error chain
	if !ok {
		return assert.Equal(t, expected, actual)
	}

	if len(we.WriteErrors) != 1 {
		return assert.Equal(t, expected, actual)
	}

	a := we.WriteErrors[0]

	// set expected fields that might be helpful in the test output
	require.Nil(t, expected.Raw)
	expected.Raw = a.Raw

	return assert.Equal(t, expected, a)
}

// AssertMatchesError asserts that both errors are of same type and
// are equal in value, except the message and Raw part.
func AssertMatchesError(t testing.TB, expected, actual error) {
	t.Helper()

	switch expected := expected.(type) { //nolint:errorlint // do not inspect error chain
	case mongo.CommandError:
		AssertMatchesCommandError(t, expected, actual)
	case mongo.WriteException:
		AssertMatchesWriteError(t, expected, actual)
	case mongo.BulkWriteException:
		AssertMatchesBulkException(t, expected, actual)
	default:
		t.Fatalf("unknown error type %T, expected one of [CommandError, WriteException, BulkWriteException]", expected)
	}
}

// AssertMatchesCommandError asserts that both errors are equal CommandErrors,
// except messages (and ignoring the Raw part).
func AssertMatchesCommandError(t testing.TB, expected, actual error) {
	t.Helper()

	a, ok := actual.(mongo.CommandError) //nolint:errorlint // do not inspect error chain
	require.Truef(t, ok, "actual is %[1]T (%[1]v), not mongo.CommandError", actual)

	e, ok := expected.(mongo.CommandError) //nolint:errorlint // do not inspect error chain
	require.Truef(t, ok, "expected is %[1]T (%[1]v), not mongo.CommandError", expected)

	a.Raw = nil
	e.Raw = nil

	actualMessage := a.Message
	a.Message = e.Message

	if !AssertEqualCommandError(t, e, a) {
		t.Logf("actual message: %s", actualMessage)
	}
}

// AssertMatchesWriteError asserts that both errors are WriteExceptions containing exactly one WriteError,
// and those WriteErrors are equal, except messages (and ignoring the Raw part).
func AssertMatchesWriteError(t testing.TB, expected, actual error) {
	t.Helper()

	a, ok := actual.(mongo.WriteException) //nolint:errorlint // do not inspect error chain
	require.Truef(t, ok, "actual is %T, not mongo.WriteException", actual)
	require.Lenf(t, a.WriteErrors, 1, "actual is %v, expected one mongo.WriteError", a.WriteErrors)

	e, ok := expected.(mongo.WriteException) //nolint:errorlint // do not inspect error chain
	require.Truef(t, ok, "expected is %T, not mongo.WriteException", expected)
	require.Lenf(t, e.WriteErrors, 1, "expected is %v, expected one mongo.WriteError", e.WriteErrors)

	aErr := a.WriteErrors[0]
	eErr := e.WriteErrors[0]

	aErr.Raw = nil
	eErr.Raw = nil

	actualMessage := aErr.Message
	aErr.Message = eErr.Message

	if !AssertEqualWriteError(t, eErr, aErr) {
		t.Logf("actual message: %s", actualMessage)
	}
}

// AssertMatchesBulkException asserts that both errors are BulkWriteExceptions containing the same number of WriteErrors,
// and those WriteErrors are equal, except messages (and ignoring the Raw part).
//
// TODO https://github.com/FerretDB/FerretDB/issues/3290
func AssertMatchesBulkException(t testing.TB, expected, actual error) {
	t.Helper()

	a, ok := actual.(mongo.BulkWriteException) //nolint:errorlint // do not inspect error chain
	require.Truef(t, ok, "actual is %T, not mongo.BulkWriteException", actual)

	e, ok := expected.(mongo.BulkWriteException) //nolint:errorlint // do not inspect error chain
	require.Truef(t, ok, "expected is %T, not mongo.BulkWriteException", expected)

	if len(a.WriteErrors) != len(e.WriteErrors) {
		assert.Equal(t, expected, actual)
		return
	}

	for i, we := range a.WriteErrors {
		expectedWe := e.WriteErrors[i]

		expectedWe.Message = we.Message
		expectedWe.Raw = we.Raw

		assert.Equal(t, expectedWe, we)
	}
}

// AssertEqualAltCommandError asserts that the expected MongoDB error is the same as the actual (ignoring the Raw part);
// the alternative error message may be provided if FerretDB is unable to produce exactly the same text as MongoDB.
//
// In general, error messages should be the same. Exceptions include:
//
//   - MongoDB typos (e.g. "sortto" instead of "sort to");
//   - MongoDB values formatting (e.g. we don't want to write additional code to format
//     `{ $slice: { a: { b: 3 }, b: "string" } }` exactly the same way).
//
// In any case, the alternative error message returned by FerretDB should not mislead users.
func AssertEqualAltCommandError(t testing.TB, expected mongo.CommandError, altMessage string, actual error) bool {
	t.Helper()

	a, ok := actual.(mongo.CommandError)
	if !ok {
		return assert.Equal(t, expected, actual)
	}

	// set expected fields that might be helpful in the test output
	require.Nil(t, expected.Raw)
	expected.Raw = a.Raw

	if setup.IsMongoDB(t) || altMessage == "" {
		return assert.Equal(t, expected, a)
	}

	if assert.ObjectsAreEqual(expected, a) {
		return true
	}

	expected.Message = altMessage
	return assert.Equal(t, expected, a)
}

// AssertEqualAltWriteError asserts that the expected MongoDB error is the same as the actual;
// the alternative error message may be provided if FerretDB is unable to produce exactly the same text as MongoDB.
func AssertEqualAltWriteError(t testing.TB, expected mongo.WriteError, altMessage string, actual error) bool {
	t.Helper()

	we, ok := actual.(mongo.WriteException)
	if !ok {
		return assert.Equal(t, expected, actual)
	}

	if len(we.WriteErrors) != 1 {
		return assert.Equal(t, expected, actual)
	}

	a := we.WriteErrors[0]

	// set expected fields that might be helpful in the test output
	require.Nil(t, expected.Raw)
	expected.Raw = a.Raw

	if setup.IsMongoDB(t) || altMessage == "" {
		return assert.Equal(t, expected, a)
	}

	if assert.ObjectsAreEqual(expected, a) {
		return true
	}

	expected.Message = altMessage
	return assert.Equal(t, expected, a)
}

// UnsetRaw returns error with all Raw fields unset. It returns nil if err is nil.
//
// Error is checked using a regular type assertion; wrapped errors (errors.As) are not checked.
func UnsetRaw(t testing.TB, err error) error {
	t.Helper()

	switch err := err.(type) {
	case mongo.CommandError:
		err.Raw = nil
		return err

	case mongo.WriteException:
		if err.WriteConcernError != nil {
			err.WriteConcernError.Raw = nil
		}
		for i, we := range err.WriteErrors {
			we.Raw = nil
			err.WriteErrors[i] = we
		}
		err.Raw = nil
		return err

	case mongo.BulkWriteException:
		if err.WriteConcernError != nil {
			err.WriteConcernError.Raw = nil
		}

		for i, we := range err.WriteErrors {
			we.Raw = nil
			err.WriteErrors[i] = we
		}

		return err

	default:
		return err
	}
}

// CollectIDs returns all _id values from given documents.
//
// The order is preserved.
func CollectIDs(t testing.TB, docs []bson.D) []any {
	t.Helper()

	ids := make([]any, len(docs))
	for i, doc := range docs {
		id, ok := doc.Map()["_id"]
		require.True(t, ok)
		ids[i] = id
	}

	return ids
}

// CollectKeys returns document keys.
//
// The order is preserved.
func CollectKeys(t testing.TB, doc bson.D) []string {
	t.Helper()

	res := make([]string, len(doc))
	for i, e := range doc {
		res[i] = e.Key
	}

	return res
}

// FetchAll fetches all documents from the cursor, closing it.
func FetchAll(t testing.TB, ctx context.Context, cursor *mongo.Cursor) []bson.D {
	t.Helper()

	var res []bson.D
	err := cursor.All(ctx, &res)
	require.NoError(t, cursor.Close(ctx))
	require.NoError(t, err)
	return res
}

// FilterAll returns filtered documented from the given collection sorted by _id.
func FilterAll(t testing.TB, ctx context.Context, collection *mongo.Collection, filter bson.D) []bson.D {
	t.Helper()

	opts := options.Find().SetSort(bson.D{{"_id", 1}})
	cursor, err := collection.Find(ctx, filter, opts)
	require.NoError(t, err)

	return FetchAll(t, ctx, cursor)
}

// FindAll returns all documents from the given collection sorted by _id.
func FindAll(t testing.TB, ctx context.Context, collection *mongo.Collection) []bson.D {
	t.Helper()

	return FilterAll(t, ctx, collection, bson.D{})
}

// GenerateDocuments generates documents with _id in a range [startID, endID).
// It returns bson.A containing bson.D documents.
func GenerateDocuments(startID, endID int32) bson.A {
	var arr bson.A

	for i := startID; i < endID; i++ {
		arr = append(arr, bson.D{{"_id", i}})
	}

	return arr
}

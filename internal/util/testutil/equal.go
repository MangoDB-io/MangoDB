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

package testutil

import (
	"bytes"
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/FerretDB/wire/wirebson"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dump returns string representation for debugging.
func dump[T wirebson.Type](tb testing.TB, v T) string {
	tb.Helper()

	return wirebson.LogMessageIndent(v)
}

// dumpSlice returns string representation for debugging.
func dumpSlice[T wirebson.Type](tb testing.TB, s []T) string {
	tb.Helper()

	arr := wirebson.MakeArray(len(s))

	for _, v := range s {
		err := arr.Add(v)
		require.NoError(tb, err)
	}

	return wirebson.LogMessageIndent(arr)
}

// AssertEqual asserts that two BSON values are equal.
func AssertEqual[T wirebson.Type](tb testing.TB, expected, actual T) bool {
	tb.Helper()

	if equal(tb, expected, actual) {
		return true
	}

	expectedS, actualS, diff := diffValues(tb, expected, actual)
	msg := fmt.Sprintf("Not equal: \nexpected: %s\nactual  : %s\n%s", expectedS, actualS, diff)

	return assert.Fail(tb, msg)
}

// AssertEqualSlices asserts that two BSON slices are equal.
func AssertEqualSlices[T wirebson.Type](tb testing.TB, expected, actual []T) bool {
	tb.Helper()

	allEqual := len(expected) == len(actual)
	if allEqual {
		for i, e := range expected {
			a := actual[i]
			if !equal(tb, e, a) {
				allEqual = false
				break
			}
		}
	}

	if allEqual {
		return true
	}

	expectedS, actualS, diff := diffSlices(tb, expected, actual)
	msg := fmt.Sprintf("Not equal: \nexpected: %s\nactual  : %s\n%s", expectedS, actualS, diff)

	return assert.Fail(tb, msg)
}

// diffValues returns a readable form of given values and the difference between them.
func diffValues[T wirebson.Type](tb testing.TB, expected, actual T) (expectedS string, actualS string, diff string) {
	expectedS = dump(tb, expected)
	actualS = dump(tb, actual)

	var err error
	diff, err = difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(expectedS),
		FromFile: "expected",
		B:        difflib.SplitLines(actualS),
		ToFile:   "actual",
		Context:  1,
	})
	require.NoError(tb, err)

	return
}

// diffSlices returns a readable form of given slices and the difference between them.
func diffSlices[T wirebson.Type](tb testing.TB, expected, actual []T) (expectedS string, actualS string, diff string) {
	expectedS = dumpSlice(tb, expected)
	actualS = dumpSlice(tb, actual)

	var err error
	diff, err = difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(expectedS),
		FromFile: "expected",
		B:        difflib.SplitLines(actualS),
		ToFile:   "actual",
		Context:  1,
	})
	require.NoError(tb, err)

	return
}

// equal compares any BSON values in a way that is useful for tests:
//   - float64 NaNs are equal to each other;
//   - float64 zero values are compared with sign (math.Copysign(0, -1) != math.Copysign(0, +1));
//   - time.Time values are compared using Equal method.
//
// This function is for tests; it should not try to convert values to different types before comparing them.
func equal(tb testing.TB, v1, v2 any) bool {
	tb.Helper()

	switch v1 := v1.(type) {
	case wirebson.AnyDocument:
		d, ok := v2.(wirebson.AnyDocument)
		if !ok {
			return false
		}

		return equalDocuments(tb, v1, d)

	case wirebson.AnyArray:
		a, ok := v2.(wirebson.AnyArray)
		if !ok {
			return false
		}

		return equalArrays(tb, v1, a)

	default:
		return equalScalars(tb, v1, v2)
	}
}

// equalDocuments compares BSON documents.
func equalDocuments(tb testing.TB, v1, v2 wirebson.AnyDocument) bool {
	tb.Helper()

	require.NotNil(tb, v1)
	require.NotNil(tb, v2)

	d1, err := v1.Decode()
	require.NoError(tb, err)

	d2, err := v2.Decode()
	require.NoError(tb, err)

	fieldNames := d1.FieldNames()
	slices.Sort(fieldNames)
	fieldNames = slices.Compact(fieldNames)
	require.Len(tb, fieldNames, len(d1.FieldNames()), "duplicate field names are not handled")

	if !slices.Equal(d1.FieldNames(), d2.FieldNames()) {
		return false
	}

	for _, n := range fieldNames {
		if !equal(tb, d1.Get(n), d2.Get(n)) {
			return false
		}
	}

	return true
}

// equalArrays compares BSON arrays.
func equalArrays(tb testing.TB, v1, v2 wirebson.AnyArray) bool {
	tb.Helper()

	require.NotNil(tb, v1)
	require.NotNil(tb, v2)

	a1, err := v1.Decode()
	require.NoError(tb, err)

	a2, err := v2.Decode()
	require.NoError(tb, err)

	l := a1.Len()
	if l != a2.Len() {
		return false
	}

	for i := range l {
		if !equal(tb, a1.Get(i), a2.Get(i)) {
			return false
		}
	}

	return true
}

// equalScalars compares BSON scalar values.
func equalScalars(tb testing.TB, v1, v2 any) bool {
	tb.Helper()

	require.NotNil(tb, v1)
	require.NotNil(tb, v2)

	switch s1 := v1.(type) {
	case float64:
		s2, ok := v2.(float64)
		if !ok {
			return false
		}

		if math.IsNaN(s1) {
			return math.IsNaN(s2)
		}

		if s1 == 0 && s2 == 0 {
			return math.Signbit(s1) == math.Signbit(s2)
		}

		return s1 == s2

	case string:
		s2, ok := v2.(string)
		if !ok {
			return false
		}

		return s1 == s2

	case wirebson.Binary:
		s2, ok := v2.(wirebson.Binary)
		if !ok {
			return false
		}

		return s1.Subtype == s2.Subtype && bytes.Equal(s1.B, s2.B)

	case wirebson.UndefinedType:
		_, ok := v2.(wirebson.UndefinedType)
		return ok

	case wirebson.ObjectID:
		s2, ok := v2.(wirebson.ObjectID)
		if !ok {
			return false
		}

		return s1 == s2

	case bool:
		s2, ok := v2.(bool)
		if !ok {
			return false
		}

		return s1 == s2

	case time.Time:
		s2, ok := v2.(time.Time)
		if !ok {
			return false
		}

		return s1.Equal(s2)

	case wirebson.NullType:
		_, ok := v2.(wirebson.NullType)
		return ok

	case wirebson.Regex:
		s2, ok := v2.(wirebson.Regex)
		if !ok {
			return false
		}

		return s1.Pattern == s2.Pattern && s1.Options == s2.Options

	case int32:
		s2, ok := v2.(int32)
		if !ok {
			return false
		}

		return s1 == s2

	case wirebson.Timestamp:
		s2, ok := v2.(wirebson.Timestamp)
		if !ok {
			return false
		}

		return s1 == s2

	case int64:
		s2, ok := v2.(int64)
		if !ok {
			return false
		}

		return s1 == s2

	case wirebson.Decimal128:
		s2, ok := v2.(wirebson.Decimal128)
		if !ok {
			return false
		}

		return s1 == s2

	default:
		tb.Fatalf("unhandled types %T, %T", v1, v2)
		panic("not reached")
	}
}

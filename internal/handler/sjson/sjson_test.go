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

package sjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/must"
	"github.com/FerretDB/FerretDB/internal/util/testutil"
	"github.com/FerretDB/FerretDB/internal/util/testutil/testtb"
	"github.com/FerretDB/FerretDB/internal/wire"
)

type testCase struct {
	name   string
	v      sjsontype
	sch    *elem
	j      string
	canonJ string // canonical form without extra object fields, zero values, etc.
	jErr   string // unwrapped
}

// assertEqualForFuzz is like assertEqual, but it also ignores the binaryType.Subtype as well
// as regexType.Options.
// We can't compare these fields since the encoded string (that can be generated by a fuzzing engine)
// does not contain this data. This would lead to two objects being reported as different.
func assertEqualForFuzz(tb testtb.TB, expected, actual any, msgAndArgs ...any) bool {
	tb.Helper()

	switch e := expected.(type) {
	case *binaryType:
		a := actual.(*binaryType)
		return assertEqual(tb, e.B, a.B, msgAndArgs...)
	case *regexType:
		a := actual.(*regexType)
		return assertEqual(tb, e.Pattern, a.Pattern, msgAndArgs...)
	}

	return assertEqual(tb, expected, actual, msgAndArgs...)
}

// assertEqual is assert.Equal that also can compare NaNs and ±0.
func assertEqual(tb testtb.TB, expected, actual any, msgAndArgs ...any) bool {
	tb.Helper()

	switch expected := expected.(type) {
	// should not be possible, check just in case
	case doubleType, float64:
		tb.Fatalf("unexpected type %[1]T: %[1]v", expected)

	case *doubleType:
		require.IsType(tb, expected, actual, msgAndArgs...)
		e := float64(*expected)
		a := float64(*actual.(*doubleType))

		if math.IsNaN(e) || math.IsNaN(a) {
			return assert.Equal(tb, math.IsNaN(e), math.IsNaN(a), msgAndArgs...)
		}

		if e == 0 && a == 0 {
			return assert.Equal(tb, math.Signbit(e), math.Signbit(a), msgAndArgs...)
		}
		// fallthrough to regular assert.Equal below
	}

	return assert.Equal(tb, expected, actual, msgAndArgs...)
}

// lastErr returns the last error in error chain.
func lastErr(err error) error {
	for {
		e := errors.Unwrap(err)
		if e == nil {
			return err
		}
		err = e
	}
}

func testJSON(t *testing.T, testCases []testCase, newFunc func() sjsontype) {
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require.NotEmpty(t, tc.name, "name should not be empty")
			require.NotEmpty(t, tc.j, "j should not be empty")

			t.Parallel()

			if tc.jErr == "" {
				var dst bytes.Buffer
				require.NoError(t, json.Compact(&dst, []byte(tc.j)))
				require.Equal(t, tc.j, dst.String(), "j should be compacted")
				if tc.canonJ != "" {
					dst.Reset()
					require.NoError(t, json.Compact(&dst, []byte(tc.canonJ)))
					require.Equal(t, tc.canonJ, dst.String(), "canonJ should be compacted")
				}
			}

			t.Run("UnmarshalJSON", func(t *testing.T) {
				t.Parallel()

				v := newFunc()
				err := unmarshalJSON(v, &tc)

				if tc.jErr == "" {
					require.NoError(t, err)
					assertEqual(t, tc.v, v)
					return
				}

				require.Error(t, err)
				require.Equal(t, tc.jErr, lastErr(err).Error())
			})

			t.Run("MarshalJSON", func(t *testing.T) {
				if tc.v == nil {
					t.Skip("v is nil")
				}

				t.Parallel()

				actualJ, err := tc.v.MarshalJSON()
				require.NoError(t, err)
				expectedJ := tc.j
				if tc.canonJ != "" {
					expectedJ = tc.canonJ
				}
				assert.Equal(t, expectedJ, string(actualJ))
			})

			t.Run("Marshal", func(t *testing.T) {
				if tc.v == nil {
					t.Skip("v is nil")
				}

				t.Parallel()

				actualJ, err := MarshalSingleValue(fromSJSON(tc.v))
				require.NoError(t, err)

				expectedJ := tc.j
				if tc.canonJ != "" {
					expectedJ = tc.canonJ
				}

				assert.Equal(t, expectedJ, string(actualJ))
			})
		})
	}
}

func isValidDocumentData(v sjsontype) bool {
	if v == nil {
		return false
	}

	// create a dummy document that wraps v so it has a chance
	// of passing a document validation
	d := must.NotFail(types.NewDocument(
		"_id", "dummy_id",
		"value", fromSJSON(v),
	))

	return d.ValidateData() == nil
}

func addRecordedFuzzDocs(f *testing.F, needDocument, needSchema bool) int {
	// TODO https://github.com/FerretDB/FerretDB/issues/3067
	records, err := wire.LoadRecords(testutil.TmpRecordsDir, 100)
	require.NoError(f, err)

	var n int

	for _, rec := range records {
		if rec.Body == nil {
			continue
		}

		var docs []*types.Document

		switch b := rec.Body.(type) {
		case *wire.OpMsg:
			doc, err := b.Document()
			require.NoError(f, err)
			docs = append(docs, doc)

		case *wire.OpQuery:
			if doc := b.Query(); doc != nil {
				docs = append(docs, doc)
			}

			if doc := b.ReturnFieldsSelector(); doc != nil {
				docs = append(docs, doc)
			}

		case *wire.OpReply:
			doc, err := b.Document()
			require.NoError(f, err)

			if doc != nil {
				docs = append(docs, doc)
			}
		}

		for _, doc := range docs {
			if doc.ValidateData() != nil {
				continue
			}

			args := make([]any, 0, 2)

			if needDocument {
				j, err := MarshalSingleValue(doc)
				require.NoError(f, err)
				args = append(args, j)
			}

			if needSchema {
				sch, err := marshalSchemaForDoc(doc)
				require.NoError(f, err)
				args = append(args, sch)
			}

			f.Add(args...)
			n++
		}
	}

	return n
}

func fuzzJSONWithFixedDocuments(f *testing.F, testCases []testCase, newFunc func() sjsontype) {
	type documentTest struct {
		value sjsontype
		j     string
	}
	documents := make([]documentTest, 0, len(testCases))

	for _, tc := range testCases {
		if tc.jErr != "" || tc.v == nil {
			continue
		}
		sch := must.NotFail(json.Marshal(tc.sch))
		f.Add(string(sch))
		documents = append(documents, documentTest{
			value: tc.v,
			j:     tc.j,
		})
	}

	if _, ok := newFunc().(*documentType); ok && !testing.Short() {
		needDocument := false
		needSchema := true
		n := addRecordedFuzzDocs(f, needDocument, needSchema)
		f.Logf("%d recorded documents were added to the seed corpus", n)
	}

	f.Fuzz(func(t *testing.T, jsch string) {
		t.Parallel()

		var sch elem
		if err := json.Unmarshal([]byte(jsch), &sch); err != nil {
			t.Skip("invalid schema")
		}

		for _, d := range documents {
			actualV := newFunc()
			tc := testCase{
				sch: &sch,
				j:   d.j,
			}
			if err := unmarshalJSON(actualV, &tc); err == nil {
				// if there was no error,
				// check that the documents match each other
				assertEqualForFuzz(t, d.value, actualV)
			}
		}
	})
}

func fuzzJSONWithFixedSchemas(f *testing.F, testCases []testCase, newFunc func() sjsontype) {
	schemas := make([]string, 0, len(testCases))

	for _, tc := range testCases {
		sch := must.NotFail(json.Marshal(tc.sch))
		schemas = append(schemas, string(sch))

		f.Add(tc.j)

		if tc.canonJ != "" {
			f.Add(tc.canonJ)
		}
	}

	if _, ok := newFunc().(*documentType); ok && !testing.Short() {
		needDocument := true
		needSchema := false
		n := addRecordedFuzzDocs(f, needDocument, needSchema)
		f.Logf("%d recorded documents were added to the seed corpus", n)
	}

	f.Fuzz(func(t *testing.T, j string) {
		t.Parallel()

		if j == "null" {
			t.Skip()
		}

		for _, jsch := range schemas {
			var sch elem
			require.NoError(t, json.Unmarshal([]byte(jsch), &sch))

			v := newFunc()
			tc := testCase{
				sch: &sch,
				j:   j,
			}
			if err := unmarshalJSON(v, &tc); err != nil {
				continue
			}
			if !isValidDocumentData(v) {
				continue
			}

			// test MarshalJSON
			var j string
			{
				b, err := v.MarshalJSON()
				require.NoError(t, err)
				j = string(b)
			}

			// test UnmarshalJSON
			{
				actualV := newFunc()
				tc := testCase{
					sch: &sch,
					j:   j,
				}
				err := unmarshalJSON(actualV, &tc)
				require.NoError(t, err)
				assertEqualForFuzz(t, v, actualV)
			}
		}
	})
}

func benchmark(b *testing.B, testCases []testCase, newFunc func() sjsontype) {
	for _, tc := range testCases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			b.Run("UnmarshalJSON", func(b *testing.B) {
				data := []byte(tc.j)
				var v sjsontype
				var err error

				b.ReportAllocs()
				b.SetBytes(int64(len(data)))
				b.ResetTimer()

				for range b.N {
					v = newFunc()
					err = unmarshalJSON(v, &tc)
				}

				b.StopTimer()

				if tc.jErr == "" {
					require.NoError(b, err)
					assertEqual(b, tc.v, v)
					return
				}

				require.Error(b, err)
				require.Equal(b, tc.jErr, lastErr(err).Error())
			})
		})
	}
}

// unmarshalJSON encapsulates type switch and calls UnmarshalJSON or UnmarshalJSONWithSchema on the given value.
func unmarshalJSON(v sjsontype, tc *testCase) error {
	var err error
	switch v := v.(type) {
	case *documentType:
		err = v.UnmarshalJSONWithSchema([]byte(tc.j), tc.sch.Schema)
	case *arrayType:
		err = v.UnmarshalJSONWithSchema([]byte(tc.j), tc.sch.Items)
	case *doubleType:
		err = v.UnmarshalJSON([]byte(tc.j))
	case *stringType:
		err = v.UnmarshalJSON([]byte(tc.j))
	case *binaryType:
		err = v.UnmarshalJSONWithSchema([]byte(tc.j), tc.sch)
	case *objectIDType:
		err = v.UnmarshalJSON([]byte(tc.j))
	case *boolType:
		err = v.UnmarshalJSON([]byte(tc.j))
	case *dateTimeType:
		err = v.UnmarshalJSON([]byte(tc.j))
	case *nullType:
		panic("not implemented")
	case *regexType:
		err = v.UnmarshalJSONWithSchema([]byte(tc.j), tc.sch)
	case *int32Type:
		err = v.UnmarshalJSON([]byte(tc.j))
	case *timestampType:
		err = v.UnmarshalJSON([]byte(tc.j))
	case *int64Type:
		err = v.UnmarshalJSON([]byte(tc.j))
	default:
		panic(fmt.Sprintf("not reached: %T", v)) // for sumtype to work
	}

	return err
}

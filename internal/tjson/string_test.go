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

package tjson

import (
	"testing"
)

var stringTestCases = []testCase{{
	name: "foo",
	v:    "foo",
	j:    `"foo"`,
	s:    stringSchema,
}, {
	name: "empty",
	v:    "",
	j:    `""`,
	s:    stringSchema,
}, {
	name: "zero",
	v:    "\x00",
	j:    `"\u0000"`,
	s:    stringSchema,
}}

func TestString(t *testing.T) {
	t.Parallel()
	testJSON(t, stringTestCases, func() tjsontype { return new(stringType) })
}

func FuzzString(f *testing.F) {
	fuzzJSON(f, stringTestCases, func() tjsontype { return new(stringType) })
}

func BenchmarkString(b *testing.B) {
	benchmark(b, stringTestCases, func() tjsontype { return new(stringType) })
}

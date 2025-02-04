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

// Package testdata provides vet tool test data.
package testdata

func testCorrect() {
	// TODO https://github.com/FerretDB/FerretDB/issues/178
}

func testIncorrectNoURL() {
	// TODO no URL // want "invalid TODO: incorrect format"
}

func testIncorrectFormat() {
	// TODO: https://github.com/FerretDB/FerretDB/issues/178 // want "invalid TODO: incorrect format"
}

// For https://github.com/github/codeql/issues/15894.
func testIncorrectDomain() {
	// TODO https://githubbcom/FerretDB/FerretDB/issues/3413 // want "invalid TODO: incorrect format"
}

func testCorrectFormatClosed() {
	// TODO https://github.com/FerretDB/FerretDB/issues/1 // want "invalid TODO: linked issue https://github.com/FerretDB/FerretDB/issues/1 is closed"
}

func testIncorrectFormatClosed() {
	// TODO: https://github.com/FerretDB/FerretDB/issues/1 // want "invalid TODO: incorrect format"
}

func testCorrectFormatNotExists() {
	// TODO https://github.com/FerretDB/FerretDB/issues/999999 // want "invalid TODO: linked issue https://github.com/FerretDB/FerretDB/issues/999999 is not found"
}

func testIncorrectFormatNotExists() {
	// TODO: https://github.com/FerretDB/FerretDB/issues/999999 // want "invalid TODO: incorrect format"
}

func testPrivateIssue() {
	// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/1
}

func testIncorrectIssueNumber() {
	// TODO https://github.com/FerretDB/FerretDB/issues/0 // want "invalid TODO: incorrect issue number"
}

func testDocumentDB() {
	// TODO https://github.com/microsoft/documentdb/issues/1 // want "invalid TODO: linked issue https://github.com/microsoft/documentdb/issues/1 is closed"
}

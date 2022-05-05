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

package shareddata

// BigNumbersData contains big numbers of different types for tests.
//
// TODO Merge into Scalars. https://github.com/FerretDB/FerretDB/issues/558
var BigNumbersData = &Values[string]{
	data: map[string]any{
		"int64-big":  int64(2 << 61),
		"double-big": float64(2 << 60),
	},
}

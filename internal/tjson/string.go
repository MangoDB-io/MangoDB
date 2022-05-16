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
	"bytes"
	"encoding/json"

	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
)

// stringType represents BSON UTF-8 string type.
type stringType string

// tjsontype implements tjsontype interface.
func (str *stringType) tjsontype() {}

var stringSchema = map[string]any{"type": "string"}

// Unmarshal implements tjsontype interface.
func (str *stringType) Unmarshal(_ map[string]any) ([]byte, error) {
	res, err := json.Marshal(string(*str))
	if err != nil {
		return nil, lazyerrors.Error(err)
	}
	return res, nil
}

// Marshal implements tjsontype interface.
func (str *stringType) Marshal(data []byte, _ map[string]any) error {
	if bytes.Equal(data, []byte("null")) {
		panic("null data")
	}

	r := bytes.NewReader(data)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var o string
	if err := dec.Decode(&o); err != nil {
		return lazyerrors.Error(err)
	}
	if err := checkConsumed(dec, r); err != nil {
		return lazyerrors.Error(err)
	}

	*str = stringType(o)
	return nil
}

// check interfaces
var (
	_ tjsontype = (*stringType)(nil)
)

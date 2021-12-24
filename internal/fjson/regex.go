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

package fjson

import (
	"bytes"
	"encoding/json"

	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
)

// Regex represents BSON Regex data type.
type Regex types.Regex

func (regex *Regex) fjsontype() {}

type regexJSON struct {
	R string `json:"$r"`
	O string `json:"o"`
}

// UnmarshalJSON implements bsontype interface.
func (regex *Regex) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		panic("null data")
	}

	r := bytes.NewReader(data)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var o regexJSON
	if err := dec.Decode(&o); err != nil {
		return err
	}
	if err := checkConsumed(dec, r); err != nil {
		return lazyerrors.Errorf("fjson.Regex.UnmarshalJSON: %s", err)
	}

	*regex = Regex{
		Pattern: o.R,
		Options: o.O,
	}
	return nil
}

// MarshalJSON implements bsontype interface.
func (regex Regex) MarshalJSON() ([]byte, error) {
	return json.Marshal(regexJSON{
		R: regex.Pattern,
		O: regex.Options,
	})
}

// check interfaces
var (
	_ fjsontype = (*Regex)(nil)
)

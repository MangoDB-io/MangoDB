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

package pjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
)

// documentType represents BSON Document type.
type documentType types.Document

// pjsontype implements pjsontype interface.
func (doc *documentType) pjsontype() {}

// UnmarshalJSON implements pjsontype interface.
func (doc *documentType) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		panic("null data")
	}

	r := bytes.NewReader(data)
	dec := json.NewDecoder(r)

	var rawMessages map[string]json.RawMessage
	if err := dec.Decode(&rawMessages); err != nil {
		return lazyerrors.Error(err)
	}

	if err := checkConsumed(dec, r); err != nil {
		return lazyerrors.Error(err)
	}

	b, ok := rawMessages["$k"]
	if !ok {
		return lazyerrors.Errorf("pjson.documentType.UnmarshalJSON: missing $k")
	}

	var keys []string
	if err := json.Unmarshal(b, &keys); err != nil {
		return lazyerrors.Error(err)
	}

	if len(keys)+1 != len(rawMessages) {
		return lazyerrors.Errorf("pjson.documentType.UnmarshalJSON: %d elements in $k, %d in total", len(keys), len(rawMessages))
	}

	td := must.NotFail(types.NewDocument())

	for _, key := range keys {
		b, ok = rawMessages[key]

		if !ok {
			return lazyerrors.Errorf("pjson.documentType.UnmarshalJSON: missing key %q", key)
		}

		v, err := Unmarshal(b)
		if err != nil {
			return lazyerrors.Error(err)
		}

		td.Set(key, v)
	}

	*doc = documentType(*td)

	return nil
}

// MarshalJSON implements pjsontype interface.
func (doc *documentType) MarshalJSON() ([]byte, error) {
	td := types.Document(*doc)

	var buf bytes.Buffer

	buf.WriteString(`{"$s":`)
	buf.Write(must.NotFail(marshalSchema(&td)))

	keys := td.Keys()

	for _, key := range keys {
		buf.WriteByte(',')

		var b []byte
		var err error

		if b, err = json.Marshal(key); err != nil {
			return nil, lazyerrors.Error(err)
		}

		buf.Write(b)
		buf.WriteByte(':')

		value, err := td.Get(key)
		if err != nil {
			return nil, lazyerrors.Error(err)
		}

		b, err = Marshal(value)
		if err != nil {
			return nil, lazyerrors.Error(err)
		}

		buf.Write(b)
	}

	buf.WriteByte('}')

	return buf.Bytes(), nil
}

// marshalSchema marshals document's schema.
func marshalSchema(td *types.Document) (json.RawMessage, error) {
	var buf bytes.Buffer

	buf.WriteString(`{"$k":`)

	keys := td.Keys()
	if keys == nil {
		keys = []string{}
	}

	b, err := json.Marshal(keys)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	buf.Write(b)

	for _, key := range keys {
		buf.WriteByte(',')

		if b, err = json.Marshal(key); err != nil {
			return nil, lazyerrors.Error(err)
		}

		buf.Write(b)
		buf.WriteByte(':')

		value := must.NotFail(td.Get(key))

		switch val := value.(type) {
		case *types.Document:
			buf.WriteString(`{"t": "object", "$s":`)

			b, err := marshalSchema(val)
			if err != nil {
				return nil, lazyerrors.Error(err)
			}

			buf.Write(b)

			buf.WriteByte('}')

		case *types.Array:
			buf.WriteString(`{"t": "array", "$s":`)

			// todo recursive schema for each element

			buf.WriteByte('}')

		case float64:
			buf.WriteString(`{"t": "double"}`)

		case string:
			buf.WriteString(`{"t": "string"}`)

		case types.Binary:
			buf.WriteString(`{"t": "binData", "s": 0}`) // todo

		case types.ObjectID:
			buf.WriteString(`{"t": "objectId"}`)

		case bool:
			buf.WriteString(`{"t": "bool"}`)

		case time.Time:
			buf.WriteString(`{"t": "date"}`)

		case types.NullType:
			buf.WriteString(`{"t": "null"}`)

		case types.Regex:
			buf.WriteString(`{"t": "regex", "o": ""}`) // todo

		case int32:
			buf.WriteString(`{"t": "int"}`)

		case types.Timestamp:
			buf.WriteString(`{"t": "timestamp"}`)

		case int64:
			buf.WriteString(`{"t": "long"}`)

		default:
			panic(fmt.Sprintf("pjson.marshalSchema: unknown type %[1]T (value %[1]q)", val))
		}

		b, err := Marshal(value)
		if err != nil {
			return nil, lazyerrors.Error(err)
		}

		buf.Write(b)
	}

	buf.WriteByte('}')

	return buf.Bytes(), nil
}

// check interfaces
var (
	_ pjsontype = (*documentType)(nil)
)

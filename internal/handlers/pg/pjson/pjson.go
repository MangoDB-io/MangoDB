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

// Package pjson provides converters from/to jsonb with some extensions for built-in and `types` types.
//
// See contributing guidelines and documentation for package `types` for details.
//
// # Mapping
//
// PJSON uses schema to map values to data types.
//
// Composite types
//
//	Alias      types package    pjson package         JSON representation
//
//	object     *types.Document  *pjson.documentType   {"<key 1>": <value 1>, "<key 2>": <value 2>, ...}
//	array      *types.Array     *pjson.arrayType      JSON array
//
// Scalar types
//
//	Alias      types package    pjson package         JSON representation
//
//	double     float64          *pjson.doubleType     {"$f": JSON number}
//	string     string           *pjson.stringType     JSON string
//	binData    types.Binary     *pjson.binaryType     {"$b": "<base 64 string>", "s": <subtype number>}
//	objectId   types.ObjectID   *pjson.objectIDType   {"$o": "<ObjectID as 24 character hex string"}
//	bool       bool             *pjson.boolType       JSON true / false values
//	date       time.Time        *pjson.dateTimeType   {"$d": milliseconds since epoch as JSON number}
//	null       types.NullType   *pjson.nullType       JSON null
//	regex      types.Regex      *pjson.regexType      {"$r": "<string without terminating 0x0>", "o": "<string without terminating 0x0>"}
//	int        int32            *pjson.int32Type      JSON number
//	timestamp  types.Timestamp  *pjson.timestampType  {"$t": "<number as string>"}
//	long       int64            *pjson.int64Type      {"$l": "<number as string>"}
//
//nolint:lll // for readability
//nolint:dupword // false positive
package pjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/AlekSi/pointer"

	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
)

// pjsontype is a type that can be marshaled from/to pjson.
type pjsontype interface {
	pjsontype() // seal for go-sumtype

	json.Marshaler
}

//go-sumtype:decl pjsontype

// checkConsumed returns error if decoder or reader have buffered or unread data.
func checkConsumed(dec *json.Decoder, r *bytes.Reader) error {
	if dr := dec.Buffered().(*bytes.Reader); dr.Len() != 0 {
		b, _ := io.ReadAll(dr)
		return lazyerrors.Errorf("%d bytes remains in the decoded: %s", dr.Len(), b)
	}

	if l := r.Len(); l != 0 {
		b, _ := io.ReadAll(r)
		return lazyerrors.Errorf("%d bytes remains in the reader: %s", l, b)
	}

	return nil
}

// fromPJSON converts pjsontype value to matching built-in or types' package value.
func fromPJSON(v pjsontype) any {
	switch v := v.(type) {
	case *documentType:
		return pointer.To(types.Document(*v))
	case *arrayType:
		return pointer.To(types.Array(*v))
	case *doubleType:
		return float64(*v)
	case *stringType:
		return string(*v)
	case *binaryType:
		return types.Binary(*v)
	case *objectIDType:
		return types.ObjectID(*v)
	case *boolType:
		return bool(*v)
	case *dateTimeType:
		return time.Time(*v)
	case *nullType:
		return types.Null
	case *regexType:
		return types.Regex(*v)
	case *int32Type:
		return int32(*v)
	case *timestampType:
		return types.Timestamp(*v)
	case *int64Type:
		return int64(*v)
	}

	panic(fmt.Sprintf("not reached: %T", v)) // for go-sumtype to work
}

// toPJSON converts built-in or types' package value to pjsontype value.
func toPJSON(v any) pjsontype {
	switch v := v.(type) {
	case *types.Document:
		return pointer.To(documentType(*v))
	case *types.Array:
		return pointer.To(arrayType(*v))
	case float64:
		return pointer.To(doubleType(v))
	case string:
		return pointer.To(stringType(v))
	case types.Binary:
		return pointer.To(binaryType(v))
	case types.ObjectID:
		return pointer.To(objectIDType(v))
	case bool:
		return pointer.To(boolType(v))
	case time.Time:
		return pointer.To(dateTimeType(v))
	case types.NullType:
		return pointer.To(nullType(v))
	case types.Regex:
		return pointer.To(regexType(v))
	case int32:
		return pointer.To(int32Type(v))
	case types.Timestamp:
		return pointer.To(timestampType(v))
	case int64:
		return pointer.To(int64Type(v))
	}

	panic(fmt.Sprintf("not reached: %T", v)) // for go-sumtype to work
}

// unmarshalDoc decodes the top-level document.
// It decodes document's schema from the `$s` field and uses it to decode the data of the document.
func unmarshalDoc(data []byte) (any, error) {
	var v map[string]json.RawMessage
	r := bytes.NewReader(data)
	dec := json.NewDecoder(r)

	err := dec.Decode(&v)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	if err := checkConsumed(dec, r); err != nil {
		return nil, lazyerrors.Error(err)
	}

	sch, ok := v["$s"]
	if !ok {
		return nil, lazyerrors.Errorf("schema is not set")
	}

	var schema schema
	err = json.Unmarshal(sch, &schema)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	delete(v, "$s")

	if len(schema.Keys) != len(v) {
		return nil, lazyerrors.Errorf("document must have the same number of keys and values (keys: %d, values: %d)", len(schema.Keys), len(v))
	}

	var d documentType
	err = d.UnmarshalJSONWithSchema(data, &schema)

	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return &d, nil
}

// UnmarshalElem decodes the given pjson-encoded data element by the given schema.
func UnmarshalElem(data []byte, sch *elem) (any, error) {
	var v json.RawMessage
	r := bytes.NewReader(data)
	dec := json.NewDecoder(r)

	err := dec.Decode(&v)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	if err := checkConsumed(dec, r); err != nil {
		return nil, lazyerrors.Error(err)
	}

	var res pjsontype

	switch sch.Type {
	case elemTypeObject:
		var d documentType
		err = d.UnmarshalJSONWithSchema(data, sch.Schema)
		res = &d
	case elemTypeArray:
		var a arrayType
		err = a.UnmarshalJSONWithSchema(data, sch.Items)
		res = &a
	case elemTypeDouble:
		var d doubleType
		err = d.UnmarshalJSON(data)
		res = &d
	case elemTypeString:
		var s stringType
		err = s.UnmarshalJSON(data)
		res = &s
	case elemTypeBinData:
		var b binaryType
		err = b.UnmarshalJSON(data)
		b.Subtype = types.BinarySubtype(sch.Subtype)
		res = &b
	case elemTypeBool:
		var b boolType
		err = b.UnmarshalJSON(data)
		res = &b
	case elemTypeDate:
		var d dateTimeType
		err = d.UnmarshalJSON(data)
		res = &d
	case elemTypeNull:
		var n nullType
		err = n.UnmarshalJSON(data)
		res = &n
	case elemTypeRegex:
		var r regexType
		err = r.UnmarshalJSON(data)
		r.Options = sch.Options
		res = &r
	case elemTypeInt:
		var i int32Type
		err = i.UnmarshalJSON(data)
		res = &i
	case elemTypeTimestamp:
		var t timestampType
		err = t.UnmarshalJSON(data)
		res = &t
	case elemTypeLong:
		var l int64Type
		err = l.UnmarshalJSON(data)
		res = &l
	default:
		return nil, lazyerrors.Errorf("pjson.unmarshalElem: unhandled type %q", sch.Type)
	}

	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return fromPJSON(res), nil
}

// Marshal encodes given built-in or types' package value into pjson.
func Marshal(v any) ([]byte, error) {
	if v == nil {
		panic("v is nil")
	}

	b, err := toPJSON(v).MarshalJSON()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return b, nil
}

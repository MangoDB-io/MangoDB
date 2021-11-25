// Copyright 2021 Baltoro OÜ.
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

package bson

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"

	"github.com/MangoDB-io/MangoDB/internal/util/lazyerrors"
)

// Int64 data type.
type Int64 int64

func (i *Int64) bsontype() {}

// ReadFrom bufio.Reader Int64 type.
func (i *Int64) ReadFrom(r *bufio.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, i); err != nil {
		return lazyerrors.Errorf("bson.Int64.ReadFrom (binary.Read): %w", err)
	}

	return nil
}

// WriteTo bufio.Writer Int64.
func (i Int64) WriteTo(w *bufio.Writer) error {
	v, err := i.MarshalBinary()
	if err != nil {
		return lazyerrors.Errorf("bson.Int64.WriteTo: %w", err)
	}

	_, err = w.Write(v)
	if err != nil {
		return lazyerrors.Errorf("bson.Int64.WriteTo: %w", err)
	}

	return nil
}

// MarshalBinary converts Int64 to byte array.
func (i Int64) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.LittleEndian, i)

	return buf.Bytes(), nil
}

type int64JSON struct {
	L int64 `json:"$l,string"`
}

// UnmarshalJSON Int64 in JSON format to byte array.
func (i *Int64) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		panic("null data")
	}

	r := bytes.NewReader(data)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var o int64JSON
	if err := dec.Decode(&o); err != nil {
		return err
	}
	if err := checkConsumed(dec, r); err != nil {
		return lazyerrors.Errorf("bson.Int64.UnmarshalJSON: %s", err)
	}

	*i = Int64(o.L)
	return nil
}

// MarshalJSON Int64 to JSON format in a byte array.
func (i Int64) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64JSON{
		L: int64(i),
	})
}

// check interfaces
var (
	_ bsontype = (*Int64)(nil)
)

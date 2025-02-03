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

import (
	"time"

	"./bson"
)

type bsonType byte

const (
	TypeDouble           = bsonType(0x01)
	TypeString           = bsonType(0x02)
	TypeEmbeddedDocument = bsonType(0x03)
	TypeArray            = bsonType(0x04)
	TypeBinary           = bsonType(0x05)
	TypeUndefined        = bsonType(0x06)
	TypeObjectID         = bsonType(0x07)
	TypeBoolean          = bsonType(0x08)
	TypeDateTime         = bsonType(0x09)
	TypeNull             = bsonType(0x0A)
	TypeRegex            = bsonType(0x0B)
	TypeDBPointer        = bsonType(0x0C)
	TypeJavaScript       = bsonType(0x0D)
	TypeSymbol           = bsonType(0x0E)
	TypeCodeWithScope    = bsonType(0x0F)
	TypeInt32            = bsonType(0x10)
	TypeTimestamp        = bsonType(0x11)
	TypeInt64            = bsonType(0x12)
	TypeDecimal128       = bsonType(0x13)
	TypeMinKey           = bsonType(0xFF)
	TypeMaxKey           = bsonType(0x7F)
)

func testCorrect(v any) {
	switch v.(type) {
	case bson.AnyDocument:
	case *bson.Document:
	case bson.RawDocument:
	case bson.AnyArray:
	case *bson.Array:
	case bson.RawArray:
	case float64, int32, int64: // multiple types
	case int8: // unexpected type
	case string:
	case bson.Binary:
	case bson.UndefinedType:
	case bson.ObjectID:
	case bool:
	case time.Time:
	case bson.NullType:
	case bson.Regex:
	case bson.Timestamp:
	}
}

func testIncorrectSimple(v any) {
	switch v.(type) { // want "Document should go before Array in the switch"
	case *bson.Array:
	case *bson.Document:
	}
}

func testIncorrectMixed(v any) {
	switch v.(type) { // want "Document should go before Time in the switch"
	case time.Time:
	case *bson.Document:
	}
}

func testIncorrectMultiple(v any) {
	switch v.(type) { // want "int32 should go before int64 in the switch"
	case float64, int64, int32:
	}
}

func testCorrectType(v bsonType) {
	switch v {
	case TypeEmbeddedDocument:
	case TypeArray:
	case TypeDouble:
	case TypeString:
	case TypeBinary:
	case TypeUndefined:
	case TypeObjectID:
	case TypeBoolean:
	case TypeDateTime:
	case TypeNull:
	case TypeRegex:
	case TypeDBPointer:
	case TypeJavaScript:
	case TypeSymbol:
	case TypeCodeWithScope:
	case TypeInt32:
	case TypeTimestamp:
	case TypeInt64:
	case TypeDecimal128:
	case TypeMinKey:
	case TypeMaxKey:
	}
}

func testIncorrectSimpleType(v bsonType) {
	switch v { // want "TypeDouble should go before TypeString in the switch"
	case TypeString:
	case TypeDouble:
	}
}

func testIncorrectMixedType(v bsonType) {
	switch v { // want "TypeEmbeddedDocument should go before TypeDateTime in the switch"
	case TypeDateTime:
	case TypeEmbeddedDocument:
	}
}

func testCorrectMultipleType(v bsonType) {
	switch v {
	case TypeArray, TypeBinary, TypeUndefined:
	}
}

func testIncorrectMultipleType(v bsonType) {
	switch v { // want "TypeBinary should go before TypeUndefined in the switch"
	case TypeArray, TypeUndefined, TypeBinary:
	}
}

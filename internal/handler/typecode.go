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

package handler

import (
	"fmt"
	"time"

	"github.com/FerretDB/wire/wirebson"
)

//go:generate ../../bin/stringer -linecomment -type typeCode

// typeCode represents BSON type codes with corresponding codes in BSON specification.
// Type code `number` is added to support MongoDB surrogate alias `number` which matches double, int and long type values.
type typeCode int32

//nolint:godot //sentence with unexported constant do not start with a capital letter
const (
	// typeCodeDouble is a double type code.
	typeCodeDouble = typeCode(1) // double
	// typeCodeString is a string type code.
	typeCodeString = typeCode(2) // string
	// typeCodeObject is an object type code.
	typeCodeObject = typeCode(3) // object
	// typeCodeArray is an array type code.
	typeCodeArray = typeCode(4) // array
	// typeCodeBinData is a binary data type code.
	typeCodeBinData = typeCode(5) // binData

	typeCodeFIXME = typeCode(6) // FIXME

	// typeCodeObjectID is an object id type code.
	typeCodeObjectID = typeCode(7) // objectId
	// typeCodeBool is a boolean type code.
	typeCodeBool = typeCode(8) // bool
	// typeCodeDate is a date type code.
	typeCodeDate = typeCode(9) // date
	// typeCodeNull is a null type code.
	typeCodeNull = typeCode(10) // null
	// typeCodeRegex is a regex type code.
	typeCodeRegex = typeCode(11) // regex
	// typeCodeInt is an int type code.
	typeCodeInt = typeCode(16) // int
	// typeCodeTimestamp is a timestamp type code.
	typeCodeTimestamp = typeCode(17) // timestamp
	// typeCodeLong is a long type code.
	typeCodeLong = typeCode(18) // long
	// typeCodeDecimal is a decimal type code.
	typeCodeDecimal = typeCode(19) // decimal

	// Not implemented.

	// typeCodeMinKey is a minKey type code.
	typeCodeMinKey = typeCode(-1) // minKey
	// typeCodeMaxKey is a maxKey type code.
	typeCodeMaxKey = typeCode(127) // maxKey

	// Not actual type code. `number` matches double, int and long.

	// typeCodeNumber is a number type code.
	typeCodeNumber = typeCode(-128) // number
)

// aliasFromType returns BSON type alias name for given value.
func aliasFromType(v any) string {
	switch v := v.(type) {
	case *wirebson.Document, wirebson.RawDocument:
		return typeCodeObject.String()
	case *wirebson.Array, wirebson.RawArray:
		return typeCodeArray.String()
	case float64:
		return typeCodeDouble.String()
	case string:
		return typeCodeString.String()
	case wirebson.Binary:
		return typeCodeBinData.String()
	case wirebson.UndefinedType:
		return typeCodeNull.String()
	case wirebson.ObjectID:
		return typeCodeObjectID.String()
	case bool:
		return typeCodeBool.String()
	case time.Time:
		return typeCodeDate.String()
	case wirebson.NullType:
		return typeCodeNull.String()
	case wirebson.Regex:
		return typeCodeRegex.String()
	case int32:
		return typeCodeInt.String()
	case wirebson.Timestamp:
		return typeCodeTimestamp.String()
	case int64:
		return typeCodeLong.String()
	case wirebson.Decimal128:
		return typeCodeDecimal.String()
	default:
		panic(fmt.Sprintf("not supported type %T", v))
	}
}

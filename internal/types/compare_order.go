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

package types

import (
	"fmt"
	"math"
	"time"
)

//go:generate ../../bin/stringer -linecomment -type compareTypeOrderResult
//go:generate ../../bin/stringer -linecomment -type numericOrderResult
//go:generate ../../bin/stringer -linecomment -type SortType

// compareTypeOrderResult represents the comparison order of data types.
type compareTypeOrderResult uint8

const (
	_ compareTypeOrderResult = iota
	nullDataType
	nanDataType
	numbersDataType
	stringDataType
	objectDataType
	arrayDataType
	binDataType
	objectIdDataType
	booleanDataType
	dateDataType
	timestampDataType
	regexDataType
)

// detectDataType returns a sequence for build-in type.
func detectDataType(value any) compareTypeOrderResult {
	switch value := value.(type) {
	case float64:
		if math.IsNaN(value) {
			return nanDataType
		}
		return numbersDataType
	case string:
		return stringDataType
	case Binary:
		return binDataType
	case ObjectID:
		return objectIdDataType
	case bool:
		return booleanDataType
	case time.Time:
		return dateDataType
	case NullType:
		return nullDataType
	case Regex:
		return regexDataType
	case int32:
		return numbersDataType
	case Timestamp:
		return timestampDataType
	case int64:
		return numbersDataType
	default:
		panic(fmt.Sprintf("value cannot be defined, value is %[1]v, data type of value is %[1]T", value))
	}
}

// numericOrderResult represents the comparison order of numbers.
type numericOrderResult uint8

const (
	_ numericOrderResult = iota
	doubleNegativeZero
	doubleDT
	int32DT
	int64DT
)

// defineNumberDataType returns a sequence for float64, int32 and int64 types.
func defineNumberDataType(value any) numericOrderResult {
	switch value := value.(type) {
	case float64:
		if value == 0 && math.Signbit(value) {
			return doubleNegativeZero
		}
		return doubleDT
	case int32:
		return int32DT
	case int64:
		return int64DT
	default:
		panic(fmt.Sprintf("defineNumberDataType: value cannot be defined, value is %[1]v, data type of value is %[1]T", value))
	}
}

// SortType represents sort type for $sort aggregation.
type SortType int8

const (
	// Ascending is used for sort in ascending order.
	Ascending SortType = 1

	// Descending is used for sort in descending order.
	Descending SortType = -1
)

// CompareOrder defines the data type for the two values and compares them.
// When the types are equal, it compares their values using Compare.
func CompareOrder(a, b any, order SortType) CompareResult {
	if a == nil {
		panic("CompareOrder: a is nil")
	}
	if b == nil {
		panic("CompareOrder: b is nil")
	}

	aType := detectDataType(a)
	bType := detectDataType(b)
	switch {
	case aType == bType:
		res := Compare(a, b)

		if res == Equal && aType == numbersDataType {
			aNumberType := defineNumberDataType(a)
			bNumberType := defineNumberDataType(b)
			switch {
			case aNumberType < bNumberType && order == Ascending:
				return Less
			case aNumberType > bNumberType && order == Ascending:
				return Greater
			case aNumberType < bNumberType && order == Descending:
				return Greater
			case aNumberType > bNumberType && order == Descending:
				return Less
			default:
				return res
			}
		}

		return res

	case aType < bType:
		return Less

	case aType > bType:
		return Greater

	default:
		panic("CompareOrder: not reached")
	}
}

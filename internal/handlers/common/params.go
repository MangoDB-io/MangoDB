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

package common

import (
	"fmt"
	"math"

	"github.com/FerretDB/FerretDB/internal/types"
)

// GetRequiredParam returns doc's value for key or protocol error for missing or invalid parameter.
func GetRequiredParam[T types.Type](doc *types.Document, key string) (T, error) {
	var zero T

	v, err := doc.Get(key)
	if err != nil {
		msg := fmt.Sprintf("required parameter %q is missing", key)
		return zero, NewErrorMsg(ErrBadValue, msg)
	}

	res, ok := v.(T)
	if !ok {
		msg := fmt.Sprintf("required parameter %q has type %T (expected %T)", key, v, zero)
		return zero, NewErrorMsg(ErrBadValue, msg)
	}

	return res, nil
}

// GetOptionalParam returns doc's value for key, default value for missing parameter, or protocol error for invalid parameter.
func GetOptionalParam[T types.Type](doc *types.Document, key string, defaultValue T) (T, error) {
	v, err := doc.Get(key)
	if err != nil {
		return defaultValue, nil
	}

	res, ok := v.(T)
	if !ok {
		msg := fmt.Sprintf("parameter %q has type %T (expected %T)", key, v, defaultValue)
		return defaultValue, NewErrorMsg(ErrBadValue, msg)
	}

	return res, nil
}

// AssertType asserts value's type, returning protocol error for unexpected types.
func AssertType[T types.Type](value any) (T, error) {
	res, ok := value.(T)
	if !ok {
		msg := fmt.Sprintf("got type %T, expected %T", value, res)
		return res, NewErrorMsg(ErrBadValue, msg)
	}

	return res, nil
}

var (
	errUnexpectedType = fmt.Errorf("GetWholeNumberParam: unexpected type")
	errNotWholeNumber = fmt.Errorf("GetWholeNumberParam: not a whole number")
)

// GetWholeNumberParam checks if the given value is int32, int64, or float64 containing a whole number,
// such as used in the limit, $size, etc.
func GetWholeNumberParam(value any) (int64, error) {
	switch value := value.(type) {
	case int32:
		return int64(value), nil
	case int64:
		return value, nil
	case float64:
		// TODO check float negative zero (math.Copysig(0, -1))
		if value != math.Trunc(value) || math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, errNotWholeNumber
		}
		return int64(value), nil
	default:
		return 0, errUnexpectedType
	}
}

// getBinaryMaskParam matches value type, returning types.Binary and error if match failed.
// Possible values are: position array ([1,3,5] == 010101), whole number value and types.Binary value.
func getBinaryMaskParam(value any) (types.Binary, error) {
	var mask types.Binary
	var err error

	switch value := value.(type) {
	case *types.Array:
		// {field: {$bitsAllClear: [position1, position2]}}
		mask, err = types.BinaryFromArray(value)
		if err != nil {
			return types.Binary{}, NewError(ErrBadValue, err)
		}

	case int32:
		// {field: {$bitsAllClear: bitmask}}
		if value < 0 {
			return types.Binary{}, NewErrorMsg(ErrFailedToParse,
				fmt.Sprintf(`Expected a positive number in: $bitsAllClear: %#v`, value),
			)
		}

		mask, err = types.BinaryFromInt(int64(value))
		if err != nil {
			return types.Binary{}, NewError(ErrBadValue, err)
		}
	case int64:
		// {field: {$bitsAllClear: bitmask}}
		if value < 0 {
			return types.Binary{}, NewErrorMsg(ErrFailedToParse,
				fmt.Sprintf(`Expected a positive number in: $bitsAllClear: %#v`, value),
			)
		}

		mask, err = types.BinaryFromInt(value)
		if err != nil {
			return types.Binary{}, NewError(ErrBadValue, err)
		}
	case float64:
		// TODO check float negative zero
		if value != math.Trunc(value) || math.IsNaN(value) || math.IsInf(value, 0) {
			return types.Binary{}, NewErrorMsg(ErrFailedToParse,
				fmt.Sprintf("Expected an integer: $bitsAllClear: %#v", value),
			)
		}
		mask, err = types.BinaryFromInt(int64(value))
		if err != nil {
			return types.Binary{}, err
		}
	case types.Binary:
		// {field: {$bitsAllClear: BinData()}}
		mask = value
	default:
		return types.Binary{}, NewErrorMsg(ErrBadValue,
			fmt.Sprintf(
				`value takes an Array, a number, or a BinData but received: $bitsAllClear: %#v`, value),
		)
	}
	return mask, nil
}

// getBinaryParam matches value type, returning types.Binary and error if match failed.
func getBinaryParam(value any) (types.Binary, error) {
	var res types.Binary
	var err error

	switch value := value.(type) {
	case int32:
		res, err = types.BinaryFromInt(int64(value))
		if err != nil {
			return types.Binary{}, err
		}
	case int64:
		res, err = types.BinaryFromInt(value)
		if err != nil {
			return types.Binary{}, err
		}
	case float64:
		// TODO check float negative zero
		if value != math.Trunc(value) || math.IsNaN(value) || math.IsInf(value, 0) {
			return types.Binary{}, errNotWholeNumber
		}
		res, err = types.BinaryFromInt(int64(value))
		if err != nil {
			return types.Binary{}, err
		}
	default:
		return types.Binary{}, NewErrorMsg(ErrBadValue, "not matched")
	}
	return res, nil
}

func getBinaryParams(fieldValue any, maskValue any) (types.Binary, types.Binary, error) {
	maskBinary, err := getBinaryMaskParam(maskValue)
	if err != nil {
		return types.Binary{}, types.Binary{}, err
	}

	fieldBinary, err := getBinaryParam(fieldValue)
	if err != nil {
		return types.Binary{}, types.Binary{}, err
	}

	if len(fieldBinary.B) != len(maskBinary.B) {
		panic("fieldBinary and maskBinary sizes should be equal")
	}

	return fieldBinary, maskBinary, nil
}

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
	"errors"
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

// GetInt64Value matches value's type returning protocol error for bad float values and unexpected results.
// valueType parameter used to generate error message.
func GetInt64Value(valueType string, value any) (int64, error) {
	var sortValue int64

	switch value := value.(type) {
	case int32:
		sortValue = int64(value)
	case int64:
		sortValue = int64(value)
	case float64:
		if value != math.Trunc(value) || math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, errors.New(fmt.Sprintf("%s must be a whole number", valueType))
		}
		sortValue = int64(value)
	default:
		return 0, errors.New(fmt.Sprintf("Illegal key in %s specification", valueType))
	}
	return sortValue, nil
}

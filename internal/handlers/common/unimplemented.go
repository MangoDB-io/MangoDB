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
	"strings"

	"go.uber.org/zap"

	"github.com/FerretDB/FerretDB/internal/types"
)

// Unimplemented returns ErrNotImplemented if doc has any of the given fields.
func Unimplemented(doc *types.Document, fields ...string) error {
	for _, field := range fields {
		if v, err := doc.Get(field); err == nil || v != nil {
			err = fmt.Errorf("%s: support for field %q is not implemented yet", doc.Command(), field)
			return NewError(ErrNotImplemented, err)
		}
	}

	return nil
}

// UnimplementedNonDefault returns ErrNotImplemented if doc has given field,
// and isDefault, called with the actual value, returns false.
func UnimplementedNonDefault(doc *types.Document, field string, isDefault func(v any) bool) error {
	v, err := doc.Get(field)
	if err != nil {
		return nil
	}

	if isDefault(v) {
		return nil
	}

	err = fmt.Errorf("%s: support for field %q with non-default value %v is not implemented yet", doc.Command(), field, v)
	return NewError(ErrNotImplemented, err)
}

// UnimplementedDot returns ErrNotImplemented if document's field contains a dot-separated path.
func UnimplementedDot(doc *types.Document, fields ...string) error {
	for _, field := range fields {
		v, err := doc.Get(field)
		if err != nil || v == nil {

			continue
		}

		reqField, ok := v.(*types.Document)
		if !ok {
			continue
		}

		for _, k := range reqField.Keys() {
			if strings.ContainsRune(k, '.') {
				err = fmt.Errorf("%s: dot notation support for field %q is not implemented yet", doc.Command(), field)
				return NewError(ErrNotImplemented, err)
			}
		}
	}

	return nil
}

// UnsupportedDot returns ErrDottedFieldName if field contains dot.
func UnsupportedDot(field string) error {
	if strings.ContainsRune(field, '.') {
		err := fmt.Errorf("can't use dot notation for: %s", field)
		return NewError(ErrDottedFieldName, err)
	}
	return nil
}

// Ignored logs a message if doc has any of the given fields.
func Ignored(doc *types.Document, l *zap.Logger, fields ...string) {
	for _, field := range fields {
		if v, err := doc.Get(field); err == nil || v != nil {
			l.Debug("ignoring field", zap.String("command", doc.Command()), zap.String("field", field))
		}
	}
}

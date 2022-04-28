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
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/exp/slices"

	"github.com/FerretDB/FerretDB/internal/util/must"
)

// Common interface with bson.Document.
//
// TODO Remove this type.
type document interface {
	Map() map[string]any
	Keys() []string
}

// Document represents BSON document.
//
// Duplicate field names are not supported.
type Document struct {
	m    map[string]any
	keys []string
}

// ConvertDocument converts bson.Document to *types.Document and validates it.
// It references the same data without copying it.
//
// TODO Remove this function.
func ConvertDocument(d document) (*Document, error) {
	if d == nil {
		panic("types.ConvertDocument: d is nil")
	}

	doc := &Document{
		m:    d.Map(),
		keys: d.Keys(),
	}

	if err := doc.validate(); err != nil {
		return doc, fmt.Errorf("types.ConvertDocument: %w", err)
	}

	return doc, nil
}

// MustConvertDocument is a ConvertDocument that panics in case of error.
//
// Deprecated: use `must.NotFail(ConvertDocument(...))` instead.
func MustConvertDocument(d document) *Document {
	return must.NotFail(ConvertDocument(d))
}

// NewDocument creates a document with the given key/value pairs.
func NewDocument(pairs ...any) (*Document, error) {
	l := len(pairs)
	if l%2 != 0 {
		return nil, fmt.Errorf("types.NewDocument: invalid number of arguments: %d", l)
	}

	if l == 0 {
		return new(Document), nil
	}

	doc := &Document{
		m:    make(map[string]any, l/2),
		keys: make([]string, 0, l/2),
	}
	for i := 0; i < l; i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("types.NewDocument: invalid key type: %T", pairs[i])
		}

		value := pairs[i+1]
		if err := doc.add(key, value); err != nil {
			return nil, fmt.Errorf("types.NewDocument: %w", err)
		}
	}

	if err := doc.validate(); err != nil {
		return nil, fmt.Errorf("types.NewDocument: %w", err)
	}

	return doc, nil
}

// MustNewDocument is a NewDocument that panics in case of error.
//
// TODO Remove this function.
//
// Deprecated: use `must.NotFail(NewDocument(...))` instead.
func MustNewDocument(pairs ...any) *Document {
	return must.NotFail(NewDocument(pairs...))
}

func (*Document) compositeType() {}

// DeepCopy returns a deep copy of this Document.
func (d *Document) DeepCopy() *Document {
	if d == nil {
		panic("types.Document.DeepCopy: nil document")
	}
	return deepCopy(d).(*Document)
}

// isValidKey returns false if key is not a valid document field key.
func isValidKey(key string) bool {
	if key == "" {
		return false
	}

	// forbid keys like $k (used by fjson representation), but allow $db (used by many commands)
	if key[0] == '$' && len(key) <= 2 {
		return false
	}

	return utf8.ValidString(key)
}

// validate checks if the document is valid.
func (d *Document) validate() error {
	if d == nil {
		panic("types.Document.validate: d is nil")
	}

	if len(d.m) != len(d.keys) {
		return fmt.Errorf("types.Document.validate: keys and values count mismatch: %d != %d", len(d.m), len(d.keys))
	}

	// TODO check that _id is not regex or array

	prevKeys := make(map[string]struct{}, len(d.keys))
	for _, key := range d.keys {
		if !isValidKey(key) {
			return fmt.Errorf("types.Document.validate: invalid key: %q", key)
		}

		value, ok := d.m[key]
		if !ok {
			return fmt.Errorf("types.Document.validate: key not found: %q", key)
		}

		if _, ok := prevKeys[key]; ok {
			return fmt.Errorf("types.Document.validate: duplicate key: %q", key)
		}
		prevKeys[key] = struct{}{}

		if err := validateValue(value); err != nil {
			return fmt.Errorf("types.Document.validate: %w", err)
		}
	}

	return nil
}

// Len returns the number of elements in the document.
//
// It returns 0 for nil Document.
func (d *Document) Len() int {
	if d == nil {
		return 0
	}
	return len(d.keys)
}

// Map returns this document as a map. Do not modify it.
//
// It returns nil for nil Document.
func (d *Document) Map() map[string]any {
	if d == nil {
		return nil
	}
	return d.m
}

// Keys returns document's keys. Do not modify it.
//
// It returns nil for nil Document.
func (d *Document) Keys() []string {
	if d == nil {
		return nil
	}
	return d.keys
}

// Command returns the first document's key. This is often used as a command name.
// It returns an empty string if document is nil or empty.
func (d *Document) Command() string {
	keys := d.Keys()
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

// add adds the value for the given key, returning error if that key is already present.
//
// As a special case, _id always becomes the first key.
func (d *Document) add(key string, value any) error {
	if _, ok := d.m[key]; ok {
		return fmt.Errorf("types.Document.add: key already present: %q", key)
	}

	if !isValidKey(key) {
		return fmt.Errorf("types.Document.add: invalid key: %q", key)
	}

	if err := validateValue(value); err != nil {
		return fmt.Errorf("types.Document.validate: %w", err)
	}

	// update keys slice
	if key == "_id" {
		// TODO check that value is not regex or array

		// ensure that _id is the first field
		d.keys = slices.Insert(d.keys, 0, key)
	} else {
		d.keys = append(d.keys, key)
	}

	d.m[key] = value

	return nil
}

// Get returns a value at the given key.
//
// The only possible error is missing key.
// Callers can rely on that and use `if err != nil` check.
func (d *Document) Get(key string) (any, error) {
	if value, ok := d.m[key]; ok {
		return value, nil
	}

	return nil, fmt.Errorf("types.Document.Get: key not found: %q", key)
}

// GetPairByPath returns key/index and value pair by path - a sequence of indexes and keys separated by dots.
func (d *Document) GetPairByPath(path string) (string, any, error) {
	return getPairByPath(d, path)
}

// Set sets the value for the given key, replacing any existing value.
//
// As a special case, _id always becomes the first key.
func (d *Document) Set(key string, value any) error {
	if !isValidKey(key) {
		return fmt.Errorf("types.Document.Set: invalid key: %q", key)
	}

	if err := validateValue(value); err != nil {
		return fmt.Errorf("types.Document.validate: %w", err)
	}

	// update keys slice
	if key == "_id" {
		// TODO check that value is not regex or array

		// ensure that _id is the first field
		if i := slices.Index(d.keys, key); i >= 0 {
			d.keys = slices.Delete(d.keys, i, i+1)
		}
		d.keys = slices.Insert(d.keys, 0, key)
	} else {
		if _, ok := d.m[key]; !ok {
			d.keys = append(d.keys, key)
		}
	}

	if d.m == nil {
		d.m = map[string]any{
			key: value,
		}
		return nil
	}

	d.m[key] = value
	return nil
}

// Remove the given key, doing nothing if the key does not exist.
func (d *Document) Remove(key string) {
	if _, ok := d.m[key]; !ok {
		return
	}

	delete(d.m, key)

	for i, k := range d.keys {
		if k == key {
			d.keys = append(d.keys[:i], d.keys[i+1:]...)
			return
		}
	}

	// should not be reached
	panic(fmt.Sprintf("types.Document.Remove: key not found: %q", key))
}

// RemoveByPath removes document by path, doing nothing if the key does not exist.
func (d *Document) RemoveByPath(path string) {
	keys := strings.Split(path, ".")

	if len(keys) == 0 {
		return
	}

	key := keys[0]
	if _, ok := d.m[key]; !ok {
		return
	}

	if len(keys) == 1 {
		d.Remove(key)
		return
	}

	key2 := keys[1]
	switch x := d.m[key].(type) {
	case *Document:
		d.Remove(key2)

	case *Array:
		i, err := strconv.Atoi(key2)
		if err != nil {
			panic("wrong path " + key2)
			// return
		}
		if i > len(x.s)-1 {
			return
		}
		x.s = append(x.s[:i], x.s[i+1:]...)
	default:
		// no path further: scalar value
	}
	return
}

// check interfaces
var (
	_ document = (*Document)(nil)
)

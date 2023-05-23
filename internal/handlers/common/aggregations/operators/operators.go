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

// Package operators provides aggregation operators.
// Operators are used in aggregation stages to filter and model data.
// This package contains all operators apart from the accumulation operators,
// which are stored and described in accumulators package.
//
// Accumulators that can be used outside of accumulation with different behaviour (like `$sum`),
// should be stored in both operators and accumulators packages.
package operators

import "github.com/FerretDB/FerretDB/internal/types"

// newOperatorFunc is a type for a function that creates a standard aggregation operator.
//
// By standard aggregation operator we mean any operator that is not accumulator.
// While accumulators perform operations on multiple documents
// (for example `$count` can count documents in each `$group` group),
// standard operators perfrom operations on single documents.
type newOperatorFunc func(expression *types.Document) (Operator, error)

// Operator is a common interface for standard aggregation operators.
type Operator interface {
	// Process document and returns the result of applying operator.
	Process(in *types.Document) (any, error)
}

// NewOperator returns operator from provided document.
// The document should look like: `{<$operator>: <operator-value>}`
func NewOperator(doc any) (Operator, error) {
	operatorDoc, ok := doc.(*types.Document)

	switch {
	case !ok:
		return nil, newOperatorError(ErrWrongType)
	case operatorDoc.Len() == 0:
		return nil, newOperatorError(ErrEmptyField)
	case operatorDoc.Len() > 1:
		return nil, newOperatorError(ErrTooManyFields)
	}

	operator := operatorDoc.Command()

	newOperator, ok := Operators[operator]
	if !ok {
		return nil, newOperatorError(ErrNotImplemented)
	}

	return newOperator(operatorDoc)
}

// Operators maps all standard aggregation operators.
var Operators = map[string]newOperatorFunc{
	// sorted alphabetically
	// TODO https://github.com/FerretDB/FerretDB/issues/2680
	"$type": newType,
	// please keep sorted alphabetically
}

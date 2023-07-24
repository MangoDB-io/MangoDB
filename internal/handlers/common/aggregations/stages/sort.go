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

package stages

import (
	"context"
	"errors"

	"github.com/FerretDB/FerretDB/internal/handlers/common"
	"github.com/FerretDB/FerretDB/internal/handlers/common/aggregations"
	"github.com/FerretDB/FerretDB/internal/handlers/commonerrors"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/iterator"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
)

// sort represents $sort stage.
type sort struct {
	fields *types.Document
	query  aggregations.AggregateQuery
}

// newSort creates a new $sort stage.
func newSort(params newStageParams) (aggregations.Stage, error) {
	fields, err := common.GetRequiredParam[*types.Document](params.stage, "$sort")
	if err != nil {
		return nil, commonerrors.NewCommandErrorMsgWithArgument(
			commonerrors.ErrSortBadExpression,
			"the $sort key specification must be an object",
			"$sort (stage)",
		)
	}

	if fields.Len() == 0 {
		return nil, commonerrors.NewCommandErrorMsgWithArgument(
			commonerrors.ErrSortMissingKey,
			"$sort stage must have at least one sort key",
			"$sort (stage)",
		)
	}

	// TODO: https://github.com/FerretDB/FerretDB/issues/2090

	return &sort{
		fields: fields,
		query:  params.query,
	}, nil
}

// FetchDocuments implements Stage interface.
func (s *sort) FetchDocuments(ctx context.Context, closer *iterator.MultiCloser) (types.DocumentsIterator, error) {
	return s.query.QueryDocuments(ctx, closer)
}

// Process implements Stage interface.
//
// If sort path is invalid, it returns a possibly wrapped types.DocumentPathError.
func (s *sort) Process(ctx context.Context, iter types.DocumentsIterator, closer *iterator.MultiCloser) (types.DocumentsIterator, error) { //nolint:lll // for readability
	var err error
	if iter, err = common.SortIterator(iter, closer, s.fields); err != nil {
		var pathErr *types.DocumentPathError
		if errors.As(err, &pathErr) && pathErr.Code() == types.ErrDocumentPathEmptyKey {
			return nil, commonerrors.NewCommandErrorMsgWithArgument(
				commonerrors.ErrPathContainsEmptyElement,
				"FieldPath field names may not be empty strings.",
				"$sort (stage)",
			)
		}

		return nil, lazyerrors.Error(err)
	}

	return iter, nil
}

// check interfaces
var (
	_ aggregations.Stage = (*sort)(nil)
)

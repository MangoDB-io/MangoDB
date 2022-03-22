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
	"math"
	"sort"
	"strings"

	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
)

// sortType represents sort type for $sort aggregation.
type sortType int

const (
	ascending sortType = iota
	descending
)

// SortDocuments sorts given documents in place according to the given sorting conditions.
func SortDocuments(docs []*types.Document, sort *types.Document) error {
	if sort.Len() == 0 {
		return nil
	}

	if sort.Len() > 32 {
		return lazyerrors.Errorf("maximum sort keys exceeded: %v", sort.Len())
	}

	sortFuncs := make([]sortFunc, len(sort.Keys()))
	for i, sortKey := range sort.Keys() {
		sortField, err := sort.Get(sortKey)
		if err != nil {
			return err
		}
		sortType, err := getSortType(sortField)
		if err != nil {
			return err
		}

		sortKey := sortKey
		sortFuncs[i] = func(a, b *types.Document) bool {
			sortType := sortType

			aField, err := a.Get(sortKey)
			if err != nil {
				return false
			}

			bField, err := b.Get(sortKey)
			if err != nil {
				return false
			}

			switch aField.(type) {
			case string:
				aField, err := AssertType[string](aField)
				if err != nil {
					return false
				}
				bField, err := AssertType[string](bField)
				if err != nil {
					return false
				}

				return strings.Compare(aField, bField) == -1
			default:
				result := compareScalars(aField, bField)
				return matchSortResult(sortType, result)
			}
		}
	}

	sorter := &docsSorter{docs: docs, sorts: sortFuncs}
	sorter.Sort(docs)

	return nil
}

type sortFunc func(a, b *types.Document) bool

type docsSorter struct {
	docs  []*types.Document
	sorts []sortFunc
}

func (ds *docsSorter) Sort(docs []*types.Document) {
	ds.docs = docs
	sort.Sort(ds)
}

func (ds *docsSorter) Len() int {
	return len(ds.docs)
}

func (ds *docsSorter) Swap(i, j int) {
	ds.docs[i], ds.docs[j] = ds.docs[j], ds.docs[i]
}

func (ds *docsSorter) Less(i, j int) bool {
	p, q := ds.docs[i], ds.docs[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ds.sorts)-1; k++ {
		sortFunc := ds.sorts[k]
		switch {
		case sortFunc(p, q):
			// p < q, so we have a decision.
			return true
		case sortFunc(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return ds.sorts[k](p, q)
}

// getSortType determines sortType from input sort value.
func getSortType(value any) (sortType, error) {
	var sortValue int

	switch value := value.(type) {
	case int32:
		sortValue = int(value)
	case int64:
		sortValue = int(value)
	case float64:
		if value != math.Trunc(value) || math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, NewErrorMsg(ErrBadValue, "$size must be a whole number")
		}
		sortValue = int(value)
	default:
		return 0, lazyerrors.New("failed to determine sort type")
	}

	switch sortValue {
	case 1:
		return ascending, nil
	case -1:
		return descending, nil
	default:
		return 0, lazyerrors.New("failed to determine sort type")
	}

}

// matchSortResult matching sort type and compare result.
func matchSortResult(sort sortType, result compareResult) bool {
	cmp := false
	switch result {
	case less:
		switch sort {
		case ascending:
			cmp = true
		case descending:
			cmp = false
		}
	case greater:
		switch sort {
		case ascending:
			cmp = false
		case descending:
			cmp = true
		}
	case notEqual, equal:
		return false // ???
	}

	return cmp
}

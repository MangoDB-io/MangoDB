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

// Package aggregations provides aggregation pipelines.
package aggregations

import (
	"context"

	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/iterator"
)

// Stage is a common interface for all aggregation stages.
type Stage interface {
	// Process applies an aggregate stage on documents from iterator.
	Process(ctx context.Context, iter types.DocumentsIterator, closer *iterator.MultiCloser) (types.DocumentsIterator, error)
}

// Aggregation is a common interface for fetching from database.
type Aggregation interface {
	// Query fetches documents from the database.
	Query(ctx context.Context, params QueryParams, closer *iterator.MultiCloser) (types.DocumentsIterator, error)

	// CollStats fetches collection statistics from the database.
	CollStats(ctx context.Context, closer *iterator.MultiCloser) (*CollStatsResult, error)
}

// QueryParams describes query parameter to fetch from the database.
type QueryParams struct {
	Filter *types.Document
	Sort   *types.Document
	Limit  int64
}

// CollStatsResult describes collection statistics retrieved from the database.
type CollStatsResult struct {
	CountObjects   int64
	CountIndexes   int64
	SizeTotal      int64
	SizeIndexes    int64
	SizeCollection int64
}

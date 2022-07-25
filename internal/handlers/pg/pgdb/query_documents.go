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

package pgdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgtype/pgxtype"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"

	"github.com/FerretDB/FerretDB/internal/fjson"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
)

const (
	// FetchedChannelBufSize is the size of the buffer of the channel that is used in QueryDocuments.
	FetchedChannelBufSize = 3
	// FetchedSliceCapacity is the capacity of the slice in FetchedDocs.
	FetchedSliceCapacity = 2
)

// FetchedDocs is a struct that contains a list of documents and an error.
// It is used in the fetched channel returned by QueryDocuments.
type FetchedDocs struct {
	Docs []*types.Document
	Err  error
}

// SQLParam represents options/parameters used for sql query.
type SQLParam struct {
	DB         string
	Collection string
	Comment    string
	Explain    bool
}

// QueryDocuments returns a channel with buffer FetchedChannelBufSize
// to fetch list of documents for given FerretDB database and collection.
//
// If an error occurs before the fetching, the error is returned immediately.
// The returned channel is always non-nil.
//
// The channel is closed when all documents are sent; the caller should always drain the channel.
// If an error occurs during fetching, the last message before closing the channel contains an error.
// Context cancellation is not considered an error.
//
// If the collection doesn't exist, fetch returns a closed channel and no error.
func (pgPool *Pool) QueryDocuments(ctx context.Context, querier pgxtype.Querier, sp SQLParam) (<-chan FetchedDocs, error) {
	fetchedChan := make(chan FetchedDocs, FetchedChannelBufSize)
	db := sp.DB
	collection := sp.Collection

	sql, err := pgPool.buildQuery(ctx, querier, sp)
	if err != nil {
		close(fetchedChan)
		if err == ErrTableNotExist {
			return fetchedChan, nil
		}
		return fetchedChan, lazyerrors.Error(err)
	}

	rows, err := querier.Query(ctx, sql)
	if err != nil {
		close(fetchedChan)
		return fetchedChan, lazyerrors.Error(err)
	}

	go func() {
		defer close(fetchedChan)
		defer rows.Close()

		err := iterateFetch(ctx, fetchedChan, rows)
		switch {
		case err == nil:
			// nothing
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			pgPool.logger.Warn(
				fmt.Sprintf("caught %v, stop fetching", err),
				zap.String("db", db), zap.String("collection", collection),
			)
		default:
			pgPool.logger.Error("exiting fetching with an error", zap.Error(err))
		}
	}()

	return fetchedChan, nil
}

// Explain returns document list with explain analyze results.
// We don't expect much documents in results here.
func (pgPool *Pool) Explain(ctx context.Context, querier pgxtype.Querier, sp SQLParam) ([]*types.Document, error) {
	sql, err := pgPool.buildQuery(ctx, querier, sp)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}
	rows, err := querier.Query(ctx, sql)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}
	var res []*types.Document
	for ctx.Err() == nil {
		if !rows.Next() {
			break
		}
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}

		var plans []map[string]any
		if err := json.Unmarshal(b, &plans); err != nil {
			return nil, err
		}

		for _, m := range plans {
			doc := new(types.Document)
			for k, mapval := range m {
				must.NoError(doc.Set(k, toInternalType(mapval)))
			}
			res = append(res, doc)
		}
	}
	return res, nil
}

// buildQuery builds query.
func (pgPool *Pool) buildQuery(ctx context.Context, querier pgxtype.Querier, sp SQLParam) (string, error) {
	db := sp.DB
	collection := sp.Collection
	comment := sp.Comment

	// Special case: check if collection exists at all
	collectionExists, err := CollectionExists(ctx, querier, db, collection)
	if err != nil {
		return "", lazyerrors.Error(err)
	}
	if !collectionExists {
		pgPool.logger.Info(
			"Collection doesn't exist, handling a case to deal with a non-existing collection (return empty list)",
			zap.String("db", db), zap.String("collection", collection),
		)
		return "", ErrTableNotExist
	}

	table, err := getTableName(ctx, querier, db, collection)
	if err != nil {
		return "", lazyerrors.Error(err)
	}

	sql := `SELECT _jsonb `
	if comment != "" {
		comment = strings.ReplaceAll(comment, "/*", "/ *")
		comment = strings.ReplaceAll(comment, "*/", "* /")

		sql += `/* ` + comment + ` */ `
	}
	sql += `FROM ` + pgx.Identifier{db, table}.Sanitize()

	if sp.Explain {
		sql = "EXPLAIN (VERBOSE true, FORMAT JSON) " + sql
	}
	return sql, nil
}

// iterateFetch iterates over the rows returned by the query and sends FetchedDocs to fetched channel.
// It returns ctx.Err() if context cancellation was received.
func iterateFetch(ctx context.Context, fetched chan FetchedDocs, rows pgx.Rows) error {
	for ctx.Err() == nil {
		var allFetched bool
		res := make([]*types.Document, 0, FetchedSliceCapacity)
		for i := 0; i < FetchedSliceCapacity; i++ {
			if !rows.Next() {
				allFetched = true
				break
			}

			var b []byte
			if err := rows.Scan(&b); err != nil {
				return writeFetched(ctx, fetched, FetchedDocs{Err: lazyerrors.Error(err)})
			}

			doc, err := fjson.Unmarshal(b)
			if err != nil {
				return writeFetched(ctx, fetched, FetchedDocs{Err: lazyerrors.Error(err)})
			}
			res = append(res, doc.(*types.Document))
		}

		if len(res) > 0 {
			if err := writeFetched(ctx, fetched, FetchedDocs{Docs: res}); err != nil {
				return err
			}
		}

		if allFetched {
			if err := rows.Err(); err != nil {
				if ferr := writeFetched(ctx, fetched, FetchedDocs{Err: lazyerrors.Error(err)}); ferr != nil {
					return ferr
				}
			}

			return nil
		}
	}

	return ctx.Err()
}

// writeFetched sends FetchedDocs to fetched channel or handles context cancellation.
// It returns ctx.Err() if context cancellation was received.
func writeFetched(ctx context.Context, fetched chan FetchedDocs, doc FetchedDocs) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case fetched <- doc:
		return nil
	}
}

// toInternalType transforms map[string]any, []any and scalars into internal type representation.
func toInternalType(v any) any {
	switch v := v.(type) {
	case map[string]any:
		m := new(types.Document)
		for k, mapval := range v {
			must.NoError(m.Set(k, toInternalType(mapval)))
		}
		return m

	case []any:
		a := new(types.Array)
		for _, arrval := range v {
			must.NoError(a.Append(toInternalType(arrval)))
		}
		return a

	case nil:
		return types.Null

	case float64,
		string,
		bool,
		time.Time,
		int32,
		int64:
		return v
	}
	panic(fmt.Sprintf("unsupported type: %T", v))
}

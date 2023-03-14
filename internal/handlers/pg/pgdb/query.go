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

	"github.com/jackc/pgx/v4"
	"golang.org/x/exp/maps"

	"github.com/FerretDB/FerretDB/internal/handlers/pg/pjson"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/iterator"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
)

// FetchedDocs is a struct that contains a list of documents and an error.
// It is used in the fetched channel returned by QueryDocuments.
type FetchedDocs struct {
	Docs []*types.Document
	Err  error
}

// QueryParams represents options/parameters used for SQL query.
type QueryParams struct {
	// Query filter for possible pushdown; may be ignored in part or entirely.
	Filter     *types.Document
	DB         string
	Collection string
	Comment    string
	Explain    bool
}

// Explain returns SQL EXPLAIN results for given query parameters.
func Explain(ctx context.Context, tx pgx.Tx, qp *QueryParams) (*types.Document, error) {
	exists, err := CollectionExists(ctx, tx, qp.DB, qp.Collection)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	if !exists {
		return nil, lazyerrors.Error(ErrTableNotExist)
	}

	table, err := getMetadata(ctx, tx, qp.DB, qp.Collection)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	var query string

	if qp.Explain {
		query = `EXPLAIN (VERBOSE true, FORMAT JSON) `
	}

	query += `SELECT _jsonb `

	if c := qp.Comment; c != "" {
		// prevent SQL injections
		c = strings.ReplaceAll(c, "/*", "/ *")
		c = strings.ReplaceAll(c, "*/", "* /")

		query += `/* ` + c + ` */ `
	}

	query += ` FROM ` + pgx.Identifier{qp.DB, table}.Sanitize()

	where, args, err := prepareWhereClause(qp.Filter)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	query += where

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}
	defer rows.Close()

	var res *types.Document

	if !rows.Next() {
		return nil, lazyerrors.Error(errors.New("no rows returned from EXPLAIN"))
	}

	var b []byte
	if err = rows.Scan(&b); err != nil {
		return nil, lazyerrors.Error(err)
	}

	var plans []map[string]any
	if err = json.Unmarshal(b, &plans); err != nil {
		return nil, lazyerrors.Error(err)
	}

	if len(plans) == 0 {
		return nil, lazyerrors.Error(errors.New("no execution plan returned"))
	}

	res = convertJSON(plans[0]).(*types.Document)

	if err = rows.Err(); err != nil {
		return nil, lazyerrors.Error(err)
	}

	return res, nil
}

// QueryDocuments returns an queryIterator to fetch documents for given SQLParams.
// If the collection doesn't exist, it returns an empty iterator and no error.
// If an error occurs, it returns nil and that error, possibly wrapped.
//
// Transaction is not closed by this function. Use iterator.WithClose if needed.
func QueryDocuments(ctx context.Context, tx pgx.Tx, qp *QueryParams) (iterator.Interface[int, *types.Document], error) {
	table, err := getMetadata(ctx, tx, qp.DB, qp.Collection)

	switch {
	case err == nil:
		// do nothing
	case errors.Is(err, ErrTableNotExist):
		return newIterator(ctx, nil), nil
	default:
		return nil, lazyerrors.Error(err)
	}

	iter, err := buildIterator(ctx, tx, &iteratorParams{
		schema:  qp.DB,
		table:   table,
		comment: qp.Comment,
		explain: qp.Explain,
		filter:  qp.Filter,
	})
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return iter, nil
}

// queryById returns the first found document by its ID from the given PostgreSQL schema and table.
// If the document is not found, it returns nil and no error.
func queryById(ctx context.Context, tx pgx.Tx, schema, table string, id any) (*types.Document, error) {
	query := `SELECT _jsonb FROM ` + pgx.Identifier{schema, table}.Sanitize()

	where, args, err := prepareWhereClause(must.NotFail(types.NewDocument("_id", id)))
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	query += where

	var b []byte
	err = tx.QueryRow(ctx, query, args...).Scan(&b)

	switch {
	case err == nil:
		// do nothing
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	default:
		return nil, lazyerrors.Error(err)
	}

	doc, err := pjson.Unmarshal(b)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return doc, nil
}

// iteratorParams contains parameters for building an iterator.
type iteratorParams struct {
	schema  string
	table   string
	comment string
	explain bool
	filter  *types.Document
}

// buildIterator returns an iterator to fetch documents for given iteratorParams.
func buildIterator(ctx context.Context, tx pgx.Tx, p *iteratorParams) (iterator.Interface[int, *types.Document], error) {
	var query string

	if p.explain {
		query = `EXPLAIN (VERBOSE true, FORMAT JSON) `
	}

	query += `SELECT _jsonb `

	if c := p.comment; c != "" {
		// prevent SQL injections
		c = strings.ReplaceAll(c, "/*", "/ *")
		c = strings.ReplaceAll(c, "*/", "* /")

		query += `/* ` + c + ` */ `
	}

	query += ` FROM ` + pgx.Identifier{p.schema, p.table}.Sanitize()

	where, args, err := prepareWhereClause(p.filter)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	query += where

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return newIterator(ctx, rows), nil
}

// prepareWhereClause adds WHERE clause with given filters to the query and returns the query and arguments.
func prepareWhereClause(sqlFilters *types.Document) (string, []any, error) {
	var filters []string
	var args []any
	var p Placeholder

	iter := sqlFilters.Iterator()
	defer iter.Close()

	// iterate through root document
	for {
		rootKey, rootVal, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.ErrIteratorDone) {
				break
			}

			return "", nil, lazyerrors.Error(err)
		}

		// don't pushdown $comment, it's attached to query in handlers
		if strings.HasPrefix(rootKey, "$") {
			continue
		}

		path, err := types.NewPathFromString(rootKey)

		var pe *types.DocumentPathError

		switch {
		case err == nil:
			// TODO dot notation https://github.com/FerretDB/FerretDB/issues/2069
			if path.Len() > 1 {
				continue
			}
		case errors.As(err, &pe):
			// ignore empty key error, otherwise return error
			if pe.Code() != types.ErrDocumentPathEmptyKey {
				return "", nil, lazyerrors.Error(err)
			}
		default:
			panic("Invalid error type: DocumentPathError expected ")
		}

		switch v := rootVal.(type) {
		case *types.Document:
			iter := v.Iterator()
			defer iter.Close()

			// iterate through subdocument, as it may contain operators
			for {
				k, v, err := iter.Next()
				if err != nil {
					if errors.Is(err, iterator.ErrIteratorDone) {
						break
					}

					return "", nil, lazyerrors.Error(err)
				}

				switch k {
				case "$eq":
					switch v := v.(type) {
					case *types.Document, *types.Array, types.Binary,
						types.NullType, types.Regex, types.Timestamp:
						// type not supported for pushdown
					case float64:
						sql := `_jsonb->%[1]s @> %[2]s`

						if v > 2<<53 {
							sql = `_jsonb->%[1]s > %[2]s`
							v = 2 << 53
						}

						if v < -(2 << 53) {
							sql = `_jsonb->%[1]s < %[2]s`
							v = -(2 << 53)
						}

						filters = append(filters, fmt.Sprintf(sql, p.Next(), p.Next()))
						args = append(args, rootKey, string(must.NotFail(pjson.MarshalSingleValue(v))))

					case int64:
						sql := `_jsonb->%[1]s @> %[2]s`

						if v > 2<<53 {
							sql = `_jsonb->%[1]s > %[2]s`
							v = 2 << 53
						}

						if v < -(2 << 53) {
							sql = `_jsonb->%[1]s < %[2]s`
							v = -(2 << 53)
						}

						filters = append(filters, fmt.Sprintf(sql, p.Next(), p.Next()))
						args = append(args, rootKey, string(must.NotFail(pjson.MarshalSingleValue(v))))

					case string, types.ObjectID, bool, time.Time, int32:
						// Select if value under the key is equal to provided value.
						sql := `_jsonb->%[1]s @> %[2]s`

						filters = append(filters, fmt.Sprintf(sql, p.Next(), p.Next()))
						args = append(args, rootKey, string(must.NotFail(pjson.MarshalSingleValue(v))))
					default:
						panic(fmt.Sprintf("Unexpected type of value: %v", v))
					}

				case "$ne":
					switch v := v.(type) {
					case *types.Document, *types.Array, types.Binary,
						types.NullType, types.Regex, types.Timestamp:
						// type not supported for pushdown
					case float64, string, types.ObjectID, bool, time.Time, int32, int64:
						sql := `NOT ( ` +
							// does document contain the key,
							// it is necessary, as NOT won't work correctly if the key does not exist.
							`_jsonb ? %[1]s AND ` +
							// does the value under the key is equal to filter value
							`_jsonb->%[1]s @> %[2]s AND ` +
							// does the value type is equal to the filter's one
							`_jsonb->'$s'->'p'->%[1]s->'t' = '"%[3]s"' )`

						filters = append(filters, fmt.Sprintf(sql, p.Next(), p.Next(), pjson.GetTypeOfValue(v)))
						args = append(args, rootKey, must.NotFail(pjson.MarshalSingleValue(v)))
					default:
						panic(fmt.Sprintf("Unexpected type of value: %v", v))
					}

				default:
					// TODO $gt and $lt https://github.com/FerretDB/FerretDB/issues/1875
					continue
				}
			}

		case *types.Array, types.Binary, types.NullType, types.Regex, types.Timestamp:
			// type not supported for pushdown
			continue

		case float64:
			sql := `_jsonb->%[1]s @> %[2]s`

			if v > 2<<53 {
				sql = `_jsonb->%[1]s > %[2]s`
				v = 2 << 53
			}

			if v < -(2 << 53) {
				sql = `_jsonb->%[1]s < %[2]s`
				v = -(2 << 53)
			}

			filters = append(filters, fmt.Sprintf(sql, p.Next(), p.Next()))
			args = append(args, rootKey, string(must.NotFail(pjson.MarshalSingleValue(v))))

		case int64:
			sql := `_jsonb->%[1]s @> %[2]s`

			if v > 2<<53 {
				sql = `_jsonb->%[1]s > %[2]s`
				v = 2 << 53
			}

			if v < -(2 << 53) {
				sql = `_jsonb->%[1]s < %[2]s`
				v = -(2 << 53)
			}

			filters = append(filters, fmt.Sprintf(sql, p.Next(), p.Next()))
			args = append(args, rootKey, string(must.NotFail(pjson.MarshalSingleValue(v))))

		case int32:
			// Select if value under the key is equal to provided value.

			// TODO  send simpler query for <16digits numbers

			sql := `_jsonb->%[1]s @> %[2]s OR CASE ` + // If the field or field's array contain the value, just return it (Long("2305843009213693952") == Long("2305843009213693952"))
				`WHEN _jsonb->'$s'->'p'->%[1]s->'t' = '"array"' OR ` + // If the db's row is the array ...
				`_jsonb->'$s'->'p'->%[1]s->'t' = '"double"' THEN ` + // ... or double, we compare them with rounded filter (doubles that are already in database are already rounded).
				`(_jsonb->'v')::jsonb @> to_jsonb( ROUND(%[2]s::numeric, 16 - (floor(log(abs(%[2]s::numeric)))+1)::int ) )` + // comparison with rounded filter
				`END`

			filters = append(filters, fmt.Sprintf(sql, p.Next(), p.Next()))
			args = append(args, rootKey, string(must.NotFail(pjson.MarshalSingleValue(v))))

		case string, bool, time.Time, types.ObjectID:
			// Select if value under the key is equal to provided value.
			sql := `_jsonb->%[1]s @> %[2]s`

			filters = append(filters, fmt.Sprintf(sql, p.Next(), p.Next()))
			args = append(args, rootKey, string(must.NotFail(pjson.MarshalSingleValue(v))))

		default:
			panic(fmt.Sprintf("Unexpected type of value: %v", v))
		}
	}

	var filter string
	if len(filters) > 0 {
		filter = ` WHERE ` + strings.Join(filters, " AND ")
	}

	return filter, args, nil
}

// convertJSON transforms decoded JSON map[string]any value into *types.Document.
func convertJSON(value any) any {
	switch value := value.(type) {
	case map[string]any:
		d := types.MakeDocument(len(value))
		keys := maps.Keys(value)
		for _, k := range keys {
			v := value[k]
			d.Set(k, convertJSON(v))
		}
		return d

	case []any:
		a := types.MakeArray(len(value))
		for _, v := range value {
			a.Append(convertJSON(v))
		}
		return a

	case nil:
		return types.Null

	case float64, string, bool:
		return value

	default:
		panic(fmt.Sprintf("unsupported type: %[1]T (%[1]v)", value))
	}
}

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
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"golang.org/x/exp/slices"

	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/iterator"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
)

// validateCollectionNameRe validates collection names.
var validateCollectionNameRe = regexp.MustCompile("^[a-zA-Z_-][a-zA-Z0-9_-]{0,119}$")

// Collections returns a sorted list of FerretDB collection names.
//
// It returns (possibly wrapped) ErrSchemaNotExist if FerretDB database / PostgreSQL schema does not exist.
func Collections(ctx context.Context, tx pgx.Tx, db string) ([]string, error) {
	settingsExist, err := tableExists(ctx, tx, db, settingsTableName)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	// if settings table doesn't exist, there are no collections in the database
	if !settingsExist {
		return []string{}, nil
	}

	it, err := buildIterator(ctx, tx, iteratorParams{
		schema: db,
		table:  settingsTableName,
	})
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	var collections []string

	defer it.Close()

	for {
		var doc *types.Document
		_, doc, err = it.Next()

		// if the context is canceled, we don't need to continue processing documents
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		switch {
		case err == nil:
			// do nothing
		case errors.Is(err, iterator.ErrIteratorDone):
			// no more documents
			slices.Sort(collections)
			return collections, nil
		default:
			return nil, err
		}

		collections = append(collections, must.NotFail(doc.Get("_id")).(string))
	}
}

// CollectionExists returns true if FerretDB collection exists.
func CollectionExists(ctx context.Context, tx pgx.Tx, db, collection string) (bool, error) {
	_, err := getSettings(ctx, tx, db, collection)

	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, ErrTableNotExist):
		return false, nil
	default:
		return false, lazyerrors.Error(err)
	}
}

// CreateCollection creates a new FerretDB collection with the given name in the given database.
// If the database does not exist, it will be created.
//
// It returns possibly wrapped error:
//   - ErrInvalidDatabaseName - if the given database name doesn't conform to restrictions.
//   - ErrInvalidCollectionName - if the given collection name doesn't conform to restrictions.
//   - ErrAlreadyExist - if a FerretDB collection with the given name already exists.
//   - transactionConflictError - if a PostgreSQL conflict occurs (the caller could retry the transaction).
func CreateCollection(ctx context.Context, tx pgx.Tx, db, collection string) error {
	if !validateCollectionNameRe.MatchString(collection) ||
		strings.HasPrefix(collection, reservedPrefix) {
		return ErrInvalidCollectionName
	}

	table, err := upsertSettings(ctx, tx, db, collection)
	if err != nil {
		return lazyerrors.Error(err)
	}

	err = createTableIfNotExists(ctx, tx, db, table)
	if err != nil {
		return lazyerrors.Error(err)
	}

	return nil
}

// CreateCollectionIfNotExist ensures that given FerretDB database and collection exist.
// If the database does not exist, it will be created.
//
// It returns possibly wrapped error:
//   - ErrInvalidDatabaseName - if the given database name doesn't conform to restrictions.
//   - ErrInvalidCollectionName - if the given collection name doesn't conform to restrictions.
//   - transactionConflictError - if a PostgreSQL conflict occurs (the caller could retry the transaction).
func CreateCollectionIfNotExist(ctx context.Context, tx pgx.Tx, db, collection string) error {
	if !validateCollectionNameRe.MatchString(collection) ||
		strings.HasPrefix(collection, reservedPrefix) {
		return ErrInvalidCollectionName
	}

	var err error

	table, err := upsertSettings(ctx, tx, db, collection)
	if err != nil {
		return lazyerrors.Error(err)
	}

	err = createTableIfNotExists(ctx, tx, db, table)
	if err != nil {
		return lazyerrors.Error(err)
	}

	return nil
}

// DropCollection drops FerretDB collection.
//
// It returns (possibly wrapped) ErrTableNotExist if database or collection does not exist.
// Please use errors.Is to check the error.
func DropCollection(ctx context.Context, tx pgx.Tx, db, collection string) error {
	schemaExists, err := schemaExists(ctx, tx, db)
	if err != nil {
		return lazyerrors.Error(err)
	}

	if !schemaExists {
		return ErrSchemaNotExist
	}

	table := formatCollectionName(collection)
	tables, err := tables(ctx, tx, db)
	if err != nil {
		return lazyerrors.Error(err)
	}
	if !slices.Contains(tables, table) {
		return ErrTableNotExist
	}

	err = removeSettings(ctx, tx, db, collection)
	if err != nil && !errors.Is(err, ErrTableNotExist) {
		return lazyerrors.Error(err)
	}
	if errors.Is(err, ErrTableNotExist) {
		return ErrTableNotExist
	}

	// TODO https://github.com/FerretDB/FerretDB/issues/811
	sql := `DROP TABLE IF EXISTS ` + pgx.Identifier{db, table}.Sanitize() + ` CASCADE`
	_, err = tx.Exec(ctx, sql)
	if err != nil {
		return lazyerrors.Error(err)
	}

	return nil
}

// createTableIfNotExists creates the given PostgreSQL table in the given schema if the table doesn't exist.
// If the table already exists, it does nothing.
// If PostgreSQL can't create a table due to a concurrent connection (conflict), it returns errTransactionConflict.
func createTableIfNotExists(ctx context.Context, tx pgx.Tx, schema, table string) error {
	var err error

	sql := `CREATE TABLE IF NOT EXISTS ` + pgx.Identifier{schema, table}.Sanitize() + ` (_jsonb jsonb)`
	if _, err = tx.Exec(ctx, sql); err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return lazyerrors.Error(err)
	}

	switch pgErr.Code {
	case pgerrcode.UniqueViolation, pgerrcode.DuplicateObject, pgerrcode.DuplicateTable:
		// https://www.postgresql.org/message-id/CA+TgmoZAdYVtwBfp1FL2sMZbiHCWT4UPrzRLNnX1Nb30Ku3-gg@mail.gmail.com
		// Reproducible by integration tests.
		return newTransactionConflictError(err)
	default:
		return lazyerrors.Error(err)
	}
}

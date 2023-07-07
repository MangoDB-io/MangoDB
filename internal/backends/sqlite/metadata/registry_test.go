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

package metadata

import (
	"context"
	"database/sql"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FerretDB/FerretDB/internal/util/testutil"
)

func TestInTransactionRollback(t *testing.T) {
	r, err := NewRegistry("file:"+t.TempDir()+"/", testutil.Logger(t))
	require.NoError(t, err)

	t.Run("Panic", func(t *testing.T) {
		ctx := testutil.Ctx(t)
		db, err := r.DatabaseGetOrCreate(ctx, "RollbackOnPanic")
		require.NoError(t, err)

		require.Panics(t, func() {
			err = inTransaction(ctx, db, func(tx *sql.Tx) error {
				_, err = tx.ExecContext(ctx, "CREATE TABLE test (foo TEXT)")
				require.NoError(t, err)

				rows, err := tx.QueryContext(ctx, "SELECT name FROM sqlite_schema WHERE type='table'")
				require.NoError(t, err)
				defer rows.Close()

				var tables []string

				for rows.Next() {
					var name string
					err = rows.Scan(&name)
					require.NoError(t, err)

					tables = append(tables, name)
				}

				// Check if table was actually created
				require.Equal(t, []string{"_ferretdb_collections", "test"}, tables)

				panic(nil)
			})
		})

		rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_schema WHERE type='table'")
		require.NoError(t, err)
		defer rows.Close()

		var tables []string

		for rows.Next() {
			var name string
			err = rows.Scan(&name)
			require.NoError(t, err)

			tables = append(tables, name)
		}

		assert.Equal(t, []string{"_ferretdb_collections"}, tables)
	})

	t.Run("ContextCancelled", func(t *testing.T) {
		ctx := testutil.Ctx(t)

		db, err := r.DatabaseGetOrCreate(ctx, "RollbackOnContextCancelled")
		require.NoError(t, err)

		// we need to have separate context for transaction, to access
		// database tables after context cancelation.
		txCtx, cancel := context.WithCancel(ctx)

		inTransaction(txCtx, db, func(tx *sql.Tx) error {
			_, err = tx.ExecContext(txCtx, "CREATE TABLE test (foo TEXT)")
			require.NoError(t, err)

			rows, err := tx.QueryContext(txCtx, "SELECT name FROM sqlite_schema WHERE type='table'")
			require.NoError(t, err)
			defer rows.Close()

			var tables []string

			for rows.Next() {
				var name string
				err = rows.Scan(&name)
				require.NoError(t, err)

				tables = append(tables, name)
			}

			// Check if table was actually created
			require.Equal(t, []string{"_ferretdb_collections", "test"}, tables)

			cancel()

			return nil
		})

		rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_schema WHERE type='table'")
		require.NoError(t, err)
		defer rows.Close()

		var tables []string

		for rows.Next() {
			var name string
			err = rows.Scan(&name)
			require.NoError(t, err)

			tables = append(tables, name)
		}

		assert.Equal(t, []string{"_ferretdb_collections"}, tables)
	})

	t.Run("Goexit", func(t *testing.T) {
		ctx := testutil.Ctx(t)

		db, err := r.DatabaseGetOrCreate(ctx, "RollbackOnGoexit")
		require.NoError(t, err)

		var wg sync.WaitGroup

		// We run transaction in subroutine to make sure that it'll still perform the rollback even on Goexit()
		// (which is also called from testing.FailNow())
		wg.Add(1)
		go func() {
			defer wg.Done()

			inTransaction(ctx, db, func(tx *sql.Tx) error {
				_, err = tx.ExecContext(ctx, "CREATE TABLE test (foo TEXT)")
				require.NoError(t, err)

				rows, err := tx.QueryContext(ctx, "SELECT name FROM sqlite_schema WHERE type='table'")
				require.NoError(t, err)
				defer rows.Close()

				var tables []string

				for rows.Next() {
					var name string
					err = rows.Scan(&name)
					require.NoError(t, err)

					tables = append(tables, name)
				}

				// Check if table was actually created
				require.Equal(t, []string{"_ferretdb_collections", "test"}, tables)

				runtime.Goexit()
				return nil
			})
		}()

		wg.Wait()

		rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_schema WHERE type='table'")
		require.NoError(t, err)
		defer rows.Close()

		var tables []string

		for rows.Next() {
			var name string
			err = rows.Scan(&name)
			require.NoError(t, err)

			tables = append(tables, name)
		}

		assert.Equal(t, []string{"_ferretdb_collections"}, tables)
	})
}

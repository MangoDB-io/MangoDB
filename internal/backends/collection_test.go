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

package backends_test // to avoid import cycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"github.com/FerretDB/FerretDB/internal/backends"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/must"
	"github.com/FerretDB/FerretDB/internal/util/testutil"
)

func TestCollectionUpdateAll(t *testing.T) {
	t.Parallel()

	ctx := testutil.Ctx(t)

	for _, b := range getBackends(t) {
		b := b
		t.Run(b.Name(), func(t *testing.T) {
			t.Parallel()

			t.Run("DatabaseDoesNotExist", func(t *testing.T) {
				t.Parallel()

				dbName, collName := testutil.DatabaseName(t), testutil.CollectionName(t)

				db, err := b.Database(dbName)
				require.NoError(t, err)

				coll, err := db.Collection(collName)
				require.NoError(t, err)

				updateRes, err := coll.UpdateAll(ctx, &backends.UpdateAllParams{
					Docs: []*types.Document{
						must.NotFail(types.NewDocument("_id", int32(42))),
					},
				})
				assert.NoError(t, err)
				require.NotNil(t, updateRes)
				assert.Zero(t, updateRes.Updated)

				dbRes, err := b.ListDatabases(ctx, nil)
				require.NoError(t, err)
				require.NotNil(t, dbRes)
				present := slices.ContainsFunc(dbRes.Databases, func(di backends.DatabaseInfo) bool {
					return di.Name == dbName
				})
				assert.False(t, present)

				collRes, err := db.ListCollections(ctx, nil)
				require.NoError(t, err)
				require.NotNil(t, dbRes)
				present = slices.ContainsFunc(collRes.Collections, func(ci backends.CollectionInfo) bool {
					return ci.Name == collName
				})
				assert.False(t, present)
			})
		})
	}
}

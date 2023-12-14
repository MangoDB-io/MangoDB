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

package cursors

import (
	"errors"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/FerretDB/FerretDB/integration"
	"github.com/FerretDB/FerretDB/integration/setup"
	"github.com/FerretDB/FerretDB/internal/util/testutil/testfail"
	"github.com/FerretDB/FerretDB/internal/util/testutil/teststress"
	"github.com/FerretDB/FerretDB/internal/util/testutil/testtb"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestTailableStress(t *testing.T) {
	t.Parallel()

	var tt testtb.TB = t
	if !setup.IsMongoDB(tt) {
		tt = testfail.Expected(t, "https://github.com/FerretDB/FerretDB/issues/2283")
	}

	urlOpts := url.Values{}

	s := setup.SetupWithOpts(tt, &setup.SetupOpts{
		ExtraOptions: urlOpts,
	})

	db, ctx := s.Collection.Database(), s.Ctx

	opts := options.CreateCollection().SetCapped(true).SetSizeInBytes(10000)
	err := db.CreateCollection(s.Ctx, tt.Name(), opts)
	require.NoError(tt, err)

	collection := db.Collection(tt.Name())

	bsonArr, _ := integration.GenerateDocuments(0, 50)

	_, err = collection.InsertMany(ctx, bsonArr)
	require.NoError(tt, err)

	var passed atomic.Bool
	//pass := make(chan bool)
	//go func() {
	//	<-pass
	//	return
	//}()

	teststress.StressN(t, 10, func(ready chan<- struct{}, start <-chan struct{}) {
		findCmd := bson.D{
			{"find", collection.Name()},
			{"batchSize", 1},
			{"tailable", true},
			{"noCursorTimeout", true},
		}

		var res bson.D
		err := db.RunCommand(ctx, findCmd).Decode(&res)
		require.NoError(t, err)

		firstBatch, cursorID := getFirstBatch(t, res)
		require.Equal(t, 1, firstBatch.Len())

		t.Cleanup(func() {
			killCursorCmd := bson.D{
				{"killCursors", collection.Name()},
				{"cursors", bson.A{cursorID}},
			}

			err = db.RunCommand(ctx, killCursorCmd).Decode(&res)
			require.NoError(t, err)
		})

		getMoreCmd := bson.D{
			{"getMore", cursorID},
			{"collection", collection.Name()},
			{"batchSize", 1},
		}

		ready <- struct{}{}
		<-start

		for i := 1; i < 50; i++ {
			t.Log(i)
			t.Logf("s: %d", db.Client().NumberSessionsInProgress())
			var getMoreRes bson.D
			err := db.RunCommand(ctx, getMoreCmd).Decode(&getMoreRes)

			if err != nil {
				var ce mongo.CommandError

				require.True(t, errors.As(err, &ce))
				require.Equal(t, int32(50738), ce.Code)
				require.Equal(t, "Location50738", ce.Name)

				passed.Store(true)
				return
			}
		}

		err = db.RunCommand(ctx, getMoreCmd).Decode(&res)
		require.NoError(t, err)

		nextBatch, nextID := getNextBatch(t, res)

		require.Equal(t, cursorID, nextID)
		require.Equal(t, 0, nextBatch.Len())
	})

	require.True(t, passed.Load(), "Expected cross-session cursor error in at least one goroutine")
}
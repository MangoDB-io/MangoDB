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

package integration

import (
	"errors"
	"net/url"
	"sync"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/FerretDB/FerretDB/integration/setup"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCursor(t *testing.T) {
	t.Parallel()

	// use a single connection pool
	s := setup.SetupWithOpts(t, &setup.SetupOpts{
		ExtraOptions: url.Values{
			"minPoolSize": []string{"1"},
			"maxPoolSize": []string{"1"},
		},
	})

	opts := &options.FindOptions{
		BatchSize: pointer.ToInt32(1),
	}

	ctx := s.Ctx
	collection1 := s.Collection
	databaseName := s.Collection.Database().Name()
	collectionName := s.Collection.Name()

	u, err := url.Parse(s.MongoDBURI)
	require.NoError(t, err)

	// client2 uses the same connection pool
	client2, err := mongo.Connect(ctx, options.Client().ApplyURI(u.String()))
	require.NoError(t, err)

	collection2 := client2.Database(databaseName).Collection(collectionName)

	arr, _ := generateDocuments(0, 100)
	_, err = collection2.InsertMany(ctx, arr)
	require.NoError(t, err)

	cur, err := collection2.Find(ctx, bson.D{}, opts)
	require.NoError(t, err)

	cursorID := cur.ID()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		command := bson.D{
			{"getMore", cursorID},
			{"collection", collection1.Name()},
		}

		for {
			result := bson.M{}
			err := collection1.Database().RunCommand(ctx, command).Decode(result)
			require.NoError(t, err)

			if errors.Is(err, mongo.ErrNoDocuments) {
				break
			}
		}

	}()

	err = client2.Disconnect(ctx)
	require.NoError(t, err)

	wg.Wait()
}

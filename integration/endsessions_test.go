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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/FerretDB/wire"
	"github.com/FerretDB/wire/wirebson"
	"github.com/FerretDB/wire/wireclient"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/FerretDB/FerretDB/v2/internal/util/must"
	"github.com/FerretDB/FerretDB/v2/internal/util/testutil"

	"github.com/FerretDB/FerretDB/v2/integration/setup"
)

func TestEndSessions(t *testing.T) {
	t.Parallel()

	s := setup.SetupWithOpts(t, &setup.SetupOpts{WireConn: setup.WireConnAuth})

	ctx, collection, db, conn := s.Ctx, s.Collection, s.Collection.Database(), s.WireConn
	cName, dbName := collection.Name(), db.Name()

	_, err := collection.InsertMany(ctx, bson.A{
		bson.D{{"_id", "a"}},
		bson.D{{"_id", "b"}},
	})
	require.NoError(t, err)

	// test cases are not run in parallel as they use the same conn and would cause datarace

	t.Run("OneSession", func(t *testing.T) {
		sessionID := startSession(t, ctx, conn)

		sessions := wirebson.MustArray(wirebson.MustDocument("id", sessionID))

		endSessions(t, ctx, conn, dbName, sessions, nil)
	})

	t.Run("NonExistentSession", func(t *testing.T) {
		randomUUID := must.NotFail(uuid.NewRandom())

		sessionID := wirebson.Binary{
			B:       randomUUID[:],
			Subtype: wirebson.BinaryUUID,
		}

		sessions := wirebson.MustArray(wirebson.MustDocument("id", sessionID))

		endSessions(t, ctx, conn, dbName, sessions, nil)
	})

	t.Run("MultipleSessions", func(t *testing.T) {
		sessionID1 := startSession(t, ctx, conn)
		sessionID2 := startSession(t, ctx, conn)

		sessions := wirebson.MustArray(
			wirebson.MustDocument("id", sessionID1),
			wirebson.MustDocument("id", sessionID2),
		)

		endSessions(t, ctx, conn, dbName, sessions, nil)
	})

	t.Run("CursorOfEndedSession", func(t *testing.T) {
		sessionID := startSession(t, ctx, conn)

		cursorID := find(t, ctx, conn, dbName, cName, sessionID)

		sessions := wirebson.MustArray(wirebson.MustDocument("id", sessionID))

		endSessions(t, ctx, conn, dbName, sessions, nil)

		getMore(t, ctx, conn, dbName, cName, sessionID, cursorID, nil)
	})
}

// TestEndSessionsImmediateCleanup tests ended session is deleted similarly to killed sessions.
func TestEndSessionsImmediateCleanup(t *testing.T) {
	setup.SkipForMongoDB(t, "MongoDB eventually deletes ended sessions and cannot configure how long it takes")

	t.Parallel()

	cleanupInterval := time.Millisecond

	opts := &setup.SetupOpts{
		WireConn:     setup.WireConnAuth,
		ListenerOpts: &setup.ListenerOpts{SessionCleanupInterval: cleanupInterval},
	}

	s := setup.SetupWithOpts(t, opts)

	ctx, collection, db, conn := s.Ctx, s.Collection, s.Collection.Database(), s.WireConn
	cName, dbName := collection.Name(), db.Name()

	_, err := collection.InsertMany(ctx, bson.A{
		bson.D{{"_id", "a"}},
		bson.D{{"_id", "b"}},
	})
	require.NoError(t, err)

	sessionID := startSession(t, ctx, conn)

	cursorID := find(t, ctx, conn, dbName, cName, sessionID)

	sessions := wirebson.MustArray(wirebson.MustDocument("id", sessionID))

	endSessions(t, ctx, conn, dbName, sessions, nil)

	time.Sleep(10 * cleanupInterval)

	expectedErr := must.NotFail(wirebson.NewDocument(
		"ok", float64(0),
		"errmsg", fmt.Sprintf("cursor id %d not found", cursorID),
		"code", int32(43),
		"codeName", "CursorNotFound",
	))

	getMore(t, ctx, conn, dbName, cName, sessionID, cursorID, expectedErr)
}

func TestEndSessionsErrors(t *testing.T) {
	t.Parallel()

	s := setup.SetupWithOpts(t, &setup.SetupOpts{WireConn: setup.WireConnAuth})

	ctx, db, conn := s.Ctx, s.Collection, s.WireConn
	dbName := db.Name()

	// test cases are not run in parallel as they use the same conn and would cause datarace

	t.Run("NotArray", func(t *testing.T) {
		sessions := "invalid"
		expectedErr := must.NotFail(wirebson.NewDocument(
			"ok", float64(0),
			"errmsg", "BSON field 'endSessions.endSessions' is the wrong type 'string', expected type 'array'",
			"code", int32(14),
			"codeName", "TypeMismatch",
		))

		endSessions(t, ctx, conn, dbName, sessions, expectedErr)
	})
}

func TestEndSessionsUnauthenticated(t *testing.T) {
	t.Parallel()

	s := setup.SetupWithOpts(t, &setup.SetupOpts{WireConn: setup.WireConnNoAuth})

	ctx, db, conn := s.Ctx, s.Collection, s.WireConn

	randomUUID := must.NotFail(uuid.NewRandom())
	sessionID := wirebson.Binary{B: randomUUID[:], Subtype: wirebson.BinaryUUID}

	_, resBody, err := conn.Request(ctx, wire.MustOpMsg(
		"endSessions", wirebson.MustArray(wirebson.MustDocument("id", sessionID)),
		"$db", db.Name(),
	))
	require.NoError(t, err)

	res, err := must.NotFail(resBody.(*wire.OpMsg).RawDocument()).DecodeDeep()
	require.NoError(t, err)

	fixCluster(t, res)

	expected := must.NotFail(wirebson.NewDocument(
		"ok", float64(0),
		"errmsg", "Command endSessions requires authentication",
		"code", int32(13),
		"codeName", "Unauthorized",
	))

	testutil.AssertEqual(t, expected, res)
}

// endSession sends a request to end the given session.
// If expectedErr is not nil, the error is checked, otherwise it checks the response.
func endSessions(t testing.TB, ctx context.Context, conn *wireclient.Conn, dbName string, sessions any, expectedErr *wirebson.Document) {
	msg := wire.MustOpMsg(
		"endSessions", sessions,
		"$db", dbName,
	)

	_, resBody, err := conn.Request(ctx, msg)
	require.NoError(t, err)

	res, err := must.NotFail(resBody.(*wire.OpMsg).RawDocument()).DecodeDeep()
	require.NoError(t, err)

	fixCluster(t, res)

	if expectedErr != nil {
		testutil.AssertEqual(t, expectedErr, res)

		return
	}

	expected := must.NotFail(wirebson.NewDocument(
		"ok", float64(1),
	))

	testutil.AssertEqual(t, expected, res)
}

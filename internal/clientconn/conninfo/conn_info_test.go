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

package conninfo

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/iterator"
)

func TestConnInfo(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		peerAddr string
	}{
		"EmptyPeerAddr": {
			peerAddr: "",
		},
		"NonEmptyPeerAddr": {
			peerAddr: "127.0.0.8:1234",
		},
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			connInfo := &ConnInfo{
				PeerAddr: tc.peerAddr,
			}
			ctx = WithConnInfo(ctx, connInfo)
			actual := Get(ctx)
			assert.Equal(t, connInfo, actual)
		})
	}

	// special cases: if context is not set or something wrong is set in context, it panics.
	for name, tc := range map[string]struct {
		ctx context.Context
	}{
		"EmptyContext": {
			ctx: context.Background(),
		},
		"WrongValueType": {
			ctx: context.WithValue(context.Background(), connInfoKey, "wrong value type"),
		},
		"NilValue": {
			ctx: context.WithValue(context.Background(), connInfoKey, nil),
		},
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Panics(t, func() {
				Get(tc.ctx)
			})
		})
	}
}

func TestConnInfoCursorParallelWork(t *testing.T) {
	t.Parallel()

	connInfo := NewConnInfo()

	runs := runtime.GOMAXPROCS(-1) * 10
	wg := sync.WaitGroup{}
	start := make(chan struct{})
	ready := make(chan struct{}, runs)

	// Test parallel set of cursor.
	for i := 0; i < runs; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			ready <- struct{}{}

			<-start

			array := types.MakeArray(10)
			for j := 0; j < 10; j++ {
				array.Append(fmt.Sprintf("%d:%d", i, j))
			}

			connInfo.SetCursor(fmt.Sprintf("cursor %d", i), array.Iterator())
			connInfo.Cursor(fmt.Sprintf("cursor %d", i))
		}(i)
	}

	close(start)

	wg.Wait()

	assert.Equal(t, runs, len(connInfo.cursor))

	// Test parallel read of cursor.

	start = make(chan struct{})
	ready = make(chan struct{}, runs)

	for i := 0; i < runs; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			ready <- struct{}{}

			<-start

			cursor := connInfo.Cursor(fmt.Sprintf("cursor %d", i))

			for {
				j, value, err := cursor.Next()
				if err != nil {
					if errors.Is(err, iterator.ErrIteratorDone) {
						break
					}

					panic(err)
				}

				assert.Equal(t, fmt.Sprintf("%d:%d", i, j), value)
			}
		}(i)
	}

	close(start)

	wg.Wait()

	// Test parallel read and write.

	ready = make(chan struct{}, runs)
	start = make(chan struct{})

	for i := 0; i < runs/2; i++ {
		wg.Add(2)

		go func(i int) {
			defer wg.Done()

			ready <- struct{}{}

			<-start

			array := types.MakeArray(10)
			for j := 0; j < 10; j++ {
				array.Append(fmt.Sprintf("%d:%d", i, j))
			}

			connInfo.SetCursor(fmt.Sprintf("cursor %d", i), array.Iterator())
		}(i + 1000) // avoid setting the same cursor names.

		go func(i int) {
			defer wg.Done()

			ready <- struct{}{}

			<-start

			cursor := connInfo.Cursor(fmt.Sprintf("cursor %d", i))

			for {
				j, value, err := cursor.Next()
				if err != nil {
					if errors.Is(err, iterator.ErrIteratorDone) {
						break
					}

					panic(err)
				}

				assert.Equal(t, fmt.Sprintf("%d:%d", i, j), value)
			}
		}(i)
	}

	close(start)

	wg.Wait()
}

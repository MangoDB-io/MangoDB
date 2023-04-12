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
// See the License for the specific language governing permissions and limitations under the License.

package shareddata

import (
	"sync"

	"github.com/FerretDB/FerretDB/internal/util/iterator"
	"github.com/FerretDB/FerretDB/internal/util/resource"
	"go.mongodb.org/mongo-driver/bson"
)

// newBenchmarkIterator creates iterator that iterates through bson.D documents generated
// by generator function.
// Generator should return different bson.D documents on every execution.
// To stop iterator generator must return nil.
func newBenchmarkIterator(generator func() bson.D) iterator.Interface[struct{}, bson.D] {
	iter := &benchmarkIterator{
		token:     resource.NewToken(),
		generator: generator,
	}

	resource.Track(iter, iter.token)

	return iter
}

// benchmarkIterator iterates through bson.D documents generated by provided generator function.
type benchmarkIterator struct {
	generator func() bson.D
	m         sync.Mutex
	token     *resource.Token
}

// Next implements iterator.Interface.
func (iter *benchmarkIterator) Next() (struct{}, bson.D, error) {
	iter.m.Lock()
	defer iter.m.Unlock()

	var unused struct{}

	doc := iter.generator()
	if doc == nil {
		return unused, nil, iterator.ErrIteratorDone
	}

	return unused, doc, nil
}

// Close implements iterator.Interface.
func (iter *benchmarkIterator) Close() {
	iter.m.Lock()
	defer iter.m.Unlock()

	resource.Untrack(iter, iter.token)
}

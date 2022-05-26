// Copyright 2022 FerretDB Inc.
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

package types

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"sync/atomic"
	"time"

	"github.com/FerretDB/FerretDB/internal/util/must"
)

// ObjectID represents BSON type ObjectID.
//
// Normally, it is generated by the driver, but in some cases (like upserts) FerretDB has to do itself.
type ObjectID [12]byte

// NewObjectID returns a new ObjectID.
func NewObjectID() ObjectID {
	return newObjectIDTime(time.Now())
}

// newObjectIDTime returns a new ObjectID with given time.
func newObjectIDTime(t time.Time) ObjectID {
	var res ObjectID

	binary.BigEndian.PutUint32(res[0:4], uint32(t.Unix()))
	copy(res[4:9], objectIDProcess[:])

	c := atomic.AddUint32(&objectIDCounter, 1)

	// ignore the most significant byte for correct wraparound
	res[9] = byte(c >> 16)
	res[10] = byte(c >> 8)
	res[11] = byte(c)

	return res
}

var (
	objectIDProcess [5]byte
	objectIDCounter uint32
)

func init() {
	must.NotFail(io.ReadFull(rand.Reader, objectIDProcess[:]))
	must.NoError(binary.Read(rand.Reader, binary.BigEndian, &objectIDCounter))
}

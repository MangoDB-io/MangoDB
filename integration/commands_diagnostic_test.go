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
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/FerretDB/FerretDB/integration/setup"
	"github.com/FerretDB/FerretDB/internal/util/testutil"
)

func TestCommandsDiagnosticGetLog(t *testing.T) {
	t.Parallel()
	s := setup.SetupWithOpts(t, &setup.SetupOpts{
		DatabaseName: "admin",
	})

	var actual bson.D
	err := s.TargetCollection.Database().RunCommand(s.Ctx, bson.D{{"getLog", "startupWarnings"}}).Decode(&actual)
	require.NoError(t, err)

	m := actual.Map()
	t.Log(m)

	assert.Equal(t, float64(1), m["ok"])
	assert.Equal(t, []string{"totalLinesWritten", "log", "ok"}, CollectKeys(t, actual))

	assert.IsType(t, int32(0), m["totalLinesWritten"])
}

func TestCommandsDiagnosticHostInfo(t *testing.T) {
	t.Parallel()
	ctx, collection := setup.Setup(t)

	var actual bson.D
	err := collection.Database().RunCommand(ctx, bson.D{{"hostInfo", 42}}).Decode(&actual)
	require.NoError(t, err)

	m := actual.Map()
	t.Log(m)

	assert.Equal(t, float64(1), m["ok"])
	assert.Equal(t, []string{"system", "os", "extra", "ok"}, CollectKeys(t, actual))

	os := m["os"].(bson.D)
	assert.Equal(t, []string{"type", "name", "version"}, CollectKeys(t, os))

	if runtime.GOOS == "linux" {
		require.NotEmpty(t, os.Map()["name"], "os name should not be empty")
		require.NotEmpty(t, os.Map()["version"], "os version should not be empty")
	}

	system := m["system"].(bson.D)
	keys := CollectKeys(t, system)
	assert.Contains(t, keys, "currentTime")
	assert.Contains(t, keys, "hostname")
	assert.Contains(t, keys, "cpuAddrSize")
	assert.Contains(t, keys, "numCores")
	assert.Contains(t, keys, "cpuArch")
}

func TestCommandsDiagnosticListCommands(t *testing.T) {
	t.Parallel()
	ctx, collection := setup.Setup(t)

	var actual bson.D
	err := collection.Database().RunCommand(ctx, bson.D{{"listCommands", 42}}).Decode(&actual)
	require.NoError(t, err)

	m := actual.Map()
	t.Log(m)

	assert.Equal(t, float64(1), m["ok"])
	assert.Equal(t, []string{"commands", "ok"}, CollectKeys(t, actual))

	commands := m["commands"].(bson.D)
	listCommands := commands.Map()["listCommands"].(bson.D)
	assert.NotEmpty(t, listCommands.Map()["help"].(string))
}

func TestCommandsDiagnosticConnectionStatus(t *testing.T) {
	t.Parallel()
	ctx, collection := setup.Setup(t)

	var actual bson.D
	err := collection.Database().RunCommand(ctx, bson.D{{"connectionStatus", "*"}}).Decode(&actual)
	require.NoError(t, err)

	ok := actual.Map()["ok"]

	assert.Equal(t, float64(1), ok)
}

func TestCommandsDiagnosticExplain(t *testing.T) {
	t.Parallel()
	ctx, collection := setup.Setup(t)
	err := collection.Database().CreateCollection(ctx, testutil.CollectionName(t))
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		command  any
		response int32
	}{
		"CountQueryPlanner": {
			command: bson.D{
				{
					"explain", bson.D{
						{"count", testutil.CollectionName(t)},
						{"query", bson.D{{"value", bson.D{{"$type", "array"}}}}},
					},
				},
				{"verbosity", "queryPlanner"},
			},
			response: 11,
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var actual bson.D
			err := collection.Database().RunCommand(ctx, tc.command).Decode(&actual)
			require.NoError(t, err)
			t.Logf("%#v", actual)
			// t.FailNow()
		})
	}
}

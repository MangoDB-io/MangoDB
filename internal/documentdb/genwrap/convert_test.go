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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FerretDB/FerretDB/v2/internal/util/testutil"
)

func TestCase(t *testing.T) {
	t.Parallel()

	c := converter{
		l: testutil.Logger(t),
	}

	assert.Equal(t, "cursorGetMore", c.camelCase("cursor_get_more"))
	assert.Equal(t, "CursorGetMore", c.pascalCase("cursor_get_more"))
}

func TestParameterName(t *testing.T) {
	t.Parallel()

	c := converter{
		l: testutil.Logger(t),
	}

	assert.Equal(t, "validateSpec", c.parameterName("validatespec"))
}

func TestRoutine(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}

	t.Parallel()

	ctx := testutil.Ctx(t)
	uri := "postgres://username:password@127.0.0.1:5432/postgres"

	rows, err := Extract(ctx, uri, []string{"documentdb_api"})
	require.NoError(t, err)

	c := converter{
		l: testutil.Logger(t),
	}

	expected := templateData{
		FuncName:    "AggregateCursorFirstPage",
		SQLFuncName: "documentdb_api.aggregate_cursor_first_page",
	}

	assert.Equal(t, expected, c.routine(rows["documentdb_api.aggregate_cursor_first_page_19111"]))
}

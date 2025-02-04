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

package github

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheFilePath(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	require.NoError(t, err)
	expected := filepath.Join(wd, "..", "..", "tmp", "githubcache", "cache.json")

	actual, err := CacheFilePath()
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}

	t.Parallel()

	cacheFilePath := filepath.Join(t.TempDir(), "cache.json")
	ctx := context.Background()

	t.Run("CheckIssueStatus", func(t *testing.T) {
		t.Parallel()

		c, err := NewClient(cacheFilePath, t.Logf, t.Logf, t.Logf)
		require.NoError(t, err)

		actual, err := c.checkIssueStatus(ctx, "FerretDB", "FerretDB", 10)
		require.NoError(t, err)
		assert.Equal(t, IssueOpen, actual)

		actual, err = c.checkIssueStatus(ctx, "FerretDB", "FerretDB", 1)
		require.NoError(t, err)
		assert.Equal(t, IssueClosed, actual)

		actual, err = c.checkIssueStatus(ctx, "FerretDB", "FerretDB", 999999)
		require.NoError(t, err)
		assert.Equal(t, IssueNotFound, actual)

		actual, err = c.checkIssueStatus(ctx, "microsoft", "documentdb", 1)
		require.NoError(t, err)
		assert.Equal(t, IssueClosed, actual)
	})

	t.Run("IssueStatus", func(t *testing.T) {
		t.Parallel()

		c, err := NewClient(cacheFilePath, t.Logf, t.Logf, t.Logf)
		require.NoError(t, err)

		actual, err := c.IssueStatus(ctx, "https://github.com/FerretDB/FerretDB/issues/10", "FerretDB", "FerretDB", 10)
		require.NoError(t, err)
		assert.Equal(t, IssueOpen, actual)

		actual, err = c.IssueStatus(ctx, "https://github.com/FerretDB/FerretDB/issues/1", "FerretDB", "FerretDB", 1)
		require.NoError(t, err)
		assert.Equal(t, IssueClosed, actual)

		actual, err = c.IssueStatus(ctx, "https://github.com/FerretDB/FerretDB/issues/999999", "FerretDB", "FerretDB", 999999)
		require.NoError(t, err)
		assert.Equal(t, IssueNotFound, actual)

		actual, err = c.IssueStatus(ctx, "https://github.com/microsoft/documentdb/issues/1", "microsoft", "documentdb", 1)
		require.NoError(t, err)
		assert.Equal(t, IssueClosed, actual)

		// The following tests should use cache and not the client,
		// but it may be empty if tests above failed for some reason.

		if t.Failed() {
			return
		}

		c.c = nil

		actual, err = c.IssueStatus(ctx, "https://github.com/FerretDB/FerretDB/issues/10", "FerretDB", "FerretDB", 10)
		require.NoError(t, err)
		assert.Equal(t, IssueOpen, actual)

		actual, err = c.IssueStatus(ctx, "https://github.com/FerretDB/FerretDB/issues/1", "FerretDB", "FerretDB", 1)
		require.NoError(t, err)
		assert.Equal(t, IssueClosed, actual)

		actual, err = c.IssueStatus(ctx, "https://github.com/FerretDB/FerretDB/issues/999999", "FerretDB", "FerretDB", 999999)
		require.NoError(t, err)
		assert.Equal(t, IssueNotFound, actual)

		actual, err = c.IssueStatus(ctx, "https://github.com/microsoft/documentdb/issues/1", "microsoft", "documentdb", 1)
		require.NoError(t, err)
		assert.Equal(t, IssueClosed, actual)
	})
}

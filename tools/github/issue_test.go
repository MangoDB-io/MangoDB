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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sampleIssueURL = "https://github.com/FerretDB/FerretDB/issues/1"
	issueReference = "// TODO"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("OpenIssue", func(t *testing.T) {
		openIssue := issueOpen
		msg := openIssue.Validate(issueReference, sampleIssueURL)
		require.Empty(t, msg)
	})

	t.Run("ClosedIssue", func(t *testing.T) {
		closedIssue := issueClosed
		expectedMsg := fmt.Sprintf("invalid %s linked issue %s is closed", issueReference, sampleIssueURL)

		msg := closedIssue.Validate(issueReference, sampleIssueURL)
		assert.Equal(t, expectedMsg, msg)
	})

	t.Run("NotFoundIssue", func(t *testing.T) {
		closedIssue := issueNotFound
		expectedMsg := fmt.Sprintf("invalid %s linked issue %s is not found", issueReference, sampleIssueURL)

		msg := closedIssue.Validate(issueReference, sampleIssueURL)
		assert.Equal(t, expectedMsg, msg)
	})

	t.Run("UnknownIssue", func(t *testing.T) {
		var issue issueStatus

		defer func() { _ = recover() }()
		issue.Validate(issueReference, sampleIssueURL)

		t.Errorf("should panic")
	})
}
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
	"bytes"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FerretDB/gh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FerretDB/FerretDB/tools/github"
)

func TestReal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}

	blogFiles, err := filepath.Glob(filepath.Join("..", "..", "website", "blog", "*.md"))
	if err != nil {
		t.Fatal(err)
	}

	err = checkBlogFiles(blogFiles)
	if err != nil {
		t.Fatal(err)
	}
}

var fm = bytes.TrimSpace([]byte(`
slug: using-ferretdb-with-studio-3t
date: 2023-04-18
title: Using FerretDB 1.0 with Studio 3T
authors: [alex]
description: >
	Discover how to use FerretDB 1.0 with Studio 3T, and explore ways to leverage FerretDB for MongoDB GUI applications.
image: /img/blog/ferretdb-studio3t.png
tags:
	[
		tutorial,
		mongodb compatible,
		mongodb gui,
		compatible applications,
		documents databases
	]
	`))

func TestVerifySlug(t *testing.T) {
	err := verifySlug("2023-04-18-using-ferretdb-with-studio.md", fm)
	assert.EqualError(t, err, `slug "using-ferretdb-with-studio-3t" doesn't match the file name`)
}

func TestVerifyDateNotPresent(t *testing.T) {
	err := verifyDateNotPresent(fm)
	assert.EqualError(t, err, `date field should not be present in the front matter`)
}

func TestVerifyTags(t *testing.T) {
	err := verifyTags(fm)
	assert.EqualError(t, err, `tag "documents databases" is not in the allowed list`)
}

func TestVerifyTruncateString(t *testing.T) {
	err := verifyTruncateString(fm)
	assert.EqualError(t, err, "<!--truncate--> must be included to have \"Read more\" link on the homepage")
}

func TestCheckSupportedCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}

	var buf bytes.Buffer
	l := log.New(&buf, "", 0)

	p, err := github.CacheFilePath()
	require.NoError(t, err)

	client, err := github.NewClient(p, log.Printf, gh.NoopPrintf, gh.NoopPrintf)
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		Payload        string
		ExpectedOutput string
	}{
		"OpenIssueLink": {
			Payload:        "|                 | `openIssueLink`          | ❌     | [Issue](https://github.com/FerretDB/FerretDB/issues/178) |", //nolint:lll // for readability
			ExpectedOutput: "",
		},

		"ClosedIssueLink": {
			Payload:        "|                 | `closedIssueLink`          | ❌     | [Issue](https://github.com/FerretDB/FerretDB/issues/1) |", //nolint:lll // for readability
			ExpectedOutput: "linked issue https://github.com/FerretDB/FerretDB/issues/1 is closed\n",
		},

		"AnyLabelClosedIssue": {
			Payload:        "[IssueLabel](https://github.com/FerretDB/FerretDB/issues/1)",
			ExpectedOutput: "linked issue https://github.com/FerretDB/FerretDB/issues/1 is closed\n",
		},
		"ClosedIssue": {
			Payload:        "https://github.com/FerretDB/FerretDB/issues/1",
			ExpectedOutput: "linked issue https://github.com/FerretDB/FerretDB/issues/1 is closed\n",
		},
		"IncorrectIssueNumber": {
			Payload:        "https://github.com/FerretDB/FerretDB/issues/0",
			ExpectedOutput: "incorrect issue number: https://github.com/FerretDB/FerretDB/issues/0\n",
		},
		"DocumentDBIssue": {
			Payload:        "An example issue is https://github.com/microsoft/documentdb/issues/1.",
			ExpectedOutput: "linked issue https://github.com/microsoft/documentdb/issues/1 is closed\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			buf.Reset()
			r := strings.NewReader(tc.Payload)

			var failed bool

			failed, err = checkIssueURLs(client, r, l)
			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedOutput != "", failed)

			actualOutput := buf.String()
			assert.Equal(t, tc.ExpectedOutput, actualOutput)
		})
	}
}

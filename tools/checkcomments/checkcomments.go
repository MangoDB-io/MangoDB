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

// Package main contains linter for comments.
package main

import (
	"context"
	"flag"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/FerretDB/gh"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/FerretDB/FerretDB/tools/github"
)

// todoRE represents correct "// TODO" comment format.
// It returns url, owner, repo and issue number as submatches.
var todoRE = regexp.MustCompile(`^// TODO (\Qhttps://github.com/\E(FerretDB|microsoft)/([-\w]+)/issues/(\d+))$`)

// analyzer represents the checkcomments analyzer.
var analyzer = &analysis.Analyzer{
	Name:  "checkcomments",
	Doc:   "check TODO comments",
	Run:   run,
	Flags: *flag.NewFlagSet("", flag.ExitOnError),
}

// init initializes the analyzer flags.
func init() {
	analyzer.Flags.Bool("offline", false, "do not check issues open/closed status")
	analyzer.Flags.Bool("cache-debug", false, "log cache hits/misses")
	analyzer.Flags.Bool("client-debug", false, "log GitHub API requests/responses")
	analyzer.Flags.Bool("skip-private", true, "do not check https://github.com/FerretDB/FerretDB-DocumentDB/issues/XXX URLs")
}

// main runs the analyzer.
func main() {
	singlechecker.Main(analyzer)
}

// run analyses TODO comments.
func run(pass *analysis.Pass) (any, error) {
	var client *github.Client

	if !pass.Analyzer.Flags.Lookup("offline").Value.(flag.Getter).Get().(bool) {
		p, err := github.CacheFilePath()
		if err != nil {
			log.Panic(err)
		}

		cacheDebugF := gh.NoopPrintf
		if pass.Analyzer.Flags.Lookup("cache-debug").Value.(flag.Getter).Get().(bool) {
			cacheDebugF = log.New(log.Writer(), "", log.Flags()).Printf
		}

		clientDebugF := gh.NoopPrintf
		if pass.Analyzer.Flags.Lookup("client-debug").Value.(flag.Getter).Get().(bool) {
			clientDebugF = log.New(log.Writer(), "client-debug: ", log.Flags()).Printf
		}

		if client, err = github.NewClient(p, log.Printf, cacheDebugF, clientDebugF); err != nil {
			log.Panic(err)
		}
	}

	skipPrivate := pass.Analyzer.Flags.Lookup("skip-private").Value.(flag.Getter).Get().(bool)

	for _, f := range pass.Files {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				line := c.Text

				// the space between `//` and `TODO` is always added by `task fmt`
				if !strings.HasPrefix(line, "// TODO") {
					continue
				}

				if f.Name.Name == "testdata" {
					line, _, _ = strings.Cut(line, ` // want "`)
				}

				match := todoRE.FindStringSubmatch(line)

				if len(match) != 5 {
					pass.Reportf(c.Pos(), "invalid TODO: incorrect format")
					continue
				}

				url, owner, repo := match[1], match[2], match[3]

				num, err := strconv.Atoi(match[4])
				if err != nil {
					panic(err)
				}

				if num <= 0 {
					pass.Reportf(c.Pos(), "invalid TODO: incorrect issue number")
					continue
				}

				if skipPrivate && repo == "FerretDB-DocumentDB" {
					continue
				}

				status, err := client.IssueStatus(context.TODO(), url, owner, repo, num)
				if err != nil {
					log.Panic(err)
				}

				switch status {
				case github.IssueOpen:
					// nothing
				case github.IssueClosed:
					pass.Reportf(c.Pos(), "invalid TODO: linked issue %s is closed", url)
				case github.IssueNotFound:
					pass.Reportf(c.Pos(), "invalid TODO: linked issue %s is not found", url)
				default:
					log.Panicf("unknown issue status: %s", status)
				}
			}
		}
	}

	return nil, nil
}

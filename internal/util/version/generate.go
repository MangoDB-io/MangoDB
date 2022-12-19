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

//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/FerretDB/FerretDB/internal/util/must"
)

// runGit runs `git` with given arguments and returns stdout.
func runGit(args ...string) []byte {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr

	b, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("Failed to run %q: %s", strings.Join(cmd.Args, " "), err))
	}

	return b
}

func saveFile(b []byte, filename string) {
	log.Printf("%s: %s", filename, b)
	must.NoError(os.WriteFile(filepath.Join("gen", filename), b, 0o666))
}

func main() {
	log.SetFlags(0)

	var wg sync.WaitGroup

	// git describe --tags --dirty > gen/version.txt
	wg.Add(1)
	go func() {
		defer wg.Done()

		saveFile(runGit("describe", "--tags", "--dirty"), "version.txt")
	}()

	// git rev-parse HEAD > gen/commit.txt
	wg.Add(1)
	go func() {
		defer wg.Done()

		saveFile(runGit("rev-parse", "HEAD"), "commit.txt")
	}()

	// git branch --show-current > gen/branch.txt
	wg.Add(1)
	go func() {
		defer wg.Done()

		saveFile(runGit("branch", "--show-current"), "branch.txt")
	}()

	wg.Wait()
}

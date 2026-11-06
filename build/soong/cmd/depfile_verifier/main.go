// Copyright 2025 Google Inc. All rights reserved.
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

// This tool reads a depfile and a file that contains a list of input files to an action,
// and ensures that every input in the depfile is also in the input file list. This is
// essentially a lite form of action sandboxing that can be used on tools that produce depfiles.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"android/soong/makedeps"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <depfile.d> <input_files_list_file>", os.Args[0])
		flag.PrintDefaults()
	}
	skip := flag.Bool("skip", false, "Causes this check to be skipped, always returning success. "+
		"Adding this flag is easier than doing a bash/ninja conditional in some cases.")
	checkSuffixOnly := flag.Bool("check-suffix-only", false, "If set, only checks that files in "+
		"depfile have a suffix that matches a file from the input files list. Rustc outputs some "+
		"dependencies (such as those from include! macros) with absolute paths, but the input "+
		"files are always relative paths.")
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	if skip != nil && *skip {
		return
	}

	depfileFilePath := flag.Args()[0]
	depfileFile, err := os.ReadFile(depfileFilePath)
	if err != nil {
		log.Fatalf("Error opening %q: %v", depfileFilePath, err)
	}

	deps, err := makedeps.Parse(depfileFilePath, bytes.NewBuffer(append([]byte(nil), depfileFile...)))
	if err != nil {
		log.Fatalf("Failed to parse: %v", err)
	}

	fileListFilePath := flag.Args()[1]
	fileListFile, err := os.ReadFile(fileListFilePath)
	if err != nil {
		log.Fatalf("Error opening %q: %v", fileListFilePath, err)
	}
	fileListLines := strings.Split(strings.TrimSpace(string(fileListFile)), "\n")
	expectedFiles := make(map[string]struct{})
	for _, x := range fileListLines {
		expectedFiles[x] = struct{}{}
	}

	var inputs []string
	for _, input := range deps.Inputs {
		// Proguard files sometimes use -include ../another_file.flags, and the ../ appears in
		// the depfile. Clean the path.
		input = filepath.Clean(input)
		inputs = append(inputs, input)
	}

	// The depfile may contain duplicate inputs, remove them
	slices.Sort(inputs)
	inputs = slices.Compact(inputs)

	hasFailures := false
	for _, input := range inputs {
		if checkSuffixOnly != nil && *checkSuffixOnly {
			found := false
			for _, expectedFile := range fileListLines {
				if strings.HasSuffix(input, expectedFile) {
					found = true
					break
				}
			}
			if !found {
				fmt.Fprintf(os.Stderr, "Command depended on %q, which was not listed as an input\n", input)
				hasFailures = true
			}
		} else {
			if _, ok := expectedFiles[input]; !ok {
				fmt.Fprintf(os.Stderr, "Command depended on %q, which was not listed as an input\n", input)
				hasFailures = true
			}
		}
	}

	if hasFailures {
		os.Exit(1)
	}
}

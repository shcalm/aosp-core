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

package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: package_check <jar-file> <package-list>\nChecks that the "+
			"class files in the <jar file> are in <package-list> or sub-packages.")
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	jarFile := args[0]

	var prefixes []string
	for _, arg := range args[1:] {
		if strings.Contains(arg, "/") {
			log.Fatalf("Invalid package %q, Use dot notation for packages.", arg)
		}
		// Transform to a slash-separated path and add a trailing slash to enforce
		// package name boundary.
		prefixes = append(prefixes, strings.ReplaceAll(arg, ".", "/")+"/")
	}

	zipReader, err := zip.OpenReader(jarFile)
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer zipReader.Close()
	failed := false
	for _, file := range zipReader.File {
		name := file.Name
		if strings.HasSuffix(name, ".class") {
			found := false
			for _, prefix := range prefixes {
				if strings.HasPrefix(name, prefix) {
					found = true
					break
				}
			}
			if !found {
				fmt.Fprintf(os.Stderr, "Class file %s is outside specified packages.\n", name)
				failed = true
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}

// Copyright 2026 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <path to symlink>", os.Args[0])
		fmt.Fprintf(os.Stderr, "  This tool will resolve symlinks recursively. It's like \n")
		fmt.Fprintf(os.Stderr, "  realpath, except that it won't produce absolute paths or \n")
		fmt.Fprintf(os.Stderr, "  paths that escape the build directory.\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	f := filepath.Clean(flag.Arg(0))
	if strings.HasPrefix(f, "/") {
		fmt.Fprintf(os.Stderr, "Absolute paths aren't allowed: %s\n", f)
		os.Exit(1)
	}
	if strings.HasPrefix(f, "../") {
		fmt.Fprintf(os.Stderr, "Paths escaping source directory with ../ aren't allowed: %s\n", f)
		os.Exit(1)
	}

	seenLinks := []string{}

	for true {
		if slices.Contains(seenLinks, f) {
			fmt.Fprintf(os.Stderr, "Symlink cycle:\n")
			for _, l := range seenLinks {
				fmt.Fprintf(os.Stderr, "  %s\n", l)
			}
			fmt.Fprintf(os.Stderr, "  %s\n", f)
			os.Exit(1)
		}

		seenLinks = append(seenLinks, f)

		fi := must2(os.Lstat(f))

		if fi.Mode()&os.ModeSymlink == 0 {
			fmt.Printf("%s\n", f)
			return
		}

		l := must2(os.Readlink(f))
		if strings.HasPrefix(l, "/") {
			fmt.Fprintf(os.Stderr, "Absolute symlinks aren't allowed: %s -> %s\n", f, l)
			os.Exit(1)
		}
		f2 := filepath.Join(filepath.Dir(f), l)
		if strings.HasPrefix(f2, "../") {
			fmt.Fprintf(os.Stderr, "Symlink escapes source root with ../'s: %s -> %s\n", f, l)
			os.Exit(1)
		}
		f = f2
	}
}

func must2[T any](x T, err error) T {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	return x
}

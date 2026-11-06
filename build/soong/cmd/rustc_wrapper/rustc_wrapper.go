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
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: rustc_wrapper <rustc command...>\n"+
			"A wrapper around rustc/clippy to set up the correct absolute path for OUT_DIR.\n"+
			"This has to be done on invocation rather than precalucalted in Soong to\n"+
			"ensure that RBE calculates its own absolute path rather than inheriting the\n"+
			"local host absolute path.\n\n"+
			"The first argument should be the path to rustc/clippy")
		flag.PrintDefaults()
	}

	flag.Parse()

	cwd := must2(os.Getwd())

	// Check if $SOONG_RUST_GEN_DIR is set, otherwise don't do anything.
	soongRustGenDir := os.Getenv("SOONG_RUST_GEN_DIR")
	if soongRustGenDir != "" {
		if strings.HasPrefix(soongRustGenDir, "/") {
			os.Setenv("OUT_DIR", soongRustGenDir)
		} else {
			os.Setenv("OUT_DIR", filepath.Join(cwd, soongRustGenDir))
		}
	}

	args := flag.Args()

	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			os.Exit(err.ExitCode())
		}
		must(err)
	}

	for i, arg := range args {
		if arg == "--emit" && i < len(args)-1 {
			key, val, _ := strings.Cut(args[i+1], "=")
			if key == "dep-info" {
				// soong will tell rustc to emit a depfile to $out.d.raw, strip off the extra
				// extensions to get the normal depfile and rustc output file.
				rawDeps := val
				depsFile := strings.TrimSuffix(rawDeps, ".raw")
				outFile := strings.TrimSuffix(depsFile, ".d")
				outFileColon := outFile + ":"

				var depsContent strings.Builder
				rawDepsContent := string(must2(os.ReadFile(rawDeps)))
				for _, line := range strings.Split(rawDepsContent, "\n") {
					// Rustc deps-info writes out make compatible dep files:
					// https://github.com/rust-lang/rust/issues/7633
					// Rustc emits unneeded dependency lines for the .d and input .rs files.
					// Those extra lines cause ninja warning:
					//     "warning: depfile has multiple output paths"
					// For ninja, we keep/grep only the dependency rule for the rust $out file.
					if strings.HasPrefix(line, outFileColon) {
						// Absolute paths from include! macros (and similar) can cause a mismatch
						// between RBE and local dep info files, so strip out $ANDROID_BUILD_TOP
						// Also we need to make sure stripping only happens at the beginning of each file path
						prefix := strings.TrimPrefix(outFileColon, cwd+"/")
						suffix := strings.ReplaceAll(strings.TrimPrefix(line, outFileColon), " "+cwd+"/", " ")
						line = prefix + suffix
						depsContent.WriteString(line)
						depsContent.WriteRune('\n')
					}
				}

				must(os.WriteFile(depsFile, []byte(depsContent.String()), 0666))
				break
			}
		}
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func must2[T any](x T, err error) T {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	return x
}

// Copyright 2026 Google Inc. All rights reserved.
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
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	androidClangBin, newClangArgs, err := rewriteArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	err = runClang(androidClangBin, newClangArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "running %q failed: %s\n", androidClangBin, err)
		os.Exit(-1)
	}
}

// runClang attempts to run clang with the given arguments directly, falling back to using an rspfile if it gets E2BIG.
func runClang(androidClangBin string, newClangArgs []string) error {
	err := runCommand(androidClangBin, newClangArgs...)
	if errors.Is(err, syscall.E2BIG) {
		argsFile, err := writeArgsToTempFile(newClangArgs)
		defer os.Remove(argsFile)
		if err != nil {
			return fmt.Errorf("writing args to rspfile %s: %w", argsFile, err)
		}
		err = runCommand(androidClangBin, "@"+argsFile)
	}
	return err
}

func rewriteArgs(oldClangArgs []string) (androidClangBin string, newClangArgs []string, err error) {
	var replacementVersionScript string

	if len(oldClangArgs) == 1 && strings.HasPrefix(oldClangArgs[0], "@") {
		// The command line was too long and rustc put the arguments into an rsp file, one per line.
		argsFile := strings.TrimPrefix(oldClangArgs[0], "@")
		oldClangArgs, err = readArgsFromFile(argsFile)
		if err != nil {
			return "", nil, fmt.Errorf("reading arguments from file %q: %w", argsFile, err)
		}
	}

	const androidClangBinFlag = "--android-clang-bin="
	const androidVersionScriptFlag = "-Wl,--android-version-script="

	for _, arg := range oldClangArgs {
		if strings.HasPrefix(arg, androidClangBinFlag) {
			androidClangBin = strings.TrimPrefix(arg, androidClangBinFlag)
		} else if strings.HasPrefix(arg, androidVersionScriptFlag) {
			// Record and remove the custom android-version-script arg
			replacementVersionScript = strings.TrimPrefix(arg, androidVersionScriptFlag)
		} else if arg == "rsend.o" || arg == "rsbegin.o" {
			// Remove object files rustc emits for Windows target
			// We provide these as an rlib.
		} else {
			// Keep the arg
			newClangArgs = append(newClangArgs, arg)
		}
	}

	if androidClangBin == "" {
		err = fmt.Errorf("--android-clang-bin= argument is required")
		return
	}

	// Modify args
	if replacementVersionScript != "" {
		var versionScriptFound bool
		for i, arg := range newClangArgs {
			if strings.HasPrefix(arg, "-Wl,--version-script=") {
				newClangArgs[i] = "-Wl,--version-script=" + replacementVersionScript
				versionScriptFound = true
				break
			}
		}

		if !versionScriptFound {
			// If rustc did not emit a version script, just append the arg
			newClangArgs = append(newClangArgs, "-Wl,--version-script="+replacementVersionScript)
		}
	}

	return
}

// readArgsFromFile reads a list of arguments from a file, unescaping spaces and backslashes that have been escaped
// by rustc.
func readArgsFromFile(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var args []string
	scanner := bufio.NewScanner(f)
	unescaper := strings.NewReplacer(`\ `, ` `, `\\`, `\`)
	for scanner.Scan() {
		args = append(args, unescaper.Replace(scanner.Text()))
	}
	return args, scanner.Err()
}

// writeArgsToTempFile writes a list of arguments to a temporary file, one per line and escaping spaces and backslashes.
// It returns the name of the temporary file.
func writeArgsToTempFile(args []string) (string, error) {
	argsFile, err := os.CreateTemp("", "rustc_linker_args_")
	if err != nil {
		return "", err
	}
	defer argsFile.Close()

	w := bufio.NewWriter(argsFile)
	defer w.Flush()

	argFileEscaper := strings.NewReplacer(`\`, `\\`, ` `, `\ `)
	for _, arg := range args {
		argFileEscaper.WriteString(w, arg)
		w.WriteRune('\n')
	}

	return argsFile.Name(), nil
}

// runCommand runs a command, connecting stdin, stdout and stderr to the corresponding descriptors in this process,
// and waiting for it to exit.
func runCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

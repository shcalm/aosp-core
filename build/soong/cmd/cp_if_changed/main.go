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
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func run(args []string) error {
	fs := flag.NewFlagSet("cp_if_changed", flag.ContinueOnError)
	helpFlag := fs.Bool("help", false, "Show this help message and exit")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: cp_if_changed [OPTIONS] <source_file>... <destination>\n")
		fmt.Fprintf(os.Stderr, "Copies <source_file>s to <destination> only if their contents differ.\n")
		fmt.Fprintf(os.Stderr, "If <destination> is a directory, multiple source files are accepted.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *helpFlag {
		fs.Usage()
		return nil
	}

	if fs.NArg() < 2 {
		fs.Usage()
		return fmt.Errorf("at least one source and one destination argument are required")
	}

	allArgs := fs.Args()
	srcPaths := allArgs[:len(allArgs)-1]
	destPath := allArgs[len(allArgs)-1]

	destStat, err := os.Stat(destPath)
	isDir := err == nil && destStat.IsDir()

	if !isDir && len(srcPaths) > 1 {
		return fmt.Errorf("target '%s' is not a directory, cannot provide multiple source files", destPath)
	}

	for _, srcPath := range srcPaths {
		targetPath := destPath
		if isDir {
			targetPath = filepath.Join(destPath, filepath.Base(srcPath))
		}

		differ, err := filesDiffer(srcPath, targetPath)
		if err != nil {
			return err
		}

		if differ {
			if err := copyFile(srcPath, targetPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// filesDiffer checks if two files have different contents.
func filesDiffer(inPath, destPath string) (bool, error) {
	f1, err := os.Open(inPath)
	if err != nil {
		return false, err
	}
	defer f1.Close()

	f2, err := os.Open(destPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}
	defer f2.Close()

	f1Stat, err := f1.Stat()
	if err != nil {
		return false, err
	}
	f2Stat, err := f2.Stat()
	if err != nil {
		return false, err
	}
	if f1Stat.Size() != f2Stat.Size() {
		return true, nil
	}

	buf1 := make([]byte, 4096)
	buf2 := make([]byte, 4096)

	for {
		n1, err1 := f1.Read(buf1)
		n2, err2 := f2.Read(buf2)

		if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return true, nil
		}

		if err1 == io.EOF && err2 == io.EOF {
			return false, nil
		}

		if err1 != nil && err1 != io.EOF {
			return false, err1
		}
		if err2 != nil && err2 != io.EOF {
			return false, err2
		}
	}
}

func copyFile(inPath, destPath string) error {
	in, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := out.Chmod(0666); err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	return err
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

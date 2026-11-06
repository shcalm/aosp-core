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
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCopyIfChanged(t *testing.T) {
	tmpDir := t.TempDir()

	inputFile := filepath.Join(tmpDir, "input.txt")
	destFile := filepath.Join(tmpDir, "dest.txt")

	content := []byte("hello world")
	if err := os.WriteFile(inputFile, content, 0644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	// 1. Run the copy tool
	if err := run([]string{inputFile, destFile}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// 2. Verify new file is created
	info, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("dest file not created: %v", err)
	}
	mtime1 := info.ModTime()

	// 3. Run the copy tool again
	past := mtime1.Add(-1 * time.Hour)
	if err := os.Chtimes(destFile, past, past); err != nil {
		t.Fatalf("failed to change times: %v", err)
	}

	if err := run([]string{inputFile, destFile}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// 4. Verify that the new file's mtime is not updated
	info, err = os.Stat(destFile)
	if err != nil {
		t.Fatalf("failed to stat dest file: %v", err)
	}
	mtime2 := info.ModTime()

	if !mtime2.Equal(past) {
		t.Errorf("mtime updated even though file didn't change: %v != %v", mtime2, past)
	}
}

func TestCopyToDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	destDir := filepath.Join(tmpDir, "dest")
	if err := os.Mkdir(destDir, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	file1 := filepath.Join(srcDir, "file1.txt")
	file2 := filepath.Join(srcDir, "file2.txt")

	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	// Copy multiple files to directory
	if err := run([]string{file1, file2, destDir}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// Verify files are copied
	for _, name := range []string{"file1.txt", "file2.txt"} {
		if _, err := os.Stat(filepath.Join(destDir, name)); err != nil {
			t.Errorf("expected file %s to be copied to %s", name, destDir)
		}
	}

	// Verify mtime doesn't change on re-copy
	destFile1 := filepath.Join(destDir, "file1.txt")
	past := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(destFile1, past, past); err != nil {
		t.Fatalf("failed to change times: %v", err)
	}

	if err := run([]string{file1, file2, destDir}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	info, err := os.Stat(destFile1)
	if err != nil {
		t.Fatalf("failed to stat: %v", err)
	}
	if !info.ModTime().Equal(past) {
		t.Errorf("mtime updated on re-copy to directory: %v != %v", info.ModTime(), past)
	}
}

func TestFilesDifferSize(t *testing.T) {
	tmpDir := t.TempDir()

	inputFile := filepath.Join(tmpDir, "input.txt")
	destFile := filepath.Join(tmpDir, "dest.txt")

	if err := os.WriteFile(inputFile, []byte("short"), 0644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}
	if err := os.WriteFile(destFile, []byte("much longer content"), 0644); err != nil {
		t.Fatalf("failed to write dest file: %v", err)
	}

	// Capture mtime of dest file before running
	info, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("failed to stat: %v", err)
	}
	mtime1 := info.ModTime()
	// Sleep a bit to ensure mtime change if it happens
	time.Sleep(10 * time.Millisecond)

	if err := run([]string{inputFile, destFile}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// Verify dest file is updated
	info, err = os.Stat(destFile)
	if err != nil {
		t.Fatalf("failed to stat: %v", err)
	}
	if info.ModTime().Equal(mtime1) {
		t.Errorf("dest file was not updated even though sizes differ")
	}

	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}
	if string(content) != "short" {
		t.Errorf("expected content 'short', got %q", string(content))
	}
}

func TestCopyFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	inputFile := filepath.Join(tmpDir, "input.txt")
	destFile := filepath.Join(tmpDir, "dest.txt")

	if err := os.WriteFile(inputFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	// Create dest file with restricted permissions
	if err := os.WriteFile(destFile, []byte("old content"), 0600); err != nil {
		t.Fatalf("failed to write dest file: %v", err)
	}

	if err := run([]string{inputFile, destFile}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	info, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("failed to stat dest file: %v", err)
	}

	// Mode() returns FileMode, we want to check the permission bits.
	// 0666 might be affected by umask if we just created it, but Chmod(0666) should set it exactly.
	// Note: on some systems, some bits might be different, but 0666 is standard.
	if info.Mode().Perm() != 0666 {
		t.Errorf("expected permissions 0666, got %o", info.Mode().Perm())
	}
}

func TestCopyToNonDirectoryError(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	dest := filepath.Join(tmpDir, "dest.txt")

	os.WriteFile(file1, []byte("1"), 0644)
	os.WriteFile(file2, []byte("2"), 0644)
	os.WriteFile(dest, []byte("d"), 0644)

	err := run([]string{file1, file2, dest})
	if err == nil {
		t.Fatal("expected error when copying multiple files to a non-directory")
	}
	expectedErr := "target '" + dest + "' is not a directory, cannot provide multiple source files"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

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

package release_config_lib

import (
	"os"
	"reflect"
	"testing"
)

func TestReadFromFile(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected StringList
		hasError bool
	}{
		{
			name:     "basic",
			content:  "a b c",
			expected: StringList{"a", "b", "c"},
		},
		{
			name:     "comments",
			content:  "a b # c",
			expected: StringList{"a", "b"},
		},
		{
			name:     "empty lines",
			content:  "a\n\nb",
			expected: StringList{"a", "b"},
		},
		{
			name:     "nonexistent file",
			content:  "",
			hasError: true,
		},
		{
			name:     "empty file",
			content:  "",
			expected: nil,
		},
		{
			name:     "multiple lines",
			content:  "\na b\nc d e\nf\n",
			expected: StringList{"a", "b", "c", "d", "e", "f"},
		},
		{
			name:     "multiple lines with comments",
			content:  "\na b\nc d e # f\ng\n",
			expected: StringList{"a", "b", "c", "d", "e", "g"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var sl StringList
			var err error
			var f *os.File

			if tc.name != "nonexistent file" {
				f, err = os.CreateTemp(t.TempDir(), "test")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(f.Name())
				if _, err := f.WriteString(tc.content); err != nil {
					t.Fatal(err)
				}
				f.Close()
				err = sl.ReadFromFile(f.Name())
			} else {
				err = sl.ReadFromFile("nonexistent")
			}

			if tc.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !reflect.DeepEqual(sl, tc.expected) {
					t.Errorf("Expected %v, got %v", tc.expected, sl)
				}
			}
		})
	}
}

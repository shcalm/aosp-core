// Copyright 2024 Google Inc. All rights reserved.
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

package find_input_delta_lib

import (
	"bytes"
	"testing"

	// For Assert*.
	"android/soong/android"
)

func (fl *FileList) Equal(other *FileList) bool {
	if fl.Name != other.Name {
		return false
	}
	if !compareFileListSlices(fl.Additions, other.Additions) {
		return false
	}
	if !compareFileListSlices(fl.Deletions, other.Deletions) {
		return false
	}
	if !compareFileListSlices(fl.Changes, other.Changes) {
		return false
	}
	return true
}

func compareFileListSlices(a, b []FileList) bool {
	if len(a) != len(b) {
		return false
	}
	for idx, ch := range a {
		if !ch.Equal(&b[idx]) {
			return false
		}
	}

	return true
}

func TestFormat(t *testing.T) {
	testCases := []struct {
		Name     string
		Template string
		Input    FileList
		Expected string
		Err      error
	}{
		{
			Name:     "no contents",
			Template: DefaultTemplate,
			Input: FileList{
				Name: "target",
				Additions: []FileList{
					FileList{Name: "add1"},
					FileList{Name: "add2"},
				},
				Deletions: []FileList{
					FileList{Name: "del1"},
					FileList{Name: "del2"},
				},
				Changes: []FileList{
					FileList{Name: "mod1"},
					FileList{Name: "mod2"},
				},
			},
			Expected: "-del1 -del2 +add1 +add2 +mod1 +mod2 ",
			Err:      nil,
		},
		{
			Name:     "adds",
			Template: DefaultTemplate,
			Input: FileList{
				Name: "target",
				Additions: []FileList{
					FileList{Name: "add1"},
					FileList{Name: "add2"},
				},
			},
			Expected: "+add1 +add2 ",
			Err:      nil,
		},
		{
			Name:     "deletes",
			Template: DefaultTemplate,
			Input: FileList{
				Name: "target",
				Deletions: []FileList{
					FileList{Name: "del1"},
					FileList{Name: "del2"},
				},
			},
			Expected: "-del1 -del2 ",
			Err:      nil,
		},
		{
			Name:     "changes",
			Template: DefaultTemplate,
			Input: FileList{
				Name: "target",
				Changes: []FileList{
					FileList{Name: "mod1"},
					FileList{Name: "mod2"},
				},
			},
			Expected: "+mod1 +mod2 ",
			Err:      nil,
		},
		{
			Name:     "with contents",
			Template: DefaultTemplate,
			Input: FileList{
				Name: "target",
				Additions: []FileList{
					FileList{
						Name: "add1",
						Additions: []FileList{
							FileList{Name: "a1a1"},
						},
					},
					FileList{Name: "add2"},
				},
				Deletions: []FileList{
					FileList{Name: "del1"},
					FileList{
						Name: "del2",
						Deletions: []FileList{
							FileList{Name: "d2d1"},
						},
					},
				},
				Changes: []FileList{
					FileList{
						Name: "mod1",
					},
					FileList{
						Name: "mod2",
						Additions: []FileList{
							FileList{Name: "m2a1"},
						},
						Deletions: []FileList{
							FileList{Name: "m2d1"},
						},
					},
				},
			},
			Expected: "-del1 -del2 +add1 +add2 +mod1 +mod2 --file del2 -d2d1 --endfile --file add1 +a1a1 --endfile --file mod2 -m2d1 +m2a1 --endfile ",
			Err:      nil,
		},
	}
	for _, tc := range testCases {
		buf := bytes.NewBuffer([]byte{})
		err := tc.Input.Format(buf, tc.Template)
		android.AssertSame(t, tc.Name, tc.Err, err)
		android.AssertSame(t, tc.Name, tc.Expected, buf.String())
	}
}

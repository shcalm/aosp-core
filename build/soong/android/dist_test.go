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

package android

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestGetDistContributions(t *testing.T) {
	compareContributions := func(d1 *distContributions, d2 *distContributions) error {
		if d1 == nil || d2 == nil {
			if d1 != d2 {
				return fmt.Errorf("pointer mismatch, expected both to be nil but they were %p and %p", d1, d2)
			} else {
				return nil
			}
		}
		if expected, actual := len(d1.copiesForGoals), len(d2.copiesForGoals); expected != actual {
			return fmt.Errorf("length mismatch, expected %d found %d", expected, actual)
		}

		for i, copies1 := range d1.copiesForGoals {
			copies2 := d2.copiesForGoals[i]
			if expected, actual := copies1.goals, copies2.goals; !slices.Equal(expected, actual) {
				return fmt.Errorf("goals mismatch at position %d: expected %q found %q", i, expected, actual)
			}

			if expected, actual := len(copies1.copies), len(copies2.copies); expected != actual {
				return fmt.Errorf("length mismatch in copy instructions at position %d, expected %d found %d", i, expected, actual)
			}

			for j, c1 := range copies1.copies {
				c2 := copies2.copies[j]
				if expected, actual := NormalizePathForTesting(c1.from), NormalizePathForTesting(c2.from); expected != actual {
					return fmt.Errorf("paths mismatch at position %d.%d: expected %q found %q", i, j, expected, actual)
				}

				if expected, actual := c1.dest, c2.dest; expected != actual {
					return fmt.Errorf("dest mismatch at position %d.%d: expected %q found %q", i, j, expected, actual)
				}
			}
		}

		return nil
	}

	formatContributions := func(d *distContributions) string {
		buf := &strings.Builder{}
		if d == nil {
			fmt.Fprint(buf, "nil")
		} else {
			for _, copiesForGoals := range d.copiesForGoals {
				fmt.Fprintf(buf, "    Goals: %q {\n", strings.Join(copiesForGoals.goals, " "))
				for _, c := range copiesForGoals.copies {
					fmt.Fprintf(buf, "        %s -> %s\n", NormalizePathForTesting(c.from), c.dest)
				}
				fmt.Fprint(buf, "    }\n")
			}
		}
		return buf.String()
	}

	testHelper := func(t *testing.T, name, bp string, expectedContributions *distContributions) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()

			ctx, module := buildContextAndCustomModuleFoo(t, bp)
			entries := AndroidMkEntriesForTest(t, ctx, module)
			if len(entries) != 1 {
				t.Errorf("Expected a single AndroidMk entry, got %d", len(entries))
			}
			distContributions := getDistContributions(ctx, module)

			if err := compareContributions(expectedContributions, distContributions); err != nil {
				t.Errorf("%s\nExpected Contributions\n%sActualContributions\n%s",
					err,
					formatContributions(expectedContributions),
					formatContributions(distContributions))
			}
		})
	}

	testHelper(t, "dist-without-tag", `
			custom {
				name: "foo",
				dist: {
					targets: ["my_goal"]
				}
			}
`,
		&distContributions{
			copiesForGoals: []*copiesForGoals{
				{
					goals: []string{"my_goal"},
					copies: []distCopy{
						distCopyForTest("one.out", "one.out"),
					},
				},
			},
		})

	testHelper(t, "dist-with-tag", `
			custom {
				name: "foo",
				dist: {
					targets: ["my_goal"],
					tag: ".another-tag",
				}
			}
`,
		&distContributions{
			copiesForGoals: []*copiesForGoals{
				{
					goals: []string{"my_goal"},
					copies: []distCopy{
						distCopyForTest("another.out", "another.out"),
					},
				},
			},
		})

	testHelper(t, "append-artifact-with-product", `
			custom {
				name: "foo",
				dist: {
					targets: ["my_goal"],
					append_artifact_with_product: true,
				}
			}
`, &distContributions{
		copiesForGoals: []*copiesForGoals{
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("one.out", "one_bar.out"),
				},
			},
		},
	})

	testHelper(t, "dists-with-tag", `
			custom {
				name: "foo",
				dists: [
					{
						targets: ["my_goal"],
						tag: ".another-tag",
					},
				],
			}
`,
		&distContributions{
			copiesForGoals: []*copiesForGoals{
				{
					goals: []string{"my_goal"},
					copies: []distCopy{
						distCopyForTest("another.out", "another.out"),
					},
				},
			},
		})

	testHelper(t, "multiple-dists-with-and-without-tag", `
			custom {
				name: "foo",
				dists: [
					{
						targets: ["my_goal"],
					},
					{
						targets: ["my_second_goal", "my_third_goal"],
					},
				],
			}
`,
		&distContributions{
			copiesForGoals: []*copiesForGoals{
				{
					goals: []string{"my_goal"},
					copies: []distCopy{
						distCopyForTest("one.out", "one.out"),
					},
				},
				{
					goals: []string{"my_second_goal", "my_third_goal"},
					copies: []distCopy{
						distCopyForTest("one.out", "one.out"),
					},
				},
			},
		})

	testHelper(t, "dist-plus-dists-without-tags", `
			custom {
				name: "foo",
				dist: {
					targets: ["my_goal"],
				},
				dists: [
					{
						targets: ["my_second_goal", "my_third_goal"],
					},
				],
			}
`,
		&distContributions{
			copiesForGoals: []*copiesForGoals{
				{
					goals: []string{"my_second_goal", "my_third_goal"},
					copies: []distCopy{
						distCopyForTest("one.out", "one.out"),
					},
				},
				{
					goals: []string{"my_goal"},
					copies: []distCopy{
						distCopyForTest("one.out", "one.out"),
					},
				},
			},
		})

	testHelper(t, "dist-plus-dists-with-tags", `
			custom {
				name: "foo",
				dist: {
					targets: ["my_goal", "my_other_goal"],
					tag: ".multiple",
				},
				dists: [
					{
						targets: ["my_second_goal"],
						tag: ".multiple",
					},
					{
						targets: ["my_third_goal"],
						dir: "test/dir",
					},
					{
						targets: ["my_fourth_goal"],
						suffix: ".suffix",
					},
					{
						targets: ["my_fifth_goal"],
						dest: "new-name",
					},
					{
						targets: ["my_sixth_goal"],
						dest: "new-name",
						dir: "some/dir",
						suffix: ".suffix",
					},
				],
			}
`,
		&distContributions{
			copiesForGoals: []*copiesForGoals{
				{
					goals: []string{"my_second_goal"},
					copies: []distCopy{
						distCopyForTest("two.out", "two.out"),
						distCopyForTest("three/four.out", "four.out"),
					},
				},
				{
					goals: []string{"my_third_goal"},
					copies: []distCopy{
						distCopyForTest("one.out", "test/dir/one.out"),
					},
				},
				{
					goals: []string{"my_fourth_goal"},
					copies: []distCopy{
						distCopyForTest("one.out", "one.suffix.out"),
					},
				},
				{
					goals: []string{"my_fifth_goal"},
					copies: []distCopy{
						distCopyForTest("one.out", "new-name"),
					},
				},
				{
					goals: []string{"my_sixth_goal"},
					copies: []distCopy{
						distCopyForTest("one.out", "some/dir/new-name.suffix"),
					},
				},
				{
					goals: []string{"my_goal", "my_other_goal"},
					copies: []distCopy{
						distCopyForTest("two.out", "two.out"),
						distCopyForTest("three/four.out", "four.out"),
					},
				},
			},
		})

	// The above test the default values of default_dist_files and use_output_file.

	// The following tests explicitly test the different combinations of those settings.
	testHelper(t, "tagged-dist-files-no-output", `
			custom {
				name: "foo",
				default_dist_files: "tagged",
				dist_output_file: false,
				dists: [
					{
						targets: ["my_goal"],
					},
					{
						targets: ["my_goal"],
						tag: ".multiple",
					},
				],
			}
`, &distContributions{
		copiesForGoals: []*copiesForGoals{
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("one.out", "one.out"),
				},
			},
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("two.out", "two.out"),
					distCopyForTest("three/four.out", "four.out"),
				},
			},
		},
	})

	testHelper(t, "default-dist-files-no-output", `
			custom {
				name: "foo",
				default_dist_files: "default",
				dist_output_file: false,
				dists: [
					{
						targets: ["my_goal"],
					},
					{
						targets: ["my_goal"],
						tag: ".multiple",
					},
				],
			}
`, &distContributions{
		copiesForGoals: []*copiesForGoals{
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("default-dist.out", "default-dist.out"),
				},
			},
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("two.out", "two.out"),
					distCopyForTest("three/four.out", "four.out"),
				},
			},
		},
	})

	testHelper(t, "no-dist-files-no-output", `
			custom {
				name: "foo",
				default_dist_files: "none",
				dist_output_file: false,
				dists: [
					// The following will dist one.out because there's no default dist file provided
					// (default_dist_files: "none") and one.out is the outputfile for the "" tag.
					{
						targets: ["my_goal"],
					},
					{
						targets: ["my_goal"],
						tag: ".multiple",
					},
				],
			}
`, &distContributions{
		copiesForGoals: []*copiesForGoals{
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("one.out", "one.out"),
				},
			},
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("two.out", "two.out"),
					distCopyForTest("three/four.out", "four.out"),
				},
			},
		},
	})

	testHelper(t, "tagged-dist-files-default-output", `
			custom {
				name: "foo",
				default_dist_files: "tagged",
				dist_output_file: true,
				dists: [
					{
						targets: ["my_goal"],
					},
					{
						targets: ["my_goal"],
						tag: ".multiple",
					},
				],
			}
`, &distContributions{
		copiesForGoals: []*copiesForGoals{
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("one.out", "one.out"),
				},
			},
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("two.out", "two.out"),
					distCopyForTest("three/four.out", "four.out"),
				},
			},
		},
	})

	testHelper(t, "default-dist-files-default-output", `
			custom {
				name: "foo",
				default_dist_files: "default",
				dist_output_file: true,
				dists: [
					{
						targets: ["my_goal"],
					},
					{
						targets: ["my_goal"],
						tag: ".multiple",
					},
				],
			}
`, &distContributions{
		copiesForGoals: []*copiesForGoals{
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("default-dist.out", "default-dist.out"),
				},
			},
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("two.out", "two.out"),
					distCopyForTest("three/four.out", "four.out"),
				},
			},
		},
	})

	testHelper(t, "no-dist-files-default-output", `
			custom {
				name: "foo",
				default_dist_files: "none",
				dist_output_file: true,
				dists: [
					{
						targets: ["my_goal"],
					},
					{
						targets: ["my_goal"],
						tag: ".multiple",
					},
				],
			}
`, &distContributions{
		copiesForGoals: []*copiesForGoals{
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("dist-output-file.out", "dist-output-file.out"),
				},
			},
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("two.out", "two.out"),
					distCopyForTest("three/four.out", "four.out"),
				},
			},
		},
	})
}

func distCopyForTest(from, to string) distCopy {
	return distCopy{PathForTesting(from), to}
}

func TestGenerateDistContributionsForSoong(t *testing.T) {
	dc := []distContributions{{
		copiesForGoals: []*copiesForGoals{
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("one.out", "one.out"),
					distCopyForTest("two.out", "other.out"),
				},
			},
		},
	}}

	s := soongDist{}
	s.parseDistContributions(dc)
	distOutput := &bytes.Buffer{}
	err := s.generateSoongDistNinja(distOutput, true)
	if err != nil {
		t.Fatalf("failed to generate Soong dist ninja: %s", err)
	}
	noDistOutput := &bytes.Buffer{}
	err = s.generateSoongNoDistNinja(noDistOutput)
	if err != nil {
		t.Fatalf("failed to generate Soong nodist ninja: %s", err)
	}

	expectedDistOutput := `rule distCp
 description = Dist $out
 command = rm -f $out && cp $in $out
 sandbox_disabled = true
 pool = local_pool

build $distdir/one.out: distCp one.out
build $distdir/other.out: distCp two.out
build my_goal__dist: phony $distdir/one.out $distdir/other.out
 phony_output = true
`

	expectedNoDistOutput := `build my_goal__dist: phony
 phony_output = true
`

	assertStringEquals(t, expectedDistOutput, distOutput.String())
	assertStringEquals(t, expectedNoDistOutput, noDistOutput.String())
}

func TestDistConflicts(t *testing.T) {
	dc := []distContributions{{
		copiesForGoals: []*copiesForGoals{
			{
				goals: []string{"my_goal"},
				copies: []distCopy{
					distCopyForTest("one.out", "one.out"),
					distCopyForTest("two.out", "one.out"),
				},
			},
		},
	}}

	s := soongDist{}
	s.parseDistContributions(dc)
	distOutput := &bytes.Buffer{}
	err := s.generateSoongConflictDistNinja(distOutput, true)
	if err != nil {
		t.Fatalf("failed to generate Soong dist ninja: %s", err)
	}
	noDistOutput := &bytes.Buffer{}
	err = s.generateSoongNoDistNinja(noDistOutput)
	if err != nil {
		t.Fatalf("failed to generate Soong nodist ninja: %s", err)
	}

	expectedDistOutput := `rule distError
 description = Error $out
 command = echo $distErrorMsg; false
 sandbox_disabled = true
 phony_output = true
 pool = local_pool
distErrorMsg = 'conflicting dist sources "two.out" and "one.out" for target "one.out"'
build my_goal__dist: distError
`

	expectedNoDistOutput := `build my_goal__dist: phony
 phony_output = true
`

	assertStringEquals(t, expectedDistOutput, distOutput.String())
	assertStringEquals(t, expectedNoDistOutput, noDistOutput.String())
}

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

package atomsapigen

import (
	"fmt"
	"strings"
	"testing"

	"android/soong/android"
)

func DirectDepsList(ctx *android.TestResult, module android.Module) []string {
	deps := []string{}
	ctx.VisitDirectDeps(module, func(dep android.Module) {
		deps = append(deps, dep.Name())
	})
	return deps
}

// Test valid cases
func TestCcAtomslogLibrary_VerifyCodeGen(t *testing.T) {
	testCases := []struct {
		name                string
		optionalParams      string
		expectedBasename    string
		expectedMinApiLevel string
		expectedInterface   string
		expectedProtosArg   string
		isTypesafe          bool
	}{
		{
			name: "basic",
			optionalParams: `
				basename: "my_atoms_out",
			`,
			expectedBasename: "my_atoms_out",
		},
		{
			name:             "default basename",
			optionalParams:   "", // no basename specified.
			expectedBasename: "statslog_myatoms",
		},
		{
			name: "min_sdk_version",
			optionalParams: `
				min_sdk_version: "30",
			`,
			expectedBasename:    "statslog_myatoms",
			expectedMinApiLevel: "30",
		},
		{
			name: "interface",
			optionalParams: `
				interface: "bootstrap",
			`,
			expectedBasename:  "statslog_myatoms",
			expectedInterface: "bootstrap",
		},
		{
			name: "proto",
			optionalParams: `
				proto_srcs: [
					"path/to/my/atoms.proto",
					":my_atom_protos",
				],
			`,
			expectedBasename: "statslog_myatoms",
			expectedProtosArg: "path/to/my/atoms.proto" +
				" path/to/my/extension_atoms.proto" +
				" some/protobuf/loc/descriptor.proto" +
				" some/random/path/atom_field_options.proto",
		},
		{
			name: "gen_type typesafe",
			optionalParams: `
				gen_type: "typesafe",
			`,
			expectedBasename: "statslog_myatoms",
			isTypesafe:       true,
		},
		{
			name: "gen_type default",
			optionalParams: `
				gen_type: "default",
			`,
			expectedBasename: "statslog_myatoms",
			isTypesafe:       false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			bp := fmt.Sprintf(`
				cc_atomslog_library {
					name: "libmyatoms_test",
					atoms_module: "myatoms",
					namespace: "test::namespace",
					%s
				}
			`, tt.optionalParams)

			result := prepareForTestWithAtomslogBuildComponents.RunTestWithBp(t, commonBp+bp)
			module := result.ModuleForTests(t, "libmyatoms_test", "android_arm64_armv8-a_static")

			outputs := strings.Join(module.AllOutputs(), " ")
			android.AssertStringDoesContain(t, "generated cpp", outputs, "gen/"+tt.expectedBasename+".cpp")
			android.AssertStringDoesContain(t, "generated header", outputs, "gen/include/"+tt.expectedBasename+".h")

			expectedCppCmd := fmt.Sprintf(
				"__SBOX_SANDBOX_DIR__/tools/out/bin/stats-log-api-gen --cpp __SBOX_SANDBOX_DIR__/out/%s.cpp --namespace test::namespace --importHeader %s.h --omitExtraSrcs --module myatoms",
				tt.expectedBasename, tt.expectedBasename)
			expectedHdrCmd := fmt.Sprintf(
				"__SBOX_SANDBOX_DIR__/tools/out/bin/stats-log-api-gen --header __SBOX_SANDBOX_DIR__/out/include/%s.h --namespace test::namespace --omitExtraSrcs --module myatoms",
				tt.expectedBasename)
			if tt.expectedMinApiLevel != "" {
				expectedCppCmd += " --minApiLevel " + tt.expectedMinApiLevel
				expectedHdrCmd += " --minApiLevel " + tt.expectedMinApiLevel
			}
			if tt.expectedInterface != "" {
				expectedCppCmd += " --interface " + tt.expectedInterface
				expectedHdrCmd += " --interface " + tt.expectedInterface
			}
			if tt.expectedProtosArg != "" {
				expectedCppCmd += " --proto " + tt.expectedProtosArg
				expectedHdrCmd += " --proto " + tt.expectedProtosArg
			}
			if tt.isTypesafe {
				expectedCppCmd += " --type-safe"
				expectedHdrCmd += " --type-safe"
			}
			expectedCmd := fmt.Sprintf("rm -rf %s && mkdir -p %[1]s && %s && %s",
				"__SBOX_SANDBOX_DIR__/out/include", expectedHdrCmd, expectedCppCmd)

			manifest := android.RuleBuilderSboxProtoForTests(t, result.TestContext, module.Output("cc.sbox.textproto"))
			cmdStr := manifest.Commands[0].GetCommand()

			android.AssertStringEquals(t, "wrong cc gen command", expectedCmd, cmdStr)
		})
	}
}

// Test that lib dependencies are added
func TestCcAtomslogLibrary_VerifyExtraCCLibAdded(t *testing.T) {
	bp := `
		cc_atomslog_library {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
		}
	`
	result := prepareForTestWithAtomslogBuildComponents.RunTestWithBp(t, commonBp+bp)
	module := result.ModuleForTests(t, "mystatslog", "android_arm64_armv8-a_static")
	rule := module.Rule("arWithLibs")
	android.EnsureListContainsSuffix(t, rule.Inputs.Strings(), "stats-log-api-gen-cc-lib.a")
}

func TestCcAtomslogLibrary_IncludeDefaultLibsFull(t *testing.T) {
	testCases := []struct {
		name string
		bp   string
	}{
		{
			name: "include_default_libs missing",
			bp: `
				cc_atomslog_library {
					name: "mystatslog",
					atoms_module: "myatoms",
					namespace: "test::namespace",
				}
			`,
		},
		{
			name: "include_default_libs full",
			bp: `
				cc_atomslog_library {
					name: "mystatslog",
					atoms_module: "myatoms",
					namespace: "test::namespace",
					include_default_libs: "full",
				}
			`,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := prepareForTestWithAtomslogBuildComponents.RunTestWithBp(t, commonBp+tt.bp)
			module := result.ModuleForTests(t, "mystatslog", "android_arm64_armv8-a_static")
			deps := DirectDepsList(result, module.Module())
			android.AssertStringListContains(t, "missing libstatssocket", deps, "libstatssocket")
			android.AssertStringListContains(t, "missing libstatspull", deps, "libstatspull")
		})
	}
}

func TestCcAtomslogLibrary_IncludeDefaultLibsHeaders(t *testing.T) {
	bp := `
		cc_atomslog_library {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
			include_default_libs: "headers_only",
		}
	`
	result := prepareForTestWithAtomslogBuildComponents.RunTestWithBp(t, commonBp+bp)
	module := result.ModuleForTests(t, "mystatslog", "android_arm64_armv8-a_static")
	deps := DirectDepsList(result, module.Module())
	android.AssertStringListContains(t, "missing libstatssocket_headers", deps, "libstatssocket_headers")
	android.AssertStringListContains(t, "missing libstatspull_headers", deps, "libstatspull_headers")
}

func TestCcAtomslogLibrary_IncludeDefaultLibsNone(t *testing.T) {
	bp := `
		cc_atomslog_library {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
			include_default_libs: "none",
		}
	`
	result := prepareForTestWithAtomslogBuildComponents.RunTestWithBp(t, commonBp+bp)
	module := result.ModuleForTests(t, "mystatslog", "android_arm64_armv8-a_static")
	deps := DirectDepsList(result, module.Module())
	android.AssertStringListDoesNotContain(t, "unexpected libstatssocket", deps, "libstatssocket")
	android.AssertStringListDoesNotContain(t, "unexpected libstatspull", deps, "libstatspull")
	android.AssertStringListDoesNotContain(t, "unexpected libstatssocket_headers", deps, "libstatssocket_headers")
	android.AssertStringListDoesNotContain(t, "unexpected libstatspull_headers", deps, "libstatspull_headers")
}

func TestCcAtomslogLibrary_IncludeDefaultLibsInvalid(t *testing.T) {
	bp := `
		cc_atomslog_library {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
			include_default_libs: "invalid",
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"include_default_libs: must be one of \"full\", \"headers_only\", or \"none\"")).
		RunTestWithBp(t, commonBp+bp)
}

func TestCcAtomslogLibrary_NoAtomsModule(t *testing.T) {
	bp := `
		cc_atomslog_library {
			name: "mystatslog",
			basename: "mystatslog",
			namespace: "test::namespace",
		}
	`
	result := prepareForTestWithAtomslogBuildComponents.RunTestWithBp(t, commonBp+bp)
	module := result.ModuleForTests(t, "mystatslog", "android_arm64_armv8-a_static")
	manifest := android.RuleBuilderSboxProtoForTests(t, result.TestContext, module.Output("cc.sbox.textproto"))
	cmdStr := manifest.Commands[0].GetCommand()

	android.AssertStringDoesNotContain(t, "no module param", cmdStr, "--module")
}

func TestCcAtomslogLibrary_VerifyStaticCannotBeLinkedAsShared(t *testing.T) {
	bp := `
		cc_atomslog_library_static {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
		}

		cc_library_static {
			name: "myclientlib",
			shared_libs: ["mystatslog"],
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"dependency \"mystatslog\" of \"myclientlib\" missing variant")).
		RunTestWithBp(t, commonBp+bp)
}

func TestCcAtomslogLibrary_VerifySharedCannotBeLinkedAsStatic(t *testing.T) {
	bp := `
		cc_atomslog_library_shared {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
		}

		cc_library_static {
			name: "myclientlib",
			static_libs: ["mystatslog"],
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"dependency \"mystatslog\" of \"myclientlib\" missing variant")).
		RunTestWithBp(t, commonBp+bp)
}

func TestCcAtomslogLibrary_MissingAtomsModuleAndBasename(t *testing.T) {
	bp := `
		cc_atomslog_library_shared {
			name: "mystatslog",
			namespace: "test::namespace",
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"At least one of atoms_module or basename must be provided")).
		RunTestWithBp(t, commonBp+bp)
}

func TestCcAtomslogLibrary_MissingNamespace(t *testing.T) {
	bp := `
		cc_atomslog_library_shared {
			name: "mystatslog",
			atoms_module: "myatoms",
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"namespace: can't be empty")).
		RunTestWithBp(t, commonBp+bp)
}

func TestCcAtomslogLibrary_BadInterface(t *testing.T) {
	bp := `
		cc_atomslog_library_shared {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
			interface: "invalid",
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"interface: must be one of \\[platform bootstrap vendor\\]")).
		RunTestWithBp(t, commonBp+bp)
}

func TestCcAtomslogLibrary_VendorIncludesAidlLib(t *testing.T) {
	testCases := []struct {
		name        string
		aidlVersion int
		expectedLib string
	}{
		{
			name:        "default aidl version",
			expectedLib: fmt.Sprintf(aidlLibFmt, 2, "ndk"),
		},
		{
			name:        "aidl version specified",
			aidlVersion: 3,
			expectedLib: fmt.Sprintf(aidlLibFmt, 3, "ndk"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			aidlVersionStr := ""
			if tt.aidlVersion > 0 {
				aidlVersionStr = fmt.Sprintf("aidl_version: %d,", tt.aidlVersion)
			}
			bp := fmt.Sprintf(`
				cc_library_shared {
					name: "%s",
				}

				cc_atomslog_library {
					name: "mystatslog",
					atoms_module: "myatoms",
					namespace: "test::namespace",
					interface: "vendor",
					%s
				}
			`, tt.expectedLib, aidlVersionStr)
			result := prepareForTestWithAtomslogBuildComponents.RunTestWithBp(t, commonBp+bp)
			module := result.ModuleForTests(t, "mystatslog", "android_arm64_armv8-a_static")
			deps := DirectDepsList(result, module.Module())
			android.AssertStringListContains(t, "missing "+tt.expectedLib, deps, tt.expectedLib)
		})
	}
}

func TestCcAtomslogLibrary_VendorWithBadIncludeDefaultLibs(t *testing.T) {
	testCases := []struct {
		name               string
		includeDefaultLibs string
	}{
		{
			name:               "include_default_libs full",
			includeDefaultLibs: includeDefaultLibsFull,
		},
		{
			name:               "include_default_libs headers_only",
			includeDefaultLibs: includeDefaultLibsHeader,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			bp := fmt.Sprintf(`
				cc_atomslog_library {
					name: "libmyatoms_test",
					atoms_module: "myatoms",
					namespace: "test::namespace",
					interface: "vendor",
					include_default_libs: "%s",
				}
			`, tt.includeDefaultLibs)
			prepareForTestWithAtomslogBuildComponents.
				ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
					"include_default_libs: cannot be set to other than \"none\" when interface is set to \"vendor\"")).
				RunTestWithBp(t, commonBp+bp)
		})
	}
}

func TestCcAtomslogLibrary_BadGenType(t *testing.T) {
	bp := `
		cc_atomslog_library_shared {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
			gen_type: "invalid",
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"gen_type: must be one of \"default\" or \"typesafe\"")).
		RunTestWithBp(t, commonBp+bp)
}

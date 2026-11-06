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

package atomsapigen

import (
	"fmt"
	"strings"
	"testing"

	"android/soong/android"
)

// Test valid genrule cases
func TestJavaAtomslogLibrary_VerifyCodeGen(t *testing.T) {
	testCases := []struct {
		name                string
		optionalParams      string
		expectedClassName   string
		expectedMinApiLevel string
		expectedInterface   string
		expectedProtosArg   string
		isTypesafe          bool
		hasWorksource       bool
		isNonStatic         bool
	}{
		{
			name: "basic",
			optionalParams: `
				class_name: "MyAtomsLog",
			`,
			expectedClassName: "MyAtomsLog",
		},
		{
			name:              "default class name",
			optionalParams:    "", // no class name specified.
			expectedClassName: "MyAtomsStatsLog",
		},
		{
			name: "min_sdk_version",
			optionalParams: `
				min_sdk_version: "30",
			`,
			expectedClassName:   "MyAtomsStatsLog",
			expectedMinApiLevel: "30",
		},
		{
			name: "proto",
			optionalParams: `
				proto_srcs: [
					"path/to/my/atoms.proto",
					":my_atom_protos",
				],
			`,
			expectedClassName: "MyAtomsStatsLog",
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
			expectedClassName: "MyAtomsStatsLog",
			isTypesafe:        true,
		},
		{
			name: "gen_type default",
			optionalParams: `
				gen_type: "default",
			`,
			expectedClassName: "MyAtomsStatsLog",
			isTypesafe:        false,
		},
		{
			name: "has worksource",
			optionalParams: `
				worksource_support: true,
			`,
			expectedClassName: "MyAtomsStatsLog",
			hasWorksource:     true,
		},
		{
			name: "interface",
			optionalParams: `
				interface: "platform",
			`,
			expectedClassName: "MyAtomsStatsLog",
			expectedInterface: "platform",
		},
		{
			name: "non static gencode",
			optionalParams: `
				omit_static_modifier: true,
			`,
			expectedClassName: "MyAtomsStatsLog",
			isNonStatic:       true,
		},
		{
			name: "interface vendor",
			optionalParams: `
				interface: "vendor",
			`,
			expectedClassName: "MyAtomsStatsLog",
			expectedInterface: "vendor",
		},
		{
			name: "interface vendor with aidl version",
			optionalParams: `
				interface: "vendor",
				aidl_version: 3,
			`,
			expectedClassName: "MyAtomsStatsLog",
			expectedInterface: "vendor",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			bp := fmt.Sprintf(`
				java_atomslog_library {
					name: "libmyatomslog",
					atoms_module: "my_atoms",
					package_name: "test.package",
					%s
				}
			`, tt.optionalParams)

			result := prepareForTestWithAtomslogBuildComponents.RunTestWithBp(t, commonBp+bp)
			module := result.ModuleForTests(t, "libmyatomslog", "android_common")

			outputs := strings.Join(module.AllOutputs(), " ")
			android.AssertStringDoesContain(t, "generated srcjar", outputs, "gen/libmyatomslog.srcjar")
			android.AssertStringDoesContain(t, "generated java", outputs, "gen/libmyatomslog.srcjar.tmp/"+tt.expectedClassName+".java")

			expectedJavaCmd := fmt.Sprintf(
				"rm -rf %[1]s.tmp"+
					" && mkdir -p %[1]s.tmp"+
					" && __SBOX_SANDBOX_DIR__/tools/out/bin/stats-log-api-gen --java %[1]s.tmp/%[2]s.java --javaClass %[2]s --javaPackage test.package --module my_atoms --omitExtraSrcs",
				"__SBOX_SANDBOX_DIR__/out/libmyatomslog.srcjar", tt.expectedClassName)

			if tt.expectedMinApiLevel != "" {
				expectedJavaCmd += " --minApiLevel " + tt.expectedMinApiLevel
			}
			if tt.isNonStatic {
				expectedJavaCmd += " --nonStatic"
			}
			if tt.hasWorksource {
				expectedJavaCmd += " --worksource"
			}
			if tt.expectedInterface != "" {
				expectedJavaCmd += " --interface " + tt.expectedInterface
			}
			if tt.expectedProtosArg != "" {
				expectedJavaCmd += " --proto " + tt.expectedProtosArg
			}
			if tt.isTypesafe {
				expectedJavaCmd += " --type-safe"
			}

			expectedJavaCmd += fmt.Sprintf(
				" && cp include_java/StatsHistogram.java %[1]s.tmp"+
					" && __SBOX_SANDBOX_DIR__/tools/out/bin/soong_zip -srcjar -o %[1]s -C %[1]s.tmp -D %[1]s.tmp",
				"__SBOX_SANDBOX_DIR__/out/libmyatomslog.srcjar")

			javaManifest := android.RuleBuilderSboxProtoForTests(t, result.TestContext, module.Output("java.sbox.textproto"))
			javaCmd := javaManifest.Commands[0].GetCommand()

			android.AssertStringEquals(t, "wrong .srcjar gen command", expectedJavaCmd, javaCmd)
		})
	}
}

func TestJavaAtomslogLibrary_MissingAtomsModule(t *testing.T) {
	bp := `
		java_atomslog_library {
			name: "libmyatomslog",
			package_name: "test.package",
			class_name: "MyAtomsLog",
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"atoms_module: can't be empty")).
		RunTestWithBp(t, commonBp+bp)
}

func TestJavaAtomslogLibrary_MissingPackageName(t *testing.T) {
	bp := `
		java_atomslog_library {
			name: "libmyatomslog",
			atoms_module: "my_atoms",
			class_name: "MyAtomsLog",
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"package_name: can't be empty")).
		RunTestWithBp(t, commonBp+bp)
}

func TestJavaAtomslogLibrary_BadGenType(t *testing.T) {
	bp := `
		java_atomslog_library {
			name: "libmyatomslog",
			atoms_module: "my_atoms",
			package_name: "test.package",
			class_name: "MyAtomsLog",
			gen_type: "invalid",
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"gen_type: must be one of \"default\" or \"typesafe\"")).
		RunTestWithBp(t, commonBp+bp)
}

func TestJavaAtomslogLibrary_BadInterface(t *testing.T) {
	bp := `
		java_atomslog_library {
			name: "libmyatomslog",
			atoms_module: "my_atoms",
			package_name: "test.package",
			class_name: "MyAtomsLog",
			interface: "invalid",
		}
	`
	prepareForTestWithAtomslogBuildComponents.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"interface: must be one of \\[platform vendor\\]")).
		RunTestWithBp(t, commonBp+bp)
}

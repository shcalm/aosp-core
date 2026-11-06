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

package cc

import (
	"strings"
	"testing"

	"android/soong/android"
)

var prepareForTestWithXom = android.GroupFixturePreparers(
	android.FixtureModifyMockFS(func(fs android.MockFS) {
		rootBp := `
		cc_library {
			name: "libxom_disabled_by_static_dep",
			static_libs: ["libxom_explicit_disabled"],
			split_all_variants: true,
		}
		cc_library {
			name: "libxom_override_static_deps",
			static_libs: ["libxom_explicit_disabled", "libxom_path_disabled"],
			xom: true,
			split_all_variants: true,
		}
		cc_library {
			name: "libxom_disabled_by_static_dep_path",
			static_libs: ["libxom_path_disabled"],
			split_all_variants: true,
		}
		cc_library {
			name: "libxom_static_dep_path_override",
			static_libs: ["libxom_path_override"],
			split_all_variants: true,
		}
		cc_library {
			name: "libxom_explicit_enable",
			xom: true,
			split_all_variants: true,
		}
		cc_library {
			name: "libxom_explicit_disabled",
			xom: false,
			split_all_variants: true,
		}
		cc_library {
			name: "libxom_implicit",
			split_all_variants: true,
		}
		`
		disabledPathBp := `
		cc_library {
			name: "libxom_path_disabled",
			split_all_variants: true,
		}
		cc_library {
			name: "libxom_path_override",
			xom: true,
			split_all_variants: true,
		}
		`

		fs.Merge(android.MockFS{
			"Android.bp":              []byte(rootBp),
			"disable_path/Android.bp": []byte(disabledPathBp),
		})
	}),
)

func TestXomDisableByDependency(t *testing.T) {

	result := android.GroupFixturePreparers(
		prepareForCcTest,
		prepareForTestWithXom,
		android.FixtureModifyProductVariables(func(variables android.FixtureProductVariables) {
			variables.EnableXOM = BoolPtr(true)
			variables.XOMExcludePaths = []string{"disable_path"}
		}),
	).RunTest(t)

	buildOs := "android_arm64_armv8-a"
	shared_suffix := "_shared"

	disabled_by_static_dep := result.ModuleForTests(t, "libxom_disabled_by_static_dep", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]
	disabled_by_static_dep_path := result.ModuleForTests(t, "libxom_disabled_by_static_dep_path", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]
	disabled_by_path := result.ModuleForTests(t, "libxom_path_disabled", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]
	static_dep_xom_path_override := result.ModuleForTests(t, "libxom_static_dep_path_override", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]
	override_static_deps_settings := result.ModuleForTests(t, "libxom_override_static_deps", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]
	if strings.Contains(disabled_by_static_dep, "--execute-only") {
		t.Errorf("XOM not disabled in dependent of a static dep which explicitly disables XOM")
	}
	if strings.Contains(disabled_by_path, "--execute-only") {
		t.Errorf("XOM not disabled in when XOM disabled by path")
	}
	if strings.Contains(disabled_by_static_dep_path, "--execute-only") {
		t.Errorf("XOM not disabled in dependent of a static dep with XOM disabled by path")
	}
	if !strings.Contains(static_dep_xom_path_override, "--execute-only") {
		t.Errorf("XOM is disabled in a dependent of a static dep which should be overriding disabling XOM by path")
	}
	if !strings.Contains(override_static_deps_settings, "--execute-only") {
		t.Errorf("XOM explicitly enabled in a module should be overriding static deps which disable XOM.")
	}
}

func TestXomEnabled(t *testing.T) {
	result := android.GroupFixturePreparers(
		prepareForCcTest,
		prepareForTestWithXom,
		android.FixtureModifyProductVariables(func(variables android.FixtureProductVariables) {
			variables.EnableXOM = BoolPtr(true)
		}),
	).RunTest(t)
	buildOs := "android_arm64_armv8-a"
	shared_suffix := "_shared"

	implicit_enabled := result.ModuleForTests(t, "libxom_implicit", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]
	explicit_disabled := result.ModuleForTests(t, "libxom_explicit_disabled", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]
	explicit_enabled := result.ModuleForTests(t, "libxom_explicit_enable", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]

	if !strings.Contains(implicit_enabled, "--execute-only") {
		t.Errorf("XOM is globally enabled but module is missing the --execute-only linker flag")
	}
	if strings.Contains(explicit_disabled, "--execute-only") {
		t.Errorf("XOM is globally enabled and module explicitly disabling it still emits --execute-only linker flag")
	}
	if !strings.Contains(explicit_enabled, "--execute-only") {
		t.Errorf("XOM is globally enabled and module explicitly enabling it does not emit --execute-only linker flag")
	}
}

func TestXomDisabled(t *testing.T) {
	result := android.GroupFixturePreparers(
		prepareForCcTest,
		prepareForTestWithXom,
		android.FixtureModifyProductVariables(func(variables android.FixtureProductVariables) {
			variables.EnableXOM = BoolPtr(false)
		}),
	).RunTest(t)
	buildOs := "android_arm64_armv8-a"
	shared_suffix := "_shared"

	implicit_disabled := result.ModuleForTests(t, "libxom_implicit", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]
	explicit_disabled := result.ModuleForTests(t, "libxom_explicit_disabled", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]
	explicit_enabled := result.ModuleForTests(t, "libxom_explicit_enable", buildOs+shared_suffix).Rule("ld").Args["ldFlags"]

	if strings.Contains(implicit_disabled, "--execute-only") {
		t.Errorf("XOM is globally disabled but module contains the --execute-only linker flag")
	}
	if strings.Contains(explicit_disabled, "--execute-only") {
		t.Errorf("XOM is globally disabled and module explicitly disabling it still emits --execute-only linker flag")
	}
	if !strings.Contains(explicit_enabled, "--execute-only") {
		t.Errorf("XOM is globally disabled and module explicitly enabling it does not emit --execute-only linker flag")
	}
}

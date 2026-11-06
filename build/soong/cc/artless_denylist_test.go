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

package cc

import (
	"testing"

	"android/soong/android"
)

func TestAllArtlessDenylistDependency(t *testing.T) {
	t.Parallel()

	result := android.GroupFixturePreparers(
		prepareForCcTest,
		android.FixtureAddTextFile("build/soong/cc/Android.bp", `
			all_artless_denylists {
				name: "libandroid_native_denylist",
			}

			all_artless_blocked_symbol_files {
				name: "all_artless_blocked_symbol_files",
			}
		`),
		android.FixtureAddTextFile("Android.bp", `
			ndk_library {
				name: "libfoo",
				first_version: "29",
				symbol_file: "libfoo.map.txt",
			}

			cc_library {
				name: "libfoo",
			}

			ndk_library {
				name: "libbar",
				first_version: "29",
				symbol_file: "libbar.map.txt",
				bypass_artless_denylist: true,
			}

			cc_library {
				name: "libbar",
			}

			cc_library {
			  name: "liblog",
			}

			cc_genrule {
				srcs: [":all_artless_blocked_symbol_files"],
				name: "artless_blocked_symbols_gen",
				out: ["artless_blocked_symbols_gen.cpp"],
				cmd: "gen_blocklist.py $(in) -o $(out)",
				sdk_version: "current",
			}

			cc_library_shared {
				name: "libfoo_blocklist_symbols_lib",
				srcs: [":artless_blocked_symbols_gen"],
				// Force SDK variant when variant-on-demand is enabled.
				sdk_variant_only: true,
				sdk_version: "current",
				stl: "none",
			}
		`),
	).RunTest(t)

	ctx := result.TestContext

	// Check if libandroid_native_denylist depends on libfoo_denylist
	allDenylists := ctx.ModuleForTests(t, "libandroid_native_denylist", "android_arm64_armv8-a_shared").Module()
	libfooDenylist := ctx.ModuleForTests(t, "libfoo_denylist", "android_arm64_armv8-a_static").Module()
	libcDenylist := ctx.ModuleForTests(t, "libc_denylist", "android_arm64_armv8-a_static").Module()
	libmDenylist := ctx.ModuleForTests(t, "libm_denylist", "android_arm64_armv8-a_static").Module()
	libdlDenylist := ctx.ModuleForTests(t, "libdl_denylist", "android_arm64_armv8-a_static").Module()

	android.AssertHasDirectDep(t, ctx, allDenylists, libfooDenylist)
	android.AssertHasDirectDep(t, ctx, allDenylists, libcDenylist)
	android.AssertHasDirectDep(t, ctx, allDenylists, libmDenylist)
	android.AssertHasDirectDep(t, ctx, allDenylists, libdlDenylist)

	denylistSymbolsLib := ctx.ModuleForTests(t, "artless_blocked_symbols_gen", "android_arm64_armv8-a_sdk").Module()
	blocklistFiles := ctx.ModuleForTests(t, "all_artless_blocked_symbol_files", "android_arm64_armv8-a_sdk").Module()
	android.AssertHasDirectDep(t, ctx, blocklistFiles, libfooDenylist)
	android.AssertHasDirectDep(t, ctx, denylistSymbolsLib, blocklistFiles)
}

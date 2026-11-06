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

package filesystem

import (
	"android/soong/android"
	"testing"
)

func setUp(t *testing.T) android.FixturePreparer {
	return android.GroupFixturePreparers(
		PrepareForTestWithAndroidDeviceComponents,
		android.PrepareForTestWithHostTools("conv_linker_config"),
		android.FixtureRegisterWithContext(func(ctx android.RegistrationContext) {
			ctx.RegisterModuleType("android_filesystem", FilesystemFactory)
		}),
	)
}

func TestVendorModuleCheck_Restriction(t *testing.T) {
	t.Helper()

	pattern := "cannot set both PRODUCT_RESTRICT_VENDOR_FILES and VENDOR_PRODUCT_RESTRICT_VENDOR_FILES"

	android.GroupFixturePreparers(
		setUp(t),
	).ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(pattern)).
		RunTestWithBp(t,
			`
				android_device {
					name: "test_device",
					product_restrict_vendor_files: "owner",
					vendor_product_restrict_vendor_files: "owner",
				}
			`)
}
func TestVendorModuleCheck_Owner(t *testing.T) {
	pattern := `vendor module "unknown" in vendor/unknown with unknown owner "unknown" in product`
	android.GroupFixturePreparers(
		setUp(t),
		android.FixtureAddTextFile("vendor/qcom/Android.bp", `
			cc_library {
				name: "allowed",
				owner: "qcom",
				vendor: true,
			}
		`),
		android.FixtureAddTextFile("vendor/unknown/Android.bp", `
			cc_library {
				name: "unknown",
				owner: "unknown",
				vendor: true,
			}
		`),
	).ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(pattern)).
		RunTestWithBp(t, `
			android_filesystem {
				name: "vendor",
				deps: ["allowed", "unknown"],
			}
			android_device {
				name: "test_device",
				vendor_partition_name: "vendor",
				product_restrict_vendor_files: "all",
			}
		`)
}

func TestVendorModuleCheck_Path(t *testing.T) {
	pattern := `vendor module "bad_path" in vendor/acme in product .* being installed to .*/test_device/system/.*, which is not in the vendor, odm, vendor_dlkm, or odm_dlkm tree`
	android.GroupFixturePreparers(
		setUp(t),
		android.FixtureAddTextFile("vendor/acme/Android.bp", `
			cc_library {
				name: "bad_path",
			}
			cc_library {
				name: "good_path",
			}
		`),
	).ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(pattern)).
		RunTestWithBp(t, `
			android_filesystem {
				name: "system",
				deps: ["bad_path"],
			}
			android_device {
				name: "test_device_1",
				vendor_product_restrict_vendor_files: "path",
				system_partition_name: "system",
			}
			android_filesystem {
				name: "vendor",
				deps: ["good_path"],
			}
			android_device {
				name: "test_device_2",
				vendor_product_restrict_vendor_files: "path",
				vendor_partition_name: "vendor",
			}
		`)
}

func TestVendorModuleCheck_Exception(t *testing.T) {
	result := android.GroupFixturePreparers(
		setUp(t),
		android.FixtureAddTextFile("vendor/unknown/Android.bp", `
			cc_library {
				name: "libfoo",
				stubs: {
					symbol_file: "libfoo.map.txt",
				},
				owner: "unknown",
			}
		`),
		android.FixtureAddTextFile("vendor/exempt/path/Android.bp", `
			cc_library {
				name: "libbar",
				owner: "exempt",
			}
		`),
	).RunTestWithBp(t, `
		android_filesystem {
			name: "system",
			deps: ["libfoo", "libbar"],
		}
		android_device {
			name: "test_device",
			vendor_product_restrict_vendor_files: "owner",
			vendor_exception_modules: ["libfoo"],
			vendor_exception_paths: ["exempt/path"],
			system_partition_name: "system",
		}
	`)
	android.FailIfErrored(t, result.Errs)
}

func TestNeedToCheckVendorInfo(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name                  string
		dir                   string
		exceptionPathPrefixes []string
		want                  bool
	}{
		{
			name:                  "not vendor path",
			dir:                   "system/app/MySystemApp",
			exceptionPathPrefixes: []string{},
			want:                  false,
		},
		{
			name:                  "vendor path",
			dir:                   "vendor/app/MyVendorApp",
			exceptionPathPrefixes: []string{},
			want:                  true,
		},
		{
			name:                  "vendor path with exact exception",
			dir:                   "vendor/app/MyVendorApp",
			exceptionPathPrefixes: []string{"vendor/app/MyVendorApp"},
			want:                  false,
		},
		{
			name:                  "vendor path with parent exception",
			dir:                   "vendor/app/MyVendorApp",
			exceptionPathPrefixes: []string{"vendor/app/"},
			want:                  false,
		},
		{
			name:                  "vendor path with different exception",
			dir:                   "vendor/app/MyVendorApp",
			exceptionPathPrefixes: []string{"vendor/other/"},
			want:                  true,
		},
		{
			name:                  "empty dir",
			dir:                   "",
			exceptionPathPrefixes: []string{},
			want:                  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := needToCheckVendorInfo(tc.dir, tc.exceptionPathPrefixes)
			if got != tc.want {
				t.Errorf("needToCheckVendorInfo(%q, %v) = %v, want %v", tc.dir, tc.exceptionPathPrefixes, got, tc.want)
			}
		})
	}
}

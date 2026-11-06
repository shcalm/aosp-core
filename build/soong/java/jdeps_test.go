// Copyright 2018 Google Inc. All rights reserved.
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

package java

import (
	"reflect"
	"testing"

	"android/soong/android"
)

func TestCollectJavaLibraryPropertiesAddLibsDeps(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		java_library {name: "Foo"}
		java_library {name: "Bar"}
		java_library {
			name: "javalib",
			libs: ["Foo", "Bar"],
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	for _, expected := range []string{"Foo", "Bar"} {
		if !android.InList(expected, dpInfo.Deps) {
			t.Errorf("Library.IDEInfo() Deps = %v, %v not found", dpInfo.Deps, expected)
		}
	}
}

func TestCollectJavaLibraryPropertiesAddStaticLibsDeps(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		java_library {name: "Foo"}
		java_library {name: "Bar"}
		java_library {
			name: "javalib",
			static_libs: ["Foo", "Bar"],
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	for _, expected := range []string{"Foo", "Bar"} {
		if !android.InList(expected, dpInfo.Deps) {
			t.Errorf("Library.IDEInfo() Deps = %v, %v not found", dpInfo.Deps, expected)
		}
	}
}

func TestCollectJavaLibraryPropertiesAddScrs(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		java_library {
			name: "javalib",
			srcs: ["Foo.java", "Bar.java"],
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	expected := []string{"Foo.java", "Bar.java"}
	if !reflect.DeepEqual(dpInfo.Srcs, expected) {
		t.Errorf("Library.IDEInfo() Srcs = %v, want %v", dpInfo.Srcs, expected)
	}
}

func TestCollectJavaLibraryPropertiesAddAidlIncludeDirs(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		java_library {
			name: "javalib",
			aidl: {
				include_dirs: ["Foo", "Bar"],
			},
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	expected := []string{"Foo", "Bar"}
	if !reflect.DeepEqual(dpInfo.Aidl_include_dirs, expected) {
		t.Errorf("Library.IDEInfo() Aidl_include_dirs = %v, want %v", dpInfo.Aidl_include_dirs, expected)
	}
}

func TestCollectJavaLibraryPropertiesAddAidlSrcs(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		filegroup {
			name: "my_aidl_files",
			srcs: ["Foo.aidl", "Bar.aidl"],
		}

		java_library {
			name: "javalib",
			srcs: [":my_aidl_files", "Baz.java"],
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	expected := []string{"Foo.aidl", "Bar.aidl"}
	if dpInfo.Aidl == nil {
		t.Fatalf("Library.IDEInfo() Aidl is nil")
	}
	if !reflect.DeepEqual(dpInfo.Aidl.Srcs, expected) {
		t.Errorf("Library.IDEInfo() Aidl.Srcs = %v, want %v", dpInfo.Aidl.Srcs, expected)
	}
}

func TestCollectJavaLibraryPropertiesAddProtoInfo(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		filegroup {
			name: "my_proto_files",
			srcs: ["Foo.proto", "Bar.proto"],
		}

		java_library {
			name: "javalib",
			srcs: [":my_proto_files", "Baz.java"],
			proto: {
				type: "lite",
				canonical_path_from_root: false,
				local_include_dirs: ["proto"],
			},
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	if dpInfo.Proto == nil {
		t.Fatal("Library.IDEInfo() Proto is nil")
	}

	android.AssertStringEquals(t, "Library.IDEInfo() Proto.Type equal to", "lite", dpInfo.Proto.Type)
	if dpInfo.Proto.CanonicalPathFromRoot == nil || *dpInfo.Proto.CanonicalPathFromRoot {
		t.Errorf("Library.IDEInfo() Proto.CanonicalPathFromRoot = %v, want %v", dpInfo.Proto.CanonicalPathFromRoot, false)
	}

	expectedSrcs := []string{"Foo.proto", "Bar.proto"}
	if !reflect.DeepEqual(dpInfo.Proto.Srcs, expectedSrcs) {
		t.Errorf("Library.IDEInfo() Proto.Srcs = %v, want %v", dpInfo.Proto.Srcs, expectedSrcs)
	}

	expectedLocalIncludes := []string{"proto"}
	if !reflect.DeepEqual(dpInfo.Proto.LocalIncludeDirs, expectedLocalIncludes) {
		t.Errorf("Library.IDEInfo() Proto.LocalIncludeDirs = %v, want %v", dpInfo.Proto.LocalIncludeDirs, expectedLocalIncludes)
	}
}

func TestCollectJavaLibraryWithJarJarRules(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		java_library {
			name: "javalib",
			srcs: ["foo.java"],
			jarjar_rules: "jarjar_rules.txt",
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	android.AssertStringEquals(t, "IdeInfo.Srcs of repackaged library should not be empty", "foo.java", dpInfo.Srcs[0])
	android.AssertStringEquals(t, "IdeInfo.Jar_rules of repackaged library should not be empty", "jarjar_rules.txt", dpInfo.Jarjar_rules[0])
	if !android.SubstringInList(dpInfo.Jars, "soong/.intermediates/javalib/android_common/jarjar/turbine/javalib.jar") {
		t.Errorf("IdeInfo.Jars of repackaged library should contain the output of jarjar-ing. All outputs: %v\n", dpInfo.Jars)
	}
}

func TestCollectJavaLibraryLinkingAgainstVersionedSdk(t *testing.T) {
	t.Parallel()
	ctx := android.GroupFixturePreparers(
		prepareForJavaTest,
		FixtureWithPrebuiltApis(map[string][]string{
			"29": {},
		})).RunTestWithBp(t,
		`
		java_library {
			name: "javalib",
			srcs: ["foo.java"],
			sdk_version: "29",
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	android.AssertStringListContains(t, "IdeInfo.Deps should contain versioned sdk module", dpInfo.Deps, "sdk_public_29_android")
}

func TestDoNotAddNoneSystemModulesToDeps(t *testing.T) {
	ctx := android.GroupFixturePreparers(
		prepareForJavaTest,
		android.FixtureMergeEnv(
			map[string]string{
				"DISABLE_STUB_VALIDATION": "true",
			},
		),
	).RunTestWithBp(t,
		`
		java_library {
			name: "javalib",
			srcs: ["foo.java"],
			sdk_version: "none",
			system_modules: "none",
		}

		java_api_library {
			name: "javalib.stubs",
			stubs_type: "everything",
			api_contributions: ["javalib-current.txt"],
			api_surface: "public",
			system_modules: "none",
		}
		java_api_contribution {
			name: "javalib-current.txt",
			api_file: "javalib-current.txt",
			api_surface: "public",
		}
	`)
	javalib := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, javalib)
	android.AssertStringListDoesNotContain(t, "IdeInfo.Deps should contain not contain `none`", dpInfo.Deps, "none")

	javalib_stubs := ctx.ModuleForTests(t, "javalib.stubs", "android_common").Module().(*ApiLibrary)
	dpInfo = getIdeInfo(ctx, javalib_stubs)
	android.AssertStringListDoesNotContain(t, "IdeInfo.Deps should contain not contain `none`", dpInfo.Deps, "none")
}

func TestCollectJavaLibraryPropertiesAddModuleType(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t, `java_library { name: "javalib" }`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)
	android.AssertStringEquals(t, "IdeInfo.ModuleType should be equal to", "java_library", dpInfo.ModuleType)
}

func TestCollectJavaLibraryPropertiesAddManifest(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t, `
		android_library { name: "androidlib", manifest: "lib/AndroidManifest.xml" }
		android_app { name: "androidapp", manifest: "app/AndroidManifest.xml", sdk_version: "current" }
	`)
	module := ctx.ModuleForTests(t, "androidlib", "android_common").Module().(*AndroidLibrary)
	dpInfo := getIdeInfo(ctx, module)
	android.AssertStringEquals(t, "IdeInfo.Manifest should be equal to", "lib/AndroidManifest.xml", dpInfo.Manifest)

	appModule := ctx.ModuleForTests(t, "androidapp", "android_common").Module().(*AndroidApp)
	appDpInfo := getIdeInfo(ctx, appModule)
	android.AssertStringEquals(t, "IdeInfo.Manifest should be equal to", "app/AndroidManifest.xml", appDpInfo.Manifest)
}

func TestCollectJavaLibraryPropertiesAddPackageName(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t, `
		android_library { name: "androidlib", package_name: "com.foo.bar" }
	`)
	module := ctx.ModuleForTests(t, "androidlib", "android_common").Module().(*AndroidLibrary)
	dpInfo := getIdeInfo(ctx, module)
	android.AssertStringEquals(t, "IdeInfo.PackageName should be equal to", "com.foo.bar", dpInfo.PackageName)
}

func getIdeInfo(ctx android.OtherModuleProviderContext, module android.ModuleOrProxy) android.IdeInfo {
	if info, ok := android.OtherModuleProvider(ctx, module, android.CommonModuleInfoProvider); ok && info.IdeInfo != nil {
		return *info.IdeInfo
	}
	return android.IdeInfo{}
}

func TestCollectJavaImportPropertiesAddImportedJars(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		java_import {
			name: "javalib",
			jars: ["foo.jar", "bar.jar"],
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Import)
	dpInfo := getIdeInfo(ctx, module)

	expected := []string{"foo.jar", "bar.jar"}
	if !reflect.DeepEqual(dpInfo.Imported_jars, expected) {
		t.Errorf("Import.IDEInfo() Imported_jars = %v, want %v", dpInfo.Imported_jars, expected)
	}
}
func TestCollectJavaAARImportPropertiesAddImportedAars(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		android_library_import {
			name: "aar_import",
			aars: ["foo.aar"],
		}
	`)
	module := ctx.ModuleForTests(t, "aar_import", "android_common").Module().(*AARImport)
	dpInfo := getIdeInfo(ctx, module)

	expected := []string{"foo.aar"}
	if !reflect.DeepEqual(dpInfo.Imported_aars, expected) {
		t.Errorf("AARImport.IDEInfo() Imported_aars = %v, want %v", dpInfo.Imported_aars, expected)
	}
}

func TestCollectJavaLibraryPropertiesAddAssociates(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
			java_library {
				name: "java_a",
				srcs: ["a.java"],
			}
			java_library {
				name: "java_b",
				srcs: ["b.java"],
				associates: ["java_a"],
			}
		`)
	module := ctx.ModuleForTests(t, "java_b", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	expected := []string{"java_a"}
	if !reflect.DeepEqual(dpInfo.Associates, expected) {
		t.Errorf("Library.IDEInfo() Associates = %v, want %v", dpInfo.Associates, expected)
	}
}

func TestCollectJavaLibraryPropertiesAddKotlincFlags(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		java_library {
			name: "javalib",
			kotlincflags: ["-flag1", "-flag2"],
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	expectedKotlincFlags := []string{"-flag1", "-flag2"}
	if !reflect.DeepEqual(dpInfo.Kotlincflags, expectedKotlincFlags) {
		t.Errorf("Library.IDEInfo() Kotlincflags = %v, want %v", dpInfo.Kotlincflags, expectedKotlincFlags)
	}
}

func TestCollectJavaLibraryPropertiesAddJavacFlags(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		java_library {
			name: "javalib",
			javacflags: ["-flag1", "-flag2"],
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	expectedJavacFlags := []string{"-flag1", "-flag2"}
	if !reflect.DeepEqual(dpInfo.Javacflags, expectedJavacFlags) {
		t.Errorf("Library.IDEInfo() Javacflags = %v, want %v", dpInfo.Javacflags, expectedJavacFlags)
	}
}

func TestCollectJavaLibraryPropertiesAddAnnotationProcessorFlags(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		java_library {
			name: "javalib",
			annotation_processor_flags: ["-apflag1", "-apflag2"],
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	expectedAnnotationProcessorFlags := []string{"-apflag1", "-apflag2"}
	if !reflect.DeepEqual(dpInfo.Annotation_processor_flags, expectedAnnotationProcessorFlags) {
		t.Errorf("Library.IDEInfo() Annotation_processor_flags = %v, want %v", dpInfo.Annotation_processor_flags, expectedAnnotationProcessorFlags)
	}
}

func TestCollectJavaLibraryPropertiesAddPlugins(t *testing.T) {
	t.Parallel()
	ctx, _ := testJava(t,
		`
		java_plugin {
			name: "plugin1",
		}
		java_plugin {
			name: "plugin2",
		}
		java_library {
			name: "javalib",
			plugins: ["plugin1", "plugin2"],
		}
	`)
	module := ctx.ModuleForTests(t, "javalib", "android_common").Module().(*Library)
	dpInfo := getIdeInfo(ctx, module)

	expectedPlugins := []string{"plugin1", "plugin2"}
	if !reflect.DeepEqual(dpInfo.Plugins, expectedPlugins) {
		t.Errorf("Library.IDEInfo() Plugins = %v, want %v", dpInfo.Plugins, expectedPlugins)
	}
}

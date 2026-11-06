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
	"android/soong/android"
	"android/soong/cc"
	"android/soong/java"
)

// Common Android.bp content needed for all tests
const commonBp = `
	filegroup {
		name: "my_atom_protos",
		srcs: ["path/to/my/extension_atoms.proto"],
	}

	filegroup {
		name: "libstats_atom_options_protos",
		srcs: ["some/random/path/atom_field_options.proto"],
	}

	filegroup {
		name: "libprotobuf-internal-descriptor-proto",
		srcs: ["some/protobuf/loc/descriptor.proto"],
	}

	filegroup {
		name: "stats-log-api-gen-java-srcs",
		srcs: ["include_java/StatsHistogram.java"],
		path: "include_java",
	}

	cc_library_static {
		name: "stats-log-api-gen-cc-lib",
		// srcs: ["dummy.cpp"], // Not strictly necessary for dependency resolution
	}

	cc_library_shared {
		name: "libstatssocket",
	}

	cc_library_shared {
		name: "libstatspull",
	}

	cc_library_headers {
		name: "libstatssocket_headers",
	}

	cc_library_headers {
		name: "libstatspull_headers",
	}

	java_library {
		name: "androidx.annotation_annotation",
		// srcs: ["dummy.java"], // Not strictly necessary for dependency resolution
	}

	java_library {
		name: "android.frameworks.stats-V2-java",
	}

	java_library {
		name: "android.frameworks.stats-V3-java",
	}
`

// Prepare the test environment for cc_atomslog_library and java_atomslog_library.
// Registers all the cc/java_atomslog_library_* build components.
var prepareForTestWithAtomslogBuildComponents = android.GroupFixturePreparers(
	cc.PrepareForTestWithCcDefaultModules,
	java.PrepareForTestWithJavaDefaultModules,
	android.FixtureRegisterWithContext(func(ctx android.RegistrationContext) {
		ctx.RegisterModuleType("cc_atomslog_library", CcAtomslogLibraryFactory)
		ctx.RegisterModuleType("cc_atomslog_library_static", CcAtomslogLibraryStaticFactory)
		ctx.RegisterModuleType("cc_atomslog_library_shared", CcAtomslogLibrarySharedFactory)
		ctx.RegisterModuleType("java_atomslog_library", JavaAtomslogLibraryFactory)
	}),
)

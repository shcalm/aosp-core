// Copyright 2015 Google Inc. All rights reserved.
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

package config

import (
	"android/soong/android"
)

type toolchainLFIArm64 struct {
	toolchainLFI
	toolchain64Bit

	toolchainLdflags string
	toolchainCflags  string
}

func (t *toolchainLFIArm64) Name() string {
	return "arm64_lfi"
}

func (t *toolchainLFIArm64) IncludeFlags() string {
	return ""
}

func (t *toolchainLFIArm64) ClangTriple() string {
	return "aarch64_lfi-unknown-linux-android30"
}

func (t *toolchainLFIArm64) Cflags() string {
	return "${config.Arm64Cflags} -mno-outline-atomics"
}

func (t *toolchainLFIArm64) Cppflags() string {
	return "${config.Arm64Cppflags}"
}

func (t *toolchainLFIArm64) Ldflags(ctx ToolchainFlagsContext) FlagsWithDeps {
	return FlagsWithDeps{
		Flags: "${config.Arm64Ldflags}",
	}
}

func (t *toolchainLFIArm64) ToolchainCflags() string {
	return t.toolchainCflags
}

func (t *toolchainLFIArm64) ToolchainLdflags() FlagsWithDeps {
	return FlagsWithDeps{
		Flags: t.toolchainLdflags,
	}
}

func (toolchainLFIArm64) LibclangRuntimeLibraryArch() string {
	return "aarch64"
}

func arm64LFIToolchainFactory(arch android.Arch) Toolchain {
	// Force armv8-a when compiling for lfi, as that's all the lfi compiler supports for now.
	arch = android.Arch{
		ArchType:     android.Arm64,
		ArchVariant:  "armv8-a",
		CpuVariant:   "cortex-a53",
		Abi:          []string{"arm64-v8a"},
		ArchFeatures: []string{"branchprot"},
	}
	toolchainCflags, toolchainLdflags := arm64ToolchainFlags(arch)
	return &toolchainLFIArm64{
		toolchainCflags:  toolchainCflags,
		toolchainLdflags: toolchainLdflags,
	}
}

func init() {
	registerLFIToolchainFactory(android.Android, android.Arm64, arm64LFIToolchainFactory)
}

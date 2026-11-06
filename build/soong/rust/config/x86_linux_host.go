// Copyright 2019 The Android Open Source Project
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
	"path/filepath"
	"strings"

	"android/soong/android"
	cc_config "android/soong/cc/config"
)

var (
	LinuxMuslRustFlags = []string{
		// disable rustc's builtin fallbacks for crt objects
		"-C link_self_contained=no",
		// force rustc to use a dynamic musl libc
		"-C target-feature=-crt-static",
		"-Z link-native-libraries=no",
	}
	LinuxRustLinkFlags = []string{
		"-B${cc_config.ClangBin}",
		"-fuse-ld=lld",
		"-Wl,--undefined-version",
	}
	LinuxRustMuslLinkFlags = []string{
		"--sysroot /dev/null",
		"-nodefaultlibs",
		"-nostdlib",
		"-Wl,--no-dynamic-linker",
	}
	linuxX86Rustflags   = []string{}
	linuxX86Linkflags   = []string{}
	linuxX8664Rustflags = []string{}
	linuxX8664Linkflags = []string{}
)

func LinuxToolchainRustFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	depsPhony := ctx.CreateNinjaPhonyOnce("linuxToolchainRustDeps", []string{
		filepath.Join(LinuxGccRoot(), LinuxGccTriple(), "lib32", "*"),
		filepath.Join(LinuxGccRoot(), LinuxGccTriple(), "lib64", "*"),
		filepath.Join(LinuxGccRoot(), "lib/gcc", LinuxGccTriple(), LinuxGccVersion(), "*"),
		filepath.Join(LinuxGccRoot(), "sysroot/usr/lib/**/*"),
	})
	return cc_config.FlagsWithDeps{
		Flags: strings.Join([]string{
			// These flags are no strictly necessary but included so RBE can discover dependencies.
			"-L${cc_config.LinuxGccRoot}/${cc_config.LinuxGccTriple}/lib32",
			"-L${cc_config.LinuxGccRoot}/${cc_config.LinuxGccTriple}/lib64",
			"-L${cc_config.LinuxGccRoot}/lib/gcc/${cc_config.LinuxGccTriple}/${cc_config.LinuxGccVersion}",
			"-L${cc_config.LinuxGccRoot}/sysroot/usr/lib",
		}, " "),
		Deps: android.Paths{depsPhony},
	}
}

func LinuxRustGlibcLinkFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	depsPhony := ctx.CreateNinjaPhonyOnce("linuxRustGlibcLinkDeps",
		[]string{filepath.Join(LinuxGccRoot(), "sysroot/**/*")})
	return cc_config.FlagsWithDeps{
		Flags: "--sysroot ${cc_config.LinuxGccRoot}/sysroot",
		Deps:  android.Paths{depsPhony},
	}
}

func init() {
	registerToolchainFactory(android.Linux, android.X86_64, linuxGlibcX8664ToolchainFactory)
	registerToolchainFactory(android.Linux, android.X86, linuxGlibcX86ToolchainFactory)

	registerToolchainFactory(android.LinuxMusl, android.X86_64, linuxMuslX8664ToolchainFactory)
	registerToolchainFactory(android.LinuxMusl, android.X86, linuxMuslX86ToolchainFactory)

	pctx.StaticVariable("LinuxMuslToolchainRustFlags", strings.Join(LinuxMuslRustFlags, " "))
	pctx.StaticVariable("LinuxToolchainLinkFlags", strings.Join(LinuxRustLinkFlags, " "))
	pctx.StaticVariable("LinuxMuslToolchainLinkFlags", strings.Join(LinuxRustMuslLinkFlags, " "))
	pctx.StaticVariable("LinuxToolchainX86RustFlags", strings.Join(linuxX86Rustflags, " "))
	pctx.StaticVariable("LinuxToolchainX86LinkFlags", strings.Join(linuxX86Linkflags, " "))
	pctx.StaticVariable("LinuxToolchainX8664RustFlags", strings.Join(linuxX8664Rustflags, " "))
	pctx.StaticVariable("LinuxToolchainX8664LinkFlags", strings.Join(linuxX8664Linkflags, " "))

}

// Base 64-bit linux rust toolchain
type toolchainLinuxX8664 struct {
	toolchain64Bit
	cc_toolchain cc_config.Toolchain
}

func (toolchainLinuxX8664) Supported() bool {
	return true
}

func (toolchainLinuxX8664) Bionic() bool {
	return false
}

func (t *toolchainLinuxX8664) Name() string {
	return "x86_64"
}

func (t *toolchainLinuxX8664) ToolchainLinkFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	// Prepend the lld flags from cc_config so we stay in sync with cc
	preFlags := cc_config.FlagsWithDeps{
		Flags: "${cc_config.LinuxLdflags}",
	}
	postFlags := cc_config.FlagsWithDeps{
		Flags: "${config.LinuxToolchainLinkFlags} ${config.LinuxToolchainX8664LinkFlags}",
	}
	ccFlags := cc_config.LinuxX8664Ldflags(ctx)
	ccFlags.Flags = strings.ReplaceAll(ccFlags.Flags, "${config.", "${cc_config.")
	return preFlags.Append(ccFlags).Append(postFlags)
}

func (t *toolchainLinuxX8664) ToolchainRustFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	return LinuxToolchainRustFlags(ctx).AppendNoDeps("${config.LinuxToolchainX8664RustFlags}")
}

// Specialization of the 64-bit linux rust toolchain for glibc.  Adds the gnu rust triple and
// sysroot linker flags.
type toolchainLinuxGlibcX8664 struct {
	toolchainLinuxX8664
}

func (t *toolchainLinuxX8664) RustTriple() string {
	return "x86_64-unknown-linux-gnu"
}

func (t *toolchainLinuxGlibcX8664) Glibc() bool {
	return true
}

func (t *toolchainLinuxGlibcX8664) ToolchainLinkFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	return t.toolchainLinuxX8664.ToolchainLinkFlags(ctx).Append(LinuxRustGlibcLinkFlags(ctx))
}

func linuxGlibcX8664ToolchainFactory(arch android.Arch) Toolchain {
	return &toolchainLinuxGlibcX8664{
		toolchainLinuxX8664{
			cc_toolchain: cc_config.FindToolchain(android.Linux, arch, false),
		},
	}
}

// Specialization of the 64-bit linux rust toolchain for musl.  Adds the musl rust triple and
// linker flags to avoid using the host sysroot.
type toolchainLinuxMuslX8664 struct {
	toolchainLinuxX8664
}

func (t *toolchainLinuxMuslX8664) RustTriple() string {
	return "x86_64-unknown-linux-musl"
}

func (t *toolchainLinuxMuslX8664) ToolchainLinkFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	extraFlags := cc_config.FlagsWithDeps{
		Flags: "${config.LinuxMuslToolchainLinkFlags}",
	}
	return t.toolchainLinuxX8664.ToolchainLinkFlags(ctx).Append(extraFlags)
}

func (t *toolchainLinuxMuslX8664) ToolchainRustFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	return t.toolchainLinuxX8664.ToolchainRustFlags(ctx).AppendNoDeps("${config.LinuxMuslToolchainRustFlags}")
}

func linuxMuslX8664ToolchainFactory(arch android.Arch) Toolchain {
	return &toolchainLinuxMuslX8664{
		toolchainLinuxX8664{
			cc_toolchain: cc_config.FindToolchain(android.LinuxMusl, arch, false),
		},
	}
}

func (t *toolchainLinuxMuslX8664) Musl() bool {
	return true
}

// Base 32-bit linux rust toolchain
type toolchainLinuxX86 struct {
	toolchain32Bit
	cc_toolchain cc_config.Toolchain
}

func (toolchainLinuxX86) Supported() bool {
	return true
}

func (toolchainLinuxX86) Bionic() bool {
	return false
}

func (t *toolchainLinuxX86) Name() string {
	return "x86"
}

func (toolchainLinuxX86) LibclangRuntimeLibraryArch() string {
	return "i386"
}

func (toolchainLinuxX8664) LibclangRuntimeLibraryArch() string {
	return "x86_64"
}

func (t *toolchainLinuxX86) ToolchainLinkFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	// Prepend the lld flags from cc_config so we stay in sync with cc
	preFlags := cc_config.FlagsWithDeps{
		Flags: "${cc_config.LinuxLdflags}",
	}
	postFlags := cc_config.FlagsWithDeps{
		Flags: "${config.LinuxToolchainLinkFlags} ${config.LinuxToolchainX86LinkFlags}",
	}
	ccFlags := cc_config.LinuxX86Ldflags(ctx)
	ccFlags.Flags = strings.ReplaceAll(ccFlags.Flags, "${config.", "${cc_config.")
	return preFlags.Append(ccFlags).Append(postFlags)
}

func (t *toolchainLinuxX86) ToolchainRustFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	return LinuxToolchainRustFlags(ctx).AppendNoDeps("${config.LinuxToolchainX86RustFlags}")
}

// Specialization of the 32-bit linux rust toolchain for glibc.  Adds the gnu rust triple and
// sysroot linker flags.
type toolchainLinuxGlibcX86 struct {
	toolchainLinuxX86
}

func (t *toolchainLinuxGlibcX86) RustTriple() string {
	return "i686-unknown-linux-gnu"
}

func (t *toolchainLinuxGlibcX86) Glibc() bool {
	return true
}

func (t *toolchainLinuxGlibcX86) ToolchainLinkFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	return t.toolchainLinuxX86.ToolchainLinkFlags(ctx).Append(LinuxRustGlibcLinkFlags(ctx))
}

func linuxGlibcX86ToolchainFactory(arch android.Arch) Toolchain {
	return &toolchainLinuxGlibcX86{
		toolchainLinuxX86{
			cc_toolchain: cc_config.FindToolchain(android.Linux, arch, false),
		},
	}
}

// Specialization of the 32-bit linux rust toolchain for musl.  Adds the musl rust triple and
// linker flags to avoid using the host sysroot.
type toolchainLinuxMuslX86 struct {
	toolchainLinuxX86
}

func (t *toolchainLinuxMuslX86) RustTriple() string {
	return "i686-unknown-linux-musl"
}

func (t *toolchainLinuxMuslX86) ToolchainLinkFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	extraFlags := cc_config.FlagsWithDeps{
		Flags: "${config.LinuxMuslToolchainLinkFlags}",
	}
	return t.toolchainLinuxX86.ToolchainLinkFlags(ctx).Append(extraFlags)
}

func (t *toolchainLinuxMuslX86) ToolchainRustFlags(ctx ToolchainFlagsContext) cc_config.FlagsWithDeps {
	return t.toolchainLinuxX86.ToolchainRustFlags(ctx).AppendNoDeps("${config.LinuxMuslToolchainRustFlags}")
}

func (t *toolchainLinuxMuslX86) Musl() bool {
	return true
}

func linuxMuslX86ToolchainFactory(arch android.Arch) Toolchain {
	return &toolchainLinuxMuslX86{
		toolchainLinuxX86{
			cc_toolchain: cc_config.FindToolchain(android.LinuxMusl, arch, false),
		},
	}
}

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
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	RegisterBuildComponents(android.InitRegistrationContext)
}

func RegisterBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("all_artless_denylists", AllArtlessDenylistsFactory)
	ctx.RegisterModuleType("all_artless_blocked_symbol_files", AllArtlessBlockedSymbolFilesFactory)
}

var (
	// ndkStubGenerator is defined in ndk_library.go.
	genNativeStubSrc = pctx.AndroidStaticRule("genNativeStubSrc",
		blueprint.RuleParams{
			Command: "$ndkStubGenerator --arch $arch --api current " +
				"--api-map $apiMap --artless-denylist $flags $in $out",
			CommandDeps:     []string{"$ndkStubGenerator"},
			SandboxDisabled: true,
		}, "arch", "apiMap", "flags")
)

// Creates a stub static library that denies access to APIs incompatible with
// native-only processes, based on the provided version file.
//
// Note: This module type is only available via direct CreateModule() calls and
// is not available in Android.bp files.
//
// Example:
//
// artless_denylist {
//
//	name: "libfoo_denylist",
//	symbol_file: "libfoo.map.txt",
//
// }
type artlessDenylistLibraryProperties struct {
	// Relative path to the symbol map.
	// See build/soong/docs/map_files.md.
	//
	// If unset, this emits an empty library and empty target for blocked symbol
	// list.
	Symbol_file *string `android:"path"`
}

type artlessDenylistDecorator struct {
	*libraryDecorator

	properties artlessDenylistLibraryProperties

	versionScriptPath     android.ModuleGenPath
	blockedSymbolListPath android.ModuleGenPath
}

func AddArtlessDenylistLibraryCompilerFlags(flags Flags) Flags {
	// All symbols in the stubs library should be visible.
	if inList("-fvisibility=hidden", flags.Local.CFlags) {
		flags.Local.CFlags = append(flags.Local.CFlags, "-fvisibility=default")
	}
	return flags
}

func (stub *artlessDenylistDecorator) compilerFlags(ctx ModuleContext, flags Flags, deps PathDeps) Flags {
	flags = stub.baseCompiler.compilerFlags(ctx, flags, deps)
	if stub.properties.Symbol_file != nil {
		return AddArtlessDenylistLibraryCompilerFlags(flags)
	}
	return flags
}

type artlessDenylistOutputs struct {
	StubSrc           android.ModuleGenPath
	VersionScript     android.ModuleGenPath
	blockedSymbolList android.ModuleGenPath
}

func genNativeStubs(ctx android.ModuleContext, symbolFile string, genstubFlags string) artlessDenylistOutputs {
	stubSrcPath := android.PathForModuleGen(ctx, "stub.c")
	versionScriptPath := android.PathForModuleGen(ctx, "stub.map")
	symbolFilePath := android.PathForModuleSrc(ctx, symbolFile)
	blockedSymbolListPath := android.PathForModuleGen(ctx, "artless_blocked_symbol_list.txt")
	apiLevelsJson := android.GetApiLevelsJson(ctx)
	ctx.Build(pctx, android.BuildParams{
		Rule:        genNativeStubSrc,
		Description: "generate native-only denylist " + symbolFilePath.Rel(),
		Outputs: []android.WritablePath{stubSrcPath, versionScriptPath,
			blockedSymbolListPath},
		Input:     symbolFilePath,
		Implicits: []android.Path{apiLevelsJson},
		Args: map[string]string{
			"arch":   ctx.Arch().ArchType.String(),
			"apiMap": apiLevelsJson.String(),
			"flags":  genstubFlags,
		},
	})

	return artlessDenylistOutputs{
		StubSrc:           stubSrcPath,
		VersionScript:     versionScriptPath,
		blockedSymbolList: blockedSymbolListPath,
	}
}

func (c *artlessDenylistDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	if c.properties.Symbol_file == nil {
		return Objects{}
	}

	if !strings.HasSuffix(String(c.properties.Symbol_file), ".map.txt") {
		ctx.PropertyErrorf("symbol_file", "must end with .map.txt")
	}

	symbolFile := String(c.properties.Symbol_file)
	nativeAbiResult := genNativeStubs(ctx, symbolFile, "")
	objs := CompileStubLibrary(ctx, flags, nativeAbiResult.StubSrc, ctx.getSharedFlags())
	c.blockedSymbolListPath = nativeAbiResult.blockedSymbolList
	c.versionScriptPath = nativeAbiResult.VersionScript
	return objs
}

func (c *artlessDenylistDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	deps = c.libraryDecorator.linkerDeps(ctx, deps)
	if c.properties.Symbol_file != nil {
		deps.HeaderLibs = append(deps.HeaderLibs, "liblog_headers")
	}
	return deps
}

func (stub *artlessDenylistDecorator) extraOutputFilePaths() map[string]android.Paths {
	if stub.properties.Symbol_file == nil {
		return map[string]android.Paths{
			"artless_blocked_symbol_list.txt": nil,
		}
	}
	return map[string]android.Paths{
		"artless_blocked_symbol_list.txt": {stub.blockedSymbolListPath},
	}
}

// artless_denylist creates a static library that redirects functions
// incompatible with native-only app processes to an aborting implementation.
//
// Note: This module type is only available via direct CreateModule() calls and
// is not available in Android.bp files.
func ArtlessDenylistFactory() android.Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyStatic()
	module.stl = nil
	module.sanitize = nil

	stub := &artlessDenylistDecorator{
		libraryDecorator: library,
	}
	module.compiler = stub
	module.linker = stub
	module.installer = stub
	module.library = stub

	module.AddProperties(&stub.properties)

	return module.Init()
}

type allArtlessDenylistsDecorator struct {
	*libraryDecorator
}

func (c *allArtlessDenylistsDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	deps = c.libraryDecorator.linkerDeps(ctx, deps)
	for _, lib := range android.SortedUniqueStrings(*getNDKKnownLibs(ctx.Config())) {
		libName := strings.TrimSuffix(lib, ndkLibrarySuffix)
		deps.WholeStaticLibs = append(deps.WholeStaticLibs, libName+"_denylist")
	}
	deps.SharedLibs = append(deps.SharedLibs, "liblog")
	return deps
}

func (c *allArtlessDenylistsDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = c.libraryDecorator.linkerFlags(ctx, flags)
	flags.Local.LdFlags = append(flags.Local.LdFlags, "-Wl,-z,global")
	return flags
}

func (c *allArtlessDenylistsDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps, objs Objects) android.Path {
	if ctx.ModuleDir() != "build/soong/cc" || ctx.ModuleName() != "libandroid_native_denylist" {
		ctx.ModuleErrorf("There can only be one libandroid_native_denylist in build/soong/cc")
		return nil
	}
	return c.libraryDecorator.link(ctx, flags, deps, objs)
}

func AllArtlessDenylistsFactory() android.Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyShared()
	module.stl = nil
	module.sanitize = nil

	decorator := &allArtlessDenylistsDecorator{
		libraryDecorator: library,
	}
	module.compiler = decorator
	module.linker = decorator
	module.installer = decorator
	module.library = decorator

	return module.Init()
}

type AllArtlessBlockedSymbolFiles struct {
	android.ModuleBase
}

func AllArtlessBlockedSymbolFilesFactory() android.Module {
	module := &AllArtlessBlockedSymbolFiles{}
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibBoth)
	return module
}

type blockedSymbolFileTag struct {
	blueprint.BaseDependencyTag
}

func (c *AllArtlessBlockedSymbolFiles) DepsMutator(ctx android.BottomUpMutatorContext) {
	for _, lib := range android.SortedUniqueStrings(*getNDKKnownLibs(ctx.Config())) {
		libName := strings.TrimSuffix(lib, ndkLibrarySuffix)
		// Ideally, we should be emitting the list of symbols from a sdk variant
		// module, but it is more convenient for now to emit it when the platform
		// denylist (libfoo_denylist) is being generated.
		//
		// Eventually, this should be moved to being generated as a part of
		// libfoo.ndk, which generates the public NDK stubs.
		ctx.AddVariationDependencies([]blueprint.Variation{{Mutator: "sdk", Variation: ""}},
			blockedSymbolFileTag{}, libName+"_denylist")
	}
}

func (c *AllArtlessBlockedSymbolFiles) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if ctx.ModuleDir() != "build/soong/cc" || ctx.ModuleName() != "all_artless_blocked_symbol_files" {
		ctx.ModuleErrorf("There can only be one all_artless_blocked_symbol_files in build/soong/cc")
		return
	}
	var srcs android.Paths
	ctx.VisitDirectDepsProxyWithTag(blockedSymbolFileTag{}, func(proxy android.ModuleProxy) {
		srcs = append(srcs, android.OutputFilesForModule(ctx, proxy, "artless_blocked_symbol_list.txt")...)
	})
	ctx.SetOutputFiles(srcs, "")
}

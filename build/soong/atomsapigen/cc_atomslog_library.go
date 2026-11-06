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
	"android/soong/android"
	"android/soong/cc"
	"fmt"
	"strconv"
	"strings"
)

const (
	includeDefaultLibsFull   = "full"
	includeDefaultLibsHeader = "headers_only"
	includeDefaultLibsNone   = "none"
)

var cppInterfaceApis = []string{"platform", "bootstrap", "vendor"}

type CcAtomslogLibraryProperties struct {
	// Atoms module annotation.
	Atoms_module string

	// C++ namespace where the generated symbols go.
	Namespace string

	// Basename of the header and cpp files generated.
	Basename string

	// Options for including libstatssocket and libstatspull dependencies. There are 3 choices:
	// "full": include libstatssocket and libstatspull as shared_libs,
	// "headers_only": include libstatssocket_headers and libstatspull_headers as header_libs,
	// "none": Nothing is included by default.
	// The default option is "full" unless interface is set to "vendor"; then the default is "none".
	// If "interface" property is set to "vendor", this should be set to "none".
	Include_default_libs string

	// The codegen API to use. Supported APIs are "platform" (default), "vendor", or "bootstrap".
	Interface string

	// Path(s) to proto srcs for atoms. Only one file must define the Atom proto message.
	Proto_srcs []string `android:"path"`

	// Type of logging sources to generate. There are 2 choices:
	// "default": Generate logging methods that have a parameter for each atom field. [default]
	// "typesafe": Generate logging methods that use a struct for creating atom events.
	Gen_type string

	// Version of AIDL lib to use for vendor logging. Default is 2.
	Aidl_version *int64
}

type CcAtomslogLibraryCallbacks struct {
	properties *CcAtomslogLibraryProperties

	genCppDir     android.ModuleGenPath
	genIncludeDir android.ModuleGenPath
	basename      string
}

func CcAtomslogLibraryFactory() android.Module {
	callbacks := &CcAtomslogLibraryCallbacks{
		properties: &CcAtomslogLibraryProperties{},
	}
	return cc.GeneratedCcLibraryModuleFactory(callbacks)
}

func CcAtomslogLibraryStaticFactory() android.Module {
	callbacks := &CcAtomslogLibraryCallbacks{
		properties: &CcAtomslogLibraryProperties{},
	}
	return cc.GeneratedCcLibraryStaticModuleFactory(callbacks)
}

func CcAtomslogLibrarySharedFactory() android.Module {
	callbacks := &CcAtomslogLibraryCallbacks{
		properties: &CcAtomslogLibraryProperties{},
	}
	return cc.GeneratedCcLibrarySharedModuleFactory(callbacks)
}

func (this *CcAtomslogLibraryCallbacks) GeneratorInit(ctx cc.BaseModuleContext) {
}

func (this *CcAtomslogLibraryCallbacks) GeneratorProps() []interface{} {
	return []interface{}{this.properties}
}

func (this *CcAtomslogLibraryCallbacks) GeneratorDeps(ctx cc.DepsContext, deps cc.Deps) cc.Deps {
	// Add the library containing the extra C++ sources like StatsHistogram.
	deps.WholeStaticLibs = append(deps.WholeStaticLibs, "stats-log-api-gen-cc-lib")

	if this.properties.Interface == "vendor" {
		// libstatssocket and libstatspull are not needed for vendor logging.
		// include_default_libs should be either not specified or "none"
		// The aidl lib containing reportVendorAtom is added instead.
		if !android.InList(this.properties.Include_default_libs, []string{"", includeDefaultLibsNone}) {
			ctx.PropertyErrorf(
				"include_default_libs",
				"cannot be set to other than \"%s\" when interface is set to \"vendor\"",
				includeDefaultLibsNone)
		} else {
			aidlVersion := int64(2) // default to V2
			if this.properties.Aidl_version != nil {
				aidlVersion = *this.properties.Aidl_version
			}
			deps.SharedLibs = append(deps.SharedLibs, fmt.Sprintf(aidlLibFmt, aidlVersion, "ndk"))
		}
	} else {
		switch this.properties.Include_default_libs {
		case "":
			fallthrough // "full" is default.
		case includeDefaultLibsFull:
			deps.SharedLibs = append(deps.SharedLibs, "libstatssocket", "libstatspull")
		case includeDefaultLibsHeader:
			deps.HeaderLibs = append(deps.HeaderLibs, "libstatssocket_headers", "libstatspull_headers")
		case includeDefaultLibsNone:
		default:
			ctx.PropertyErrorf(
				"include_default_libs", "must be one of \"%s\", \"%s\", or \"%s\"",
				includeDefaultLibsFull,
				includeDefaultLibsHeader,
				includeDefaultLibsNone)
		}
	}

	if protoSrcs := this.properties.Proto_srcs; protoSrcs != nil {
		this.properties.Proto_srcs = append(protoSrcs, protoDepSrcs...)
	}

	return deps
}

func (this *CcAtomslogLibraryCallbacks) GeneratorSources(ctx cc.ModuleContext) cc.GeneratedSource {
	result := cc.GeneratedSource{}

	atomsModule := this.properties.Atoms_module
	this.basename = this.properties.Basename

	if this.basename == "" {
		if atomsModule == "" {
			ctx.ModuleErrorf("At least one of atoms_module or basename must be provided")
			return result
		} else {
			this.basename = "statslog_" + atomsModule
		}
	}

	// Figure out the generated file paths.
	this.genIncludeDir = android.PathForModuleGen(ctx, "include")
	result.IncludeDirs = []android.Path{this.genIncludeDir}
	result.ReexportedDirs = []android.Path{this.genIncludeDir}

	this.genCppDir = android.PathForModuleGen(ctx)
	result.Sources = []android.Path{this.genCppDir.Join(ctx, this.basename+".cpp")}

	// Put the header file in an include subfolder so the .cpp file can't be included by clients.
	result.Headers = []android.Path{this.genIncludeDir.Join(ctx, this.basename+".h")}

	return result
}

func (this *CcAtomslogLibraryCallbacks) GeneratorFlags(ctx cc.ModuleContext, flags cc.Flags, deps cc.PathDeps) cc.Flags {
	return flags
}

func (this *CcAtomslogLibraryCallbacks) GeneratorBuildActions(ctx cc.ModuleContext, flags cc.Flags, deps cc.PathDeps) {
	cppNamespace := this.properties.Namespace
	if cppNamespace == "" {
		ctx.PropertyErrorf("namespace", "can't be empty")
	}

	ruleBuilder := android.NewRuleBuilder(pctx, ctx).SandboxDisabled().Sbox(this.genCppDir, android.PathForModuleOut(ctx, "cc.sbox.textproto"))
	ruleBuilder.Command().Text("rm -rf").OutputDir("include")
	ruleBuilder.Command().Text("mkdir -p").OutputDir("include")
	hdrGenCmd := ruleBuilder.Command().BuiltTool("stats-log-api-gen").
		FlagWithOutput("--header ", this.genIncludeDir.Join(ctx, this.basename+".h")).
		FlagWithArg("--namespace ", cppNamespace).
		Flag("--omitExtraSrcs")
	cppGenCmd := ruleBuilder.Command().BuiltTool("stats-log-api-gen").
		FlagWithOutput("--cpp ", this.genCppDir.Join(ctx, this.basename+".cpp")).
		FlagWithArg("--namespace ", cppNamespace).
		FlagWithArg("--importHeader ", this.basename+".h").
		Flag("--omitExtraSrcs")

	if atomsModule := this.properties.Atoms_module; atomsModule != "" {
		cppGenCmd.FlagWithArg("--module ", atomsModule)
		hdrGenCmd.FlagWithArg("--module ", atomsModule)
	}

	linkableIntf := ctx.Module().(cc.VersionedLinkableInterface)
	if minSdkStr := linkableIntf.MinSdkVersion(ctx); minSdkStr != "" {
		if minApiLevel, err := cc.NativeApiLevelFromUser(ctx, minSdkStr); err != nil {
			ctx.PropertyErrorf("min_sdk_version", "invalid value")
		} else {
			minApiLevelArg := strconv.Itoa(minApiLevel.FinalInt())
			cppGenCmd.FlagWithArg("--minApiLevel ", minApiLevelArg)
			hdrGenCmd.FlagWithArg("--minApiLevel ", minApiLevelArg)
		}
	}

	if interfaceApi := this.properties.Interface; interfaceApi != "" {
		if android.InList(interfaceApi, cppInterfaceApis) {
			cppGenCmd.FlagWithArg("--interface ", interfaceApi)
			hdrGenCmd.FlagWithArg("--interface ", interfaceApi)
		} else {
			ctx.PropertyErrorf("interface", "must be one of %v", cppInterfaceApis)
		}
	}

	if protoSrcs := this.properties.Proto_srcs; protoSrcs != nil {
		protoPaths := android.PathsForModuleSrc(ctx, protoSrcs)
		argStr := strings.Join(protoPaths.Strings(), " ")
		cppGenCmd.Implicits(protoPaths)
		cppGenCmd.FlagWithArg("--proto ", argStr)
		hdrGenCmd.Implicits(protoPaths)
		hdrGenCmd.FlagWithArg("--proto ", argStr)
	}

	switch this.properties.Gen_type {
	case "":
		fallthrough
	case genTypeDefault:
	case genTypeTypesafe:
		cppGenCmd.Flag("--type-safe")
		hdrGenCmd.Flag("--type-safe")
	default:
		ctx.PropertyErrorf(
			"gen_type", "must be one of \"%s\" or \"%s\"",
			genTypeDefault,
			genTypeTypesafe)
	}

	ruleBuilder.Build("atomslog_cc_generation", "generate .h and .cpp files")
}

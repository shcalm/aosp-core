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
	"android/soong/java"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

var javaInterfaceApis = []string{"platform", "vendor"}

type JavaAtomslogLibraryProperties struct {
	// Atoms module annotation.
	Atoms_module string

	// Java package name for the generated logging class
	Package_name string

	// Class name of the generated logging class. Default is "<atoms_module>StatsLog"
	Class_name string

	// The codegen API to use. Supported APIs are "platform" (default), or "vendor"
	Interface string

	// Path(s) to proto srcs for atoms. Only one file must define the Atom proto message.
	Proto_srcs []string `android:"path"`

	// Add support for logging WorkSource objects.
	Worksource_support bool

	// Generate logging methods without the static modifier.
	Omit_static_modifier bool

	// Type of logging sources to generate. There are 2 choices:
	// "default": Generate logging methods that have a parameter for each atom field. [default]
	// "typesafe": Generate logging methods that use a struct for creating atom events.
	Gen_type string

	// Version of AIDL lib to use for vendor logging. Default is 2.
	Aidl_version *int64

	Extra_srcs []string `android:"path" blueprint:"mutated"`
}

type JavaAtomslogLibraryCallbacks struct {
	properties JavaAtomslogLibraryProperties
}

func JavaAtomslogLibraryFactory() android.Module {
	callbacks := &JavaAtomslogLibraryCallbacks{}
	return java.GeneratedJavaLibraryModuleFactory("java_atomslog_library", callbacks, &callbacks.properties)
}

func (this *JavaAtomslogLibraryCallbacks) DepsMutator(module *java.GeneratedJavaLibraryModule, ctx android.BottomUpMutatorContext) {
	if this.properties.Interface == "vendor" {
		aidlVersion := int64(2) // default to V2
		if this.properties.Aidl_version != nil {
			aidlVersion = *this.properties.Aidl_version
		}
		module.AddStaticLibrary(fmt.Sprintf(aidlLibFmt, aidlVersion, "java"))
	}

	module.AddStaticLibrary("androidx.annotation_annotation")

	this.properties.Extra_srcs = []string{":stats-log-api-gen-java-srcs"}

	if protoSrcs := this.properties.Proto_srcs; protoSrcs != nil {
		this.properties.Proto_srcs = append(protoSrcs, protoDepSrcs...)
	}
}

func (this *JavaAtomslogLibraryCallbacks) GenerateSourceJarBuildActions(module *java.GeneratedJavaLibraryModule, ctx android.ModuleContext) (android.Path, android.Path) {
	atomsModule := this.properties.Atoms_module
	if atomsModule == "" {
		ctx.PropertyErrorf("atoms_module", "can't be empty")
	}

	packageName := this.properties.Package_name
	if packageName == "" {
		ctx.PropertyErrorf("package_name", "can't be empty")
	}

	className := this.properties.Class_name
	if className == "" {
		className = snakeCaseToPascalCase(atomsModule) + "StatsLog"
	}

	srcJarDir := android.PathForModuleGen(ctx)
	srcJarPath := android.PathForModuleGen(ctx, ctx.ModuleName()+".srcjar")

	ruleBuilder := android.NewRuleBuilder(pctx, ctx).SandboxDisabled().Sbox(srcJarDir, android.PathForModuleOut(ctx, "java.sbox.textproto"))
	ruleBuilder.Command().Text("rm -rf").OutputDir(ctx.ModuleName() + ".srcjar.tmp")
	ruleBuilder.Command().Text("mkdir -p").OutputDir(ctx.ModuleName() + ".srcjar.tmp")
	javaGenCmd := ruleBuilder.Command().BuiltTool("stats-log-api-gen").
		FlagWithOutput("--java ", srcJarPath.AddExtension(ctx, "tmp").Join(ctx, className+".java")).
		FlagWithArg("--javaClass ", className).
		FlagWithArg("--javaPackage ", packageName).
		FlagWithArg("--module ", atomsModule).
		Flag("--omitExtraSrcs")

	if minApiLevel := module.MinSdkVersion(ctx); !minApiLevel.IsPreview() && !minApiLevel.IsInvalid() && !minApiLevel.IsCurrent() {
		minApiLevelArg := strconv.Itoa(minApiLevel.FinalInt())
		javaGenCmd.FlagWithArg("--minApiLevel ", minApiLevelArg)
	}

	if this.properties.Omit_static_modifier {
		javaGenCmd.Flag("--nonStatic")
	}

	if this.properties.Worksource_support {
		javaGenCmd.Flag("--worksource")
	}

	if interfaceApi := this.properties.Interface; interfaceApi != "" {
		if android.InList(interfaceApi, javaInterfaceApis) {
			javaGenCmd.FlagWithArg("--interface ", interfaceApi)
		} else {
			ctx.PropertyErrorf("interface", "must be one of %v", javaInterfaceApis)
		}
	}

	if protoSrcs := this.properties.Proto_srcs; protoSrcs != nil {
		protoPaths := android.PathsForModuleSrc(ctx, protoSrcs)
		argStr := strings.Join(protoPaths.Strings(), " ")
		javaGenCmd.Implicits(protoPaths)
		javaGenCmd.FlagWithArg("--proto ", argStr)
	}

	switch this.properties.Gen_type {
	case "":
		fallthrough
	case genTypeDefault:
	case genTypeTypesafe:
		javaGenCmd.Flag("--type-safe")
	default:
		ctx.PropertyErrorf(
			"gen_type", "must be one of \"%s\" or \"%s\"",
			genTypeDefault,
			genTypeTypesafe)
	}

	// Copy extra srcs like StatsHistogram.java to be alongside the generated java for the srcjar.
	extraSrcsPaths := android.PathsForModuleSrc(ctx, this.properties.Extra_srcs)
	ruleBuilder.Command().Text("cp").Inputs(extraSrcsPaths).OutputDir(ctx.ModuleName() + ".srcjar.tmp")

	ruleBuilder.Command().BuiltTool("soong_zip").
		Flag("-srcjar").
		FlagWithOutput("-o ", srcJarPath).
		Flag("-C").OutputDir(ctx.ModuleName() + ".srcjar.tmp").
		Flag("-D").OutputDir(ctx.ModuleName() + ".srcjar.tmp")

	ruleBuilder.Build("atomslog_java_generation", "generate .srcjar file")

	return srcJarPath, nil
}

func (this *JavaAtomslogLibraryCallbacks) IDEInfo(module *java.GeneratedJavaLibraryModule, ctx android.BaseModuleContext, dpInfo *android.IdeInfo) {
}

func snakeCaseToPascalCase(str string) string {
	words := strings.Split(str, "_")
	var builder strings.Builder

	for _, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			rest := string(runes[1:])
			builder.WriteRune(runes[0])
			builder.WriteString(rest)
		}
	}

	return builder.String()
}

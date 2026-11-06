// Copyright 2020 Google Inc. All rights reserved.
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

package android

import (
	"github.com/google/blueprint/proptools"
)

func init() {
	RegisterModuleType("prebuilt_build_tool", NewPrebuiltBuildTool)
	RegisterModuleType("prebuilt_build_tool_defaults", PrebuiltBuildToolsDefaultsFactory)
}

type prebuiltBuildToolProperties struct {
	// Source file to be executed for this build tool
	Src *string `android:"path,arch_variant"`

	// Extra files that should trigger rules using this tool to rebuild
	Deps []string `android:"path,arch_variant"`

	// Create a make variable with the specified name that contains the path to
	// this prebuilt built tool, relative to the root of the source tree.
	Export_to_make_var *string

	// Whether to install the binary in $(HOST_OUT)/bin or not. Defaults to false.
	Installable *bool
}

type prebuiltBuildTool struct {
	DefaultableModuleBase
	ModuleBase
	prebuilt Prebuilt

	properties prebuiltBuildToolProperties

	toolPath OptionalPath
}

func (t *prebuiltBuildTool) Name() string {
	return t.prebuilt.Name(t.ModuleBase.Name())
}

func (t *prebuiltBuildTool) Prebuilt() *Prebuilt {
	return &t.prebuilt
}

func (t *prebuiltBuildTool) DepsMutator(ctx BottomUpMutatorContext) {
	if t.properties.Src == nil {
		if ctx.Config().AllowMissingDependencies() {
			ctx.AddMissingDependencies([]string{"missing_prebuilt_source_file"})
		} else {
			ctx.PropertyErrorf("src", "missing prebuilt source file")
		}
	}
}

func (t *prebuiltBuildTool) GenerateAndroidBuildActions(ctx ModuleContext) {
	if proptools.Bool(t.properties.Installable) && len(t.properties.Deps) > 0 {
		ctx.ModuleErrorf("installable and deps properties cannot be used together")
	}

	sourcePath := t.prebuilt.SingleSourcePath(ctx)
	installedPath := PathForModuleOut(ctx, t.BaseModuleName())
	deps := PathsForModuleSrc(ctx, t.properties.Deps)

	ctx.Build(pctx, BuildParams{
		Rule:      CpExecutableWithBash,
		Output:    installedPath,
		Input:     sourcePath,
		Implicits: deps,
	})

	if proptools.BoolDefault(t.properties.Installable, false) {
		installDir := PathForModuleInstall(ctx, "bin")
		ctx.InstallExecutable(installDir, installedPath.Base(), installedPath)
	} else {
		packagingDir := PathForModuleInstall(ctx, t.BaseModuleName())
		ctx.PackageFile(packagingDir, sourcePath.String(), sourcePath)
		for _, dep := range deps {
			ctx.PackageFile(packagingDir, dep.String(), dep)
		}
	}

	t.toolPath = OptionalPathForPath(installedPath)
}

func (t *prebuiltBuildTool) MakeVars(ctx MakeVarsModuleContext) []ModuleMakeVarsValue {
	if makeVar := String(t.properties.Export_to_make_var); makeVar != "" &&
		t.Target().Os == ctx.Config().BuildOS {
		return []ModuleMakeVarsValue{{makeVar, t.toolPath.String()}}
	}
	return nil
}

func (t *prebuiltBuildTool) HostToolPath() OptionalPath {
	return t.toolPath
}

var _ HostToolProvider = &prebuiltBuildTool{}

// prebuilt_build_tool is to declare prebuilts to be used during the build, particularly for use
// in genrules with the "tools" property.
func NewPrebuiltBuildTool() Module {
	module := &prebuiltBuildTool{}
	module.AddProperties(&module.properties)
	InitSingleSourcePrebuiltModule(module, &module.properties, "Src")
	InitAndroidArchModule(module, HostSupportedNoCross, MultilibFirst)
	InitDefaultableModule(module)

	return module
}

type PrebuiltBuildToolDefaults struct {
	ModuleBase
	DefaultsModuleBase
}

func PrebuiltBuildToolsDefaultsFactory() Module {
	module := &PrebuiltBuildToolDefaults{}
	module.AddProperties(
		&prebuiltBuildToolProperties{},
	)
	InitDefaultsModule(module)
	return module
}

// Copyright (C) 2025 The Android Open Source Project
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

	"github.com/google/blueprint/proptools"
)

func init() {
	RegisterPrebuiltFilesystemComponents(android.InitRegistrationContext)
}

func RegisterPrebuiltFilesystemComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("android_filesystem_prebuilt", PrebuiltFilesystemFactory)
}

type prebuiltFilesystemProperties struct {
	// A prebuilt system image file
	Src *string `android:"path"`
}

type prebuiltFilesystem struct {
	filesystem

	prebuiltProperties prebuiltFilesystemProperties
}

func (p *prebuiltFilesystem) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	input := android.PathForModuleSrc(ctx, proptools.String(p.prebuiltProperties.Src))
	rootDir := android.PathForModuleOut(ctx, p.rootDirString()).OutputPath
	output := android.PathForModuleOut(ctx, p.installFileName())
	p.output = output

	// TODO: implement FilesystemInfo correctly to replace entire android_filesystem.
	fsInfo := FilesystemInfo{
		ModuleName:       ctx.ModuleName(),
		PartitionName:    p.partitionName(),
		RootDir:          rootDir,
		Output:           p.OutputPath(),
		SignedOutputPath: p.SignedOutputPath(),
		Prebuilt:         true,
	}
	android.SetProvider(ctx, FilesystemProvider, fsInfo)

	builder := android.NewRuleBuilder(pctx, ctx).SandboxDisabled()
	rootdirTimestamp := android.PathForModuleOut(ctx, p.rootDirString()+".timestamp")

	switch p.fsType(ctx) {
	case erofsType:
		builder.Command().Textf("rm -rf %s && ", rootDir.String()).BuiltTool("fsck.erofs").Textf("--extract=%s", rootDir.String()).Input(input)
	case ext4Type:
		builder.Command().Textf("rm -rf %s && ", rootDir.String()).BuiltTool("debugfs").Flag("-R").Textf("'rdump / %s'", rootDir.String()).Input(input)
	default:
		ctx.ModuleErrorf("prebuilt filesystem only supports erofs and ext4 but was %q", p.fsType(ctx).String())
		return
	}

	builder.Command().Text("touch").Output(rootdirTimestamp)

	apexKeys := android.PathForModuleOut(ctx, "apexkeys.txt")
	builder.Command().
		Text("for f in $(find").Text(rootDir.String()).Text("-name \"*.apex\" -o -name \"*.capex\" | sort ); do").
		Text("name=$(basename $f); name=${name/.capex/.apex};").
		Text("echo \"name=\\\"$name\\\" public_key=\\\"PRESIGNED\\\" private_key=\\\"PRESIGNED\\\" container_certificate=\\\"PRESIGNED\\\" container_private_key=\\\"PRESIGNED\\\" partition=\\\"" + p.partitionName() + "\\\"\"").
		Text("; done >").Output(apexKeys)

	builder.Build("unpack_prebuilt_filesystem", "unpacking prebuilt filesystem image")

	android.SetProvider(ctx, ApexKeyPathInfoProvider, ApexKeyPathInfo{
		ApexKeyPath: apexKeys,
	})

	ctx.Build(pctx, android.BuildParams{
		Rule:        android.CpRule,
		Description: "install prebuilt filesystem",
		Output:      output,
		Input:       input,
		Implicit:    rootdirTimestamp,
	})
}

func PrebuiltFilesystemFactory() android.Module {
	module := &prebuiltFilesystem{}
	module.filesystemBuilder = module
	module.AddProperties(&module.prebuiltProperties)
	initBaseFilesystemModule(module, &module.filesystem)
	return module
}

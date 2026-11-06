// Copyright 2023 Google Inc. All rights reserved.
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

package aconfig

import (
	"fmt"
	"slices"

	"android/soong/android"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

//go:generate go run ../../blueprint/gobtools/codegen

const allAconfigDeclarationsStorage = "all_aconfig_declarations_storage"

const AllAconfigModule = "all_aconfig_declarations"

// all_aconfig_declarations creates an artifact that records all flags used
// with @FlaggedApi that are currently slated for inclusion in the next API
// finalization.
func AllAconfigDeclarationsFactory() android.Module {
	module := &allAconfigDeclarationsModule{}
	module.AddProperties(&module.properties)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	return module
}

// A singleton that collects all of the aconfig flags declared in the
// tree into a single combined file for export to the external flag setting
// server (inside Google it's Gantry).
//
// Note that this is ALL aconfig_declarations modules present in the tree, not just
// ones that are relevant to the product currently being built, so that that infra
// doesn't need to pull from multiple builds and merge them.
func AllAconfigDeclarationsSingletonFactory() android.Singleton {
	return &allAconfigDeclarationsSingleton{}
}

var aconfigFlagArtifactsDistGoals = []string{
	"docs", "droid", "sdk", "release_config_metadata", "gms",
}

// AllAconfigDeclarationsInfo contains flag storage files containing all flags from all the modules
// across the whole Android source tree. None of these files may be installed on the device.
// They should only be used or consumed as artifacts from the build servers,
// or used by host side tools/tests.
// @auto-generate: gob
type AllAconfigDeclarationsInfo struct {
	// ParsedFlagsFile contains all flags in a binary proto format.
	ParsedFlagsFile android.Path

	// TextProtoFlagsFile contains all flags in a text proto format.
	TextProtoFlagsFile android.Path

	// StorageFlagVal is a "flag_val" storage file for all flags.
	StorageFlagVal android.Path

	// StorageFlagMap is a "flag_map" storage file for all flags.
	StorageFlagMap android.Path

	// StorageFlagInfo is a "flag_info" storage file for all flags.
	StorageFlagInfo android.Path

	// StoragePackageMap is a "package_map" storage file for all flags.
	StoragePackageMap android.Path
}

var AllAconfigDeclarationsInfoProvider = blueprint.NewProvider[AllAconfigDeclarationsInfo]()

type allAconfigReleaseDeclarationsSingleton struct {
	intermediateBinaryProtoPath android.OutputPath
	intermediateTextProtoPath   android.OutputPath

	intermediateStorageFlagVal    android.OutputPath
	intermediateStorageFlagMap    android.OutputPath
	intermediateStorageFlagInfo   android.OutputPath
	intermediateStoragePackageMap android.OutputPath
}

type ApiSurfaceContributorProperties struct {
	Api_signature_files  proptools.Configurable[[]string] `android:"arch_variant,path"`
	Finalized_flags_file string                           `android:"arch_variant,path"`
}

type allAconfigDeclarationsModule struct {
	android.ModuleBase

	properties ApiSurfaceContributorProperties
}

type allAconfigDeclarationsSingleton struct{}

func (this *allAconfigDeclarationsSingleton) sortedConfigNames(releaseMap map[string]allAconfigReleaseDeclarationsSingleton) []string {
	var names []string
	for k := range releaseMap {
		names = append(names, k)
	}
	slices.Sort(names)
	return names
}

func GenerateFinalizedFlagsForApiSurface(ctx android.ModuleContext, outputPath android.WritablePath,
	parsedFlagsFile android.Path, apiSurface ApiSurfaceContributorProperties) {

	apiSignatureFiles := android.Paths{}
	for _, apiSignatureFile := range apiSurface.Api_signature_files.GetOrDefault(ctx, nil) {
		if path := android.PathForModuleSrc(ctx, apiSignatureFile); path != nil {
			apiSignatureFiles = append(apiSignatureFiles, path)
		}
	}
	finalizedFlagsFile := android.PathForModuleSrc(ctx, apiSurface.Finalized_flags_file)

	finalizeNonApiFlags := ctx.Config().GetBuildFlagBool("RELEASE_ACONFIG_FINALIZE_NON_API_FLAGS")

	intermediateMetalavaFlagsConfig := android.PathForModuleOut(ctx, "metalava-flags.config")
	intermediateFlagReport := android.PathForModuleOut(ctx, "metalava-flag-report.csv")
	intermediateNonApiFinalizedFlags := android.PathForModuleOut(ctx, "intermediate-non-api-finalized-flags.txt")
	builder := android.NewRuleBuilder(pctx, ctx).SandboxDisabled()
	builder.Command().
		BuiltTool("aconfig-to-metalava-flags").
		Input(parsedFlagsFile).
		FlagWithOutput("> ", intermediateMetalavaFlagsConfig)
	builder.Command().
		BuiltTool("metalava").
		Flag("flag-report").
		FlagWithInput("--config-file ", intermediateMetalavaFlagsConfig).
		FlagWithOutput("--output-file ", intermediateFlagReport).
		Inputs(apiSignatureFiles)
	if finalizeNonApiFlags {
		builder.Command().
			BuiltTool("finalize-non-api-flags").
			FlagWithInput("--cache ", parsedFlagsFile).
			FlagWithOutput("> ", intermediateNonApiFinalizedFlags)
		builder.Command().
			BuiltTool("record-finalized-flags").
			FlagWithInput("--finalized-flags ", finalizedFlagsFile).
			FlagWithInput("--flag-report ", intermediateFlagReport).
			FlagWithInput("--non-api-finalized-flags ", intermediateNonApiFinalizedFlags).
			FlagWithOutput("> ", outputPath)
	} else {
		builder.Command().
			BuiltTool("record-finalized-flags").
			FlagWithInput("--finalized-flags ", finalizedFlagsFile).
			FlagWithInput("--flag-report ", intermediateFlagReport).
			FlagWithOutput("> ", outputPath)
	}
	builder.Build("finalized-flags", "Record all aconfig flags used with finalized @FlaggedApi APIs")
}

func GenerateExportedFlagCheck(ctx android.ModuleContext, outputPath android.WritablePath,
	parsedFlagsFile android.Path, apiSurface ApiSurfaceContributorProperties) {

	apiSignatureFiles := android.Paths{}
	for _, apiSignatureFile := range apiSurface.Api_signature_files.GetOrDefault(ctx, nil) {
		if path := android.PathForModuleSrc(ctx, apiSignatureFile); path != nil {
			apiSignatureFiles = append(apiSignatureFiles, path)
		}
	}
	finalizedFlagsFile := android.PathForModuleSrc(ctx, apiSurface.Finalized_flags_file)

	ctx.Build(pctx, android.BuildParams{
		Rule:   ExportedFlagCheckRule,
		Inputs: append(apiSignatureFiles, finalizedFlagsFile, parsedFlagsFile),
		Output: outputPath,
		Args: map[string]string{
			"api_signature_files":  android.JoinPathsWithPrefix(apiSignatureFiles, "--api-signature-file "),
			"finalized_flags_file": "--finalized-flags-file " + finalizedFlagsFile.String(),
			"parsed_flags_file":    "--parsed-flags-file " + parsedFlagsFile.String(),
		},
	})
}

func (this *allAconfigDeclarationsModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	parsedFlagsFile := android.PathForIntermediates(ctx, "all_aconfig_declarations.pb")
	finalizedFlags := android.PathForIntermediates(ctx, "finalized-flags.txt")
	GenerateFinalizedFlagsForApiSurface(ctx, finalizedFlags, parsedFlagsFile, this.properties)
	ctx.DistForGoalWithFilename("sdk", finalizedFlags, "finalized-flags.txt")

	depsFiles := android.Paths{finalizedFlags}
	if ctx.Config().GetBuildFlagBool("RELEASE_EXPORTED_FLAG_CHECK") {
		invalidExportedFlags := android.PathForIntermediates(ctx, "invalid_exported_flags.txt")
		GenerateExportedFlagCheck(ctx, invalidExportedFlags, parsedFlagsFile, this.properties)
		depsFiles = append(depsFiles, invalidExportedFlags)
		ctx.Phony("droidcore", invalidExportedFlags)
	}

	ctx.Phony("all_aconfig_declarations", depsFiles...)
	ctx.SetOutputFiles(android.Paths{parsedFlagsFile}, "")

	android.SetProvider(ctx, AllAconfigDeclarationsInfoProvider, AllAconfigDeclarationsInfo{
		ParsedFlagsFile:    parsedFlagsFile,
		TextProtoFlagsFile: android.PathForIntermediates(ctx, "all_aconfig_declarations.textproto"),

		StoragePackageMap: android.PathForIntermediates(ctx, "all_aconfig_declarations.package.map"),
		StorageFlagMap:    android.PathForIntermediates(ctx, "all_aconfig_declarations.flag.map"),
		StorageFlagInfo:   android.PathForIntermediates(ctx, "all_aconfig_declarations.flag.info"),
		StorageFlagVal:    android.PathForIntermediates(ctx, "all_aconfig_declarations.val"),
	})
}

func (this *allAconfigDeclarationsSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	releaseMap := make(map[string]allAconfigReleaseDeclarationsSingleton)
	for _, rcName := range append([]string{""}, ctx.Config().ReleaseAconfigExtraReleaseConfigs()...) {
		// Find all of the aconfig_declarations modules
		var packages = make(map[string]int)
		var cacheFiles android.Paths
		ctx.VisitAllModuleProxies(func(module android.ModuleProxy) {
			decl, ok := android.OtherModuleProvider(ctx, module, android.AconfigReleaseDeclarationsProviderKey)
			if !ok {
				return
			}
			cacheFiles = append(cacheFiles, decl.Data[rcName].IntermediateCacheOutputPath)
			packages[decl.Data[rcName].Package]++
		})

		var numOffendingPkg = 0
		offendingPkgsMessage := ""
		for pkg, cnt := range packages {
			if cnt > 1 {
				offendingPkgsMessage += fmt.Sprintf("%d aconfig_declarations found for package %s\n", cnt, pkg)
				numOffendingPkg++
			}
		}

		if numOffendingPkg > 0 {
			panic("Only one aconfig_declarations allowed for each package.\n" + offendingPkgsMessage)
		}

		// Generate build action for aconfig (binary proto output)
		paths := allAconfigReleaseDeclarationsSingleton{
			intermediateBinaryProtoPath: android.PathForIntermediates(ctx, assembleFileName(rcName, "all_aconfig_declarations.pb")),
			intermediateTextProtoPath:   android.PathForIntermediates(ctx, assembleFileName(rcName, "all_aconfig_declarations.textproto")),

			intermediateStoragePackageMap: android.PathForIntermediates(ctx, assembleFileName(rcName, "all_aconfig_declarations.package.map")),
			intermediateStorageFlagMap:    android.PathForIntermediates(ctx, assembleFileName(rcName, "all_aconfig_declarations.flag.map")),
			intermediateStorageFlagInfo:   android.PathForIntermediates(ctx, assembleFileName(rcName, "all_aconfig_declarations.flag.info")),
			intermediateStorageFlagVal:    android.PathForIntermediates(ctx, assembleFileName(rcName, "all_aconfig_declarations.val")),
		}
		releaseMap[rcName] = paths
		ctx.Build(pctx, android.BuildParams{
			Rule:        AllDeclarationsRule,
			Inputs:      cacheFiles,
			Output:      releaseMap[rcName].intermediateBinaryProtoPath,
			Description: "all_aconfig_declarations",
			Args: map[string]string{
				"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
			},
		})
		ctx.Phony("all_aconfig_declarations", releaseMap[rcName].intermediateBinaryProtoPath)

		// Generate build action for aconfig (text proto output)
		ctx.Build(pctx, android.BuildParams{
			Rule:        AllDeclarationsRuleTextProto,
			Inputs:      cacheFiles,
			Output:      releaseMap[rcName].intermediateTextProtoPath,
			Description: "all_aconfig_declarations_textproto",
			Args: map[string]string{
				"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
			},
		})
		ctx.Phony("all_aconfig_declarations_textproto", releaseMap[rcName].intermediateTextProtoPath)

		storageFilesVersion := ctx.Config().ReleaseAconfigStorageVersion()
		const container = "all_aconfig_declarations"

		ctx.Build(pctx, android.BuildParams{
			Rule:        allDeclarationsRuleStoragePackageMap,
			Inputs:      cacheFiles,
			Output:      releaseMap[rcName].intermediateStoragePackageMap,
			Description: "all_aconfig_declarations_storage_package_map",
			Args: map[string]string{
				"container":   container,
				"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
				"version":     storageFilesVersion,
			},
		})
		ctx.Phony(allAconfigDeclarationsStorage, releaseMap[rcName].intermediateStoragePackageMap)

		ctx.Build(pctx, android.BuildParams{
			Rule:        allDeclarationsRuleStorageFlagMap,
			Inputs:      cacheFiles,
			Output:      releaseMap[rcName].intermediateStorageFlagMap,
			Description: "all_aconfig_declarations_storage_flag_map",
			Args: map[string]string{
				"container":   container,
				"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
				"version":     storageFilesVersion,
			},
		})
		ctx.Phony(allAconfigDeclarationsStorage, releaseMap[rcName].intermediateStorageFlagMap)

		ctx.Build(pctx, android.BuildParams{
			Rule:        allDeclarationsRuleStorageFlagInfo,
			Inputs:      cacheFiles,
			Output:      releaseMap[rcName].intermediateStorageFlagInfo,
			Description: "all_aconfig_declarations_storage_flag_info",
			Args: map[string]string{
				"container":   container,
				"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
				"version":     storageFilesVersion,
			},
		})
		ctx.Phony(allAconfigDeclarationsStorage, releaseMap[rcName].intermediateStorageFlagInfo)

		ctx.Build(pctx, android.BuildParams{
			Rule:        allDeclarationsRuleStorageFlagVal,
			Inputs:      cacheFiles,
			Output:      releaseMap[rcName].intermediateStorageFlagVal,
			Description: "all_aconfig_declarations_storage_flag_val",
			Args: map[string]string{
				"container":   container,
				"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
				"version":     storageFilesVersion,
			},
		})
		ctx.Phony(allAconfigDeclarationsStorage, releaseMap[rcName].intermediateStorageFlagVal)
	}

	for _, rcName := range this.sortedConfigNames(releaseMap) {
		ctx.DistForGoals(aconfigFlagArtifactsDistGoals, releaseMap[rcName].intermediateBinaryProtoPath)
		ctx.DistForGoalsWithFilename(aconfigFlagArtifactsDistGoals, releaseMap[rcName].intermediateBinaryProtoPath, assembleFileName(rcName, "flags.pb"))
		ctx.DistForGoalsWithFilename(aconfigFlagArtifactsDistGoals, releaseMap[rcName].intermediateTextProtoPath, assembleFileName(rcName, "flags.textproto"))
	}
}

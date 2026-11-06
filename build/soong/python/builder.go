// Copyright 2017 Google Inc. All rights reserved.
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

package python

// This file contains Ninja build actions for building Python program.

import (
	"strings"

	"android/soong/android"

	"github.com/google/blueprint"
)

var (
	pctx      = android.NewPackageContext("android/soong/python")
	soongZip  = android.SoongZip
	mergeZips = android.MergeZips
	sed       = android.Sed
	echo      = android.Echo
	chmod     = android.Chmod
	rm        = android.Rm
	dirname   = android.Dirname

	zip = pctx.AndroidStaticRule("zip",
		blueprint.RuleParams{
			Command2: blueprint.NewCommand(soongZip, " -o $out $args"),
		},
		"args")

	combineZip = pctx.AndroidStaticRule("combineZip",
		blueprint.RuleParams{
			Command2: blueprint.NewCommand(mergeZips, " $out $in"),
		},
	)

	hostPar = pctx.AndroidStaticRule("hostPar",
		blueprint.RuleParams{
			Command2: blueprint.NewCommand(
				sed, ` -e 's/%interpreter%/$interp/g' -e 's/%main%/__soong_entrypoint_redirector__.py/g' build/soong/python/scripts/stub_template_host.txt > $out.main && `,
				sed, " -e 's/ENTRY_POINT/$main/g' build/soong/python/scripts/main_non_embedded.py >`", dirname, " $out`/__soong_entrypoint_redirector__.py && ",
				soongZip, " -o $out.entrypoint_zip -C `", dirname, " $out` -f `", dirname, " $out`/__soong_entrypoint_redirector__.py && ",
				echo, ` "#!/usr/bin/env $interp" >${out}.prefix &&`,
				mergeZips, ` -p --prefix ${out}.prefix -pm $out.main $out $srcsZips $out.entrypoint_zip && `,
				chmod, " +x $out && ", rm, " -f $out.main ${out}.prefix $out.entrypoint_zip `", dirname, " $out`/__soong_entrypoint_redirector__.py",
			),
			CommandDeps: []string{"build/soong/python/scripts/stub_template_host.txt", "build/soong/python/scripts/main_non_embedded.py"},
		},
		"interp", "main", "srcsZips")

	embeddedPar = pctx.AndroidStaticRule("embeddedPar",
		blueprint.RuleParams{
			Command2: blueprint.NewCommand(
				rm, ` -f $out.main && `,
				sed, ` 's/ENTRY_POINT/$main/' build/soong/python/scripts/main.py >$out.main && `,
				mergeZips, ` -p -pm $out.main --prefix $launcher $out $srcsZips && `,
				chmod, ` +x $out && `,
				rm, ` -rf $out.main`,
			),
			CommandDeps: []string{"build/soong/python/scripts/main.py"},
		},
		"main", "srcsZips", "launcher")

	embeddedParNoMain = pctx.AndroidStaticRule("embeddedParNoMain",
		blueprint.RuleParams{
			Command2: blueprint.NewCommand(
				mergeZips, " -p --prefix $launcher $out $srcsZips && ",
				chmod, " +x $out",
			),
		},
		"srcsZips", "launcher")

	precompile = pctx.AndroidStaticRule("precompilePython", blueprint.RuleParams{
		Command: `LD_LIBRARY_PATH="$ldLibraryPath" ` +
			`PYTHONPATH=$stdlibZip/internal/$stdlibPkg ` +
			`$launcher build/soong/python/scripts/precompile_python.py $in $out`,
		CommandDeps: []string{
			"$stdlibZip",
			"$launcher",
			"build/soong/python/scripts/precompile_python.py",
		},
	}, "stdlibZip", "stdlibPkg", "launcher", "ldLibraryPath")
)

func init() {
	pctx.Import("android/soong/android")
}

func registerBuildActionForParFile(ctx android.ModuleContext, embeddedLauncher bool,
	launcherPath android.OptionalPath, interpreter, main, binName string,
	srcsZips android.Paths) android.Path {

	// .intermediate output path for bin executable.
	binFile := android.PathForModuleOut(ctx, binName)

	// implicit dependency for parFile build action.
	implicits := srcsZips

	if !embeddedLauncher {
		ctx.Build(pctx, android.BuildParams{
			Rule:        hostPar,
			Description: "host python archive",
			Output:      binFile,
			Implicits:   implicits,
			Args: map[string]string{
				"interp":   strings.Replace(interpreter, "/", `\/`, -1),
				"main":     strings.Replace(strings.TrimSuffix(main, pyExt), "/", ".", -1),
				"srcsZips": strings.Join(srcsZips.Strings(), " "),
			},
		})
	} else if launcherPath.Valid() {
		// added launcherPath to the implicits Ninja dependencies.
		implicits = append(implicits, launcherPath.Path())

		if main == "" {
			ctx.Build(pctx, android.BuildParams{
				Rule:        embeddedParNoMain,
				Description: "embedded python archive",
				Output:      binFile,
				Implicits:   implicits,
				Args: map[string]string{
					"srcsZips": strings.Join(srcsZips.Strings(), " "),
					"launcher": launcherPath.String(),
				},
			})
		} else {
			ctx.Build(pctx, android.BuildParams{
				Rule:        embeddedPar,
				Description: "embedded python archive",
				Output:      binFile,
				Implicits:   implicits,
				Args: map[string]string{
					"main":     strings.Replace(strings.TrimSuffix(main, pyExt), "/", ".", -1),
					"srcsZips": strings.Join(srcsZips.Strings(), " "),
					"launcher": launcherPath.String(),
				},
			})
		}
	}

	return binFile
}

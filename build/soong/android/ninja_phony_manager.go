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
	"slices"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/uniquelist"
)

func init() {
	RegisterParallelSingletonType("ninja_phony_manager", ninjaPhonyManagerFactory)
}

type CreateNinjaPhonyOnceContext interface {
	CreateNinjaPhonyOnce(name string, globs []string) Path
}

// CreateNinjaPhonyOnce creates a ninja phony with the given name and the dependencies
// returned from globbing all files with the given globs. It will only create the phony once even
// if this function is called multiple times. Thus you have to ensure that the mapping
// from name -> globs remains consistent.
//
// This can be used to make the ninja file smaller, by creating aliases for many dependency files.
// Then many different modules can reuse the same alias.
//
// This function returns the PathForPhony(name).
func (m *moduleContext) CreateNinjaPhonyOnce(name string, globs []string) Path {
	if len(globs) == 0 {
		return nil
	}
	depsUnique := uniquelist.Make(globs)
	if m.ninjaPhonies == nil {
		m.ninjaPhonies = make(map[string]NinjaPhoniesGlobsInfo)
	} else if oldDeps, ok := m.ninjaPhonies[name]; ok {
		if oldDeps.Globs != depsUnique {
			m.ModuleErrorf("CreateNinjaPhonyOnce called twice with a different list of deps for phony %q:\n  %s\nand:\n  %s\n",
				name, oldDeps.Globs.ToSlice(), globs)
		}
		return PathForPhony(m, name)
	}
	m.ninjaPhonies[name] = NinjaPhoniesGlobsInfo{depsUnique}
	return PathForPhony(m, name)
}

func ninjaPhonyManagerFactory() Singleton {
	return &ninjaPhonyManager{}
}

type ninjaPhonyManager struct{}

// We use a singleton to actually create the phonies, because in builds with
// AllowMissingDependencies enabled, a module missing dependencies will have all its ctx.Build()
// calls replaced with error rules. So a module with missing dependencies may create a phony,
// and then a module without missing dependencies tries to use it if we create the phonies
// directly in CreateNinjaPhonyOnce().
func (n *ninjaPhonyManager) GenerateBuildActions(ctx SingletonContext) {
	phonies := make(map[string]uniquelist.UniqueList[string])

	ctx.VisitAllModuleProxies(func(proxy ModuleProxy) {
		commoninfo, ok := OtherModuleProvider(ctx, proxy, CommonModuleInfoProvider)
		if !ok {
			return
		}
		for name, deps := range commoninfo.NinjaPhonies {
			if oldDeps, ok := phonies[name]; ok {
				if oldDeps != deps.Globs {
					ctx.Errorf("CreateNinjaPhonyOnce called twice with a different list of deps for phony %q:\n  %s\nand:\n  %s\n",
						name, oldDeps.ToSlice(), deps.Globs.ToSlice())
				}
				continue
			}
			phonies[name] = deps.Globs
		}
	})

	placeholderFile := PathForSource(ctx, "build/soong/android/ninja_phony_manager.go")

	for _, phonyName := range SortedKeys(phonies) {
		globs := phonies[phonyName]
		globResults := make([][]string, 0, globs.Len())
		globResultCount := 0
		for glob := range globs.Iter() {
			globResult, err := ctx.GlobWithDeps(glob, nil)
			if err != nil {
				ctx.Errorf("%s", err)
				continue
			}
			// ignore directories
			globResult = slices.DeleteFunc(globResult, func(p string) bool {
				return strings.HasSuffix(p, "/")
			})
			globResults = append(globResults, globResult)
			globResultCount += len(globResult)
		}
		deps := make([]Path, 0, max(globResultCount, 1))
		for _, globResult := range globResults {
			for _, p := range globResult {
				path, err := pathForSource(ctx, p)
				if err != nil {
					ctx.Errorf("%s", err)
					continue
				}
				deps = append(deps, path)
			}
		}

		if len(deps) == 0 {
			// ninja phonies must have at least one dependency, add a placeholder file as a dep
			// if there are no other deps.
			deps = append(deps, placeholderFile)
		}

		ctx.Build(pctx, BuildParams{
			Rule:   blueprint.Phony,
			Output: PathForPhony(ctx, phonyName),
			Inputs: deps,
		})
	}
}

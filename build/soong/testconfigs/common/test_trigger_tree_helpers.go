// Copyright 2025 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"android/soong/testconfigs/protos"

	"github.com/google/blueprint/pathtools"
)

// AddTestTrigger steps through the tree to the test_trigger's module directory,
// then converts the test_trigger module into its protobuf counterpart, placing it
// as a child within the directory of the tree.
func AddTestTrigger(tree *TestTriggerTree, testTrigger *protos.TestTrigger) {
	// Convert the test_trigger map into a tree structure
	treeRef := tree

	// Step through the tree by the test_trigger's module path.
	for _, subPath := range strings.Split(testTrigger.Path, "/") {
		if _, ok := treeRef.Children[subPath]; !ok {
			treeRef.Children[subPath] = NewTestTriggerTree()
		}
		treeRef = treeRef.Children[subPath]
	}

	treeRef.Configs = append(treeRef.Configs, testTrigger)
}

func extractImports(configs []*protos.TestTrigger) []string {
	imports := make(map[string]bool)
	for _, config := range configs {
		for _, importPath := range config.GetImports() {
			if !strings.HasPrefix(importPath, "..") {
				imports[importPath] = true
			} else {
				importAbs := filepath.Join(config.Path, importPath)
				imports[importAbs] = true
			}
		}
	}
	return slices.Collect(maps.Keys(imports))
}

func extractPathTriggeredConfigs(configs []*protos.TestTrigger, paths ...string) []*protos.TestTrigger {
	triggeredConfigs := []*protos.TestTrigger{}

	for _, config := range configs {
		// Trigger always for empty patterns.
		if len(config.GetFilePatterns()) == 0 {
			triggeredConfigs = append(triggeredConfigs, config)
			continue
		}

		// Trigger if a file pattern matches a provided path.
		for _, filePattern := range config.GetFilePatterns() {
			if matched := slices.ContainsFunc(paths, func(path string) bool {
				matched, _ := pathtools.Match(filePattern, path)
				return matched
			}); matched {
				triggeredConfigs = append(triggeredConfigs, config)
				break
			}
		}
	}

	return triggeredConfigs
}

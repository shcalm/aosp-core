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
	"strings"

	"android/soong/testconfigs/protos"
)

type TestTriggerTree struct {
	Children map[string]*TestTriggerTree

	Configs []*protos.TestTrigger

	visited bool
}

func NewTestTriggerTree() *TestTriggerTree {
	return &TestTriggerTree{
		Children: make(map[string]*TestTriggerTree),
		Configs:  []*protos.TestTrigger{},
		visited:  false,
	}
}

func (tree *TestTriggerTree) GetTriggeredConfigs(configsFound map[string]*protos.TestTrigger, importsOnly bool, paths ...string) {
	if len(paths) == 0 {
		return
	}

	triggeredConfigs := tree.getConfigsFrom(importsOnly, paths...)
	for _, triggeredConfig := range triggeredConfigs {
		configsFound[triggeredConfig.Name] = triggeredConfig
	}

	tree.GetTriggeredConfigs(configsFound, true, extractImports(triggeredConfigs)...)
}

func (tree *TestTriggerTree) getConfigsFrom(importsOnly bool, paths ...string) []*protos.TestTrigger {
	triggeredConfigs := []*protos.TestTrigger{}

	if !tree.visited {
		tree.visited = true
		if importsOnly {
			triggeredConfigs = append(triggeredConfigs, extractPathTriggeredConfigs(tree.Configs)...)
		} else {
			triggeredConfigs = append(triggeredConfigs, extractPathTriggeredConfigs(tree.Configs, paths...)...)
		}
	}

	// Group the paths by top, for each child, recursive call with subset of paths.
	pathsByTop := make(map[string][]string)
	for _, path := range paths {
		pathParts := strings.Split(path, "/")
		if !importsOnly && len(pathParts) == 1 {
			continue
		}
		top := pathParts[0]
		if _, ok := pathsByTop[top]; !ok {
			pathsByTop[top] = []string{}
		}
		if len(pathParts) > 0 {
			pathsByTop[top] = append(pathsByTop[top], strings.Join(pathParts[1:], "/"))
		}
	}
	for subpath, subtree := range tree.Children {
		if paths, ok := pathsByTop[subpath]; ok {
			triggeredConfigs = append(triggeredConfigs, subtree.getConfigsFrom(importsOnly, paths...)...)
		}
	}

	return triggeredConfigs
}

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

package testconfigs

import (
	"maps"
	"slices"

	"android/soong/android"
)

var validTestSuites = map[string]any{
	"general-tests": nil,
	"device-tests":  nil,
}

func isValidTestSuite(testSuite string) bool {
	_, found := validTestSuites[testSuite]
	return found
}

func (zipper *TestConfigZipper) validateTestSuites(ctx android.SingletonContext) {
	validate := func(plan *TestExecutionPlanProperties, moduleType, name string) {
		for testModule := range plan.GetTestModules() {
			if !ctx.Config().AllowMissingDependencies() && !ctx.DeviceConfig().NativeCoverageEnabled() {
				testSuiteInfo, ok := zipper.testModulesTestSuiteInfo[testModule]
				if !ok || testSuiteInfo == nil {
					ctx.Errorf("%s \"%s\": could not find test suite info for module \"%s\"", moduleType, name, testModule)
					continue // Skip to the next module
				}
				if !slices.ContainsFunc(testSuiteInfo.TestSuites, isValidTestSuite) {
					ctx.Errorf("%s \"%s\": referenced test module \"%s\" that is not in a valid test suite: %s", moduleType, name, testModule, maps.Keys(validTestSuites))
				}
			}
		}
	}

	for name, testExecutionPlan := range zipper.testExecutionPlans {
		validate(testExecutionPlan, "test_execution_plan", name)
	}

	for name, testTrigger := range zipper.testTriggers {
		testExecutionPlan := &TestExecutionPlanProperties{
			Tests: testTrigger.TestTriggerInlineProperties.Tests,
		}
		validate(testExecutionPlan, "test_trigger", name)
	}
}

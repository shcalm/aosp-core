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
	"android/soong/android"
	"strings"

	"github.com/google/blueprint"
)

// TestExecutionPlan defines the information to be used when kicking off a test.
// This includes the test modules to be called and the arguments
// associated with a specific testing invocation.
type TestExecutionPlan struct {
	android.ModuleBase

	configProperties TestExecutionPlanProperties
}

//go:generate go run ../../blueprint/gobtools/codegen

// @auto-generate: gob
type TestExecutionPlanProperties struct {
	// List of tests that should be kicked off.
	Tests []ModuleProperties

	// List of arguments to be utilized for this execution plan.
	Args []string

	// The metadata values associated the test execution plan.
	TestExecutionPlanMetadataProperties
}

// @auto-generate: gob
type TestExecutionPlanMetadataProperties struct {
	// List of source code files that is noted for test coverage.
	Code_under_test []string
}

func (metadata *TestExecutionPlanMetadataProperties) IsEmpty() bool {
	return len(metadata.Code_under_test) == 0
}

// @auto-generate: gob
type ModuleProperties struct {
	// Name of the module under testing.
	Module string

	// List of the subsets within the module that should be tested.
	Include []string

	// List of the subsets within the module that should not be tested.
	Exclude []string

	// List of arguments specific to this module under testing.
	Module_args []string
}

func (plan *TestExecutionPlanProperties) Validate(ctx android.ModuleContext) {
	// Check if test modules exist and are in valid suites.
	for _, moduleExecution := range plan.Tests {
		if moduleExecution.Module == "" {
			ctx.ModuleErrorf("module must have a name")
		}
		if !ctx.OtherModuleExists(moduleExecution.Module) {
			if !ctx.Config().AllowMissingDependencies() {
				ctx.ModuleErrorf("failed to find referenced test module %s", moduleExecution.Module)
			}
		}
		validateArgs(ctx, moduleExecution.Module_args)
	}

	// Ensure test args are valid
	validateArgs(ctx, plan.Args)
}

func validateArgs(ctx android.ModuleContext, args []string) {
	for _, arg := range args {
		argSplit := strings.SplitN(arg, "=", 2)
		if len(argSplit) < 2 {
			ctx.ModuleErrorf("contains invalid argument %s, must be format key=value", arg)
		}
	}
}

func (plan *TestExecutionPlanProperties) GetTestModules() map[string]any {
	testModules := map[string]any{}
	for _, moduleExecution := range plan.Tests {
		testModule := moduleExecution.Module
		testModules[testModule] = nil
	}
	return testModules
}

func (plan *TestExecutionPlanProperties) IsEmpty() bool {
	if len(plan.Tests) > 0 ||
		len(plan.Args) > 0 {
		return false
	}
	return true
}

// @auto-generate: gob
type TestExecutionPlanInlinable struct {
	TestExecutionPlanProperties
	Name string
}

var TestExecutionPlanProvider = blueprint.NewProvider[TestExecutionPlanProperties]()

func (plan *TestExecutionPlan) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	plan.configProperties.Validate(ctx)

	// Create provider for TestExecutionPlan information.
	android.SetProvider(ctx, TestExecutionPlanProvider, plan.configProperties)
}

func TestExecutionPlanFactory() android.Module {
	module := &TestExecutionPlan{}
	module.AddProperties(&module.configProperties)
	android.InitAndroidModule(module)
	return module
}

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

	"github.com/google/blueprint"
)

// TestWorkflow describes a one to one relationship between
// and execution and scheduling plan.
type TestWorkflow struct {
	android.ModuleBase

	configProperties TestWorkflowProperties
}

//go:generate go run ../../blueprint/gobtools/codegen

// @auto-generate: gob
type TestWorkflowProperties struct {
	// The execution plan utilized for the test workflow.
	Execution_plan TestExecutionPlanInlinable

	// The scheduling plan utilized for the test workflow.
	Scheduling_plan TestSchedulingPlanInlinable
}

func (workflow *TestWorkflowProperties) Validate(ctx android.ModuleContext) {
	if workflow.Execution_plan.Name == "" {
		ctx.ModuleErrorf("execution_plan must have a name")
	}

	if workflow.Scheduling_plan.Name == "" {
		ctx.ModuleErrorf("scheduling_plan must have a name")
	}

	if workflow.Execution_plan.IsEmpty() && !ctx.OtherModuleExists(workflow.Execution_plan.Name) {
		ctx.ModuleErrorf("failed to find referenced execution_plan %s", workflow.Execution_plan.Name)
	} else {
		workflow.Execution_plan.Validate(ctx)
	}
}

var TestWorkflowProvider = blueprint.NewProvider[TestWorkflowProperties]()

func (workflow *TestWorkflow) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	workflow.configProperties.Validate(ctx)

	// Create provider for TestExecutionPlan information.
	android.SetProvider(ctx, TestWorkflowProvider, workflow.configProperties)
}

func TestWorkflowFactory() android.Module {
	module := &TestWorkflow{}
	module.AddProperties(&module.configProperties)
	android.InitAndroidModule(module)
	return module
}

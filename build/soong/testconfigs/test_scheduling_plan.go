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

// TestSchedulingPlan defines the information necessary to schedule within the CI testing stack.
// Currently acts as a link to ATP test configurations.
type TestSchedulingPlan struct {
	android.ModuleBase

	configProperties TestSchedulingPlanProperties
}

//go:generate go run ../../blueprint/gobtools/codegen

// @auto-generate: gob
type TestSchedulingPlanProperties struct {
	// Test Scheduling Plans that relate to this plan and form a grouping.
	// Restricted to "presubmit" and "postsubmit" test_scheduling_plan modules
	// and not generally available for regular usage.
	Related_plans []string
}

func (plan *TestSchedulingPlanProperties) Validate(ctx android.ModuleContext) {
	if len(plan.Related_plans) > 0 {
		switch ctx.ModuleName() {
		case "presubmit", "postsubmit":
			// Valid
		default:
			ctx.ModuleErrorf("field `Related_plans` is restricted to the presubmit and postsubmit test scheduling plans")
			return
		}
	}

	for _, relatedPlan := range plan.Related_plans {
		if !ctx.OtherModuleExists(relatedPlan) {
			ctx.ModuleErrorf("failed to find related test_scheduling_plan %s", relatedPlan)
		}
	}
}

func (plan *TestSchedulingPlanProperties) IsEmpty() bool {
	return true
}

// @auto-generate: gob
type TestSchedulingPlanInlinable struct {
	TestSchedulingPlanProperties
	Name string
}

var TestSchedulingPlanProvider = blueprint.NewProvider[TestSchedulingPlanProperties]()

func (plan *TestSchedulingPlan) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	plan.configProperties.Validate(ctx)

	// Create provider for TestSchedulingPlan information.
	android.SetProvider(ctx, TestSchedulingPlanProvider, plan.configProperties)
}
func TestSchedulingPlanFactory() android.Module {
	module := &TestSchedulingPlan{}
	module.AddProperties(&module.configProperties)
	android.InitAndroidModule(module)
	return module
}

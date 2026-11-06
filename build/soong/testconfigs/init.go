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

import "android/soong/android"

var pctx = android.NewPackageContext("android/soong/testconfigs")

func init() {
	RegisterTestConfigComponents(android.InitRegistrationContext)
}

func RegisterTestConfigComponents(ctx android.RegistrationContext) {
	// Modules
	ctx.RegisterModuleType("test_trigger", TestTriggerFactory)
	ctx.RegisterModuleType("test_workflow", TestWorkflowFactory)
	ctx.RegisterModuleType("test_execution_plan", TestExecutionPlanFactory)
	ctx.RegisterModuleType("test_scheduling_plan", TestSchedulingPlanFactory)

	// Singletons
	ctx.RegisterSingletonType("test_config_zipper", TestConfigZipperFactory)
}

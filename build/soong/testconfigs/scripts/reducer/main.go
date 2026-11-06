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

package main

import (
	"os"
)

func (reducer *TestConfigReducer) run() error {
	if err := reducer.parse(os.Args[1:]); err != nil {
		println("failed to parse")
		return err
	}

	closer, err := reducer.setup()
	if err != nil {
		return err
	}
	defer closer()

	if err := reducer.load(); err != nil {
		return err
	}

	reducer.TestTriggerTree.GetTriggeredConfigs(reducer.TriggeredConfigs, false, reducer.ModifiedFiles...)

	for _, config := range reducer.TriggeredConfigs {
		if schedulingPlanName := config.GetInline().GetSchedulingPlan().GetName(); schedulingPlanName != "" {
			reducer.TriggeredSchedulingPlans[schedulingPlanName] = reducer.SchedulingPlans[schedulingPlanName]
		}
		for _, workflow := range config.GetList().GetWorkflows() {
			workflow := reducer.TestWorkflows[workflow.GetName()]
			executionPlan := reducer.ExecutionPlans[workflow.GetExecutionPlan().GetName()]
			schedulingPlan := reducer.SchedulingPlans[workflow.GetSchedulingPlan().GetName()]

			reducer.TriggeredWorkflows[workflow.GetName()] = workflow
			reducer.TriggeredExecutionPlans[workflow.GetExecutionPlan().GetName()] = executionPlan
			reducer.TriggeredSchedulingPlans[workflow.GetSchedulingPlan().GetName()] = schedulingPlan
		}
	}

	if err := reducer.write(); err != nil {
		return err
	}

	return nil
}

func main() {
	packager := NewTestConfigReducer()
	if err := packager.run(); err != nil {
		println(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

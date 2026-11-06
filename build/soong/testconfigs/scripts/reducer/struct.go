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
	"android/soong/testconfigs/common"
	"android/soong/testconfigs/protos"
)

type TestConfigReducer struct {
	// Args
	Top       string
	Projects  string
	Filepaths string

	// Parsed Data
	ExecutionPlans        map[string]*protos.TestExecutionPlan
	SchedulingPlans       map[string]*protos.TestSchedulingPlan
	TestWorkflows         map[string]*protos.TestWorkflow
	TestTriggerTree       *common.TestTriggerTree
	ModifiedFiles         []string
	TestConfigsDir        string
	TestConfigsReducedDir string
	DistDir               string

	// Built Data
	TriggeredConfigs         map[string]*protos.TestTrigger
	TriggeredWorkflows       map[string]*protos.TestWorkflow
	TriggeredExecutionPlans  map[string]*protos.TestExecutionPlan
	TriggeredSchedulingPlans map[string]*protos.TestSchedulingPlan
}

func NewTestConfigReducer() *TestConfigReducer {
	return &TestConfigReducer{
		ExecutionPlans:           make(map[string]*protos.TestExecutionPlan),
		SchedulingPlans:          make(map[string]*protos.TestSchedulingPlan),
		TestWorkflows:            make(map[string]*protos.TestWorkflow),
		TestTriggerTree:          common.NewTestTriggerTree(),
		ModifiedFiles:            []string{},
		TriggeredConfigs:         make(map[string]*protos.TestTrigger),
		TriggeredWorkflows:       make(map[string]*protos.TestWorkflow),
		TriggeredExecutionPlans:  make(map[string]*protos.TestExecutionPlan),
		TriggeredSchedulingPlans: make(map[string]*protos.TestSchedulingPlan),
	}
}

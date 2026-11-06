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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/blueprint/pathtools"

	"android/soong/testconfigs/common"
	"android/soong/testconfigs/protos"
)

func (reducer *TestConfigReducer) parse(args []string) error {
	fs := flag.NewFlagSet("reduce-test-configs", flag.ExitOnError)

	fs.StringVar(&reducer.Top, "top", "", "The top level directory")
	fs.StringVar(&reducer.Projects, "projects", "", "The project repositories to consider when loading relevant modified files")
	fs.StringVar(&reducer.Filepaths, "filepaths", "", "The explicit filepaths to consider when reducing test configurations.")

	return fs.Parse(args)
}

func (reducer *TestConfigReducer) setup() (closer func() error, err error) {
	outDirCmd := exec.Command(SoongUiScript, "--dumpvar-mode", "--abs", "OUT_DIR")
	outDirCmd.Dir = reducer.Top

	outDir, err := outDirCmd.Output()
	if err != nil {
		return nil, err
	}
	soongOutDir := filepath.Join(strings.Trim(string(outDir), " \n"), "soong")

	reducer.TestConfigsDir = filepath.Join(soongOutDir, "test-configs")
	reducer.TestConfigsReducedDir = filepath.Join(soongOutDir, "test-configs-reduced")
	if err := os.MkdirAll(reducer.TestConfigsReducedDir, 0755); err != nil {
		return nil, err
	}

	distDirCmd := exec.Command(SoongUiScript, "--dumpvar-mode", "--abs", "DIST_DIR")
	distDirCmd.Dir = reducer.Top

	distDirOut, err := distDirCmd.Output()
	if err != nil {
		return nil, err
	}
	reducer.DistDir = strings.Trim(string(distDirOut), " \n")

	file, err := os.OpenFile(filepath.Join(reducer.DistDir, LogPath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	log.SetOutput(file)

	return file.Close, nil
}

func (reducer *TestConfigReducer) load() error {
	testConfigsPath := filepath.Join(reducer.TestConfigsDir, "test_configs.pb")
	testConfigs := &protos.TestConfigs{}

	loadErrs := map[string]error{
		"execution_plans": common.UnmarshalFile(testConfigsPath, testConfigs),
		"modified_files":  reducer.loadModifiedFiles(),
	}

	var loadErr error
	for name, err := range loadErrs {
		if err != nil {
			loadErr = fmt.Errorf("%w\n%w", fmt.Errorf("error loading %s: %w", name, err), loadErr)
		}
	}
	if loadErr != nil {
		return loadErr
	}

	for _, plan := range testConfigs.GetExecutionPlans() {
		reducer.ExecutionPlans[plan.GetName()] = plan
	}
	for _, plan := range testConfigs.GetSchedulingPlans() {
		reducer.SchedulingPlans[plan.GetName()] = plan
	}
	for _, workflow := range testConfigs.GetWorkflows() {
		reducer.TestWorkflows[workflow.GetName()] = workflow
	}
	for _, trigger := range testConfigs.GetTriggers() {
		if len(trigger.GetList().GetWorkflows()) > 0 {
			workflows := []*protos.TestWorkflow{}
			for _, workflow := range trigger.GetList().GetWorkflows() {
				workflows = append(workflows, reducer.TestWorkflows[workflow.GetName()])
			}
			trigger.GetList().Workflows = workflows
		}
		common.AddTestTrigger(reducer.TestTriggerTree, trigger)
	}

	return nil
}

func (reducer *TestConfigReducer) loadModifiedFiles() error {
	if reducer.Filepaths != "" {
		var err error
		reducer.ModifiedFiles, err = reducer.loadModifiedFilesFromFilepaths()
		if err != nil {
			return err
		}
	} else if changeInfoPath := os.Getenv("CHANGE_INFO"); changeInfoPath != "" {
		var err error
		reducer.ModifiedFiles, err = reducer.loadModifiedFilesFromChangeInfo(changeInfoPath)
		if err != nil {
			return err
		}
	} else {
		var err error
		reducer.ModifiedFiles, err = reducer.loadModifiedFilesFromRepoDiff()
		if err != nil {
			return err
		}
	}

	log.Println("Modified Files:")
	for _, file := range reducer.ModifiedFiles {
		log.Printf("\t%s", file)
	}

	return nil
}

func (reducer *TestConfigReducer) loadModifiedFilesFromFilepaths() ([]string, error) {
	modifiedFiles := []string{}
	for _, fp := range strings.Split(reducer.Filepaths, ",") {
		if pathtools.IsGlob(fp) {
			globResults, err := pathtools.OsFs.Glob(fp, nil, true)
			if err != nil {
				return nil, fmt.Errorf("file pattern error %s: %s", fp, err.Error())
			}
			modifiedFiles = append(modifiedFiles, globResults.Matches...)
		} else {
			modifiedFiles = append(modifiedFiles, fp)
		}
	}
	return modifiedFiles, nil
}

func (reducer *TestConfigReducer) loadModifiedFilesFromRepoDiff() ([]string, error) {
	modifiedFiles := []string{}

	modifiedFilesArgs := []string{"forall"}
	if reducer.Projects != "" {
		modifiedFilesArgs = append(modifiedFilesArgs, strings.Split(reducer.Projects, ",")...)
	}
	modifiedFilesArgs = append(modifiedFilesArgs, "-p", "-c", "git diff --name-only $(git merge-base @{u} HEAD)")

	modifiedFilesCmd := exec.Command("repo", modifiedFilesArgs...)
	modifiedFilesCmd.Dir = reducer.Top

	modifiedFilesOut, _ := modifiedFilesCmd.CombinedOutput()
	project := ""
	for _, line := range strings.Split(string(modifiedFilesOut), "\n") {
		if newProject, switchProject := strings.CutPrefix(line, "project "); switchProject {
			project = newProject
			continue
		}
		if line == "" {
			continue
		}
		modifiedFiles = append(modifiedFiles, filepath.Join(project, line))
	}

	return modifiedFiles, nil
}

func (reducer *TestConfigReducer) loadModifiedFilesFromChangeInfo(path string) ([]string, error) {
	modifiedFiles := map[string]any{}

	changeInfo, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(changeInfo, &data); err != nil {
		return nil, err
	}
	changes, ok := data["changes"].([]any)
	if !ok {
		log.Println("CHANGE_INFO does not contain changes")
		return nil, nil
	}
	for _, change := range changes {
		projectPath, ok := change.(map[string]any)["projectPath"].(string)
		if !ok {
			log.Println("CHANGE_INFO does not specify a project path in the changes")
			continue
		}
		revisions, ok := change.(map[string]any)["revisions"].([]any)
		if !ok {
			log.Println("CHANGE_INFO does not list revisions in the changes")
			continue
		}
		for _, revision := range revisions {
			fileInfos, ok := revision.(map[string]any)["fileInfos"].([]any)
			if !ok {
				log.Println("CHANGE_INFO does not list file infos in the revision")
				continue
			}
			for _, fileInfo := range fileInfos {
				path, ok := fileInfo.(map[string]any)["path"].(string)
				if !ok {
					log.Println("CHANGE_INFO does not specify a path in the file info")
					continue
				}
				fullPath := filepath.Join(projectPath, path)
				modifiedFiles[fullPath] = true
			}
		}
	}

	return slices.Collect(maps.Keys(modifiedFiles)), nil
}

func (reducer *TestConfigReducer) write() error {
	testConfigs := &protos.TestConfigs{
		Triggers:        []*protos.TestTrigger{},
		Workflows:       []*protos.TestWorkflow{},
		SchedulingPlans: []*protos.TestSchedulingPlan{},
		ExecutionPlans:  []*protos.TestExecutionPlan{},
	}

	for _, config := range reducer.TriggeredConfigs {
		testConfigs.Triggers = append(testConfigs.Triggers, config)
	}
	for _, workflow := range reducer.TriggeredWorkflows {
		testConfigs.Workflows = append(testConfigs.Workflows, workflow)
	}
	for _, plan := range reducer.TriggeredExecutionPlans {
		testConfigs.ExecutionPlans = append(testConfigs.ExecutionPlans, plan)
	}
	for _, plan := range reducer.TriggeredSchedulingPlans {
		testConfigs.SchedulingPlans = append(testConfigs.SchedulingPlans, plan)
	}

	writeErr := common.MarshalToFile(testConfigs, filepath.Join(reducer.TestConfigsReducedDir, "test_configs.pb"))
	if writeErr != nil {
		return writeErr
	}

	if err := os.MkdirAll(reducer.DistDir, 0755); err != nil {
		log.Println("Error creating dist dir", err.Error())
		return err
	}
	zipCmd := exec.Command(SoongZip, "-o", filepath.Join(reducer.DistDir, "test-configs-reduced.zip"), "-C", reducer.TestConfigsReducedDir, "-D", reducer.TestConfigsReducedDir)
	zipCmd.Dir = reducer.Top
	zipOut, err := zipCmd.CombinedOutput()
	if err != nil {
		log.Println("Zip Error", string(zipOut))
		return err
	}

	return nil

}

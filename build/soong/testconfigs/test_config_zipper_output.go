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

	"google.golang.org/protobuf/proto"
)

const (
	TestConfigsDir            = "test-configs"
	TestConfigsZip            = "test-configs.zip"
	TestConfigsZipPhonyTarget = "test-configs-zip"
)

func (zipper *TestConfigZipper) writeZip(ctx android.SingletonContext) {
	baseDir := android.PathForOutput(ctx, TestConfigsDir)

	testConfigsPath := baseDir.Join(ctx, "test_configs.pb")
	testConfigsData, err := proto.Marshal(zipper.getTestConfigs())
	if err != nil {
		ctx.Errorf("failed to marshal test configs: %v", err)
	}
	android.WriteFileRuleVerbatim(ctx, testConfigsPath, string(testConfigsData))

	builder := android.NewRuleBuilder(pctx, ctx)
	zipOut := android.PathForOutput(ctx, TestConfigsZip)
	builder.Command().BuiltTool("soong_zip").
		FlagWithOutput("-o ", zipOut).
		FlagWithArg("-C ", baseDir.String()).
		FlagWithArg("-D ", baseDir.String()).
		FlagWithInput("-f ", testConfigsPath)
	builder.Build(TestConfigsZipPhonyTarget, "Creating test-configs.zip")

	ctx.Phony(TestConfigsZipPhonyTarget, zipOut)
	ctx.DistForGoals([]string{TestConfigsZipPhonyTarget, "test_mapping", "dist_files", "apps_only"}, zipOut)
}

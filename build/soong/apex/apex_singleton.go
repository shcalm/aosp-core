/*
 * Copyright (C) 2020 The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package apex

import (
	"encoding/json"

	"android/soong/android"
)

func init() {
	registerApexPrebuiltInfoComponents(android.InitRegistrationContext)
}

func registerApexPrebuiltInfoComponents(ctx android.RegistrationContext) {
	ctx.RegisterParallelSingletonType("apex_prebuiltinfo_singleton", apexPrebuiltInfoFactory)
}

func apexPrebuiltInfoFactory() android.Singleton {
	return &apexPrebuiltInfo{}
}

type apexPrebuiltInfo struct {
	out android.WritablePath
}

func (a *apexPrebuiltInfo) GenerateBuildActions(ctx android.SingletonContext) {
	prebuiltInfos := []android.PrebuiltJsonInfo{}

	ctx.VisitAllModuleProxies(func(m android.ModuleProxy) {
		prebuiltInfo, exists := android.OtherModuleProvider(ctx, m, android.PrebuiltJsonInfoProvider)
		// Use prebuiltInfoProvider to filter out non apex soong modules.
		// Use HideFromMake to filter out the unselected variants of a specific apex.
		if exists && !android.OtherModulePointerProviderOrDefault(ctx, m, android.CommonModuleInfoProvider).HideFromMake {
			prebuiltInfos = append(prebuiltInfos, prebuiltInfo)
		}
	})

	j, err := json.Marshal(prebuiltInfos)
	if err != nil {
		ctx.Errorf("Could not convert prebuilt info of apexes to json due to error: %v", err)
	}
	a.out = android.PathForOutput(ctx, "prebuilt_info.json")
	android.WriteFileRule(ctx, a.out, string(j))
	ctx.DistForGoal("droidcore", a.out)
}

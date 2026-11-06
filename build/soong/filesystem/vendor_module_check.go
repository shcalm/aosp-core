// Copyright 2025 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filesystem

import (
	"fmt"
	"path/filepath"
	"strings"

	"android/soong/android"

	"github.com/google/blueprint/proptools"
)

var (
	_vendor_owner_allowed_list = map[string]bool{
		"asus":          true,
		"audience":      true,
		"atmel":         true,
		"broadcom":      true,
		"csr":           true,
		"elan":          true,
		"fpc":           true,
		"google":        true,
		"htc":           true,
		"huawei":        true,
		"imgtec":        true,
		"invensense":    true,
		"intel":         true,
		"lge":           true,
		"moto":          true,
		"mtk":           true,
		"nvidia":        true,
		"nxp":           true,
		"nxpsw":         true,
		"qcom":          true,
		"qti":           true,
		"samsung":       true,
		"samsung_arm":   true,
		"sony":          true,
		"synaptics":     true,
		"ti":            true,
		"trusted_logic": true,
		"verizon":       true,
		"waves":         true,
		"widevine":      true,
	}
)

func checkOverlays(ctx android.ModuleContext, product string, allOverlays android.Paths, exceptionPrefixes []string) {
	var filteredOverlays android.Paths

	for _, overlay := range allOverlays {
		if needToCheckVendorInfo(overlay.String(), exceptionPrefixes) {
			filteredOverlays = append(filteredOverlays, overlay)
		}
	}

	if len(filteredOverlays) > 0 {
		var overlays []string
		for _, overlay := range filteredOverlays {
			overlays = append(overlays, overlay.String())
		}
		ctx.ModuleErrorf("Policy violation: Product %s has unauthorized vendor overlays: %s",
			product, strings.Join(overlays, " "))
	}
}

func allowPath(installFile string, allowedPrefixes []string) bool {
	for _, allowPrefix := range allowedPrefixes {
		if strings.HasPrefix(installFile, allowPrefix) {
			return true
		}
	}
	return false
}

func checkVendorModuleInstallPaths(ctx android.ModuleContext, modulesToCheck []android.ModuleProxy) {
	// Define the allowed installation path prefixes using paths from the build config.
	allowedPrefixes := []string{
		android.PathForModuleInPartitionInstall(ctx, ctx.DeviceConfig().VendorPath(), "").String() + string(filepath.Separator),
		android.PathForModuleInPartitionInstall(ctx, ctx.DeviceConfig().OdmPath(), "").String() + string(filepath.Separator),
		android.PathForModuleInPartitionInstall(ctx, ctx.DeviceConfig().VendorDlkmPath(), "").String() + string(filepath.Separator),
		android.PathForModuleInPartitionInstall(ctx, ctx.DeviceConfig().OdmDlkmPath(), "").String() + string(filepath.Separator),
		android.PathForHostInstall(ctx, "").String() + string(filepath.Separator),
	}

	for _, mod := range modulesToCheck {
		srcDir := ctx.OtherModuleDir(mod)
		if commonInfo, ok := android.OtherModuleProvider(ctx, mod, android.CommonModuleInfoProvider); ok && commonInfo.InstallFiles != nil {
			info := commonInfo.InstallFiles
			for _, installFile := range info.InstallFiles {
				if !allowPath(installFile.String(), allowedPrefixes) {
					ctx.ModuleErrorf(
						"vendor module %q in %s in product %q being installed to %s, which is not in the vendor, odm, vendor_dlkm, or odm_dlkm tree",
						mod.Name(),
						srcDir,
						ctx.Config().DeviceProduct(),
						installFile,
					)
				}
			}
		}
	}
}

func needToCheckVendorInfo(dir string, exceptionPathPrefixes []string) bool {
	// $(if $(filter vendor/%, $(ALL_MODULES.$(m).PATH)), ...)
	if !strings.HasPrefix(dir, "vendor/") {
		return false
	}

	// $(if $(filter-out $(_vendor_exception_path_prefix), $(ALL_MODULES.$(m).PATH)), $(m))
	for _, exceptionPathPrefix := range exceptionPathPrefixes {
		if strings.HasPrefix(dir, exceptionPathPrefix) {
			return false
		}
	}

	return true
}

func (a *androidDevice) vendorModuleCheck(ctx android.ModuleContext, allInstalledModules []android.ModuleProxy) {
	var restrictions = proptools.String(a.deviceProps.Product_restrict_vendor_files)
	// The new list to hold the transformed paths (simulating _vendor_exception_path_prefix)
	var vendorExceptionPathPrefixes []string
	vendorExceptionModules := make(map[string]bool)
	if restrictions != "" {
		if proptools.String(a.deviceProps.Vendor_product_restrict_vendor_files) != "" {
			ctx.ModuleErrorf("Error: cannot set both PRODUCT_RESTRICT_VENDOR_FILES and VENDOR_PRODUCT_RESTRICT_VENDOR_FILES")
		}
	} else {
		restrictions = proptools.String(a.deviceProps.Vendor_product_restrict_vendor_files)

		for _, path := range a.deviceProps.Vendor_exception_paths {
			// This is equivalent to the Make pattern "vendor/%/"
			transformedPath := "vendor/" + path + string(filepath.Separator)
			vendorExceptionPathPrefixes = append(vendorExceptionPathPrefixes, transformedPath)
		}

		for _, mod := range a.deviceProps.Vendor_exception_modules {
			vendorExceptionModules[mod] = true
		}
	}

	if restrictions == "" {
		return
	}

	outputFile := android.PathForModuleOut(ctx, "vendor_owner_info.txt")
	var contentBuilder strings.Builder
	hasVendorInfo := false

	var needToCheckModules []android.ModuleProxy
	var allOverlays android.Paths

	for _, mod := range allInstalledModules {
		moduleDir := ctx.OtherModuleDir(mod) + string(filepath.Separator)
		if !vendorExceptionModules[mod.Name()] && needToCheckVendorInfo(moduleDir, vendorExceptionPathPrefixes) {
			needToCheckModules = append(needToCheckModules, mod)
		}

		if rroInfo, ok := android.OtherModuleProvider(ctx, mod, android.RROInfoProvider); ok {
			allOverlays = append(allOverlays, rroInfo.Paths...)
		}
	}

	// Restrict owners
	if restrictions == "true" || restrictions == "owner" || restrictions == "all" {
		// Attempt to retrieve the Overlay Info Provider data
		checkOverlays(ctx, ctx.Config().DeviceProduct(), allOverlays, vendorExceptionPathPrefixes)

		for _, mod := range needToCheckModules {
			var owner string
			commonInfo, ok := android.OtherModuleProvider(ctx, mod, android.CommonModuleInfoProvider)
			if ok {
				owner = commonInfo.Owner
			}
			moduleDir := ctx.OtherModuleDir(mod)
			if !_vendor_owner_allowed_list[owner] {
				ctx.ModuleErrorf("Error: vendor module %q in %s with unknown owner %q in product %q",
					mod.Name(), moduleDir, owner, ctx.Config().DeviceProduct())
			}
			if commonInfo != nil && commonInfo.InstallFiles != nil {
				info := commonInfo.InstallFiles
				for _, file := range info.InstallFiles {
					hasVendorInfo = true
					contentBuilder.WriteString(fmt.Sprintf("%s:%s\n", file, owner))
				}
			}
		}
	}

	// Restrict paths
	if restrictions == "path" || restrictions == "all" {
		checkVendorModuleInstallPaths(ctx, needToCheckModules)
	}

	var finalContent string
	if !hasVendorInfo {
		finalContent = "No vendor module owner info.\n"
	} else {
		finalContent = contentBuilder.String()
	}

	android.WriteFileRule(ctx, outputFile, finalContent)

	if !ctx.Config().KatiEnabled() && proptools.Bool(a.deviceProps.Main_device) {
		// Only create the phony and dist target for Soong-only builds.
		ctx.Phony("droidcore", outputFile)
		ctx.DistForGoal("droidcore", outputFile)
	}
}

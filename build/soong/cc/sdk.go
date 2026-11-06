// Copyright 2020 Google Inc. All rights reserved.
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

package cc

import (
	"android/soong/android"
	"android/soong/genrule"
)

// sdkTransitionMutator creates a platform and an SDK variant for modules
// that set sdk_version, and ignores sdk_version for the platform
// variant.  The SDK variant will be used for embedding in APKs
// that may be installed on older platforms.  Apexes use their own
// variants that enforce backwards compatibility.
type sdkTransitionMutator struct{}

func (sdkTransitionMutator) split(ctx android.BaseModuleContext) []string {
	if ctx.Os() != android.Android {
		return []string{""}
	}

	if ctx.Target().LFI {
		return []string{""}
	}

	switch m := ctx.Module().(type) {
	case LinkableInterface:
		if m.AlwaysSdk() {
			if !m.UseSdk() && !m.SplitPerApiLevel() {
				ctx.ModuleErrorf("UseSdk() must return true when AlwaysSdk is set, did the factory forget to set Sdk_version?")
			}
			return []string{"sdk"}
		} else if m.UseSdk() || m.SplitPerApiLevel() {
			return []string{"", "sdk"}
		} else {
			return []string{""}
		}
	case *genrule.Module:
		if p, ok := m.Extra.(*GenruleExtraProperties); ok {
			if String(p.Sdk_version) != "" {
				return []string{"", "sdk"}
			} else {
				return []string{""}
			}
		}
	// AllArtlessBlockedSymbolFiles is an arch-variant file group that "wraps" a
	// list of cc_library targets' tagged outputs. Unlike normal filegroups, it
	// it is OS and arch-variant, and would otherwise get a sdk:"" split that
	// prevents usage from CTS (which is always sdk:"sdk").
	//
	// The list of symbols is a part of API contract and effectively AlwaysSDK,
	// so treat it as such here.
	case *AllArtlessBlockedSymbolFiles:
		return []string{"sdk"}
	}

	return []string{""}
}

func (s sdkTransitionMutator) Split(ctx android.BaseModuleContext) []string {
	allSplits := s.split(ctx)
	if ctx.Config().GetBuildFlagBool("RELEASE_SOONG_SDK_VARIANT_ON_DEMAND") {
		return allSplits[0:1]
	} else {
		return allSplits
	}
}

func (s sdkTransitionMutator) SplitOnDemand(ctx android.BaseModuleContext) []string {
	allSplits := s.split(ctx)
	if len(allSplits) <= 1 || !ctx.Config().GetBuildFlagBool("RELEASE_SOONG_SDK_VARIANT_ON_DEMAND") {
		return nil
	} else {
		return allSplits[1:]
	}
}

func (sdkTransitionMutator) OutgoingTransition(ctx android.OutgoingTransitionContext, sourceVariation string) string {
	if _, ok := ctx.DepTag().(android.UsesUnbundledVariantDepTag); ok {
		return "sdk"
	}
	return sourceVariation
}

func (sdkTransitionMutator) IncomingTransition(ctx android.IncomingTransitionContext, incomingVariation string) string {
	if ctx.Os() != android.Android {
		return ""
	}
	switch m := ctx.Module().(type) {
	case LinkableInterface:
		if m.AlwaysSdk() {
			return "sdk"
		} else if m.UseSdk() || m.SplitPerApiLevel() {
			return incomingVariation
		}
	case *genrule.Module:
		if p, ok := m.Extra.(*GenruleExtraProperties); ok {
			if String(p.Sdk_version) != "" {
				return incomingVariation
			}
		}
	case *AllArtlessBlockedSymbolFiles:
		return "sdk"
	}
	_, usesUnbundledVariantDepTag := ctx.DepTag().(android.UsesUnbundledVariantDepTag)
	// If we've reached this point, the module doesn't have an sdk variant. If we're adding
	// a dependency, we want to pass the sdk variant through to cause a missing dependency error,
	// so that sdk modules can't depend on non-sdk modules and smuggle the use of private apis.
	// However, when the unbundled_builder depends on modules, it wants to prefer the sdk variant
	// but fall back to non-sdk if it doesn't exist. It's ok in this case because the
	// unbundled_builder is just a module for disting other modules, it doesn't have any code of its
	// own.
	if ctx.IsAddingDependency() && !usesUnbundledVariantDepTag {
		return incomingVariation
	} else {
		return ""
	}
}

func (sdkTransitionMutator) Mutate(ctx android.BottomUpMutatorContext, variation string) {
	if ctx.Os() != android.Android {
		return
	}

	switch m := ctx.Module().(type) {
	case VersionedLinkableInterface:
		if m.AlwaysSdk() {
			if variation != "sdk" {
				ctx.ModuleErrorf("tried to create variation %q for module with AlwaysSdk set, expected \"sdk\"", variation)
			}

			m.SetSdkVariant()
		} else if m.UseSdk() || m.SplitPerApiLevel() {
			if variation == "" {
				// Clear the sdk_version property for the platform (non-SDK) variant so later code
				// doesn't get confused by it.
				m.SetSdkVersion(nil)
			} else {
				// Mark the SDK variant.
				m.SetSdkVariant()

				// SDK variant never gets installed because the variant is to be embedded in
				// APKs, not to be installed to the platform.
				m.SetPreventInstall()
			}

			if ctx.Config().HasUnbundledBuildApps() {
				if variation == "" {
					// For an unbundled apps build, hide the platform variant from Make
					// so that other Make modules don't link against it, but against the
					// SDK variant.
					m.SetHideFromMake()
				}
			} else {
				if variation == "sdk" {
					// For a platform build, mark the SDK variant so that it gets a ".sdk" suffix when
					// exposed to Make.
					m.SetSdkAndPlatformVariantVisibleToMake()
				}
			}
		} else {
			// Clear the sdk_version property for modules that don't have an SDK variant so
			// later code doesn't get confused by it.
			m.SetSdkVersion(nil)
		}
	}
}

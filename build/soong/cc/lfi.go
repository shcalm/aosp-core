// Copyright 2026 Google Inc. All rights reserved.
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

	"github.com/google/blueprint"
)

// LTO (Light-weight Fault Isolation) enables in-process sandboxing via an
// assembly rewriting pass that runs during clang
//
// To properly build a module with LTO, the module and all recursive static
// dependencies should be compiled with -flto which directs the compiler to emit
// bitcode rather than native object files. These bitcode files are then passed
// by the linker to the LLVM plugin for compilation at link time. Static
// dependencies not built as bitcode will still function correctly but cannot be
// optimized at link time and may not be compatible with features that require
// LTO, such as CFI.
//
// This file adds support to soong to automatically propagate LTO options to a
// new variant of all static dependencies for each module with LTO enabled.

//go:generate go run ../../blueprint/gobtools/codegen

type LFIProperties struct {
	Lfi struct {
		Enabled     *bool `android:"arch_variant"`
		Stores_only *bool `android:"arch_variant"`
		Use_rlbox   *bool `android:"arch_variant"`
	} `android:"arch_variant"`
}

type Lfi struct {
	Properties        LFIProperties
	MutatedProperties struct {
		LFIEnabled    bool `blueprint:"mutated"`
		LFIStoresOnly bool `blueprint:"mutated"`
		LFIVariation  bool `blueprint:"mutated"`
		LFIUseRlbox   bool `blueprint:"mutated"`
	}
}

type LfiSupportingModule interface {
	GetLfi() *Lfi
}

func (m *Module) GetLfi() *Lfi {
	return m.lfi
}

var _ LfiSupportingModule = (*Module)(nil)

func (lfi *Lfi) Props(isBinary bool) []interface{} {
	// The non-mutated properties should only be added to cc_binary objects.
	// They decide how LFI is built, the libraries just follow what the binary says.
	if isBinary {
		return []interface{}{&lfi.Properties, &lfi.MutatedProperties}
	} else {
		return []interface{}{&lfi.MutatedProperties}
	}
}

func (lfi *Lfi) begin(ctx BaseModuleContext) {
	lfiEnabled := Bool(lfi.Properties.Lfi.Enabled)
	if ctx.Host() || ctx.Arch().ArchType != android.Arm64 || ctx.isSdkVariant() {
		lfiEnabled = false
	}

	if lfiEnabled && !ctx.Module().(*Module).IsLFISupported() {
		ctx.PropertyErrorf("lfi.enabled", "lfi_supported: true must be set if LFI is enabled for the module.")
	}

	if Bool(lfi.Properties.Lfi.Stores_only) {
		ctx.PropertyErrorf("lfi.stores_only", "stores_only mode not supported yet")
	}
	if Bool(lfi.Properties.Lfi.Use_rlbox) {
		ctx.PropertyErrorf("lfi.use_rlbox", "use_rlbox not supported yet")
	}

	lfi.MutatedProperties.LFIEnabled = lfiEnabled
	lfi.MutatedProperties.LFIStoresOnly = Bool(lfi.Properties.Lfi.Stores_only)
	lfi.MutatedProperties.LFIUseRlbox = Bool(lfi.Properties.Lfi.Use_rlbox)
}

func (lfi *Lfi) flags(ctx ModuleContext, flags Flags, deps PathDeps) (Flags, PathDeps) {
	return flags, deps
}

func (lfi *Lfi) deps(ctx DepsContext, deps Deps) Deps {
	if !lfi.MutatedProperties.LFIVariation && lfi.MutatedProperties.LFIEnabled && !ctx.Target().LFI {
		if ctx.static() && !ctx.staticBinary() {
			// the lfi init.c code uses liblog for LFI_VERBOSE=1
			deps.SharedLibs = append(deps.SharedLibs, "liblog")
		}
	}
	return deps
}

func (lfi *Lfi) IsLFIVariation() bool {
	return lfi != nil && lfi.MutatedProperties.LFIVariation
}

// @auto-generate: gob
type lfiInfo struct {
	includeDir android.Path
	header     android.Path
	srcs       android.Paths
}

var lfiInfoProvider = blueprint.NewProvider[lfiInfo]()

type ShouldPropagateLfiDepTag interface {
	ShouldPropagateLfi() bool
}

func lfiPropagateViaDepTag(tag blueprint.DependencyTag) bool {
	if tag == LFIDepTag {
		return true
	}
	libTag, isLibTag := tag.(libraryDependencyTag)
	// Do not recurse down non-static dependencies
	if isLibTag {
		return libTag.static() || libTag.shared() || libTag.header()
	} else if tag == objDepTag || tag == reuseObjTag || tag == staticVariantTag {
		return true
	}

	if t, ok := tag.(ShouldPropagateLfiDepTag); ok && t.ShouldPropagateLfi() {
		return true
	}

	return false
}

// lfiTransitionMutator creates LFI variants of cc modules.  Variant "" is the default variant, which may
// or may not have LFI enabled depending on the config and the module's type and properties.  "lfi-stores-only" or
// "lfi-stores-and-loads" variants are created when a module needs to compile in the non-default state for that module.
type lfiTransitionMutator struct{}

const LFI_STORES_ONLY_VARIATION = "lfi_stores_only"
const LFI_STORES_AND_LOADS_VARIATION = "lfi_stores_and_loads"

var LFIDepTag = dependencyTag{name: "lfi"}

func (l *lfiTransitionMutator) SplitOnDemand(ctx android.BaseModuleContext) []string {
	return nil
}

func (l *lfiTransitionMutator) Split(ctx android.BaseModuleContext) []string {
	m, ok := ctx.Module().(*Module)
	if !ok || m.lfi == nil {
		return []string{""}
	}

	if m.IsSdkVariant() {
		return []string{""}
	}

	if m.lfi.MutatedProperties.LFIEnabled && ctx.Target().LFI {
		if m.lfi.MutatedProperties.LFIStoresOnly {
			return []string{LFI_STORES_ONLY_VARIATION}
		} else {
			return []string{LFI_STORES_AND_LOADS_VARIATION}
		}
	}

	return []string{""}
}

func (l *lfiTransitionMutator) OutgoingTransition(ctx android.OutgoingTransitionContext, sourceVariation string) string {
	if ctx.DepTag() == LFIDepTag {
		return LFI_STORES_AND_LOADS_VARIATION
	}
	if m, ok := ctx.Module().(LfiSupportingModule); ok && m.GetLfi() != nil {
		if !lfiPropagateViaDepTag(ctx.DepTag()) {
			return ""
		}

		return sourceVariation
	}
	return ""
}

func (l *lfiTransitionMutator) IncomingTransition(ctx android.IncomingTransitionContext, incomingVariation string) string {
	if !ctx.Target().LFI {
		return ""
	}
	if m, ok := ctx.Module().(LfiSupportingModule); ok && m.GetLfi() != nil {
		return incomingVariation
	}
	return ""
}

func (l *lfiTransitionMutator) Mutate(ctx android.BottomUpMutatorContext, variation string) {
	mod := ctx.Module()
	if !mod.Enabled(ctx) {
		return
	}

	if ctx.Target().LFI && variation == "" {
		mod.Disable()
		return
	}

	var lfi *Lfi
	if m, ok := ctx.Module().(LfiSupportingModule); ok {
		lfi = m.GetLfi()
	}
	if lfi == nil {
		if ctx.Target().LFI {
			ctx.Module().Disable()
		}
		return
	}

	// If a module is lfi_enabled: true, disable its non-lfi variants, as they likely won't
	// compile after taking into account all the other lfi-specific properties set on it.
	if lfi.MutatedProperties.LFIEnabled && variation == "" {
		mod.Disable()
		return
	}

	lfi.MutatedProperties.LFIVariation = variation != ""

	if m, ok := ctx.Module().(*Module); ok {
		if m.IsSdkVariant() {
			if ctx.Target().LFI {
				mod.Disable()
			}
			return
		}
	}

	if lfi.MutatedProperties.LFIVariation {
		if variation == LFI_STORES_ONLY_VARIATION {
			lfi.MutatedProperties.LFIStoresOnly = true
		}
		mod.SkipInstall()
		mod.HideFromMake()
	}
}

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

package android

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

//go:generate go run ../../blueprint/gobtools/codegen

// The contributions to the dist.
type distContributions struct {
	// Path to license metadata file.
	licenseMetadataFile Path
	// List of goals and the dist copy instructions.
	copiesForGoals []*copiesForGoals
}

// getCopiesForGoals returns a copiesForGoals into which copy instructions that
// must be processed when building one or more of those goals can be added.
func (d *distContributions) getCopiesForGoals(goals []string) *copiesForGoals {
	copiesForGoals := &copiesForGoals{goals: goals}
	d.copiesForGoals = append(d.copiesForGoals, copiesForGoals)
	return copiesForGoals
}

// Associates a list of dist copy instructions with a set of goals for which they
// should be run.
type copiesForGoals struct {
	// goals are build targets that will trigger the copy instructions.
	goals []string

	// A list of instructions to copy a module's output files to somewhere in the
	// dist directory.
	copies []distCopy
}

// Adds a copy instruction.
func (d *copiesForGoals) addCopyInstruction(from Path, dest string) {
	d.copies = append(d.copies, distCopy{from, dest})
}

// Instruction on a path that must be copied into the dist.
// @auto-generate: gob
type distCopy struct {
	// The path to copy from.
	from Path

	// The destination within the dist directory to copy to.
	dest string
}

func (d *distCopy) String() string {
	if len(d.dest) == 0 {
		return d.from.String()
	}
	return fmt.Sprintf("%s:%s", d.from.String(), d.dest)
}

type distCopies []distCopy

func (d *distCopies) Strings() (ret []string) {
	if d == nil {
		return
	}
	for _, dist := range *d {
		ret = append(ret, dist.String())
	}
	return
}

// This gets the dist contributions from the given module that were specified in the Android.bp
// file using the dist: property. It does not include contributions that the module's
// implementation may have defined with ctx.DistForGoals(), for that, see DistProvider.
func getDistContributions(ctx ConfigAndOtherModuleProviderContext, mod ModuleOrProxy) *distContributions {
	name := mod.Name()

	commonInfo := OtherModulePointerProviderOrDefault(ctx, mod, CommonModuleInfoProvider)

	info := GetInstallFilesCommon(commonInfo)
	availableTaggedDists := info.DistFiles

	if len(availableTaggedDists) == 0 {
		// Nothing dist-able for this module.
		return nil
	}

	// Collate the contributions this module makes to the dist.
	distContributions := &distContributions{}

	if !exemptFromRequiredApplicableLicensesProperty(mod) {
		distContributions.licenseMetadataFile = info.LicenseMetadataFile
	}

	// Iterate over this module's dist structs, merged from the dist and dists properties.
	for _, dist := range commonInfo.Dists {
		// Get the list of goals this dist should be enabled for. e.g. sdk, droidcore
		goals := dist.Targets

		// Get the tag representing the output files to be dist'd. e.g. ".jar", ".proguard_map"
		var tag string
		if dist.Tag == nil {
			// If the dist struct does not specify a tag, use the default output files tag.
			tag = DefaultDistTag
		} else {
			tag = *dist.Tag
		}

		// Get the paths of the output files to be dist'd, represented by the tag.
		// Can be an empty list.
		tagPaths := availableTaggedDists[tag]
		if len(tagPaths) == 0 {
			// Nothing to dist for this tag, continue to the next dist.
			continue
		}

		if len(tagPaths) > 1 && (dist.Dest != nil || dist.Suffix != nil) {
			errorMessage := "%s: Cannot apply dest/suffix for more than one dist " +
				"file for %q goals tag %q in module %s. The list of dist files, " +
				"which should have a single element, is:\n%s"
			panic(fmt.Errorf(errorMessage, mod, goals, tag, name, tagPaths))
		}

		copiesForGoals := distContributions.getCopiesForGoals(goals)

		// Iterate over each path adding a copy instruction to copiesForGoals
		for _, path := range tagPaths {
			// It's possible that the Path is nil from errant modules. Be defensive here.
			if path == nil {
				tagName := "default" // for error message readability
				if dist.Tag != nil {
					tagName = *dist.Tag
				}
				panic(fmt.Errorf("Dist file should not be nil for the %s tag in %s", tagName, name))
			}

			dest := filepath.Base(path.String())

			if dist.Dest != nil {
				var err error
				if dest, err = validateSafePath(*dist.Dest); err != nil {
					// This was checked in ModuleBase.GenerateBuildActions
					panic(err)
				}
			}

			ext := filepath.Ext(dest)
			suffix := ""
			if dist.Suffix != nil {
				suffix = *dist.Suffix
			}

			prependProductString := ""
			if proptools.Bool(dist.Prepend_artifact_with_product) {
				prependProductString = fmt.Sprintf("%s-", ctx.Config().DeviceProduct())
			}

			appendProductString := ""
			if proptools.Bool(dist.Append_artifact_with_product) {
				appendProductString = fmt.Sprintf("_%s", ctx.Config().DeviceProduct())
			}

			if suffix != "" || appendProductString != "" || prependProductString != "" {
				dest = prependProductString + strings.TrimSuffix(dest, ext) + suffix + appendProductString + ext
			}

			if dist.Dir != nil {
				var err error
				if dest, err = validateSafePath(*dist.Dir, dest); err != nil {
					// This was checked in ModuleBase.GenerateBuildActions
					panic(err)
				}
			}

			copiesForGoals.addCopyInstruction(path, dest)
		}
	}

	return distContributions
}

// soongDist implements collecting and writing out dist rules for Soong-only builds.
//
// Any phony target that is used as a dist goal gets an extra intermediate phony target
// added as a dependency.
//
// An extra ninja file is written out to be used when dist is enabled, which contains copy
// rules for all the dist files, and the intermediate phony targets with dependencies on
// the copy rules.
//
// Another extra ninja file is written out to be used when dist is not enabled, which contains
// empty phony rules for the intermediate phony targets.
type soongDist struct {
	// copies is the list of copy pairs from all dist rules.
	copies []distCopy
	// goals is the list of goals that have dist rules attached.
	goals []string
	// goalToDests maps dist goals to the list of copy destinations for that goal.
	goalToDests map[string][]string
	// conflicts is the list of copy pairs that have conflicting destinations.  If any
	// conflicts are found then all copy rules will be replaced with error messages.
	conflicts [][2]distCopy
}

const distIntermediateGoalSuffix = "__dist"

// collectDists visits all singletons and modules to collect the dist goals and copies.
func (s *soongDist) collectDists(ctx SingletonContext) {
	var allDistContributions []distContributions
	ctx.VisitAllSingletons(func(s blueprint.SingletonProxy) {
		if info, ok := OtherSingletonProvider(ctx, s, SingletonDistInfoProvider); ok {
			if contribution := distsToDistContributions(info.Dists); contribution != nil {
				allDistContributions = append(allDistContributions, *contribution)
			}
		}
	})

	ctx.VisitAllModuleProxies(func(mod ModuleProxy) {
		if distInfo, ok := OtherModuleProvider(ctx, mod, DistProvider); ok {
			if contribution := distsToDistContributions(distInfo.Dists); contribution != nil {
				allDistContributions = append(allDistContributions, *contribution)
			}
		}

		commonInfo := OtherModulePointerProviderOrDefault(ctx, mod, CommonModuleInfoProvider)
		if commonInfo.SkipAndroidMkProcessing {
			return
		}
		if contribution := getDistContributions(ctx, mod); contribution != nil {
			allDistContributions = append(allDistContributions, *contribution)
		}
	})

	s.parseDistContributions(allDistContributions)
}

func distsToDistContributions(dists []dist) *distContributions {
	if len(dists) == 0 {
		return nil
	}

	copyGoals := []*copiesForGoals{}
	for _, dist := range dists {
		copyGoals = append(copyGoals, &copiesForGoals{
			goals:  dist.goals,
			copies: dist.paths,
		})
	}

	return &distContributions{
		copiesForGoals: copyGoals,
	}
}

func (s *soongDist) parseDistContributions(allDistContributions []distContributions) {
	s.goalToDests = map[string][]string{}
	for _, dist := range allDistContributions {
		for _, distCopy := range dist.copiesForGoals {
			for _, goal := range distCopy.goals {
				for _, dest := range distCopy.copies {
					s.goalToDests[goal] = append(s.goalToDests[goal], dest.dest)
				}
			}
			s.copies = append(s.copies, distCopy.copies...)
		}
	}

	// Kick off some work that can be done in parallel.
	wg := sync.WaitGroup{}

	// Collect all the dist goals and sort them.
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.goals = slices.Grow(s.goals, len(s.goalToDests))
		s.goals = slices.AppendSeq(s.goals, maps.Keys(s.goalToDests))
		slices.Sort(s.goals)
	}()

	// Sort the copy rules and detect duplicates.
	wg.Add(1)
	go func() {
		defer wg.Done()
		slices.SortFunc(s.copies, func(a, b distCopy) int {
			return cmp.Or(cmp.Compare(a.dest, b.dest), cmp.Compare(a.from.String(), b.from.String()))
		})
		s.copies = slices.CompactFunc(s.copies, func(a, b distCopy) bool {
			if a.dest == b.dest {
				if as, bs := a.from.String(), b.from.String(); as != bs {
					s.conflicts = append(s.conflicts, [2]distCopy{a, b})
				}
				return true
			}
			return false
		})
	}()

	// Wait for the parallel work to finish.  distGoals, distCopies and ctx.Errorf must not be used
	// until the parallel work finishes.
	wg.Wait()
}

// addDistsToPhonyMap adds the intermediate phony targets as dependencies of the dist goals.
func (s *soongDist) addDistsToPhonyMap(ctx SingletonContext, phonyMap phonyMap) {
	for _, goal := range s.goals {
		phonyMap[goal] = append(phonyMap[goal], PathForPhony(ctx, goal+distIntermediateGoalSuffix))
	}
}

// writeNinjaFiles writes the two extra ninja files for use when dist is and is not enabled.
func (s *soongDist) writeNinjaFiles(soongDistFile, soongNoDistFile Path, rewrapperRemoteBuild bool) []error {
	wg := WaitGroupWithErrorCollector{}

	wg.Go(func() error {
		f, err := openBufferedFile(absolutePath(soongDistFile.String()))
		if err != nil {
			return err
		}
		defer f.Close()
		if len(s.conflicts) == 0 {
			return s.generateSoongDistNinja(f, rewrapperRemoteBuild)
		} else {
			return s.generateSoongConflictDistNinja(f, rewrapperRemoteBuild)
		}
	})

	wg.Go(func() error {
		f, err := openBufferedFile(absolutePath(soongNoDistFile.String()))
		if err != nil {
			return err
		}
		defer f.Close()
		return s.generateSoongNoDistNinja(f)
	})

	wg.Wait()
	return wg.Errors()
}

// generateSoongDistNinja generates a ninja file to be used when dist is enabled which contains
// copy rules for all the dist files, and the intermediate phony targets with dependencies
// on the copy rules.
func (s *soongDist) generateSoongDistNinja(w io.StringWriter, rewrapperRemoteBuild bool) error {
	var err error
	write := func(s string) {
		if err == nil {
			_, err = w.WriteString(s)
		}
	}
	write("" +
		"rule distCp\n" +
		" description = Dist $out\n" +
		" command = rm -f $out && cp $in $out\n" +
		" sandbox_disabled = true\n")
	if rewrapperRemoteBuild {
		write(" pool = local_pool\n")
	}
	write("\n")

	insertFileNameTagReplacer := strings.NewReplacer("FILE_NAME_TAG_PLACEHOLDER", "$distFileNameTag")

	for _, distCopy := range s.copies {
		write("build $distdir/")
		write(insertFileNameTagReplacer.Replace(distCopy.dest))
		write(": distCp ")
		write(distCopy.from.String())
		write("\n")
	}

	for _, goal := range s.goals {
		intermediateGoal := goal + distIntermediateGoalSuffix
		write("build ")
		write(intermediateGoal)
		write(": phony")
		for _, dest := range s.goalToDests[goal] {
			write(" $distdir/")
			write(insertFileNameTagReplacer.Replace(dest))
		}
		write("\n phony_output = true\n")
	}
	return err
}

// generateSoongConflictDistNinja generates a ninja file to be used when dist is enabled
// but conflicting copying rules are present.  The ninja file will contain errors for all
// intermediate phony targets, causing any build with `dist` on the command line to fail.
// This matches the behavior of old builds with Make-based dist rules that were only
// generated when `dist` was on the command line, and is necessary because mainline builds
// that configure multiple targets with conflicting multilib values (e.g. arm64 and x86_64)
// exist but don't use `dist.
func (s *soongDist) generateSoongConflictDistNinja(w io.StringWriter, rewrapperRemoteBuild bool) error {
	var err error
	write := func(s string) {
		if err == nil {
			_, err = w.WriteString(s)
		}
	}
	write("" +
		"rule distError\n" +
		" description = Error $out\n" +
		" command = echo $distErrorMsg; false\n" +
		" sandbox_disabled = true\n" +
		" phony_output = true\n")
	if rewrapperRemoteBuild {
		write(" pool = local_pool\n")
	}

	conflictA := s.conflicts[0][0]
	conflictB := s.conflicts[0][1]
	errorMsg := fmt.Sprintf("conflicting dist sources %q and %q for target %q",
		conflictA.from.String(), conflictB.from.String(), conflictA.dest)

	write("distErrorMsg = ")
	write(proptools.NinjaAndShellEscape(errorMsg))
	write("\n")

	for _, goal := range s.goals {
		intermediateGoal := goal + distIntermediateGoalSuffix
		write("build ")
		write(intermediateGoal)
		write(": distError\n")
	}
	return err
}

// writeSoongNoDistNinja generates a ninja file to be used when dist is not enabled, which contains
// empty phony rules for the intermediate phony targets.
func (s *soongDist) generateSoongNoDistNinja(w io.StringWriter) error {
	var err error
	write := func(s string) {
		if err == nil {
			_, err = w.WriteString(s)
		}
	}
	for _, goal := range s.goals {
		intermediateGoal := goal + distIntermediateGoalSuffix
		write("build ")
		write(intermediateGoal)
		write(": phony\n phony_output = true\n")
	}
	return err
}

// WaitGroupWithErrorCollector wraps sync.WaitGroup with helpers to collect errors
// from the parallel tasks.
type WaitGroupWithErrorCollector struct {
	sync.WaitGroup
	lock   sync.Mutex
	errors []error
	done   bool
}

// Error reports an error from a parallel task.
func (wg *WaitGroupWithErrorCollector) Error(err error) {
	wg.lock.Lock()
	defer wg.lock.Unlock()
	wg.errors = append(wg.errors, err)
}

// Go runs a task, calling Add, Done and Error as necessary.
func (wg *WaitGroupWithErrorCollector) Go(f func() error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := f()
		if err != nil {
			wg.Error(err)
		}
	}()
}

// GoWithMultipleErrors runs a task, calling Add, Done and Error as necessary.
func (wg *WaitGroupWithErrorCollector) GoWithMultipleErrors(f func() []error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		errs := f()
		for _, err := range errs {
			wg.Error(err)
		}
	}()
}

// Wait waits for all parallel tasks to finish and makes any errors visible to Errors.
func (wg *WaitGroupWithErrorCollector) Wait() {
	wg.WaitGroup.Wait()
	wg.done = true
}

// Errors returns any errors reported by the parallel tasks.  Calling Errors before Wait
// returns will panic.
func (wg *WaitGroupWithErrorCollector) Errors() []error {
	if !wg.done {
		panic(fmt.Errorf("Errors called before Wait"))
	}
	return wg.errors
}

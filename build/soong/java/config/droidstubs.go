// Copyright 2023 Google Inc. All rights reserved.
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

package config

import "strings"

var (
	metalavaFlags = []string{
		"--color",
		"--quiet",
		"--repeat-errors-max 10",
		"--hide UnresolvedImport",
	}

	// formatVersion is the default version used for new files.
	formatVersion = "2.0"

	// formatProperties are the default format properties that are combined with the formatVersion
	// and used for new and existing files (unless explicitly overridden by the file).
	//
	// Force metalava to sort overloaded methods by their order in the source code.
	// See b/285312164 for more information.
	// And add concrete overrides of abstract methods.
	// See b/299366704 for more information.
	formatProperties = "overloaded-method-order=source,add-additional-overrides=yes"

	defaultMetalavaFormatFlags = []string{
		"--format " + formatVersion,
		"--format-defaults " + formatProperties,
	}

	// Flags common to all metalava invocations.
	MetalavaFlags = strings.Join(metalavaFlags, " ")

	// DefaultMetalavaEverythingFormatFlags are the default format flags used by
	// the "everything" Metalava invocations.
	DefaultMetalavaEverythingFormatFlags = strings.Join(defaultMetalavaFormatFlags, " ")

	// DefaultMetalavaExportableFormatSpecifier is the default format used by the "exportable" Metalava
	// invocations.
	DefaultMetalavaExportableFormatSpecifier = formatVersion + ":" + formatProperties

	metalavaAnnotationsFlags = []string{
		"--include-annotations",
		"--exclude-annotation androidx.annotation.RequiresApi",
	}

	MetalavaAnnotationsFlags = strings.Join(metalavaAnnotationsFlags, " ")

	metalavaAnnotationsWarningsFlags = []string{
		// TODO(tnorbye): find owners to fix these warnings when annotation was enabled.
		"--hide HiddenTypedefConstant",
		"--hide SuperfluousPrefix",
	}

	MetalavaAnnotationsWarningsFlags = strings.Join(metalavaAnnotationsWarningsFlags, " ")

	metalavaVmFlags = []string{
		"-J-XX:ReservedCodeCacheSize=128m",
	}

	MetalavaVmFlags = strings.Join(metalavaVmFlags, " ")
)

const (
	MetalavaAddOpens = "-J--add-opens=java.base/java.util=ALL-UNNAMED"
)

func init() {
	pctx.StaticVariable("MetalavaAnnotationsFlags", strings.Join(metalavaAnnotationsFlags, " "))

	pctx.StaticVariable("MetalavaAnnotationWarningsFlags", strings.Join(metalavaAnnotationsWarningsFlags, " "))
}

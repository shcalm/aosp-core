// Copyright (C) 2025 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	idi_lib "android/soong/cmd/incremental_dex_input/incremental_dex_input_lib"
)

type multiString []string

func (ms *multiString) String() string     { return strings.Join(*ms, ", ") }
func (ms *multiString) Set(s string) error { *ms = append(*ms, s); return nil }

func main() {
	var classesJar, deps, outputDir, packageOutputDir, dexTarget string
	var tools multiString

	flag.StringVar(&classesJar, "classesJar", "", "jar file containing compiled java classes")
	flag.StringVar(&deps, "deps", "", "rsp file enlisting all module deps")
	flag.StringVar(&dexTarget, "dexTarget", "", "dex output")
	flag.StringVar(&outputDir, "outputDir", "", "root directory for creating dex entries")
	flag.StringVar(&packageOutputDir, "packageOutputDir", "", "root directory for creating package based dex entries")
	flag.Var(&tools, "tool", "tool dependency that causes all java classes to be recompiled when changed")

	flag.Parse()

	if classesJar == "" {
		panic("must specify --classesJar")
	}

	if deps == "" {
		panic("must specify --deps")
	}

	if dexTarget == "" {
		panic("must specify --dexTarget")
	}

	if outputDir == "" {
		panic("must specify --outputDir")
	}

	if packageOutputDir == "" {
		panic("must specify --packageOutputDir")
	}

	executable, err := os.Executable()
	if err != nil {
		panic(fmt.Errorf("failed to get path to executable: %w", err))
	}
	tools = append(tools, executable)

	idi_lib.GenerateIncrementalInput(classesJar, outputDir, packageOutputDir, dexTarget, deps, tools)
}

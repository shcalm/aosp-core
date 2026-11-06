/*
 * Copyright (C) 2025 The Android Open Source Project
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

package com.android.kotlin.compiler.ksp

import com.android.kotlin.compiler.cli.BooleanArgument
import com.android.kotlin.compiler.cli.InputFileListArgument
import com.android.kotlin.compiler.cli.NoArgument
import com.android.kotlin.compiler.cli.ReadableDirectoryArgument
import com.android.kotlin.compiler.cli.SourceDeltaArgument
import com.android.kotlin.compiler.cli.SrcJarsDirArgument
import com.android.kotlin.compiler.cli.StringArgument
import com.android.kotlin.compiler.cli.StringMapArgument
import com.android.kotlin.compiler.cli.WritableDirectoryArgument
import java.io.File

class ProjectBaseDirArgument : ReadableDirectoryArgument<KspOptions>() {
    override val argumentName = "project-base-dir"

    override val helpText =
        """
        The base directory.
        """
            .trimIndent()

    override val default: String? = null

    override fun setDirectory(dir: File, opts: KspOptions) {
        opts.projectBaseDir = dir
    }
}

class ModuleNameArgument : StringArgument<KspOptions>() {
    override val argumentName = "module-name"

    override val helpText =
        """
        A name for the artifact that is generated.
        """
            .trimIndent()

    override val default: String? = null

    override fun setOption(option: String, opts: KspOptions) {
        opts.moduleName = option
    }
}

class JvmTargetArgument : StringArgument<KspOptions>() {
    override val argumentName = "jvm-target"

    override val helpText =
        """
        The jvm version being compiled for.
        """
            .trimIndent()

    override val default: String? = null

    override fun setOption(option: String, opts: KspOptions) {
        opts.jvmTarget = option
    }
}

class LanguageVersionArgument : StringArgument<KspOptions>() {
    override val argumentName = "language-version"

    override val helpText =
        """
        Kotlin language version being compiled for.
        """
            .trimIndent()

    override val default: String? = null

    override fun setOption(option: String, opts: KspOptions) {
        opts.languageVersion = option
    }
}

class ApiVersionArgument : StringArgument<KspOptions>() {
    override val argumentName = "api-version"

    override val helpText =
        """
        api version
        """
            .trimIndent()

    override val default: String? = null

    override fun setOption(option: String, opts: KspOptions) {
        opts.apiVersion = option
    }
}

class SourceRootsArgument : InputFileListArgument<KspOptions>() {
    override val argumentName = "source-roots"
    override val helpText =
        """
            List of files or directories containing sources to be processed.
            Entries prefixed with '@' are read in as text files where each white-space separated entry
            is treated as an additional source file.
        """
            .trimIndent()
    override val default = ""

    override fun setFiles(files: List<File>, opts: KspOptions) {
        opts.sourceRoots = files
    }
}

class LibrariesArgument : InputFileListArgument<KspOptions>() {
    override val argumentName = "libraries"
    override val helpText =
        """
            List of jars to link against.
        """
            .trimIndent()
    override val default = ""

    override fun setFiles(files: List<File>, opts: KspOptions) {
        opts.libraries = files
    }
}

class FriendsArgument : InputFileListArgument<KspOptions>() {
    override val argumentName = "friends"
    override val helpText =
        """
            List of friend jars.
        """
            .trimIndent()
    override val default = ""

    override fun setFiles(files: List<File>, opts: KspOptions) {
        opts.friends = files
    }
}

class CommonSourceRootsArgument : InputFileListArgument<KspOptions>() {
    override val argumentName = "common-src-roots"
    override val helpText =
        """
            List of sources that should be used as common sources when compiling
            for multiplatform.
        """
            .trimIndent()
    override val default = ""

    override fun setFiles(files: List<File>, opts: KspOptions) {
        opts.commonSourceRoots = files
    }
}

class OutputBaseDirArgument : WritableDirectoryArgument<KspOptions>() {
    override val argumentName = "output-base-dir"

    override val helpText =
        """
        The base directory for outputs. Useful for incremental.
        """
            .trimIndent()

    override val default: String? = null

    override fun setDirectory(dir: File, opts: KspOptions) {
        opts.outputBaseDir = dir
    }
}

class JavaOutputDirArgument : WritableDirectoryArgument<KspOptions>() {
    override val argumentName = "java-output-dir"

    override val helpText =
        """
        Directory to output generated java code.
        """
            .trimIndent()

    override val default: String? = null

    override fun setDirectory(dir: File, opts: KspOptions) {
        opts.javaOutputDir = dir
    }
}

class CachesDirArgument : WritableDirectoryArgument<KspOptions>() {
    override val argumentName = "caches-dir"

    override val helpText =
        """
        Directory to store intermediate work.
        """
            .trimIndent()

    override val default: String? = null

    override fun setDirectory(dir: File, opts: KspOptions) {
        opts.cachesDir = dir
    }
}

class ClassOutputDirArgument : WritableDirectoryArgument<KspOptions>() {
    override val argumentName = "class-output-dir"

    override val helpText =
        """
        Directory to output generated class files.
        """
            .trimIndent()

    override val default: String? = null

    override fun setDirectory(dir: File, opts: KspOptions) {
        opts.classOutputDir = dir
    }
}

class KotlinOutputDirArgument : WritableDirectoryArgument<KspOptions>() {
    override val argumentName = "kotlin-output-dir"

    override val helpText =
        """
        Directory to output generated kotlin files.
        """
            .trimIndent()

    override val default: String? = null

    override fun setDirectory(dir: File, opts: KspOptions) {
        opts.kotlinOutputDir = dir
    }
}

class ResourceOutputDirArgument : WritableDirectoryArgument<KspOptions>() {
    override val argumentName = "resource-output-dir"

    override val helpText =
        """
        Directory to output generated resource files.
        """
            .trimIndent()

    override val default: String? = null

    override fun setDirectory(dir: File, opts: KspOptions) {
        opts.resourceOutputDir = dir
    }
}

class ProcessorOptionsArgument : StringMapArgument<KspOptions>(":") {
    override val argumentName = "processor-options"

    override val helpText =
        """
            Flags to pass through to the annotation processors. Should take
            the form: a=b:c=d:e=f.
        """
            .trimIndent()

    override val default = mapOf<String, String>()

    override fun setOption(option: Map<String, String>, opts: KspOptions) {
        opts.processorOptions.putAll(option)
    }
}

class SourceDeltaArgument : SourceDeltaArgument<KspOptions>() {
    override fun setOption(option: String, opts: KspOptions) {
        opts.sourceDeltaFileName = option
    }
}

class IncrementalArgument : BooleanArgument<KspOptions>() {
    override val argumentName = "incremental"
    override val default = false
    override val helpText =
        """
            Whether to compile incrementally based on a prior invocation.
        """
            .trimIndent()

    override fun setOption(option: Boolean, opts: KspOptions) {
        opts.incremental = option
    }
}

class SrcJarsDirArgument : SrcJarsDirArgument<KspOptions>() {
    override fun setDirectory(dir: File, opts: KspOptions) {
        opts.srcJarsDir = dir
    }
}

class HelpArgument : NoArgument<KspOptions>() {
    override val argumentName = "h"

    override val helpText =
        """
            Runs ksp on the supplied sources. See
            https://github.com/google/ksp/blob/main/docs/ksp2.md
            for an example reference. This wrapper includes the ability
            to accept .srcjars as input, as well as a few other
            Soong specific details.
        """
            .trimIndent()

    override fun setOption(option: Boolean, opts: KspOptions) {}
}

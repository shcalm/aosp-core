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

import com.android.kotlin.compiler.cli.parseArgs
import com.google.devtools.ksp.impl.KotlinSymbolProcessing
import com.google.devtools.ksp.processing.KSPJvmConfig
import com.google.devtools.ksp.processing.KSPLogger
import com.google.devtools.ksp.processing.SymbolProcessorProvider
import java.io.File
import java.net.URLClassLoader
import java.util.ServiceLoader
import kotlin.system.exitProcess

private val ARGUMENT_PARSERS =
    listOf(
        ApiVersionArgument(),
        CachesDirArgument(),
        ClassOutputDirArgument(),
        CommonSourceRootsArgument(),
        FriendsArgument(),
        IncrementalArgument(),
        JavaOutputDirArgument(),
        JvmTargetArgument(),
        KotlinOutputDirArgument(),
        LanguageVersionArgument(),
        LibrariesArgument(),
        ModuleNameArgument(),
        OutputBaseDirArgument(),
        ProjectBaseDirArgument(),
        ProcessorOptionsArgument(),
        ResourceOutputDirArgument(),
        SourceDeltaArgument(),
        SourceRootsArgument(),
        SrcJarsDirArgument(),
        HelpArgument(),
    )

val USAGE_TEXT =
    """
        Usage: kotlin-ksp-client ...

        See https://github.com/google/ksp/blob/main/docs/ksp2cmdline.md
    """
        .trimIndent()

fun main(args: Array<String>) {
    val opts = KspOptions()
    ARGUMENT_PARSERS.forEach { it.setupDefault(opts) }

    if (!parseArgs(args, opts, ARGUMENT_PARSERS, System.out, System.err, USAGE_TEXT)) {
        exitProcess(-1)
    }

    val logger = Logger(Logger.Level.WARN)
    runKsp(logger, opts)
}

fun runKsp(logger: KSPLogger, opts: KspOptions): KotlinSymbolProcessing.ExitCode {
    val processorClassloader =
        URLClassLoader(opts.passThroughArgs.map { File(it).toURI().toURL() }.toTypedArray())

    val processorProviders =
        ServiceLoader.load(SymbolProcessorProvider::class.java, processorClassloader).toList()

    // TODO(b/442809933): Make incremental work. See http://ag/36521483
    val inc = false // opts.incremenetal

    if (!inc) {
        // Delete any prior output.
        listOf(
                opts.javaOutputDir!!,
                opts.cachesDir!!,
                opts.classOutputDir!!,
                opts.kotlinOutputDir!!,
                opts.resourceOutputDir!!,
            )
            .forEach { outDir -> outDir.listFiles()?.forEach { it.deleteRecursively() } }
    }
    val kspConfig =
        KSPJvmConfig.Builder()
            .apply {
                moduleName = opts.moduleName!!
                sourceRoots = opts.sourceRoots
                jvmTarget = opts.jvmTarget!!
                projectBaseDir = opts.projectBaseDir!!
                libraries = opts.libraries
                friends = opts.friends
                commonSourceRoots = opts.commonSourceRoots
                outputBaseDir = opts.outputBaseDir!!
                javaOutputDir = opts.javaOutputDir!!
                cachesDir = opts.cachesDir!!
                classOutputDir = opts.classOutputDir!!
                kotlinOutputDir = opts.kotlinOutputDir!!
                resourceOutputDir = opts.resourceOutputDir!!
                incremental = inc
                incrementalLog = opts.incremental
                languageVersion = opts.languageVersion!!
                apiVersion = opts.apiVersion!!
                processorOptions = opts.processorOptions
                if (inc) {
                    modifiedSources = opts.modifiedFiles ?: listOf()
                    removedSources = opts.removedFiles ?: listOf()
                }
            }
            .build()

    return KotlinSymbolProcessing(kspConfig, processorProviders, logger).execute()
}

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

import com.android.kotlin.compiler.cli.FileDeltaOptions
import com.android.kotlin.compiler.cli.Options
import java.io.File

class KspOptions : Options, FileDeltaOptions {
    override val passThroughArgs = mutableListOf<String>()

    var moduleName: String? = null

    var sourceRoots: List<File> = listOf()

    var jvmTarget: String? = null

    var projectBaseDir: File? = null

    var libraries: List<File> = listOf()

    var friends: List<File> = listOf()

    var commonSourceRoots: List<File> = listOf()

    private var _srcJarsDir: File? = null
    override var srcJarsDir: File
        get() {
            return _srcJarsDir
                ?: throw IllegalStateException("Can not read srcJarsDir before it is set")
        }
        set(value) {
            _srcJarsDir = value
        }

    var srcJarsDirLocation: String
        get() {
            return _srcJarsDir?.absolutePath
                ?: throw IllegalStateException("Can not read srcJarsDirLocation before it is set")
        }
        set(value) {
            if (value == "") {
                _srcJarsDir = null
            } else {
                _srcJarsDir = File(value)
            }
        }

    var outputBaseDir: File? = null

    var javaOutputDir: File? = null

    var cachesDir: File? = null

    var classOutputDir: File? = null

    var kotlinOutputDir: File? = null

    var resourceOutputDir: File? = null

    var languageVersion: String? = null

    var apiVersion: String? = null

    val processorOptions = mutableMapOf<String, String>()

    var sourceDeltaFileName: String? = null
    val sourceDeltaFile: File?
        get() = if (sourceDeltaFileName != null) File(sourceDeltaFileName!!) else null

    override val modifiedFiles: List<File>?
        get() =
            FileDeltaOptions.parseSourceChanges(sourceDeltaFile, srcJarsDirLocation)?.modifiedFiles

    override val removedFiles: List<File>?
        get() =
            FileDeltaOptions.parseSourceChanges(sourceDeltaFile, srcJarsDirLocation)?.removedFiles

    var incremental: Boolean = false
}

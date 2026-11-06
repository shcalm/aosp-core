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

package com.android.kotlin.compiler.cli

import java.io.File

interface FileDeltaOptions {
    val modifiedFiles: List<File>?

    val removedFiles: List<File>?

    var srcJarsDir: File

    companion object {

        enum class SourceChangeParseState {
            STANDARD,
            ZIP_FILE,
            CONTENTS,
        }

        fun parseSourceChanges(sourceDeltaFile: File?, zipSrcsDir: String): FileChanges? {
            if (sourceDeltaFile == null) {
                return null
            }
            var state = SourceChangeParseState.STANDARD
            val modifiedList = mutableListOf<File>()
            val removedList = mutableListOf<File>()
            for (entry in sourceDeltaFile.readText().split(" ")) {
                if (entry.length < 1) {
                    continue
                }

                // Prepare the file name as it is used in different branches.
                // Not that this file name is nonsense for a few entries, but won't be used for
                // those entries.
                var filename = entry.substring(1)
                if (state == SourceChangeParseState.CONTENTS) {
                    filename = zipSrcsDir + File.separator + filename
                }
                when {
                    entry == "--file" -> {
                        state = SourceChangeParseState.ZIP_FILE
                        // We'll skip the next entry
                    }
                    entry == "--endfile" -> {
                        state = SourceChangeParseState.STANDARD
                    }
                    state == SourceChangeParseState.ZIP_FILE -> {
                        state = SourceChangeParseState.CONTENTS
                        // we now need to prefix our entries with the srcJars directory.
                    }
                    entry.startsWith("+") -> {
                        val f = File(filename)
                        if (f.extension == "kt" || f.extension == "java") {
                            if (!f.exists()) {
                                throw RuntimeException(
                                    "Supplied file diff contains modified file that does not exist: $entry"
                                )
                            }
                            modifiedList.add(f.absoluteFile)
                        }
                    }

                    entry.startsWith("-") -> {
                        val f = File(filename)
                        removedList.add(f.absoluteFile)
                    }

                    else -> {
                        throw RuntimeException(
                            "Supplied file diff contains entry that can not be parsed: $entry"
                        )
                    }
                }
            }
            return FileChanges(modifiedFiles = modifiedList, removedFiles = removedList)
        }
    }
}

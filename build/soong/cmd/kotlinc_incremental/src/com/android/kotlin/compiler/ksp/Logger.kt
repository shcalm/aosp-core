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

import com.google.devtools.ksp.processing.KSPLogger
import com.google.devtools.ksp.symbol.FileLocation
import com.google.devtools.ksp.symbol.KSNode
import com.google.devtools.ksp.symbol.NonExistLocation
import java.io.PrintStream

class Logger(val level: Level = Level.WARN, val output: PrintStream = System.out) : KSPLogger {
    enum class Level {
        LOGGING,
        INFO,
        WARN,
        ERROR,
    }

    override fun logging(message: String, symbol: KSNode?) {
        if (level <= Level.LOGGING) {
            printMessage("logging", message, symbol)
        }
    }

    override fun info(message: String, symbol: KSNode?) {
        if (level <= Level.INFO) {
            printMessage("info", message, symbol)
        }
    }

    override fun warn(message: String, symbol: KSNode?) {
        if (level <= Level.WARN) {
            printMessage("warn", message, symbol)
        }
    }

    override fun error(message: String, symbol: KSNode?) {
        if (level <= Level.ERROR) {
            printMessage("logging", message, symbol)
        }
    }

    override fun exception(e: Throwable) {
        error(e)
    }

    private fun printMessage(level: String, message: String, symbol: KSNode?) {
        val m =
            "[ksp:$level] " +
                when (val location = symbol?.location) {
                    is FileLocation -> "${location.filePath}:${location.lineNumber}: $message"
                    is NonExistLocation,
                    null -> message
                }

        output.println(m)
    }
}

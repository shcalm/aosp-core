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

import com.google.common.truth.Truth.assertThat
import java.io.ByteArrayOutputStream
import java.io.File
import java.io.PrintStream
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder

class ArgumentTest {

    class TestOpts : Options {
        override val passThroughArgs = mutableListOf<String>()
        val mapArgs = mutableMapOf<String, String>()
        var directory: File? = null
    }

    private val opts = TestOpts()
    private val stdoutStreamCaptor = ByteArrayOutputStream()
    private val stderrStreamCaptor = ByteArrayOutputStream()

    @get:Rule val tFolder = TemporaryFolder()

    @Before fun setup() {}

    @Test
    fun testParseArgs_NoError() {
        val argParsers = listOf(HelpArgument<TestOpts>())
        val args = arrayOf("-h", "foo", "bar", "baz")

        val result =
            parseArgs(
                args,
                opts,
                argParsers,
                PrintStream(stdoutStreamCaptor),
                PrintStream(stderrStreamCaptor),
                "usage",
            )

        assertThat(result).isTrue()
        assertThat(opts.passThroughArgs).isEqualTo(listOf("foo", "bar", "baz"))
        assertThat(stdoutStreamCaptor.toString()).startsWith("usage\n")
        assertThat(stderrStreamCaptor.toString()).isEmpty()
    }

    @Test
    fun testParseArgs_Error() {
        class ErrorArg : NoArgument<TestOpts>() {
            override val argumentName = "error"
            override val helpText = "This argument forces an error"

            override fun setOption(option: Boolean, opts: TestOpts) {
                error = "A forced error"
            }
        }

        val argParsers = listOf(ErrorArg())
        val args = arrayOf("-error", "foo", "bar", "baz")

        val result =
            parseArgs(
                args,
                opts,
                argParsers,
                PrintStream(stdoutStreamCaptor),
                PrintStream(stderrStreamCaptor),
                "usage",
            )

        assertThat(result).isFalse()
        assertThat(stdoutStreamCaptor.toString()).isEmpty()
        assertThat(stderrStreamCaptor.toString()).isEqualTo(argParsers.get(0).error + "\n\n")
    }

    @Test
    fun testHelpArgument() {
        val ha = HelpArgument<TestOpts>()
        assertThat(ha.matches("-h")).isTrue()
        val args = listOf("foo").iterator()
        ha.parse("-h", args, opts)
        assertThat(args.hasNext()).isTrue()
    }

    @Test
    fun testSingleArgument_NotEmpty() {
        val sa =
            object : SingleArgument<String, TestOpts>(false) {
                override fun stringToType(arg: String) = arg

                override val argumentName = "test"
                override val helpText = "help"
                override val default = null

                override fun setOption(option: String, opts: TestOpts) {
                    opts.passThroughArgs.add(option)
                }
            }

        val arg = "-test=foobar"
        assertThat(sa.matches(arg)).isTrue()
        sa.parse(arg, emptyList<String>().iterator(), opts)
        assertThat(opts.passThroughArgs).hasSize(1)
        assertThat(opts.passThroughArgs).contains("foobar")
        assertThat(sa.error).isNull()

        sa.parse("-test=", emptyList<String>().iterator(), opts)
        assertThat(sa.error).isNotEmpty()
    }

    @Test
    fun testSingleArgument_AllowEmpty() {
        val sa =
            object : SingleArgument<String, TestOpts>(true) {
                override fun stringToType(arg: String) = arg

                override val argumentName = "test"
                override val helpText = "help"
                override val default = null

                override fun setOption(option: String, opts: TestOpts) {
                    opts.passThroughArgs.add(option)
                }
            }

        val arg = "-test="
        assertThat(sa.matches(arg)).isTrue()
        sa.parse(arg, emptyList<String>().iterator(), opts)
        assertThat(opts.passThroughArgs).hasSize(1)
        assertThat(opts.passThroughArgs).contains("")
        assertThat(sa.error).isNull()
    }

    @Test
    fun testMapArgument() {
        val ma =
            object : MapArgument<String, String, TestOpts>(":") {
                override fun kToType(arg: String) = arg

                override fun vToType(arg: String) = arg

                override val argumentName = "test"
                override val helpText = "help"
                override val default = null

                override fun setOption(option: Map<String, String>, opts: TestOpts) {
                    opts.mapArgs.putAll(option)
                }
            }

        val arg = "-test=a=b:c=d"
        assertThat(ma.matches(arg)).isTrue()
        ma.parse(arg, emptyList<String>().iterator(), opts)
        assertThat(opts.mapArgs).hasSize(2)
        assertThat(opts.mapArgs).containsEntry("a", "b")
        assertThat(opts.mapArgs).containsEntry("c", "d")
    }

    @Test
    fun testInputFileListArgument_MultipleFiles() {
        val ifa = createInputFileListArgument()

        val fileA = tFolder.newFile("a")
        val fileB = tFolder.newFile("b")

        val arg = "-test=${fileA.path}:${fileB.path}"

        assertThat(ifa.matches(arg)).isTrue()

        ifa.parse(arg, emptyList<String>().iterator(), opts)

        assertThat(opts.passThroughArgs).hasSize(2)
        assertThat(opts.passThroughArgs).contains(fileA.absolutePath)
        assertThat(opts.passThroughArgs).contains(fileB.absolutePath)
    }

    @Test
    fun testInputFileListArgument_FileList() {
        val ifa = createInputFileListArgument()

        val list = tFolder.newFile("list")
        val fileA = tFolder.newFile("a")
        val fileB = tFolder.newFile("b")

        list.writeText("${fileA.path}\n${fileB.path}")

        val arg = "-test=@${list.path}"

        assertThat(ifa.matches(arg)).isTrue()

        ifa.parse(arg, emptyList<String>().iterator(), opts)

        assertThat(opts.passThroughArgs).hasSize(2)
        assertThat(opts.passThroughArgs).contains(fileA.absolutePath)
        assertThat(opts.passThroughArgs).contains(fileB.absolutePath)
    }

    @Test
    fun testWritableDirectoryArgument_usesCanonicalPath() {
        // Verifies that WritableDirectoryArgument resolves the path to its canonical form before
        // passing it to setDirectory. This is important for path comparisons and to avoid issues
        // with relative paths containing "." or "..".

        // Setup: Create a directory structure that allows for a non-canonical path.
        val subDir = tFolder.newFolder("subDir")
        val targetDir = tFolder.newFolder("targetDir")
        val nonCanonicalPath = "${tFolder.root.path}/${subDir.name}/../${targetDir.name}"

        // Setup: Create a concrete implementation of WritableDirectoryArgument to test.
        // The setDirectory method will capture the File object passed to it.
        val wda =
            object : WritableDirectoryArgument<TestOpts>() {
                override val argumentName = "testDir"
                override val helpText = "help"
                override val default: String? = null

                override fun setDirectory(dir: File, opts: TestOpts) {
                    opts.directory = dir
                }
            }

        // Action: Parse an argument with the non-canonical path.
        val arg = "-testDir=$nonCanonicalPath"
        wda.parse(arg, emptyList<String>().iterator(), opts)

        // Assertion: Verify that the path was canonicalized before being set.
        assertThat(wda.error).isNull()
        assertThat(opts.directory).isNotNull()
        assertThat(opts.directory).isEqualTo(targetDir.canonicalFile)
    }

    @Test
    fun testMapArgument_EscapedSeparatorInKey() {
        // Validates that keys can contain colons if escaped.
        // Input string: key\:part=true
        val ma = createMapArgument(":")
        val arg = "-test=key\\:part=true"

        assertThat(ma.matches(arg)).isTrue()
        ma.parse(arg, emptyList<String>().iterator(), opts)

        assertThat(opts.mapArgs).hasSize(1)
        assertThat(opts.mapArgs).containsEntry("key:part", "true")
    }

    @Test
    fun testMapArgument_EscapedSeparatorInValue() {
        // Validates that paths containing colons work if escaped
        // Input string: path=foo\:bar
        val ma = createMapArgument(":")
        val arg = "-test=path=foo\\:bar"

        ma.parse(arg, emptyList<String>().iterator(), opts)

        assertThat(opts.mapArgs).hasSize(1)
        assertThat(opts.mapArgs).containsEntry("path", "foo:bar")
    }

    @Test
    fun testMapArgument_EscapedEqualsSign() {
        // Validates that keys can contain equals signs if escaped
        // Input string: key\=name=value
        val ma = createMapArgument(":")
        val arg = "-test=key\\=name=value"

        ma.parse(arg, emptyList<String>().iterator(), opts)

        assertThat(opts.mapArgs).hasSize(1)
        assertThat(opts.mapArgs).containsEntry("key=name", "value")
    }

    @Test
    fun testMapArgument_LiteralBackslash() {
        // Validates that a double backslash is parsed as a single literal backslash
        // Input string: path=C:\\Windows
        val ma = createMapArgument(":")
        // Runtime String: "-test=path=C\:\\Windows"
        val argBackslash = "-test=path=C\\:\\\\Windows"

        ma.parse(argBackslash, emptyList<String>().iterator(), opts)

        assertThat(opts.mapArgs).hasSize(1)
        assertThat(opts.mapArgs).containsEntry("path", "C:\\Windows")
    }

    @Test
    fun testMapArgument_MixedList() {
        // Validates that standard lists still work alongside the new logic
        val ma = createMapArgument(":")
        val arg = "-test=a=b:c=d"

        ma.parse(arg, emptyList<String>().iterator(), opts)

        assertThat(opts.mapArgs).hasSize(2)
        assertThat(opts.mapArgs).containsEntry("a", "b")
        assertThat(opts.mapArgs).containsEntry("c", "d")
    }

    // Helper method to create the MapArgument instance for testing
    private fun createMapArgument(separator: String) =
        object : MapArgument<String, String, TestOpts>(separator) {
            override fun kToType(arg: String) = arg
            override fun vToType(arg: String) = arg
            override val argumentName = "test"
            override val helpText = "help"
            override val default = null

            override fun setOption(option: Map<String, String>, opts: TestOpts) {
                opts.mapArgs.putAll(option)
            }
        }

    private fun createInputFileListArgument() =
        object : InputFileListArgument<TestOpts>() {
            override fun setFiles(files: List<File>, opts: TestOpts) {
                opts.passThroughArgs.addAll(files.map { it.absolutePath })
            }

            override val argumentName = "test"
            override val helpText = "help"
            override val default = null
        }
}

#!/usr/bin/env python
#
# Copyright (C) 2024 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Script to debug the protobuf data read by derive_classpath.

If the Android device was compiled against a REL SDK, none of its jar files on
the classpaths may have been compiled against a non-REL SDK. If this is the
case, derive_classpath will crash to indicate an error in the way the system
was configured. This script helps detect that scenario.
"""

import classpaths_pb2
import subprocess
import sys
import textwrap

RESET_CURSOR_AND_CLEAR_LINE = "\033[G\033[2K"


def progress(msg):
    if not sys.stdout.isatty():
        return
    if msg is None:
        msg = RESET_CURSOR_AND_CLEAR_LINE
    else:
        msg = RESET_CURSOR_AND_CLEAR_LINE + "> " + msg
    sys.stdout.write(msg)
    sys.stdout.flush()


def find_codename_versions_in_protobuf(binary_proto_data):
    def is_codename(s):
        return s != "" and not s.isdigit()

    jars = classpaths_pb2.ExportedClasspathsJars()
    jars.ParseFromString(binary_proto_data)

    # each jar's {min,max}_sdk_version is a string that is either
    #   - the empty string (value not set)
    #   - a numerical API level
    #   - a string codename
    # we're only interested in the codename cases
    return [
        jar
        for jar in jars.jars
        if is_codename(jar.min_sdk_version) or is_codename(jar.max_sdk_version)
    ]


def exec(cmd, encoding="UTF-8"):
    completed_proc = subprocess.run(
        cmd,
        capture_output=True,
        encoding=encoding,
    )
    completed_proc.check_returncode()
    return completed_proc.stdout


def get_protobuf_paths():
    preinstalled_paths = exec(
        [
            "adb",
            "exec-out",
            "find",
            "/system/etc/classpaths",
            "-type",
            "f",
            "-name",
            "*.pb",
        ]
    ).splitlines()

    stdout = exec(
        [
            "adb",
            "exec-out",
            "find",
            "/apex",
            "-type",
            "f",
            "-path",
            "*/etc/classpaths/*.pb",
        ]
    ).splitlines()
    apex_paths = [dir_ for dir_ in stdout if "@" not in dir_]

    return set(preinstalled_paths + apex_paths)


def main():
    if not exec(["adb", "exec-out", "id"]).startswith("uid=0(root)"):
        raise Exception("must run adb as root")

    total_jars_with_codename = 0
    for path in get_protobuf_paths():
        progress(path)
        stdout = exec(
            ["adb", "exec-out", "cat", path],
            encoding=None,
        )
        jars_with_codename = find_codename_versions_in_protobuf(stdout)
        if len(jars_with_codename) > 0:
            progress(None)
            print(path)
            for jar in jars_with_codename:
                print(textwrap.indent(str(jar), "    "))
            total_jars_with_codename += len(jars_with_codename)
    progress(None)

    if total_jars_with_codename > 0:
        if exec(["adb", "exec-out", "getprop", "ro.build.version.codename"]) == "REL":
            print(
                f"{total_jars_with_codename} jar(s) with codename version(s) found on REL device: derive_classpath will detect this configuration error and crash during boot"
            )
            sys.exit(total_jars_with_codename)
        else:
            print(
                f"{total_jars_with_codename} jar(s) with codename version(s) found on non-REL device: this configuration would be an issue on a REL device"
            )


if __name__ == "__main__":
    main()

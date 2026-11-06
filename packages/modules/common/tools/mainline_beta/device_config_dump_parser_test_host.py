#!/usr/bin/env python3
#
# Copyright 2025, The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import io
import unittest
import textwrap
import zipfile

import device_config_dump_parser


def create_in_memory_zip(file_contents: dict[str, str]) -> zipfile.ZipFile:
  """Creates an in-memory zip file from a dictionary of filenames and content."""
  zip_buffer = io.BytesIO()
  with zipfile.ZipFile(zip_buffer, "w", zipfile.ZIP_DEFLATED) as zf:
    for filename, content in file_contents.items():
      zf.writestr(filename, content.encode("utf-8"))

  # Reset buffer position to the beginning to be read from.
  zip_buffer.seek(0)
  return zipfile.ZipFile(zip_buffer, "r")


class DeviceConfigDumpParserTest(unittest.TestCase):
  """Python host unit test for device_config_dump_parser."""

  def test_parses_dumpsys_device_config(self):
    actual = device_config_dump_parser.parse_device_config_dumpsys(
        io.StringIO(textwrap.dedent("""
            DeviceConfig flags:
            aaos_core_contrib/com.android.window.flags.safe_region_letterboxing_v1=true
            aaos_sdv/com.android.apex.flags.enable_brand_new_apex=false
            accessibility/android.content.res.font_scale_converter_public=
            accessibility/android.view.accessibility.a11y_character_in_window_api=true
            accessibility/android.view.accessibility.foobar=trueish
            """))
    )
    self.assertListEqual(
        actual,
        [
            device_config_dump_parser.Flag(
                "aaos_core_contrib",
                "com.android.window.flags.safe_region_letterboxing_v1",
                "true",
            ),
            device_config_dump_parser.Flag(
                "aaos_sdv",
                "com.android.apex.flags.enable_brand_new_apex",
                "false",
            ),
            device_config_dump_parser.Flag(
                "accessibility",
                "android.content.res.font_scale_converter_public",
                "false",
            ),
            device_config_dump_parser.Flag(
                "accessibility",
                "android.view.accessibility.a11y_character_in_window_api",
                "true",
            ),
            device_config_dump_parser.Flag(
                "accessibility",
                "android.view.accessibility.foobar",
                "trueish",
            ),

        ],
    )

  def test_parses_dumpsys_device_config_broken_flags(self):
    """Tests cases (that happen) where the value of a flag has multilines or is not a bool flag"""
    actual = device_config_dump_parser.parse_device_config_dumpsys(
        io.StringIO(textwrap.dedent("""
            DeviceConfig flags:
            aaos_core_contrib/com.android.window.flags.safe_region_letterboxing_v1=true

            bbb/=true
            bbb/=
            /=
            /a.b.c=false
            false
            bbb/a.b.c
            bbb
            line
            value/
            flag
            ==============
            accessibility/android.view.accessibility.a11y_character_in_window_api=true
            """))
    )
    self.assertListEqual(
        actual,
        [
            device_config_dump_parser.Flag(
                "aaos_core_contrib",
                "com.android.window.flags.safe_region_letterboxing_v1",
                "true",
            ),
            device_config_dump_parser.Flag(
                "accessibility",
                "android.view.accessibility.a11y_character_in_window_api",
                "true",
            ),
        ],
    )

  def test_parses_dumpsys_device_config_with_duplicate_flags_calls_exit(self):
    file_contents = io.StringIO(textwrap.dedent("""
        DeviceConfig flags:
        aaos_core_contrib/com.android.window.flags.safe_region_letterboxing_v1=true
        aaos_sdv/com.android.apex.flags.enable_brand_new_apex=false
        aaos_core_contrib/com.android.window.flags.safe_region_letterboxing_v1=false
        accessibility/android.content.res.font_scale_converter_public=
        accessibility/android.view.accessibility.a11y_character_in_window_api=true
        """))
    with self.assertRaises(SystemExit) as cm:
      device_config_dump_parser.parse_device_config_dumpsys(file_contents)
    self.assertEqual(cm.exception.code, 1)

  def test_parses_dumpsys_device_config_with_duplicate_flags_allows_same_value(self):
    """Test that if it finds the same flag twice (with the same value) everything works"""
    file_contents = io.StringIO(textwrap.dedent("""
        DeviceConfig flags:
        aaos_core_contrib/com.android.window.flags.safe_region_letterboxing_v1=true
        com_android_tethering/android.view.accessibility.a11y_character_in_window_api=true
        com_android_mainline_beta_mockup/com.android.apex.flags.enable_brand_new_apex=false
        com_android_tethering/android.view.accessibility.a11y_character_in_window_api=true
        """))
    actual = device_config_dump_parser.parse_device_config_dumpsys(
        file_contents
    )
    self.assertListEqual(
        actual,
        [
            device_config_dump_parser.Flag(
                "aaos_core_contrib",
                "com.android.window.flags.safe_region_letterboxing_v1",
                "true",
            ),
            device_config_dump_parser.Flag(
                "com_android_tethering",
                "android.view.accessibility.a11y_character_in_window_api",
                "true",
            ),
            device_config_dump_parser.Flag(
                "com_android_mainline_beta_mockup",
                "com.android.apex.flags.enable_brand_new_apex",
                "false",
            ),
        ],
    )

  def test_rejects_empty_dumpsys(self):
    with self.assertRaises(SystemExit) as cm:
      device_config_dump_parser.parse_device_config_dumpsys(
          io.StringIO("")
      )
    self.assertEqual(cm.exception.code, 1)

  def surround_with_dumpsys_header_and_footer(self, contents: str) -> str:
    return textwrap.dedent(f"""
            Currently running services:
              DockObserver
              SurfaceFlinger
              SurfaceFlingerAIDL
              accessibility
              account
              activity
              activity_task
              window
            -------------------------------------------------------------------------------
            DUMP OF SERVICE DockObserver:
            Current Dock Observer Service state:
              reported state: 0
              previous state: 0
              actual state: 0
            --------- 0.003s was the duration of dumpsys DockObserver, ending at: 2025-10-27 17:07:36
            -------------------------------------------------------------------------------
            DUMP OF SERVICE SurfaceFlinger:
            Build configuration: [sf PRESENT_TIME_OFFSET=0 FORCE_HWC_FOR_RBG_TO_YUV=0 MAX_VIRT_DISPLAY_DIM=0 RUNNING_WITHOUT_SYNC_FRAMEWORK=0 NUM_FRAMEBUFFER_SURFACE_BUFFERS=3]

            DUMP OF SERVICE device_identifiers:
            --------- 0.002s was the duration of dumpsys device_identifiers, ending at: 2025-10-27 17:07:57
            -------------------------------------------------------------------------------
            DUMP OF SERVICE device_lock:
            --------- 0.008s was the duration of dumpsys device_lock, ending at: 2025-10-27 17:07:57
            -------------------------------------------------------------------------------
            DUMP OF SERVICE device_policy:
            Current Device Policy Manager state:
              Immutable state:
                mHasFeature=true
                mIsWatch=false
                mIsAutomotive=false
            -------------------------------------------------------------------------------
            {contents}
            -------------------------------------------------------------------------------
            DUMP OF SERVICE device_identifiers:
            --------- 0.002s was the duration of dumpsys device_identifiers, ending at: 2025-10-27 17:07:57
            -------------------------------------------------------------------------------
            DUMP OF SERVICE device_lock:
            --------- 0.008s was the duration of dumpsys device_lock, ending at: 2025-10-27 17:07:57
            -------------------------------------------------------------------------------
            DUMP OF SERVICE device_policy:
            Current Device Policy Manager state:
            """)

  def test_parses_full_dumpsys(self):
    actual = device_config_dump_parser.parse_device_config_dumpsys(
        io.StringIO(self.surround_with_dumpsys_header_and_footer("""
            DUMP OF SERVICE device_config:
            DeviceConfig flags:
            aaos_core_contrib/com.android.window.flags.safe_region_letterboxing_v1=true
            aaos_sdv/com.android.apex.flags.enable_brand_new_apex=false
            accessibility/android.content.res.font_scale_converter_public=true
            accessibility/android.view.accessibility.a11y_character_in_window_api=true
            --------- 0.143s was the duration of dumpsys device_config, ending at: 2025-10-27 17:07:57
            """))
    )
    self.assertListEqual(
        actual,
        [
            device_config_dump_parser.Flag(
                "aaos_core_contrib",
                "com.android.window.flags.safe_region_letterboxing_v1",
                "true",
            ),
            device_config_dump_parser.Flag(
                "aaos_sdv",
                "com.android.apex.flags.enable_brand_new_apex",
                "false",
            ),
            device_config_dump_parser.Flag(
                "accessibility",
                "android.content.res.font_scale_converter_public",
                "true",
            ),
            device_config_dump_parser.Flag(
                "accessibility",
                "android.view.accessibility.a11y_character_in_window_api",
                "true",
            ),
        ],
    )

  def test_ignores_flags_outside_device_config(self):
    actual = device_config_dump_parser.parse_device_config_dumpsys(
        io.StringIO(self.surround_with_dumpsys_header_and_footer("""
            aaos_sdv/com.android.apex.flags.enable_brand_new_apex=false
            DUMP OF SERVICE device_config:
            DeviceConfig flags:
            aaos_core_contrib/com.android.window.flags.safe_region_letterboxing_v1=true
            --------- 0.143s was the duration of dumpsys device_config, ending at: 2025-10-27 17:07:57
            accessibility/android.view.accessibility.a11y_character_in_window_api=true
            """))
    )
    self.assertListEqual(
        actual,
        [
            device_config_dump_parser.Flag(
                "aaos_core_contrib",
                "com.android.window.flags.safe_region_letterboxing_v1",
                "true",
            ),
        ],
    )

  def test_parses_full_dumpsys_with_empty_device_config(self):
    actual = device_config_dump_parser.parse_device_config_dumpsys(
        io.StringIO(self.surround_with_dumpsys_header_and_footer(textwrap.dedent("""
            DUMP OF SERVICE device_config:
            DeviceConfig flags:
            --------- 0.143s was the duration of dumpsys device_config, ending at: 2025-10-27 17:07:57
            """)))
    )
    self.assertEqual(0, len(actual))

  def test_filters_mainline_beta_namespaces(self):
    actual = device_config_dump_parser.filter_mainline_beta_flags([
        device_config_dump_parser.Flag(
            "aaos_core_contrib",
            "com.android.window.flags.safe_region_letterboxing_v1",
            "true",
        ),
        device_config_dump_parser.Flag(
            "com_android_mainline_beta_mockup",
            "com.android.apex.flags.enable_brand_new_apex",
            "false",
        ),
        device_config_dump_parser.Flag(
            "accessibility",
            "android.content.res.font_scale_converter_public",
            "true",
        ),
        device_config_dump_parser.Flag(
            "com_android_tethering",
            "android.view.accessibility.a11y_character_in_window_api",
            "true",
        ),
        device_config_dump_parser.Flag(
            "com_android_tethering",
            "android.view.accessibility.foobar",
            "trueish",
        ),
    ])
    self.assertListEqual(
        actual,
        [
            device_config_dump_parser.Flag(
                "com_android_mainline_beta_mockup",
                "com.android.apex.flags.enable_brand_new_apex",
                "false",
            ),
            device_config_dump_parser.Flag(
                "com_android_tethering",
                "android.view.accessibility.a11y_character_in_window_api",
                "true",
            ),
        ],
    )

  def test_find_bugreport_inside_zip_with_main_entry_file_with_main_entry_txt(
      self,
  ):
    zip_file = create_in_memory_zip({
        "abc.txt": "ignored",
        "main_entry.txt": "bugreport-123.txt",
        # when "main_entry.txt" exists, it only returns the contents of that file
    })
    actual = device_config_dump_parser.find_bugreport_inside_zip(zip_file)
    self.assertEqual(actual, "bugreport-123.txt")

  def test_find_bugreport_inside_zip_with_main_entry_file_without_main_entry_txt(
      self,
  ):
    zip_file = create_in_memory_zip({
        "abc.txt": "ignored",
        "bugreport-123.txt": "ignored",
    })
    actual = device_config_dump_parser.find_bugreport_inside_zip(zip_file)
    self.assertEqual(actual, "bugreport-123.txt")

  def test_find_bugreport_inside_zip_without_options(self):
    zip_file = create_in_memory_zip({
        "abc.txt": "ignored",  # not a likely candidate
    })

    with self.assertRaises(SystemExit) as cm:
      device_config_dump_parser.find_bugreport_inside_zip(zip_file)

    self.assertEqual(cm.exception.code, 1)

  def test_write_flags_to_file(self):
    output = io.StringIO()
    device_config_dump_parser.write_flags_to_file(
        [
            device_config_dump_parser.Flag(
                "com_android_mainline_beta_mockup",
                "com.android.apex.flags.enable_brand_new_apex",
                "false",
            ),
            device_config_dump_parser.Flag(
                "com_android_tethering",
                "android.view.accessibility.a11y_character_in_window_api",
                "true",
            ),
        ],
        output,
    )
    self.assertEqual(
        output.getvalue(),
        textwrap.dedent("""\
            com_android_mainline_beta_mockup com.android.apex.flags.enable_brand_new_apex false
            com_android_tethering android.view.accessibility.a11y_character_in_window_api true
            """),
    )


if __name__ == "__main__":
  unittest.main(verbosity=2)

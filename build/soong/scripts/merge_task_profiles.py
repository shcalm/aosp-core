#!/usr/bin/env python
#
# Copyright (C) 2025 The Android Open Source Project
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

import argparse
import collections
import json
import sys

def merge_data(base, overlay):
    """
    Merges overlay into base.
    - Lists are appended.
    - Dictionaries are updated recursively.
    """
    for key, value in overlay.items():
        if key in base:
            # If both are lists (like "Profiles"), append the new ones
            if isinstance(base[key], list) and isinstance(value, list):
                base[key].extend(value)
            elif isinstance(base[key], dict) and isinstance(value, dict):
                merge_data(base[key], value)
            else:
                base[key] = value
        else:
            base[key] = value
    return base

def main():
    parser = argparse.ArgumentParser(description="Merge multiple task profile files into one.")
    parser.add_argument("output", help="output JSON file path")
    parser.add_argument('inputs', nargs='+', help='Input JSON files to merge (in order)')

    args = parser.parse_args()
    merged_output_profile = collections.OrderedDict()

    for input_file in args.inputs:
        try:
            with open(input_file, 'r') as f:
                overlay_profile = json.load(f, object_pairs_hook=collections.OrderedDict)
                merge_data(merged_output_profile, overlay_profile)
        except Exception as e:
            sys.stderr.write(f"Error reading {input_file}: {e}\n")
            sys.exit(1)

    with open(args.output, "w") as f:
        json.dump(merged_output_profile, f, indent=2, separators=(',', ': '))
        f.write('\n')

if __name__ == '__main__':
    main()

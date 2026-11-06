#!/usr/bin/env python3
#
# Copyright 2025 The Android Open Source Project
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

"""A script to analyze ninjago logs for hidden dependencies.

This script provides two main functionalities:
1. Analyze a .ninja_log file (or its content fetched via a SHA hash) to find
   hidden dependencies for each target. It runs 'abfs inspect' on the response
   hash of each target to discover these dependencies. The results are saved
   to sharded binary protobuf files.

   Usage for analysis:
   python3 analyzer.py analyze <path_to_.ninja_log | ninja_log_hash> \\
       [--output_proto <output_path>] \\
       [--shard_size <size>]

   Example with a path:
   python3 analyzer.py analyze out/.ninja_log

   Example with a hash:
   python3 analyzer.py analyze 3d05466d0a1b0a41948b90fc3e9c021905d85c00

2. Convert the binary protobuf files into a human-readable textproto format.
   This is useful for debugging and manual inspection of the results.

   Usage for conversion:
   python3 analyzer.py convert <input_proto_file_1> [<input_proto_file_2> ...]

   Example:
   python3 analyzer.py convert out/ninjago_log.pb.0 out/ninjago_log.pb.1
"""

import argparse
import asyncio
import json
import os
import re
import sys
from dataclasses import dataclass
from typing import List, Optional

from google.protobuf import text_format
import ninjago_log_pb2

@dataclass
class NinjaLogEntry:
    """A dataclass to hold a single entry from a ninjago log."""
    name: str
    request_hash: str
    response_hash: str

def parse_ninja_log_content(log_content: str) -> List[NinjaLogEntry]:
    """Parses the content of a .ninja_log file and returns a list of targets."""
    targets = []
    header_re = re.compile(r"# ninjago log v\d+")
    lines = log_content.splitlines()

    in_ninjago_log = False
    for line in lines:
        if header_re.match(line):
            in_ninjago_log = True
            continue
        if not in_ninjago_log or line.startswith('#'):
            continue

        parts = line.strip().split()
        if len(parts) < 7:
            continue

        # The format of a ninjago log entry is:
        # Timestamp Duration Cached Request-Hash Response-Hash Mismatches Target
        # We are interested in Request-Hash (3), Response-Hash (4), and Target (6 onwards)
        request_hash = parts[3]
        response_hash = parts[4]
        target_name = ' '.join(parts[6:])

        targets.append(NinjaLogEntry(
            name=target_name,
            request_hash=request_hash,
            response_hash=response_hash,
        ))

    if not in_ninjago_log:
        sys.exit("Error: Not a ninjago log file. Missing ninjago log header.")

    return targets


async def get_ninja_log_from_hash_async(log_hash: str) -> str:
    """Runs 'abfs cat' asynchronously and returns the content of the ninja log."""
    try:
        # Create a subprocess to run 'abfs cat'.
        proc = await asyncio.create_subprocess_exec(
            'abfs', 'cat', log_hash,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        stdout, stderr = await proc.communicate()

        if proc.returncode == 0:
            return stdout.decode()
        else:
            sys.exit(f"Error getting ninja log from hash {log_hash}: {stderr.decode()}")
    except Exception as e:
        sys.exit(f"Exception while getting ninja log from hash {log_hash}: {e}")

async def get_hidden_deps_async(response_hash: str) -> List[str]:
    """Runs 'abfs inspect -j' asynchronously and returns the hidden dependencies."""
    hidden_deps = []
    try:
        # Create a subprocess to run 'abfs inspect -j'.
        proc = await asyncio.create_subprocess_exec(
            'abfs', 'inspect', '-j', response_hash,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        stdout, stderr = await proc.communicate()

        if proc.returncode == 0:
            data = json.loads(stdout)
            if 'ReadPatch' in data:
                prev_path = ""
                for item in data['ReadPatch']:
                    if len(item) < 5:
                        continue

                    relative_path = item[0]
                    # Reconstruct the full path.
                    if relative_path.startswith('/'):
                        current_path = relative_path[1:]
                    else:
                        current_path = os.path.normpath(os.path.join(os.path.dirname(prev_path), relative_path))
                    prev_path = current_path

                    # The 5th element (index 4) is the state. '7' indicates a hidden dependency.
                    # This is defined in `abfs/entities/patch.go`.
                    if item[4] == '7':
                        hidden_deps.append(current_path)
        else:
            print(f"Error inspecting response hash {response_hash}: {stderr.decode()}", file=sys.stderr)
    except json.JSONDecodeError as e:
        print(f"Error decoding JSON for response hash {response_hash}: {e}", file=sys.stderr)
    except Exception as e:
        print(f"Exception while inspecting response hash {response_hash}: {e}", file=sys.stderr)
    return hidden_deps

async def process_target_async(
    target_info: NinjaLogEntry, semaphore: asyncio.Semaphore
) -> Optional[ninjago_log_pb2.NinjaTarget]:
    """
    Processes a single target asynchronously to find its hidden dependencies.
    Returns a NinjaTarget protobuf message if hidden dependencies are found,
    otherwise returns None.
    """
    async with semaphore:
        hidden_deps = await get_hidden_deps_async(target_info.response_hash)
        if not hidden_deps:
            return None

        # Create a protobuf message for the target.
        ninja_target = ninjago_log_pb2.NinjaTarget()
        ninja_target.name = target_info.name
        ninja_target.request_hash = target_info.request_hash
        ninja_target.response_hash = target_info.response_hash
        ninja_target.hidden_deps.extend(hidden_deps)

        return ninja_target

async def run_analyze(args: argparse.Namespace) -> None:
    """Runs the analysis of the ninjago log."""
    ninja_log_content = None
    if os.path.exists(args.ninja_log_or_hash):
        with open(args.ninja_log_or_hash, 'r') as f:
            ninja_log_content = f.read()
    else:
        # Check if it's a valid SHA1 hash.
        if not re.fullmatch(r'[0-9a-fA-F]{40}', args.ninja_log_or_hash):
            sys.exit(f"Error: '{args.ninja_log_or_hash}' is not a valid file path or a SHA1 hash.")
        ninja_log_content = await get_ninja_log_from_hash_async(args.ninja_log_or_hash)

    # Determine the output path for the protobuf file.
    output_proto_path = args.output_proto
    if not output_proto_path:
        out_dir = os.environ.get('OUT_DIR', 'out')
        output_proto_path = os.path.join(out_dir, 'ninjago_log.pb')

    # Parse the ninja log to get the list of targets.
    targets = parse_ninja_log_content(ninja_log_content)

    # Use a semaphore to limit the number of concurrent `abfs inspect` calls.
    # Without this, the script can spawn thousands of subprocesses simultaneously,
    # exhausting system resources and causing 'Resource temporarily unavailable'
    # errors. This semaphore ensures that no more than a set number of
    # subprocesses run at once, making the analysis more stable.
    semaphore = asyncio.Semaphore(100)
    total_targets_written = 0

    # Process targets in shards to avoid creating a single large protobuf file.
    for i in range(0, len(targets), args.shard_size):
        shard_targets = targets[i:i + args.shard_size]
        shard_index = i // args.shard_size
        shard_output_path = f"{output_proto_path}.{shard_index}"

        # Process the shard of targets concurrently.
        tasks = [process_target_async(t, semaphore) for t in shard_targets]
        results = await asyncio.gather(*tasks)

        # Filter out targets that don't have hidden dependencies.
        targets_with_hidden_deps = [res for res in results if res]
        if not targets_with_hidden_deps:
            print(f"Shard {shard_index}: No targets with hidden deps found. Skipping file write.")
            continue

        # Write the targets with hidden dependencies to a protobuf file.
        ninja_log = ninjago_log_pb2.NinjaLog()
        ninja_log.targets.extend(targets_with_hidden_deps)

        with open(shard_output_path, 'wb') as f:
            f.write(ninja_log.SerializeToString())

        print(f"Successfully wrote {len(targets_with_hidden_deps)} targets to {shard_output_path}")
        total_targets_written += len(targets_with_hidden_deps)

    print(f"\nTotal targets written across all shards: {total_targets_written}")

async def run_convert(args: argparse.Namespace) -> None:
    """Converts protobuf file(s) to textproto."""
    # Process each input protobuf file.
    for input_path in args.input_proto:
        if not os.path.exists(input_path):
            sys.exit(f"Error: {input_path} not found.")

        # Determine the output path for the textproto file.
        output_textproto_path = args.output_textproto
        if len(args.input_proto) > 1 or not output_textproto_path:
            output_textproto_path = input_path + '.textproto'
        elif not output_textproto_path:
            base_name = os.path.splitext(input_path)[0]
            output_textproto_path = base_name + '.textproto'

        # Read the protobuf file and write it to a textproto file.
        ninja_log = ninjago_log_pb2.NinjaLog()
        with open(input_path, 'rb') as f:
            ninja_log.ParseFromString(f.read())

        with open(output_textproto_path, 'w') as f:
            f.write(text_format.MessageToString(ninja_log))

        print(f"Successfully converted {input_path} to {output_textproto_path}")

async def main() -> None:
    # Set up argument parser.
    parser = argparse.ArgumentParser(description="Analyze ninjago logs for hidden dependencies.")
    subparsers = parser.add_subparsers(dest='command', required=True)

    # --analyze mode: finds hidden dependencies and saves them to a protobuf file.
    parser_analyze = subparsers.add_parser('analyze', help="Analyze a .ninja_log file.")
    parser_analyze.add_argument('ninja_log_or_hash', help="Path to the .ninja_log file or its SHA hash.")
    parser_analyze.add_argument('--output_proto', help="Path to write the output protobuf file.")
    parser_analyze.add_argument('--shard_size', type=int, default=50000, help="Number of targets per shard.")
    parser_analyze.set_defaults(func=run_analyze)

    # --convert mode: converts a protobuf file to a human-readable textproto file.
    parser_convert = subparsers.add_parser('convert', help="Convert protobuf file(s) to textproto.")
    parser_convert.add_argument('input_proto', nargs='+', help="Path(s) to the input protobuf file(s).")
    parser_convert.add_argument('--output_textproto', help="Path to write the output textproto file (only used if a single input file is provided).")
    parser_convert.set_defaults(func=run_convert)

    args = parser.parse_args()
    await args.func(args)

if __name__ == "__main__":
    asyncio.run(main())

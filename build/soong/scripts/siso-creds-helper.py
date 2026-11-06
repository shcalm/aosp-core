#!/usr/bin/env python3

import os
import shlex
import subprocess
import sys


def run_credshelper():
  """Executes the 'credshelper' command and parses its stdout as JSON."""

  out_dir = os.environ.get("OUT_DIR")
  if not out_dir:
    out_dir = "out"
  cmd_file = os.path.join(out_dir, "soong", "rbe", "soong-convert-command")

  command = []
  try:
    with open(cmd_file, "r") as f:
      command_string = f.read().strip()
    if not command_string:
      print(f"Error: Command file '{cmd_file}' is empty.", file=sys.stderr)
      sys.exit(3)
    # Use shlex.split to correctly handle quoted arguments
    command = shlex.split(command_string)

  except FileNotFoundError:
    print(f"Error: Command file not found: '{cmd_file}'", file=sys.stderr)
    sys.exit(3)
  except Exception as e:
    print(
        f"Error reading or parsing command file '{cmd_file}': {e}",
        file=sys.stderr,
    )
    sys.exit(3)

  subprocess.run(command)

if __name__ == "__main__":
  run_credshelper()

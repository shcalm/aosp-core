#!/usr/bin/env python3

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

"""Converts a set of aconfig protobuf flags to a Metalava config file."""

# Formatted using `pyformat -i scripts/aconfig-to-metalava-flags.py`

import argparse
import sys
import xml.etree.ElementTree as ET

from protos import aconfig_pb2

_READ_ONLY = aconfig_pb2.flag_permission.READ_ONLY
_READ_WRITE = aconfig_pb2.flag_permission.READ_WRITE
_ENABLED = aconfig_pb2.flag_state.ENABLED
_DISABLED = aconfig_pb2.flag_state.DISABLED

# The namespace of the Metalava config file.
CONFIG_NS = 'http://www.google.com/tools/metalava/config'


def config_name(tag: str):
  """Create a QName in the config namespace.

  :param:tag the name of the entity in the config namespace.
  """
  return f'{{{CONFIG_NS}}}{tag}'

def bool_to_str(b: bool):
  """Convert a bool to a lower-case only string

  :param:b the bool to convert
  """
  return str(b).lower()

def permission_to_mutability(permission):
  if permission == _READ_ONLY:
    mutability = 'immutable'
  else:
    mutability = 'mutable'
  return mutability

def state_to_status(state):
  if state == _ENABLED:
    status = 'enabled'
  else:
    status = 'disabled'
  return status


def main():
  """Program entry point."""
  args_parser = argparse.ArgumentParser(
      description='Generate Metalava flags config from aconfig protobuf',
  )
  args_parser.add_argument(
      '--override-flag-state',
      choices=("ENABLED", "DISABLED"),
      help='Pretend all flags have the given state',
  )
  args_parser.add_argument(
      '--override-flag-permission',
      choices=("READ_ONLY", "READ_WRITE"),
      help='Pretend all flags have the given state',
  )
  args_parser.add_argument(
      'input',
      help='The path to the aconfig protobuf file',
  )
  args = args_parser.parse_args(sys.argv[1:])

  # Read the parsed_flags from the protobuf file.
  with open(args.input, 'rb') as f:
    parsed_flags = aconfig_pb2.parsed_flags.FromString(f.read())

  # Create the structure of the XML config file.
  config = ET.Element(config_name('config'))
  api_flags = ET.SubElement(config, config_name('api-flags'))

  override_flag_state = {
    "ENABLED": _ENABLED,
    "DISABLED": _DISABLED,
  }.get(args.override_flag_state)

  override_flag_permission = {
      "READ_ONLY": _READ_ONLY,
      "READ_WRITE": _READ_WRITE,
  }.get(args.override_flag_permission)

  if override_flag_state or override_flag_permission:
    # Make sure that unknown flags are treated the same way as all other flags.
    ET.SubElement(api_flags, config_name('unknown-flags'), {
        config_name('mutability'): permission_to_mutability(override_flag_permission),
        config_name('status'): state_to_status(override_flag_state),
    })

  # Create an <api-flag> element for each parsed_flag.
  for flag in parsed_flags.parsed_flag:
    state = override_flag_state or flag.state
    permission = override_flag_permission or flag.permission

    if permission == _READ_ONLY and state == _DISABLED:
      # Ignore any read only disabled flags as Metalava assumes that as the
      # default when an <api-flags/> element is provided so this reduces the
      # size of the file.
      continue

    attributes = {
        'package': flag.package,
        'name': flag.name,
        'mutability': permission_to_mutability(permission),
        'status': state_to_status(state),
        'is-exported': bool_to_str(flag.is_exported),
    }

    # Convert the attribute names into qualified names in, what will become, the
    # default namespace for the XML file. This is needed to ensure that the
    # attribute will be written in the XML file without a prefix, e.g.
    # `name="flag_name"`. Without it, a namespace prefix, e.g. `ns1`, will be
    # synthesized for the attribute when writing the XML file, i.e. it
    # will be written as `ns1:name="flag_name"`. Strictly speaking, that is
    # unnecessary as the "Namespaces in XML 1.0 (Third Edition)" specification
    # says that unprefixed attribute names have no namespace.
    qualified_attributes = {config_name(k): v for (k, v) in attributes.items()}
    ET.SubElement(api_flags, config_name('api-flag'), qualified_attributes)

  # Create a tree and add white space so it will pretty print when written out.
  tree = ET.ElementTree(config)
  ET.indent(tree)

  # Write the tree using the config namespace as the default.
  tree.write(sys.stdout, encoding='unicode', default_namespace=CONFIG_NS)
  sys.stdout.close()


if __name__ == '__main__':
  main()

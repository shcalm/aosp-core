#!/usr/bin/env python
#
# Copyright (C) 2016 The Android Open Source Project
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
#
"""Generates source for stub shared libraries for the NDK."""
import argparse
import json
import logging
from pathlib import Path
import sys
from typing import Iterable, List, TextIO, Tuple

import symbolfile
from symbolfile import Arch, Symbol, Version


class BaseGenerator:
    """Base output generator that writes stub source files and version scripts."""
    def __init__(self, src_file: TextIO, version_script: TextIO,
                 filt: symbolfile.Filter) -> None:
        self.src_file = src_file
        self.version_script = version_script
        self.filter = filt
        self.api = filt.api

    def get_pruned_symbols(self, version: Version) -> Tuple[List[Symbol], bool]:
        """Returns a list of symbols to write and a boolean indicating if the version is empty."""
        version_empty = True
        pruned_symbols = []
        for symbol in version.symbols:
            if self.filter.should_omit_symbol(symbol):
                continue

            if symbolfile.symbol_versioned_in_api(symbol.tags, self.api):
                version_empty = False

            pruned_symbols.append(symbol)

        return pruned_symbols, version_empty


class Generator(BaseGenerator):
    """Output generator that writes stub source files and version scripts."""
    def __init__(self, src_file: TextIO, version_script: TextIO,
                 symbol_list: TextIO, filt: symbolfile.Filter) -> None:
        super().__init__(src_file, version_script, filt)
        self.symbol_list = symbol_list

    def write(self, versions: Iterable[Version]) -> None:
        """Writes all symbol data to the output files."""
        self.symbol_list.write('[abi_symbol_list]\n')
        for version in versions:
            self.write_version(version)

    def write_version(self, version: Version) -> None:
        """Writes a single version block's data to the output files."""
        if self.filter.should_omit_version(version):
            return

        section_versioned = symbolfile.symbol_versioned_in_api(
            version.tags, self.api)
        pruned_symbols, version_empty = self.get_pruned_symbols(version)

        if len(pruned_symbols) > 0:
            if not version_empty and section_versioned:
                self.version_script.write(version.name + ' {\n')
                self.version_script.write('    global:\n')
            for symbol in pruned_symbols:
                emit_version = symbolfile.symbol_versioned_in_api(
                    symbol.tags, self.api)
                if section_versioned and emit_version:
                    self.version_script.write('        ' + symbol.name + ';\n')

                weak = ''
                if 'weak' in symbol.tags:
                    weak = '__attribute__((weak)) '

                if 'var' in symbol.tags:
                    self.src_file.write(f'{weak}int {symbol.name} = 0;\n')
                else:
                    self.src_file.write(f'{weak}void {symbol.name}() {{}}\n')

                self.symbol_list.write(f'{symbol.name}\n')

            if not version_empty and section_versioned:
                base = '' if version.base is None else ' ' + version.base
                self.version_script.write('}' + base + ';\n')


class ArtlessDenylistGenerator(BaseGenerator):
    """Output generator that writes stub source files and version scripts for artless denylist."""
    def __init__(self, src_file: TextIO, version_script: TextIO,
                 blocked_symbol_list: TextIO, filt: symbolfile.Filter,
                 symbol_file_name: str) -> None:
        super().__init__(src_file, version_script, filt)
        self.blocked_symbol_list = blocked_symbol_list
        self.symbol_file_name = symbol_file_name

    def write(self, versions: Iterable[Version]) -> None:
        """Writes all symbol data to the output files."""
        self.src_file.write('#include <log/log.h>\n')
        self.blocked_symbol_list.write(f'# {self.symbol_file_name}\n')
        for version in versions:
            self.write_version(version)

    def write_version(self, version: Version) -> None:
        """Writes a single version block's data to the output files."""
        # Only write stubs for public symbols. The stub applies equally to
        # app and internal calls, and we neither want to break internal calls
        # nor maintain a list of allowed internal calls.
        if self.filter.should_omit_version(version):
            return

        pruned_symbols, _ = self.get_pruned_symbols(version)

        for symbol in pruned_symbols:
            if symbol.tags.has_artless_tags:
                continue

            weak = ''
            if 'weak' in symbol.tags:
                weak = '__attribute__((weak)) '

            if 'var' not in symbol.tags:
                self.src_file.write(
                    f'{weak}void {symbol.name}() {{ '
                    f'LOG_ALWAYS_FATAL("Function {symbol.name} is not allowed in artless processes."); }}\n'
                )
                self.blocked_symbol_list.write(f'{symbol.name}\n')


def parse_args() -> argparse.Namespace:
    """Parses and returns command line arguments."""
    parser = argparse.ArgumentParser()

    def resolved_path(raw: str) -> Path:
        """Returns a resolved Path for the given string."""
        return Path(raw).resolve()

    parser.add_argument('-v', '--verbose', action='count', default=0)

    parser.add_argument(
        '--api', required=True, help='API level being targeted.')
    parser.add_argument(
        '--arch', choices=symbolfile.ALL_ARCHITECTURES, required=True,
        help='Architecture being targeted.')
    parser.add_argument(
        '--llndk', action='store_true', help='Use the LLNDK variant.')
    parser.add_argument(
        '--apex',
        action='store_true',
        help='Use the APEX variant.')
    parser.add_argument(
        '--systemapi',
        action='store_true',
        dest='systemapi',
        help='Use the SystemAPI variant.')
    parser.add_argument(
        '--no-ndk',
        action='store_false',
        dest='ndk',
        help='Do not include NDK APIs.')
    parser.add_argument(
        '--artless-denylist',
        action='store_true',
        dest='artless_denylist',
        help='Generate denylist based on the artless tag')

    parser.add_argument('--api-map',
                        type=resolved_path,
                        required=True,
                        help='Path to the API level map JSON file.')

    parser.add_argument('symbol_file',
                        type=resolved_path,
                        help='Path to symbol file.')
    parser.add_argument('stub_src',
                        type=resolved_path,
                        help='Path to output stub source file.')
    parser.add_argument('version_script',
                        type=resolved_path,
                        help='Path to output version script.')
    parser.add_argument('symbol_list',
                        type=resolved_path,
                        help='Path to output abigail symbol list (if not artless-denylist) or blocked symbol list (if artless-denylist).')

    return parser.parse_args()


def main() -> None:
    """Program entry point."""
    args = parse_args()

    with args.api_map.open() as map_file:
        api_map = json.load(map_file)
    api = symbolfile.decode_api_level(args.api, api_map)

    verbose_map = (logging.WARNING, logging.INFO, logging.DEBUG)
    verbosity = args.verbose
    if verbosity > 2:
        verbosity = 2
    logging.basicConfig(level=verbose_map[verbosity])

    filt = symbolfile.Filter(args.arch, api, args.llndk, args.apex, args.systemapi, args.ndk)
    with args.symbol_file.open() as symbol_file:
        try:
          versions = symbolfile.SymbolFileParser(symbol_file, api_map, filt).parse()
        except symbolfile.MultiplyDefinedSymbolError as ex:
            sys.exit(f'{args.symbol_file}: error: {ex}')

    with args.stub_src.open('w') as src_file:
        with args.version_script.open('w') as version_script:
            with args.symbol_list.open('w') as symbol_list:
                if args.artless_denylist:
                    generator = ArtlessDenylistGenerator(src_file, version_script,
                                                         symbol_list, filt,
                                                         args.symbol_file.name)
                else:
                    generator = Generator(src_file, version_script, symbol_list,
                                          filt)
                generator.write(versions)


if __name__ == '__main__':
    main()

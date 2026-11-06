# Copyright (C) 2026 The Android Open Source Project
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

from .env import BuildContext
from .build import build_targets
from .config import get_build_vars, get_aconfig_flag
from .ninja import query_ninja_target, depends_on, get_command, NinjaQuery, NinjaTargetNotFoundError
from .module import get_module_info, ModuleInfo
from .constants import *

__all__ = ["build_targets", "get_build_vars", "get_aconfig_flag", "query_ninja_target", "depends_on", "get_command", "NinjaQuery", "NinjaTargetNotFoundError", "BuildContext", "get_module_info", "ModuleInfo"]

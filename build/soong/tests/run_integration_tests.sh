#!/bin/bash -eu

set -o pipefail

TOP="$(readlink -f "$(dirname "$0")"/../../..)"

# Pre-warm the mock top and export it to all the subtests
source "$TOP/build/soong/tests/lib.sh"
export WARMED_UP_MOCK_TOP
export MOCK_TOP

"$TOP/build/soong/tests/androidmk_test.sh"
"$TOP/build/soong/tests/bootstrap_test.sh"
"$TOP/build/soong/tests/soong_test.sh"
"$TOP/build/soong/tests/java_partial_compile_test.sh"
"$TOP/build/soong/tests/kotlin_partial_compile_test.sh"
"$TOP/build/soong/tests/build_action_caching_test.sh"

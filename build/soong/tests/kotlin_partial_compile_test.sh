#!/bin/bash -eu

set -o pipefail

# This test checks partial_compile features
source "$(dirname "$0")/java_partial_compile_setup.sh"
source "$(dirname "$0")/compare_jars.sh"
source "$(dirname "$0")/lib.sh"

impl_library_jar=out/soong/.intermediates/soong-test/java/integration/impl-library/android_common/javac/impl-library.jar

extend_mock_top extend_mock_top_for_kotlin_partial_compile_test

function test_kt_remove_referenced_method {
  setup
  set_partial_compile_flags

  run_soong
  run_ninja ${impl_library_jar}

  sed -i".tmp" -e '/fun foo\(\)/d' soong-test/java/integration/impllib/KtClass.kt

  run_ninja ${impl_library_jar} && \
    fail "impl-library built with incorrect java compilation"

  echo "test_kt_remove_referenced_method test passed"
}

scan_and_run_tests "$@"

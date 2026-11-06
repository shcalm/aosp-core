#!/bin/bash -eu

set -o pipefail

# This test checks partial_compile features
source "$(dirname "$0")/java_partial_compile_setup.sh"
source "$(dirname "$0")/compare_jars.sh"
source "$(dirname "$0")/lib.sh"

impl_library_jar=out/soong/.intermediates/soong-test/java/integration/impl-library/android_common/javac/impl-library.jar

extend_mock_top extend_mock_top_for_java_partial_compile_test

function test_add_new_file {
  setup
  set_partial_compile_flags

  # add a new file
  create_example_impl3_file
  run_ninja ${impl_library_jar}
  # copy out jar
  mkdir -p out/test_add_new_file/partial_compile && cp ${impl_library_jar} out/test_add_new_file/partial_compile/impl-library.jar

## Now run with full_compile setup
  unset_partial_compile_flags
  run_soong
  run_ninja ${impl_library_jar}

  # copy out jar
  mkdir -p out/test_add_new_file/full_compile && cp ${impl_library_jar} out/test_add_new_file/full_compile/impl-library.jar

  # compare the two jar's for equality
  assert_jars_equal out/test_add_new_file/full_compile/impl-library.jar out/test_add_new_file/partial_compile/impl-library.jar || \
    fail "Failed: full-compile and partial-compile outputs do not match"

  echo "test_add_new_file test passed"

  remove_base_dir
}

function test_move_file {
  setup
  set_partial_compile_flags
  run_ninja ${impl_library_jar}

  # create a new directory
  mkdir -p soong-test/java/integration/impllib/newimpl
  # rename soong-test/java/integration/impllib/ExampleImpl1.java to soong-test/java/integration/impllib/newimpl/ExampleImpl1.java
  mv soong-test/java/integration/impllib/ExampleImpl1.java soong-test/java/integration/impllib/newimpl/ExampleImpl1.java
  run_ninja ${impl_library_jar}
  # copy out jar
  mkdir -p out/test_move_file/partial_compile && cp ${impl_library_jar} out/test_move_file/partial_compile/impl-library.jar

## Now run with full_compile setup
  unset_partial_compile_flags
  run_soong
  run_ninja ${impl_library_jar}
  # copy out jar
  mkdir -p out/test_move_file/full_compile && cp ${impl_library_jar} out/test_move_file/full_compile/impl-library.jar

  # compare the two jar's for equality
  assert_jars_equal out/test_move_file/full_compile/impl-library.jar out/test_move_file/partial_compile/impl-library.jar || \
    fail "Failed: full-compile and partial-compile outputs do not match"

  echo "test_move_file test passed"

  remove_base_dir
}

function test_remove_file {
  setup
  set_partial_compile_flags

  # remove soong-test/java/integration/impllib/ExampleImpl3.java
  rm -rf soong-test/java/integration/impllib/ExampleImpl3.java
  run_ninja ${impl_library_jar}
  # copy out jar
  mkdir -p out/test_remove_file/partial_compile && cp ${impl_library_jar} out/test_remove_file/partial_compile/impl-library.jar

## Now run with full_compile setup
  unset_partial_compile_flags
  run_soong
  run_ninja ${impl_library_jar}

  mkdir -p out/test_remove_file/full_compile && cp ${impl_library_jar} out/test_remove_file/full_compile/impl-library.jar

  # compare the two jar's for equality
  assert_jars_equal out/test_remove_file/full_compile/impl-library.jar out/test_remove_file/partial_compile/impl-library.jar || \
    fail "Failed: full-compile and partial-compile outputs do not match"

  echo "test_remove_file test passed"

  remove_base_dir
}

function test_bp_modification {
  setup
  set_partial_compile_flags

  #modify the bp file
  cat > soong-test/java/integration/Android.bp <<'EOF'
java_library {
    name: "impl-library",
    srcs: [
        "impllib/**/*.java",
    ],
    libs: ["provider-library-new"],
    sdk_version: "35",
}

java_library {
    name: "provider-library-new",
    srcs: ["provider/**/*.java"],
    sdk_version: "35",
}
EOF

  run_ninja ${impl_library_jar}
  # copy out jar
  mkdir -p out/test_bp_modification/partial_compile && cp ${impl_library_jar} out/test_bp_modification/partial_compile/impl-library.jar

## Now run with full_compile setup
  unset_partial_compile_flags
  run_soong
  run_ninja ${impl_library_jar}

  # copy out jar
  mkdir -p out/test_bp_modification/full_compile && cp ${impl_library_jar} out/test_bp_modification/full_compile/impl-library.jar

  # compare the two jar's for equality
  assert_jars_equal out/test_bp_modification/full_compile/impl-library.jar out/test_bp_modification/partial_compile/impl-library.jar || \
    fail "Failed: full-compile and partial-compile outputs do not match"

  echo "test_bp_modification test passed"

  remove_base_dir
}

function test_incorrect_file_modification {
  setup
  set_partial_compile_flags
  create_example_impl3_file
  run_ninja ${impl_library_jar}

  readonly ERROR_LOG=${MOCK_TOP}/out/error.log
  readonly ERROR_RULE="FAILED: //soong-test/java/integration:impl-library javac-inc"
  readonly ERROR_MESSAGE="soong-test/java/integration/impllib/ExampleImpl3.java:15"

  #modify a java file incorrectly, inc-javac should fail
  modify_example_impl1_file

  run_ninja ${impl_library_jar} && \
    fail "impl-library built with incorrect java compilation"

  if grep -q "${ERROR_RULE}" "${ERROR_LOG}" && grep -q "${ERROR_MESSAGE}" "${ERROR_LOG}" ; then
    echo Error rule and error message found in logs >/dev/null
  else
    fail "Did not find javac failure error AND file error message in error.log"
  fi

  echo "test_incorrect_file_modification test passed"
  remove_base_dir
}

function test_incorrect_cross_module_file_modification {
  setup
  set_partial_compile_flags

  local readonly ERROR_LOG=${MOCK_TOP}/out/error.log
  local readonly ERROR_RULE="FAILED: //soong-test/java/integration:impl-library javac-inc"
  local readonly ERROR_MESSAGE="soong-test/java/integration/impllib/ExampleImpl1.java:19"

  #modify a cross-module java file incorrectly, inc-javac should fail
  modify_provider_file

  run_ninja ${impl_library_jar} && \
    fail "impl-library built with incorrect java compilation"

  if grep -q "${ERROR_RULE}" "${ERROR_LOG}" && grep -q "${ERROR_MESSAGE}" "${ERROR_LOG}" ; then
    echo Error rule and error message found in logs >/dev/null
  else
    fail "Did not find javac failure error AND file error message in error.log"
  fi

  echo "test_incorrect_cross_module_file_modification test passed"
  remove_base_dir
}

function test_incorrect_api_generating_ap_modification {
  setup
  set_partial_compile_flags
  create_ap_files
  run_soong
  run_ninja ${impl_library_jar}

  local readonly ERROR_LOG=${MOCK_TOP}/out/error.log
  local readonly ERROR_RULE="FAILED: //soong-test/java/integration:impl-library javac-inc"
  local readonly ERROR_MESSAGE="soong-test/java/integration/impllib/ExampleImpl4.java:9"

  #modify the API surface generated by AP, which is being used in impl-library
  modify_annotation_api

  run_ninja ${impl_library_jar} && \
    fail "impl-library built with incorrect java compilation"

  if grep -q "${ERROR_RULE}" "${ERROR_LOG}" && grep -q "${ERROR_MESSAGE}" "${ERROR_LOG}" ; then
    echo Error rule and error message found in logs >/dev/null
  else
    fail "Did not find javac failure error AND file error message in error.log"
  fi

  echo "test_incorrect_api_generating_ap_modification test passed"
  remove_base_dir
}

function test_remove_inner_class {
  setup
  set_partial_compile_flags
  create_ap_files

  cat > soong-test/java/integration/impllib/ExampleInnerClass.java <<'EOF'
package soong.java.integration.impllib;

public class ExampleInnerClass {
  public static class InnerClass {
  }
}
EOF

  run_soong
  run_ninja ${impl_library_jar}

  if [ -z "$(zipinfo ${impl_library_jar} | grep 'soong/java/integration/impllib/ExampleInnerClass\$InnerClass.class')" ]; then
    fail "Missing ExampleInnerClass\$InnerClass.class in impl library"
  fi

  cat > soong-test/java/integration/impllib/ExampleInnerClass.java <<'EOF'
package soong.java.integration.impllib;

public class ExampleInnerClass {
}
EOF

  run_ninja ${impl_library_jar}

  if [ -n "$(zipinfo ${impl_library_jar} | grep 'soong/java/integration/impllib/ExampleInnerClass\$InnerClass.class')" ]; then
    fail "ExampleInnerClass\$InnerClass.class not removed from impl library"
  fi


  echo "test_remove_inner_class test passed"
}

scan_and_run_tests "$@"

#!/bin/bash -eu

set -o pipefail

# How to run: bash path-to-script/androidmk_test.sh
# Tests of converting license functionality of the androidmk tool

source "$(dirname "$0")/lib.sh"

function extend_mock_top_for_androidmk_test {
    # Precompile androidmk for the tests in androidmk_test.sh.
    run_ninja androidmk
}

extend_mock_top extend_mock_top_for_androidmk_test

# Expect to create a new license module
function test_rewrite_license_property_inside_current_directory {
  setup

  # Create an Android.mk file
  mkdir -p a/b
  cat > a/b/Android.mk <<'EOF'
include $(CLEAR_VARS)
LOCAL_MODULE := foo
LOCAL_LICENSE_KINDS := license_kind1 license_kind2
LOCAL_LICENSE_CONDITIONS := license_condition
LOCAL_NOTICE_FILE := $(LOCAL_PATH)/license_notice1 $(LOCAL_PATH)/license_notice2
include $(BUILD_PACKAGE)
EOF

  # Create an expected Android.bp file for the module "foo"
  cat > a/b/Android.bp <<'EOF'
package {
    // See: http://go/android-license-faq
    default_applicable_licenses: [
        "a_b_license",
    ],
}

license {
    name: "a_b_license",
    visibility: [":__subpackages__"],
    license_kinds: [
        "license_kind1",
        "license_kind2",
    ],
    license_text: [
        "license_notice1",
        "license_notice2",
    ],
}

android_app {
    name: "foo",
}
EOF

  run_androidmk_test "a/b/Android.mk" "a/b/Android.bp"
}

# Expect to reference to an existing license module
function test_rewrite_license_property_outside_current_directory {
  setup

  # Create an Android.mk file
  mkdir -p a/b/c/d
  cat > a/b/c/d/Android.mk <<'EOF'
include $(CLEAR_VARS)
LOCAL_MODULE := foo
LOCAL_LICENSE_KINDS := license_kind1 license_kind2
LOCAL_LICENSE_CONDITIONS := license_condition
LOCAL_NOTICE_FILE := $(LOCAL_PATH)/../../license_notice1 $(LOCAL_PATH)/../../license_notice2
include $(BUILD_PACKAGE)
EOF

  # Create an expected (input) Android.bp file at a/b/
  cat > a/b/Android.bp <<'EOF'
package {
    // See: http://go/android-license-faq
    default_applicable_licenses: [
        "a_b_license",
    ],
}

license {
    name: "a_b_license",
    visibility: [":__subpackages__"],
    license_kinds: [
        "license_kind1",
        "license_kind2",
    ],
    license_text: [
        "license_notice1",
        "license_notice2",
    ],
}

android_app {
    name: "bar",
}
EOF

  # Create an expected (output) Android.bp file for the module "foo"
  cat > a/b/c/d/Android.bp <<'EOF'
package {
    // See: http://go/android-license-faq
    default_applicable_licenses: [
        "a_b_license",
    ],
}

android_app {
    name: "foo",
}
EOF

  run_androidmk_test "a/b/c/d/Android.mk" "a/b/c/d/Android.bp"
}

function run_androidmk_test {
  local -r androidmk=(*/host/*/bin/androidmk)
  if [[ ${#androidmk[@]} -ne 1 ]]; then
    fail "Multiple androidmk binaries found: ${androidmk[*]}"
  fi
  local -r out=$(ANDROID_BUILD_TOP="$PWD" "${androidmk[0]}" "$1")
  local -r expected=$(cat "$2")

  if [[ "$out" != "$expected" ]]; then
    fail "The output is not the same as the expected"
  fi

  echo "Succeeded"
}

scan_and_run_tests

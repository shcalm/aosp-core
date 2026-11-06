#!/bin/bash -eu

set -o pipefail

: "${OUT_DIR:?Must set OUT_DIR}"

TOP=$(cd $(dirname $0)/../../..; pwd)

UNAME="$(uname)"
case "$UNAME" in
Linux)
    OS='linux'
    ;;
Darwin)
    OS='darwin'
    ;;
*)
    exit 1
    ;;
esac

export PATH="$TOP/prebuilts/build-tools/path/$OS-x86":$PATH

mkdir -p ${OUT_DIR}
export TMPDIR=$(cd ${OUT_DIR}; pwd)/tmp
mkdir -p ${TMPDIR}

"$TOP/build/soong/scripts/run-soong-tests-with-go-tools.sh"
"$TOP/build/soong/tests/run_integration_tests.sh"

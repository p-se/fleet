#!/bin/bash
#
# Usage
#   [FILTER=<label-filter>] dev/run-integration-tests.sh [<path-for-tests>] [go-test-arguments]

set -euxo pipefail

SETUP_ENVTEST_VER=${SETUP_ENVTEST_VER-v0.0.0-20240115093953-9e6e3b144a69}
ENVTEST_K8S_VERSION=${ENVTEST_K8S_VERSION-1.28}
FILTER=${FILTER-}

go install sigs.k8s.io/controller-runtime/tools/setup-envtest@"$SETUP_ENVTEST_VER"
# install and prepare setup-envtest
if ! command -v setup-envtest &> /dev/null
then
    go install sigs.k8s.io/controller-runtime/tools/setup-envtest@"$SETUP_ENVTEST_VER"
fi
KUBEBUILDER_ASSETS=$(setup-envtest use --use-env -p path "$ENVTEST_K8S_VERSION")
export KUBEBUILDER_ASSETS

# run integration tests
cmd="go test -v"
if [ $# -ne 0 ]; then
    has_path=0
    for arg in "$@"; do
        if [[ -d "$arg" || -f "$arg" ]]; then
            has_path=1
            break
        fi
    done

    if [ $has_path -eq 1 ]; then
        cmd="$cmd $*"
    else
        cmd="$cmd ./integrationtests/... $*"
    fi
else
    cmd="$cmd ./integrationtests/..."
fi

# For convenvience, can also be passed as argument.
filter=""
if [ -n "$FILTER" ]; then
    filter="-ginkgo.label-filter=$FILTER"
fi
$cmd "$filter"

#!/bin/bash
set -euo pipefail

# this script ensures that the `goimports` dependency is present
# and then executes goimport passing all arguments forward

./mage dependency:goimports
export GOFLAGS=""
>&2 echo "running on $@"
# exec .deps/bin/goimports -local github.com/mt-sre/devkube -w -l "$@"

#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

unset CDPATH

# Set no_proxy for localhost if behind a proxy, otherwise,
# the connections to localhost in scripts will time out
export no_proxy=127.0.0.1,localhost

# The root directory
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

export ROOT_SBIN_DIR="${ROOT_DIR}/sbin"
mkdir -p "${ROOT_SBIN_DIR}"

source "${ROOT_DIR}/hack/lib/util.sh"
source "${ROOT_DIR}/hack/lib/version.sh"
source "${ROOT_DIR}/hack/lib/log.sh"
source "${ROOT_DIR}/hack/lib/dapper.sh"
source "${ROOT_DIR}/hack/lib/docker.sh"
source "${ROOT_DIR}/hack/lib/ginkgo.sh"
source "${ROOT_DIR}/hack/lib/lint.sh"
source "${ROOT_DIR}/hack/lib/manifest-tool.sh"
source "${ROOT_DIR}/hack/lib/terraform.sh"
source "${ROOT_DIR}/hack/lib/goreleaser.sh"

cos::log::install_errexit
cos::version::get_version_vars

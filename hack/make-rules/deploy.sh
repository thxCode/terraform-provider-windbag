#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

CROSS=true ONLY_ARCHIVE=false "${ROOT_DIR}/hack/make-rules/windbag.sh" package,deploy only

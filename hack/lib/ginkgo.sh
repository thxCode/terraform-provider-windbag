#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# Ginkgo variables helpers. These functions need the
# following variables:
#
#    GINKGO_VERSION  -  The ginkgo version, default is v1.14.2.

ginkgo_version=${GINKGO_VERSION:-"v1.14.2"}

function cos::ginkgo::bin() {
  local bin="ginkgo"
  if [[ -f "${ROOT_SBIN_DIR}/ginkgo" ]]; then
    bin="${ROOT_SBIN_DIR}/ginkgo"
  fi
  echo "${bin}"
}

function cos::ginkgo::install() {
  tmp_dir=$(mktemp -d)
  pushd "${tmp_dir}" >/dev/null || exit 1
  go mod init tmp
  GOBIN="${ROOT_SBIN_DIR}" GO111MODULE=on go get "github.com/onsi/ginkgo/ginkgo@${ginkgo_version}"
  rm -rf "${tmp_dir}"
  popd >/dev/null || return
}

function cos::ginkgo::validate() {
  # shellcheck disable=SC2046
  if [[ -n "$(command -v $(cos::ginkgo::bin))" ]]; then
    if [[ $($(cos::ginkgo::bin) version 2>&1 | cut -d " " -f 3 2>&1) == "${ginkgo_version#v}" ]]; then
      return 0
    fi
  fi

  cos::log::info "installing ginkgo ${ginkgo_version}"
  if cos::ginkgo::install; then
    cos::log::info "ginkgo: $($(cos::ginkgo::bin) version)"
    return 0
  fi
  cos::log::error "no ginkgo available"
  return 1
}

function cos::ginkgo::test() {
  if ! cos::ginkgo::validate; then
    cos::log::error "cannot execute ginkgo without installed"
    return
  fi

  local dir_path="${!#}"
  local arg_idx=0
  for arg in "$@"; do
    if [[ "${arg}" == "--" ]]; then
      dir_path="${!arg_idx}"
      break
    fi
    arg_idx=$((arg_idx + 1))
  done

  if cos::util::is_empty_dir "${dir_path}"; then
    cos::log::warn "${dir_path} is an empty directory"
    return
  fi

  cos::log::debug "ginkgo -r -v -trace -tags=test -failFast -slowSpecThreshold=60 -timeout=5m " "$@"
  CGO_ENABLED=0 $(cos::ginkgo::bin) -r -v -trace -tags=test \
    -failFast -slowSpecThreshold=60 -timeout=5m "$@"
}

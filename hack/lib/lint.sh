#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# Lint variables helpers. These functions need the
# following variables:
#
#    GOLANGCI_LINT_VERSION  -  The golangci-lint version, default is v1.32.2.
#    DIRTY_CHECK            -  Specify to check the git tree is dirty or not.
#

golangci_lint_version=${GOLANGCI_LINT_VERSION:-"v1.32.2"}
dirty_check=${DIRTY_CHECK:-}

function cos::lint::bin() {
  local bin="golangci-lint"
  if [[ -f "${ROOT_SBIN_DIR}/golangci-lint" ]]; then
    bin="${ROOT_SBIN_DIR}/golangci-lint"
  fi
  echo "${bin}"
}

function cos::lint::install() {
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${ROOT_SBIN_DIR}" "${golangci_lint_version}"
}

function cos::lint::validate() {
  # shellcheck disable=SC2046
  if [[ -n "$(command -v $(cos::lint::bin))" ]]; then
    if [[ $($(cos::lint::bin) --version 2>&1 | cut -d " " -f 4 2>&1) == "${golangci_lint_version#v}" ]]; then
      return 0
    fi
  fi

  cos::log::info "installing golangci-lint ${golangci_lint_version}"
  if cos::lint::install; then
    cos::log::info "$($(cos::lint::bin) --version)"
    return 0
  fi
  cos::log::error "no golangci-lint available"
  return 1
}

function cos::lint::run() {
  if [[ "${dirty_check}" == "true" ]]; then
    if git_status=$(git status --porcelain 2>/dev/null) && [[ -n ${git_status} ]]; then
      cos::log::fatal "the git tree is dirty:\n$(git status --porcelain)"
    fi
  fi

  if cos::lint::validate; then
    for path in "$@"; do
      cos::log::debug "golangci-lint run ${path}"
      $(cos::lint::bin) run "${path}"
    done
  else
    cos::log::warn "using go fmt/vet instead ginkgo"
    for path in "$@"; do
      cos::log::debug "go fmt ${path}"
      go fmt "${path}"
      cos::log::debug "go vet -tags=test ${path}"
      go vet -tags=test "${path}"
    done
  fi
}

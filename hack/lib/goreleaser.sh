#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# Lint variables helpers. These functions need the
# following variables:
#
#    GORELEASER_VERSION  -  The goreleaser version, default is v0.155.0.
#

goreleaser_version=${GORELEASER_VERSION:-"v0.155.0"}

function cos::goreleaser::bin() {
  local bin="goreleaser"
  if [[ -f "${ROOT_SBIN_DIR}/goreleaser" ]]; then
    bin="${ROOT_SBIN_DIR}/goreleaser"
  fi
  echo "${bin}"
}

function cos::goreleaser::install() {
  curl -sSfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh -s -- -b "${ROOT_SBIN_DIR}" "${goreleaser_version}"
}

function cos::goreleaser::validate() {
  # shellcheck disable=SC2046
  if [[ -n "$(command -v $(cos::goreleaser::bin))" ]]; then
    if [[ $($(cos::goreleaser::bin) --version 2>&1 | cut -d " " -f 3 | sed -n '1p') == "${goreleaser_version#v}" ]]; then
      return 0
    fi
  fi

  cos::log::info "installing goreleaser ${goreleaser_version}"
  if cos::goreleaser::install; then
    cos::log::info "$($(cos::goreleaser::bin) --version)"
    return 0
  fi
  cos::log::error "no goreleaser available"
  return 1
}

function cos::goreleaser::build() {
  if ! cos::goreleaser::validate; then
    cos::log::error "cannot execute goreleaser as it hasn't installed"
    return
  fi

  cos::log::debug "goreleaser build $*"
  $(cos::goreleaser::bin) build "$@"
}

function cos::goreleaser::release() {
  if ! cos::goreleaser::validate; then
    cos::log::error "cannot execute goreleaser as it hasn't installed"
    return
  fi

  cos::log::debug "goreleaser release $*"
  $(cos::goreleaser::bin) release "$@"
}

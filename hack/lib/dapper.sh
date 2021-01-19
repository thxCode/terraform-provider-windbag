#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# Dapper variables helpers. These functions need the
# following variables:
#
#    DAPPER_VERSION   -  The dapper version for running, default is v0.5.3.

dapper_version=${DAPPER_VERSION:-"v0.5.3"}

function cos::dapper::bin() {
  local bin="dapper"
  if [[ -f "${ROOT_SBIN_DIR}/dapper" ]]; then
    bin="${ROOT_SBIN_DIR}/dapper"
  fi
  echo "${bin}"
}

function cos::dapper::install() {
  curl -fL "https://github.com/rancher/dapper/releases/download/${dapper_version}/dapper-$(uname -s)-$(uname -m)" -o /tmp/dapper
  chmod +x /tmp/dapper && mv /tmp/dapper "${ROOT_SBIN_DIR}/dapper"
}

function cos::dapper::validate() {
  # shellcheck disable=SC2046
  if [[ -n "$(command -v $(cos::dapper::bin))" ]]; then
    if [[ $($(cos::dapper::bin) -v 2>&1 | cut -d " " -f 3 2>&1) == "${dapper_version}" ]]; then
      return 0
    fi
  fi

  cos::log::info "installing dapper ${dapper_version}"
  if cos::dapper::install; then
    cos::log::info "dapper: $($(cos::dapper::bin) -v)"
    return 0
  fi
  cos::log::error "no dapper available"
  return 1
}

function cos::dapper::run() {
  if ! cos::docker::validate; then
    cos::log::fatal "docker hasn't been installed"
  fi
  if ! cos::dapper::validate; then
    cos::log::fatal "dapper hasn't been installed"
  fi

  cos::log::debug "dapper $*"
  $(cos::dapper::bin) "$@"
}

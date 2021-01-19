#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# Manifest tool variables helpers. These functions need the
# following variables:
#    MANIFEST_TOOL_VERSION  -  The manifest tool version for running, default is v1.0.3.
#    DOCKER_USERNAME        -  The username of Docker.
#    DOCKER_PASSWORD        -  The password of Docker.

manifest_tool_version=${MANIFEST_TOOL_VERSION:-"v1.0.3"}
docker_username=${DOCKER_USERNAME:-}
docker_password=${DOCKER_PASSWORD:-}

function cos::manifest_tool::bin() {
  local bin="manifest-tool"
  if [[ -f "${ROOT_SBIN_DIR}/manifest-tool" ]]; then
    bin="${ROOT_SBIN_DIR}/manifest-tool"
  fi
  echo "${bin}"
}

function cos::manifest_tool::install() {
  curl -fL "https://github.com/estesp/manifest-tool/releases/download/${manifest_tool_version}/manifest-tool-$(cos::util::get_os)-$(cos::util::get_arch ---full-name)" -o /tmp/manifest-tool
  chmod +x /tmp/manifest-tool && mv /tmp/manifest-tool "${ROOT_SBIN_DIR}/manifest-tool"
}

function cos::manifest_tool::validate() {
  # shellcheck disable=SC2046
  if [[ -n "$(command -v $(cos::manifest_tool::bin))" ]]; then
    if [[ $($(cos::manifest_tool::bin) --version 2>&1 | cut -d " " -f 3 2>&1) == "${manifest_tool_version}" ]]; then
      return 0
    fi
  fi

  cos::log::info "installing manifest-tool ${manifest_tool_version}"
  if cos::manifest_tool::install; then
    cos::log::info "$($(cos::manifest_tool::bin) --version 2>&1)"
    return 0
  fi
  cos::log::error "no manifest-tool available"
  return 1
}

function cos::manifest_tool::push() {
  if ! cos::manifest_tool::validate; then
    cos::log::error "cannot execute manifest-tool as it hasn't installed"
    return
  fi

  if [[ $(cos::util::get_os) == "darwin" ]]; then
    if [[ -z ${docker_username} ]] && [[ -z ${docker_password} ]]; then
      # NB(thxCode): since 17.03, Docker for Mac stores credentials in the OSX/macOS keychain and not in config.json, which means the above variables need to specify if using on Mac.
      cos::log::fatal "must set 'DOCKER_USERNAME' & 'DOCKER_PASSWORD' environment variables in Darwin platform"
    fi
  fi

  cos::log::info "manifest-tool push $*"
  if [[ -n ${docker_username} ]] && [[ -n ${docker_password} ]]; then
    cos::log::debug "manifest-tool --username=*** --password=*** push " "$@"
    $(cos::manifest_tool::bin) --username="${docker_username}" --password="${docker_password}" push "$@"
  else
    cos::log::debug "manifest-tool push " "$@"
    $(cos::manifest_tool::bin) push "$@"
  fi
}

#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# Docker variables helpers. These functions need the
# following variables:
#
#    DOCKER_VERSION  -  The docker version for running, default is 19.03.

docker_version=${DOCKER_VERSION:-"19.03"}

function cos::docker::install() {
  curl -SfL "https://get.docker.com" | sh -s VERSION="${docker_version}"
}

function cos::docker::validate() {
  if [[ -n "$(command -v docker)" ]]; then
    return 0
  fi

  cos::log::info "installing docker"
  if cos::docker::install; then
    cos::log::info "docker: $(docker version --format '{{.Server.Version}}' 2>&1)"
    return 0
  fi
  cos::log::error "no docker available"
  return 1
}

function cos::docker::get_registry(){
  local registry=()
  IFS="/" read -r -a registry <<<"$1"

  if [[ ${#registry[@]} -le 1 ]]; then
    echo -n "docker.io"
  else
    echo -n "${registry[0]}"
  fi
}

function cos::docker::login() {
  local username=${DOCKER_USERNAME:-}
  local password=${DOCKER_PASSWORD:-}
  if [[ -n ${username} ]] && [[ -n ${password} ]]; then
    local registry
    registry="$(cos::docker::get_registry "${REGISTRY:-${REPO:-}}")"
    if ! docker login -u "${username}" -p "${password}" "${registry}" >/dev/null 2>&1; then
      cos::log::error "Failed to login '${registry}' with '${username}'"
      return 1
    fi
    cos::log::debug "Logon ${registry} with ${username}"
  fi
  return 0
}

function cos::docker::build() {
  if ! cos::docker::validate; then
    cos::log::fatal "docker hasn't been installed"
  fi

  local docker_version
  docker_version="$(docker version -f '{{.Server.Version}}')"
  local versions
  IFS="." read -r -a versions <<<"${docker_version}"
  local major_version="${versions[0]}"
  local minor_version="${versions[1]}"
  local buildkit_supported="0"
  if [[ ${major_version} -ge 18 ]]; then
    if [[ ${major_version} -eq 18 ]]; then
      if [[ ${minor_version} -ge 9 ]]; then
        buildkit_supported="1"
      fi
    else
      buildkit_supported="1"
    fi
  fi
  if [[ ${buildkit_supported} == "0" ]]; then
    cos::log::warn "docker daemon doesn't support buildkit build"
  fi

  local docker_apiversion
  docker_apiversion="$(docker version -f '{{.Server.APIVersion}}')"
  local apiversions
  IFS="." read -r -a apiversions <<<"${docker_apiversion}"
  local major_apiversion="${apiversions[0]}"
  local minor_apiversion="${apiversions[1]}"
  local platform_supported="0"
  if [[ ${major_apiversion} -ge 1 ]]; then
    if [[ ${major_apiversion} -eq 1 ]]; then
      if [[ ${minor_apiversion} -ge 32 ]]; then
        platform_supported="1"
      fi
    else
      platform_supported="1"
    fi
  fi
  if [[ ${platform_supported} == "0" ]]; then
    cos::log::warn "docker daemon doesn't support platform build"
  fi

  local target_platform=""
  local build_platform="${OS:-$(go env GOOS)}/${ARCH:-$(go env GOARCH)}"
  local dockerfile="Dockerfile"
  local args=()
  for arg in "$@"; do
    if [[ "${arg}" =~ ^--platform= ]] && [[ ${platform_supported} == "0" ]]; then
      target_platform="${arg//--platform=/}"
      args+=("--build-arg=TARGETPLATFORM=${target_platform}")
      continue
    fi
    if [[ "${arg}" =~ ^--file= ]] && [[ ${platform_supported} == "0" ]]; then
      dockerfile="${arg//--file=/}"
      local tmp_dir
      tmp_dir=$(mktemp -d)
      local dockerfile_temp
      dockerfile_temp="${tmp_dir}/$(basename "${dockerfile}")"
      cp -f "${dockerfile}" "${dockerfile_temp}"
      cos::util::sed "s#--platform=\$TARGETPLATFORM##g" "${dockerfile_temp}"
      cos::util::sed "s#--platform=\$BUILDPLATFORM##g" "${dockerfile_temp}"
      if [[ -n "${target_platform}" ]]; then
        local os_arch
        IFS="/" read -r -a os_arch <<<"${target_platform}"
        args+=(
          "--build-arg=TARGETOS=${os_arch[0]}"
          "--build-arg=TARGETARCH=${os_arch[1]}"
        )
      fi
      args+=(
        "--build-arg=BUILDPLATFORM=${build_platform}"
        "--file=${dockerfile_temp}"
      )
      continue
    fi
    args+=("$arg")
  done

  # NB(thxCode): use Docker buildkit to cross build images, ref to:
  # - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope#buildkit
  cos::log::debug "docker build " "${args[@]}"
  DOCKER_BUILDKIT="${buildkit_supported}" docker build "${args[@]}"
}

function cos::docker::manifest() {
  if ! cos::docker::validate; then
    cos::log::fatal "docker hasn't been installed"
  fi
  if ! cos::docker::login; then
    cos::log::fatal "failed to login docker"
  fi

  # NB(thxCode): use Docker manifest needs to enable client experimental feature, ref to:
  # - https://docs.docker.com/engine/reference/commandline/manifest_create/
  # - https://docs.docker.com/engine/reference/commandline/cli/#experimental-features#environment-variables
  cos::log::debug "docker manifest create --amend $*"
  DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create --amend "$@"

  # NB(thxCode): use Docker manifest needs to enable client experimental feature, ref to:
  # - https://docs.docker.com/engine/reference/commandline/manifest_push/
  # - https://docs.docker.com/engine/reference/commandline/cli/#experimental-features#environment-variables
  cos::log::debug "docker manifest push --purge ${1}"
  DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push --purge "${1}"
}

function cos::docker::manifest_without_login() {
  if ! cos::docker::validate; then
    cos::log::fatal "docker hasn't been installed"
  fi

  # NB(thxCode): use Docker manifest needs to enable client experimental feature, ref to:
  # - https://docs.docker.com/engine/reference/commandline/manifest_create/
  # - https://docs.docker.com/engine/reference/commandline/cli/#experimental-features#environment-variables
  cos::log::debug "docker manifest create --amend $*"
  DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create --amend "$@"

  # NB(thxCode): use Docker manifest needs to enable client experimental feature, ref to:
  # - https://docs.docker.com/engine/reference/commandline/manifest_push/
  # - https://docs.docker.com/engine/reference/commandline/cli/#experimental-features#environment-variables
  cos::log::debug "docker manifest push --purge ${1}"
  DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push --purge "${1}"
}

function cos::docker::push() {
  if ! cos::docker::validate; then
    cos::log::fatal "docker hasn't been installed"
  fi
  if ! cos::docker::login; then
    cos::log::fatal "failed to login docker"
  fi

  for image in "$@"; do
    cos::log::debug "docker push ${image}"
    docker push "${image}"
  done
}

function cos::docker::push_without_login() {
  if ! cos::docker::validate; then
    cos::log::fatal "docker hasn't been installed"
  fi

  for image in "$@"; do
    cos::log::debug "docker push ${image}"
    docker push "${image}"
  done
}

#!/usr/bin/env bash

function cos::util::find_subdirs() {
  local path="$1"
  if [[ -z "$path" ]]; then
    path="./"
  fi
  # shellcheck disable=SC2010
  ls -l "$path" | grep "^d" | awk '{print $NF}'
}

function cos::util::is_empty_dir() {
  local path="$1"
  if [[ ! -d "${path}" ]]; then
    return 0
  fi

  # shellcheck disable=SC2012
  if [[ $(ls "${path}" | wc -l) -eq 0 ]]; then
    return 0
  fi
  return 1
}

function cos::util::join_array() {
  local IFS="$1"
  shift 1
  echo "$*"
}

function cos::util::get_os() {
  local os
  if go env GOOS >/dev/null 2>&1; then
    os=$(go env GOOS)
  else
    os=$(echo -n "$(uname -s)" | tr '[:upper:]' '[:lower:]')
  fi

  case ${os} in
  cygwin_nt*) os="windows" ;;
  mingw*) os="windows" ;;
  msys_nt*) os="windows" ;;
  esac

  echo -n "${os}"
}

function cos::util::get_arch() {
  local arch
  if go env GOARCH >/dev/null 2>&1; then
    arch=$(go env GOARCH)
    if [[ "${arch}" == "arm" ]]; then
      arch="${arch}v$(go env GOARM)"
    fi
  else
    arch=$(uname -m)
  fi

  case ${arch} in
  armv5*) arch="armv5" ;;
  armv6*) arch="armv6" ;;
  armv7*)
    if [[ "${1:-}" == "--full-name" ]]; then
      arch="armv7"
    else
      arch="arm"
    fi
    ;;
  aarch64) arch="arm64" ;;
  x86) arch="386" ;;
  i686) arch="386" ;;
  i386) arch="386" ;;
  x86_64) arch="amd64" ;;
  esac

  echo -n "${arch}"
}

function cos::util::get_random_port_start() {
  local offset="${1:-1}"
  if [[ ${offset} -le 0 ]]; then
    offset=1
  fi

  while true; do
    random_port=$((RANDOM % 10000 + 50000))
    for ((i = 0; i < offset; i++)); do
      if nc -z 127.0.0.1 $((random_port + i)); then
        random_port=0
        break
      fi
    done

    if [[ ${random_port} -ne 0 ]]; then
      echo -n "${random_port}"
      break
    fi
  done
}

function cos::util::sed() {
  if ! sed -i "$@" >/dev/null 2>&1; then
    # back off none GNU sed
    sed -i "" "$@"
  fi
}

function cos::util::decode64() {
  if [[ $# -eq 0 ]]; then
      cat | base64 --decode
  else
      printf '%s' "$1" | base64 --decode
  fi
}

function cos::util::encode64() {
  if [[ $# -eq 0 ]]; then
      cat | base64
  else
      printf '%s' "$1" | base64
  fi
}

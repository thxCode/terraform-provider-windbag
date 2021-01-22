#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
source "${ROOT_DIR}/hack/lib/init.sh"
source "${ROOT_DIR}/hack/constant.sh"

mkdir -p "${ROOT_DIR}/bin"
mkdir -p "${ROOT_DIR}/dist"

function generate() {
  cos::log::info "generating windbag..."

  cos::log::info "formatting examples"
  cos::terraform::fmt "${ROOT_DIR}/examples"

  cos::log::info "generating docs"
  cos::terraform_docs::generate

  cos::log::info "...done"
}

function mod() {
  [[ "${1:-}" != "only" && "${1:-}" != "o" ]] && generate
  cos::log::info "downloading dependencies for windbag..."

  pushd "${ROOT_DIR}" >/dev/null || exist 1
  if [[ "$(go env GO111MODULE)" == "off" ]]; then
    cos::log::warn "go mod has been disabled by GO111MODULE=off"
  else
    cos::log::info "tidying"
    go mod tidy
    cos::log::info "vending"
    go mod vendor
  fi
  popd >/dev/null || return

  cos::log::info "...done"
}

function lint() {
  [[ "${1:-}" != "only" && "${1:-}" != "o" ]] && mod
  cos::log::info "linting windbag..."

  local targets=(
    "${ROOT_DIR}/main.go"
    "${ROOT_DIR}/windbag/..."
  )
  cos::lint::run "${targets[@]}"

  cos::log::info "...done"
}

function build() {
  [[ "${1:-}" != "only" && "${1:-}" != "o" ]] && lint
  cos::log::info "building windbag(${GIT_VERSION},${GIT_COMMIT},${GIT_TREE_STATE},${BUILD_DATE})..."

  local version_flags="
    -X main.version=${GIT_VERSION}
    -X main.commit=${GIT_COMMIT}"
  local flags="
    -w -s"
  local ext_flags="
    -extldflags '-static'"

  local platforms
  if [[ "${CROSS:-false}" == "true" ]]; then
    cos::log::info "crossed building"
    platforms=("${SUPPORTED_PLATFORMS[@]}")
  else
    local os="${OS:-$(go env GOOS)}"
    local arch="${ARCH:-$(go env GOARCH)}"
    platforms=("${os}/${arch}")
  fi

  for platform in "${platforms[@]}"; do
    cos::log::info "building ${platform}"

    local os_arch
    IFS="/" read -r -a os_arch <<<"${platform}"

    local os=${os_arch[0]}
    local arch=${os_arch[1]}

    local ldflags
    local cgo
    local ext=""
    if [[ "${os}" == "darwin" ]]; then
      ldflags="${version_flags} ${flags}"
      cgo=1
    elif [[ "${os}" == "windows" ]]; then
      ldflags="${version_flags} ${flags} ${ext_flags}"
      cgo=0
      ext=".exe"
    elif [[ "${os}" =~ .*bsd$ ]]; then
      ldflags="${version_flags} ${flags}"
      cgo=0
    else
      ldflags="${version_flags} ${flags} ${ext_flags}"
      cgo=0
    fi
    GOOS=${os} GOARCH=${arch} CGO_ENABLED=${cgo} go build \
      -ldflags "${ldflags}" \
      -o "${ROOT_DIR}/bin/terraform-provider-windbag_${os}_${arch}${ext}" \
      "${ROOT_DIR}/main.go"
  done

  cos::log::info "...done"
}

function package() {
  [[ "${1:-}" != "only" && "${1:-}" != "o" ]] && build
  cos::log::info "packaging windbag..."

  local repo=${REPO:-thxcode}
  local image_name=${IMAGE_NAME:-terraform-provider-windbag}
  local tag=${TAG:-${GIT_VERSION}}

  local platforms
  if [[ "${CROSS:-false}" == "true" ]]; then
    cos::log::info "crossed packaging"
    platforms=("${SUPPORTED_PLATFORMS[@]}")
  else
    local os="${OS:-$(go env GOOS)}"
    local arch="${ARCH:-$(go env GOARCH)}"
    platforms=("${os}/${arch}")
  fi

  # archive binary
  pushd "${ROOT_DIR}/dist" >/dev/null 2>&1
  rm -f ./*.zip >/dev/null 2>&1
  rm -f ./*_SHA256SUMS >/dev/null 2>&1
  rm -f ./*_SHA256SUMS.sig >/dev/null 2>&1
  for platform in "${platforms[@]}"; do
    local os_arch
    IFS="/" read -r -a os_arch <<<"${platform}"

    local os=${os_arch[0]}
    local arch=${os_arch[1]}
    local ext=""
    if [[ "${platform}" =~ windows/* ]]; then
      ext=".exe"
    fi

    local src="terraform-provider-windbag_${os}_${arch}${ext}"
    if [[ ! -f "${ROOT_DIR}/bin/${src}" ]]; then
      cos::log::warn "skipped to archive ${src} as the binary is not found"
      continue
    fi
    cos::log::info "archiving ${src}"
    local dst_file="terraform-provider-windbag_${tag#v}${ext}"
    local dst_archive="terraform-provider-windbag_${tag#v}_${os}_${arch}.zip"
    cp -f "${ROOT_DIR}/bin/${src}" "${ROOT_DIR}/dist/${dst_file}"
    rm -f "${dst_archive}" 2>&1 && zip -1qm "${dst_archive}" "${dst_file}"
  done
  if [[ ! -f "terraform-provider-windbag_darwin_amd64.zip" ]]; then
    # shellcheck disable=SC2035
    shasum -a 256 *.zip > "terraform-provider-windbag_${tag#v}_SHA256SUMS"
    gpg --batch --detach-sign "terraform-provider-windbag_${tag#v}_SHA256SUMS"
  fi
  popd >/dev/null 2>&1

  if [[ "${ONLY_ARCHIVE:-false}" == "true" ]]; then
    cos::log::warn "skipped as packaging image is disabled by ONLY_ARCHIVE"
    return
  fi

  # package image
  for platform in "${platforms[@]}"; do
    if [[ "${platform}" =~ (windows|darwin)/* ]]; then
      cos::log::warn "skipped as packaging ${platform} image is unavailable"
      continue
    fi

    local image_tag="${repo}/${image_name}:${tag}-${platform////-}"
    if [[ ! -f "${ROOT_DIR}/bin/terraform-provider-windbag_${os}_${arch}" ]]; then
      cos::log::warn "skipped to package ${image_tag} as the binary is not found"
      continue
    fi
    cos::log::info "packaging ${image_tag}"
    cos::docker::build \
      --build-arg="WINDBAG_VERSION=${tag#v}" \
      --platform="${platform}" \
      --tag="${image_tag}" \
      "${ROOT_DIR}"
  done

  cos::log::info "...done"
}

function deploy() {
  [[ "${1:-}" != "only" && "${1:-}" != "o" ]] && package
  cos::log::info "deploying windbag..."

  local repo=${REPO:-thxcode}
  local image_name=${IMAGE_NAME:-terraform-provider-windbag}
  local tag=${TAG:-${GIT_VERSION}}

  local platforms
  if [[ "${CROSS:-false}" == "true" ]]; then
    cos::log::info "crossed deploying"
    platforms=("${SUPPORTED_PLATFORMS[@]}")
  else
    local os="${OS:-$(go env GOOS)}"
    local arch="${ARCH:-$(go env GOARCH)}"
    platforms=("${os}/${arch}")
  fi
  local images=()
  for platform in "${platforms[@]}"; do
    if [[ "${platform}" =~ (windows|darwin)/* ]]; then
      cos::log::warn "skipped as packaging ${platform} image is unavailable"
      continue
    fi

    images+=("${repo}/${image_name}:${tag}-${platform////-}")
  done
  if [[ ${#images[@]} -eq 0 ]]; then
    cos::log::warn "skipped as there are not any images to push"
    return
  fi

  if [[ "${ONLY_ARCHIVE:-false}" == "true" ]]; then
    cos::log::warn "skipped as pushing image is disabled by ONLY_ARCHIVE"
    return
  fi

  local only_manifest=${ONLY_MANIFEST:-false}
  local without_manifest=${WITHOUT_MANIFEST:-false}
  local ignore_missing=${IGNORE_MISSING:-false}

  # docker push
  if [[ "${only_manifest}" == "false" ]]; then
    cos::docker::push "${images[@]}"
  else
    cos::log::warn "deploying images has been stopped by ONLY_MANIFEST"
    # execute manifest forcibly
    without_manifest="false"
  fi

  # docker manifest
  if [[ "${without_manifest}" == "false" ]]; then
    if [[ "${ignore_missing}" == "false" ]]; then
      cos::docker::manifest "${repo}/${image_name}:${tag}" "${images[@]}"
    else
      cos::manifest_tool::push from-args \
        --ignore-missing \
        --target="${repo}/${image_name}:${tag}" \
        --template="${repo}/${image_name}:${tag}-OS-ARCH" \
        --platforms="$(cos::util::join_array "," "${platforms[@]}")"
    fi
  else
    cos::log::warn "deploying manifest images has been stopped by WITHOUT_MANIFEST"
  fi

  cos::log::info "...done"
}

function test() {
  [[ "${1:-}" != "only" && "${1:-}" != "o" ]] && build
  cos::log::info "running unit tests for windbag..."

  local unit_test_targets=(
    "${ROOT_DIR}/windbag/..."
  )

  if [[ "${CROSS:-false}" == "true" ]]; then
    cos::log::warn "crossed test is not supported"
  fi

  local os="${OS:-$(go env GOOS)}"
  local arch="${ARCH:-$(go env GOARCH)}"
  if [[ "${arch}" == "arm" ]]; then
    # NB(thxCode): race detector doesn't support `arm` arch, ref to:
    # - https://golang.org/doc/articles/race_detector.html#Supported_Systems
    TF_ACC="" GOOS=${os} GOARCH=${arch} CGO_ENABLED=1 go test \
      -tags=test \
      -v \
      -cover -coverprofile "${ROOT_DIR}/dist/test_coverage_${os}_${arch}.out" \
      "${unit_test_targets[@]}"
  else
    TF_ACC="" GOOS=${os} GOARCH=${arch} CGO_ENABLED=1 go test \
      -tags=test \
      -v \
      -race \
      -cover -coverprofile "${ROOT_DIR}/dist/test_coverage_${os}_${arch}.out" \
      "${unit_test_targets[@]}"
  fi

  cos::log::info "...done"
}

function verify() {
  [[ "${1:-}" != "only" && "${1:-}" != "o" ]] && test
  cos::log::info "running integration tests for windbag..."

  local verify_test_targets=(
    "${ROOT_DIR}/windbag/..."
  )

  local os="${OS:-$(go env GOOS)}"
  local arch="${ARCH:-$(go env GOARCH)}"
  TF_ACC=1 GOOS=${os} GOARCH=${arch} CGO_ENABLED=1 go test \
    -timeout 120m \
    -tags=test \
    -v \
    -cover -coverprofile "${ROOT_DIR}/dist/verify_coverage_${os}_${arch}.out" \
    "${verify_test_targets[@]}"

  cos::log::info "...done"
}

function e2e() {
  [[ "${1:-}" != "only" && "${1:-}" != "o" ]] && verify
  cos::log::info "running E2E tests for windbag..."

  cos::log::info "...done"
}

function entry::dapper() {
  BY="" cos::dapper::run -C="${ROOT_DIR}" -f="Dockerfile.dapper" "windbag" "$@"
}

function entry::default() {
  local stages="${1:-build}"
  shift $(($# > 0 ? 1 : 0))

  IFS="," read -r -a stages <<<"${stages}"
  local commands=$*
  if [[ ${#stages[@]} -ne 1 ]]; then
    commands="only"
  fi

  for stage in "${stages[@]}"; do
    cos::log::info "# make windbag ${stage} ${commands}"
    case ${stage} in
    g | gen | generate) generate "${commands}" ;;
    m | mod) mod "${commands}" ;;
    l | lint) lint "${commands}" ;;
    b | build) build "${commands}" ;;
    p | pkg | package) package "${commands}" ;;
    d | dep | deploy) deploy "${commands}" ;;
    t | test) test "${commands}" ;;
    v | ver | verify) verify "${commands}" ;;
    e | e2e) e2e "${commands}" ;;
    *) cos::log::fatal "unknown action '${stage}', select from generate,mod,lint,build,test,verify,package,deploy,e2e" ;;
    esac
  done
}

case ${BY:-} in
dapper) entry::dapper "$@" ;;
*) entry::default "$@" ;;
esac

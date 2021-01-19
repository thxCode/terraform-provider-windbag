#!/usr/bin/env bash

# -----------------------------------------------------------------------------
# Terraform variables helpers. These functions need the
# following variables:
#
#            TERRAFORM_VERSION   -  The terraform version for running, default is v0.14.4.
# TERRAFORM_PLUGIN_DOCS_VERSION  -  The terraform docs plugin for running, default is v0.3.1.

terraform_version=${TERRAFORM_VERSION:-"v0.14.4"}
terraform_plugin_docs_version=${TERRAFORM_PLUGIN_DOCS_VERSION:-"v0.3.1"}

function cos::terraform::bin() {
  local bin="terraform"
  if [[ -f "${ROOT_SBIN_DIR}/terraform" ]]; then
    bin="${ROOT_SBIN_DIR}/terraform"
  fi
  echo "${bin}"
}

function cos::terraform::install() {
  curl -fL "https://releases.hashicorp.com/terraform/${terraform_version#v}/terraform_${terraform_version#v}_$(cos::util::get_os)_$(cos::util::get_arch).zip" -o /tmp/terraform.zip
  unzip -o /tmp/terraform.zip -d /tmp
  chmod +x /tmp/terraform && mv /tmp/terraform "${ROOT_SBIN_DIR}/terraform"
}

function cos::terraform::validate() {
  # shellcheck disable=SC2046
  if [[ -n "$(command -v $(cos::terraform::bin))" ]]; then
    if [[ $($(cos::terraform::bin) version 2>&1 | cut -d " " -f 2 | sed -n '1p') == "${terraform_version}" ]]; then
      return 0
    fi
  fi

  cos::log::info "installing terraform"
  if cos::terraform::install; then
    cos::log::info "terraform: $($(cos::terraform::bin) version 2>&1)"
    return 0
  fi
  cos::log::error "no terraform available"
  return 1
}

function cos::terraform::fmt() {
  if ! cos::terraform::validate; then
    cos::log::error "cannot execute terraform as it hasn't installed"
    return
  fi

  cos::log::debug "terraform fmt -recursive $*"
  $(cos::terraform::bin) fmt -recursive "$@"
}

function cos::terraform_docs::bin() {
  local bin="tfplugindocs"
  if [[ -f "${ROOT_SBIN_DIR}/tfplugindocs" ]]; then
    bin="${ROOT_SBIN_DIR}/tfplugindocs"
  fi
  echo "${bin}"
}

function cos::terraform_docs::install() {
  curl -fL "https://github.com/hashicorp/terraform-plugin-docs/releases/download/${terraform_plugin_docs_version}/tfplugindocs_${terraform_plugin_docs_version#v}_$(cos::util::get_os)_$(cos::util::get_arch).zip" -o /tmp/tfplugindocs.zip
  unzip -o /tmp/tfplugindocs.zip -d /tmp
  chmod +x "/tmp/tfplugindocs" && mv "/tmp/tfplugindocs" "${ROOT_SBIN_DIR}/tfplugindocs"
}

function cos::terraform_docs::validate() {
  # shellcheck disable=SC2046
  if [[ -n "$(command -v $(cos::terraform_docs::bin))" ]]; then
    if [[ $($(cos::terraform_docs::bin) --version 2>&1 | cut -d " " -f 3) == "${terraform_plugin_docs_version#v}" ]]; then
      return 0
    fi
  fi

  cos::log::info "installing tfplugindocs"
  if cos::terraform_docs::install; then
    cos::log::info "tfplugindocs: $($(cos::terraform_docs::bin) --version 2>&1)"
    return 0
  fi
  cos::log::error "no tfplugindocs available"
  return 1
}

function cos::terraform_docs::generate() {
  if ! cos::terraform_docs::validate; then
    cos::log::error "cannot execute terraform-plugin-docs as it hasn't installed"
    return
  fi

  cos::log::debug "tfplugindocs generate"
  $(cos::terraform_docs::bin) generate >/dev/null
}

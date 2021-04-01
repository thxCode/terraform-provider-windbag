FROM --platform=$TARGETPLATFORM alpine:3.13
LABEL maintainer="thxCode <thxcode0824@gmail.com>"

RUN set -ex \
    && apk update \
    && apk add --no-cache ca-certificates bash unzip curl openssh

# NB(thxCode): automatic platform ARGs, ref to:
# - https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

# install terraform client
ENV TERRAFORM_VERSION=0.14.9
RUN curl -fL "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_${TARGETOS}_${TARGETARCH}.zip" -o "/tmp/terraform.zip" \
    && unzip -o "/tmp/terraform.zip" -d "/tmp" && chmod a+x "/tmp/terraform" \
    && mv -f "/tmp/terraform" /usr/bin/ \
    && ln -s /usr/bin/terraform /usr/bin/tf \
    && rm -rf "/tmp/*"

# prepare terraform environment
ENV TF_PLUGIN_CACHE_DIR="/root/.terraform.d/plugin-cache" \
    TF_RELEASE=1 \
    TF_DEV=true \
    TF_LOG=INFO \
    WINDBAG_LOG=INFO

# install alicloud provider
ENV TERRAFORM_ALICLOUD_VERSION=1.119.1
RUN CACHE_DIR="${TF_PLUGIN_CACHE_DIR}/registry.terraform.io/aliyun/alicloud/${TERRAFORM_ALICLOUD_VERSION}/${TARGETOS}_${TARGETARCH}"; \
    mkdir -p "${CACHE_DIR}"; \
    \
    curl -fL "https://github.com/aliyun/terraform-provider-alicloud/releases/download/v${TERRAFORM_ALICLOUD_VERSION}/terraform-provider-alicloud_${TERRAFORM_ALICLOUD_VERSION}_${TARGETOS}_${TARGETARCH}.zip" -o "/tmp/alicloud.zip" \
    && unzip -o "/tmp/alicloud.zip" -d "/tmp" && chmod a+x "/tmp/terraform-provider-alicloud_v${TERRAFORM_ALICLOUD_VERSION}" \
    && mv -f "/tmp/terraform-provider-alicloud_v${TERRAFORM_ALICLOUD_VERSION}" "${CACHE_DIR}/" \
    && rm -rf "/tmp/*"

# copy windbag provider
ARG WINDBAG_VERSION=0.0.0
ENV TERRAFORM_WINDBAG_VERSION=${WINDBAG_VERSION}
COPY bin/terraform-provider-windbag_${TARGETOS}_${TARGETARCH} ${TF_PLUGIN_CACHE_DIR}/registry.terraform.io/thxcode/windbag/${WINDBAG_VERSION}/${TARGETOS}_${TARGETARCH}/terraform-provider-windbag_v${WINDBAG_VERSION}

VOLUME /workspace
WORKDIR /workspace

---
layout: ""
page_title: "Windbag Provider"
description: |-
  The Windbag provider is used to build and manifest the Windows image crossing multi-release Windows hosts.
---

# Windbag Provider

The Windbag provider is used to build and manifest the Windows image crossing multi-release Windows hosts.

## Why I create this tool?

To package a Windows manifest Docker image(crossing multi-release) is a boring thing, honestly, I tired to setup these Windows hosts one-by-one and execute `docker build` from the lowest instance to highest one.

## How to enable SSH service on Windows?

I created a script to setup the SSH service on Windows, please have a shot if you like it:

``` powershell

# https://www.powershellgallery.com/packages/OpenSSHUtils/0.0.2.0/Content/OpenSSHUtils.psm1

# enable service
iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex

# disable service
$SSHD_ENABLED="disabled"; iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex

# configure a user with password
$env:SSH_USER="<user name>";
$env:SSH_USER_PASSWORD="<user password>";
iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex

# configure a remote administrator with public key
$SSH_USER="<user name>";
$SSH_USER_PUBLICKEY="<user public key>";
iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex

# configure a remote user with public key
$SSH_USER="<user name>";
$SSH_USER_GROUP="sshusers";
$SSH_USER_PUBLICKEY="<user public key>";
iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex

```

## Example Usage

```terraform
provider "windbag" {}

# specify the credential of dockerhub,
data "windbag_registry" "dockerhub" {
  address = [
    "docker.io"
  ]
  username = "foo"
  password = "bar"
}

# either specify the credential of multiple registries.
data "windbag_registry" "acrs" {
  address = [
    "registry.cn-hongkong.aliyuncs.com",
    "registry.cn-shenzhen.aliyuncs.com"
  ]

  # all of the above registries are using the same username and password.
  username = "foo@acr"
  password = "bar@acr"
}

# specify some windows workers
resource "windbag_worker" "windows_1809" {
  address = "192.168.1.3:22"
  ssh {
    password = "Windbag@Test"
  }
}

resource "windbag_worker" "windows_1909" {
  address = "192.168.1.4:22"
  ssh {
    password = "Windbag@Test"
  }
}

# specify the windows image to build
resource "windbag_image" "pause_window" {
  # indicate the image build context
  path = pathexpand("testdata/pause_windows")

  # indicate the image tags to build
  tag = [
    "registry.cn-hongkong.aliyuncs.com/foo/foo/pause-windows:v1.0.0",
    "registry.cn-shenzhen.aliyuncs.com/foo/foo/pause-windows:v1.0.0",
    "foo/pause-windows:v1.0.0"
  ]

  # indicate to push the image after build
  push = true

  # indicate the worker OS information,
  build_worker {
    id         = windbag_worker.windows_1809.id
    os_release = "1809"
    os_build   = "17763"
    os_type    = "windows"
    os_arch    = "amd64"
    work_dir   = "C:\\etc\\windbag"
  }

  # either use the discovered OS information by windbag_worker.
  build_worker {
    id         = windbag_worker.windows_1909.id
    os_release = windbag_worker.windows_1909.os_release
    os_build   = windbag_worker.windows_1909.os_build
    os_type    = windbag_worker.windows_1909.os_type
    os_arch    = windbag_worker.windows_1909.os_arch
    work_dir   = windbag_worker.windows_1909.work_dir
  }
}
```

> *NOTE*
>
> The `windbag_worker` only support to use `ssh` protocol to connect to the Windows host at present.

## Advance Usage with Dynamic Provisioner

```terraform
variable "acr_address" {
  type = "list"
  default = [
    "registry.cn-hangzhou.aliyuncs.com",
    "registry.cn-hongkong.aliyuncs.com",
    "registry.cn-shenzhen.aliyuncs.com"
  ]
}

variable "acr_username" {
  type = "string"
  default = "thxcode@aliyun"
}

variable "acr_password" {
  type = "string"
}

variable "host_address" {
  type = "list"
}

variable "host_username" {
  type = "string"
  default = "root"
}

variable "host_password" {
  type = "string"
}

variable "images" {
  type = "map"
  default = {
    pause_windows = "thxcode/pause-windows:v1.0.0"
    flannel_windows = "thxcode/flannel-windows:v0.13.0"
  }
}

provider "windbag" {}

# specify the credentials of registry
data "windbag_registry" "acrs" {
  address = var.acr_address
  username = var.acr_username
  password = var.acr_password
}

# specify the windows workers
resource "windbag_worker" "workers" {
  for_each = var.host_address

  address = each_key
  ssh {
    username = var.host_username
    password = var.host_password
  }
}

# specify the windows image to build
resource "windbag_image" "images" {
  for_each = var.images

  path = pathexpand(format("testdata/%s", each_key))
  tag = [
    each_value
  ]
  push = true

  build_worker = windbag_worker.workers
}
```

<!-- schema generated by tfplugindocs -->
## Schema

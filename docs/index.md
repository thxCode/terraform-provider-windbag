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

```powershell

> # https://www.powershellgallery.com/packages/OpenSSHUtils/0.0.2.0/Content/OpenSSHUtils.psm1

> # enable service
> iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex

> # disable service
> $SSHD_ENABLED="disabled"; iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex

> # configure a remote administrator with password
> $env:SSH_USER="<user name>";
> $env:SSH_USER_PASSWORD="<user password>";
> iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex

> # configure a remote administrator with public key
> $env:SSH_USER="<user name>";
> $env:SSH_USER_PUBLICKEY="<user public key>";
> iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex

> # configure a remote user with public key
> $env:SSH_USER_GROUP="sshusers";
> $env:SSH_USER="<user name>";
> $env:SSH_USER_PUBLICKEY="<user public key>";
> iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex

```

## Example Usage

```terraform
provider "windbag" {}

# specify the windows image to build
resource "windbag_image" "pause_window" {
  # indicate the image build context
  path = pathexpand("testdata/pause_windows")
  # indicate the image tags to build
  tag = [
    "registry.cn-hongkong.aliyuncs.com/foo/pause-windows:v1.0.0",
    "foo/pause-windows:v1.0.0"
  ]
  # indicate to push the image after build
  push = true

  # indicate registries
  registry {
    address  = "registry.cn-hongkong.aliyuncs.com"
    username = "foo@acr"
    password = "bar@acr"
  }
  registry {
    # address = "docker.io"
    username = "foo"
    password = "bar"
  }

  # indicate workers
  worker {
    address = "192.168.1.4:22"
    ssh {
      password = "Windbag@Test"
    }
  }
  worker {
    address = "192.168.1.3:22"
    ssh {
      password = "Windbag@Test"
    }
  }
}
```

> *NOTE*
>
> The `windbag_image` only support to use `ssh` protocol to connect to the Windows worker at present.

## Example With [Alibaba Cloud](https://registry.terraform.io/providers/aliyun/alicloud/latest/docs) Provider

```terraform
terraform {
  required_providers {
    alicloud = {
      source = "aliyun/alicloud"
    }
    windbag = {
      source = "thxcode/windbag"
    }
  }
}

# --
# configure alicloud
# --

variable "resource_group" {
  type    = string
  default = "default"
}

variable "region" {
  type    = string
  default = "cn-hongkong"
}

variable "access_key" {
  type = string
}

variable "secret_key" {
  type = string
}

variable "host_image_list" {
  type = list(string)
  default = [
    "win2019_1809_x64_dtc_en-us_40G_container_alibase_20201120.vhd",
    "wincore_1909_x64_dtc_en-us_40G_container_alibase_20200723.vhd",
    "wincore_2004_x64_dtc_en-us_40G_container_alibase_20201217.vhd"
  ]
}

variable "host_password" {
  type    = string
  default = "Windbag@Test"
}

provider "alicloud" {
  region     = var.region
  access_key = var.access_key
  secret_key = var.secret_key
}

locals {
  ecs_user_data_template = <<EOF
[powershell]
$env:SSH_USER="root";
$env:SSH_USER_PASSWORD="<PASSWORD>";
iwr -uri https://gist.githubusercontent.com/thxCode/cd8ec26795a56eb120b57675f0c067cf/raw/897f2c41df99832d6f88f663a9c2ac442dee4875/zz_sshd_manage.ps1 -UseBasicParsing | iex;
EOF
}

## resource group
data "alicloud_resource_manager_resource_groups" "default" {
  name_regex = format("^%s$", var.resource_group)
}

## zone
data "alicloud_zones" "default" {
  available_resource_creation = "Instance"
  available_instance_type     = "ecs.g6e.2xlarge"
  available_disk_category     = "cloud_essd"
  instance_charge_type        = "PostPaid"
}

## vpc
resource "alicloud_vpc" "default" {
  resource_group_id = data.alicloud_resource_manager_resource_groups.default.groups.0.id
  name              = "vpc-windbag"
  cidr_block        = "172.16.0.0/12"
}
resource "alicloud_vswitch" "default" {
  availability_zone = data.alicloud_zones.default.zones[0].id
  vpc_id            = alicloud_vpc.default.id
  name              = "vsw-windbag"
  cidr_block        = "172.16.0.0/24"
}

## security group !!!
resource "alicloud_security_group" "default" {
  resource_group_id   = data.alicloud_resource_manager_resource_groups.default.groups.0.id
  vpc_id              = alicloud_vpc.default.id
  description         = "sg-windbag"
  name                = "sg-windbag"
  security_group_type = "normal"
  inner_access_policy = "Accept"
}
resource "alicloud_security_group_rule" "all_allow_ssh" {
  security_group_id = alicloud_security_group.default.id
  description       = "sg-windbag-allow-ssh"
  type              = "ingress"
  ip_protocol       = "tcp"
  policy            = "accept"
  port_range        = "22/22"
  priority          = 1
  cidr_ip           = "0.0.0.0/0"
}
resource "alicloud_security_group_rule" "all_allow_rdp" {
  security_group_id = alicloud_security_group.default.id
  description       = "sg-windbag-allow-rdp"
  type              = "ingress"
  ip_protocol       = "tcp"
  policy            = "accept"
  port_range        = "3389/3389"
  priority          = 1
  cidr_ip           = "0.0.0.0/0"
}

## instance
resource "alicloud_instance" "default" {
  count                = length(var.host_image_list)
  description          = var.host_image_list[count.index]
  instance_name        = "ecs-windbag-${count.index}"
  image_id             = var.host_image_list[count.index]
  resource_group_id    = data.alicloud_resource_manager_resource_groups.default.groups.0.id
  availability_zone    = data.alicloud_zones.default.zones[0].id
  vswitch_id           = alicloud_vswitch.default.id
  security_groups      = alicloud_security_group.default.*.id
  instance_type        = data.alicloud_zones.default.available_instance_type
  system_disk_category = data.alicloud_zones.default.available_disk_category
  password             = var.host_password
  user_data            = replace(local.ecs_user_data_template, "<PASSWORD>", var.host_password)
}
resource "alicloud_eip" "default" {
  count                = length(var.host_image_list)
  description          = var.host_image_list[count.index]
  name                 = "eip-windbag-${count.index}"
  resource_group_id    = data.alicloud_resource_manager_resource_groups.default.groups.0.id
  bandwidth            = 100
  internet_charge_type = "PayByTraffic"
  instance_charge_type = "PostPaid"
}
resource "alicloud_eip_association" "default" {
  count         = length(var.host_image_list)
  instance_id   = alicloud_instance.default[count.index].id
  allocation_id = alicloud_eip.default[count.index].id
}
output "alicloud_eip_public_ips" {
  value = alicloud_eip.default.*.ip_address
}

# --
# configure windbag
# --

variable "image_registry_list" {
  type = list(string)
}

variable "image_repository" {
  type = string
}

variable "image_name" {
  type = string
}

variable "image_tag" {
  type = string
}

variable "image_registry_username" {
  type = string
}

variable "image_registry_password" {
  type = string
}

provider "windbag" {}

## image
resource "windbag_image" "default" {
  path = pathexpand("bar")
  tag = [
    for registry in var.image_registry_list :
    join(":", [
      join("/", [
        registry,
        var.image_repository,
      var.image_name]),
    var.image_tag])
  ]
  push = true

  dynamic "registry" {
    for_each = var.image_registry_list
    content {
      address  = registry.value
      username = var.image_registry_username
      password = var.image_registry_password
    }
  }

  dynamic "worker" {
    for_each = alicloud_eip.default.*.ip_address
    content {
      address = format("%s:22", worker.value)
      ssh {
        username = "root"
        password = var.host_password
      }
    }
  }
}
output "windbag_image_artifacts" {
  value = windbag_image.default.tag
}
```

### generate terraform plan

```bash

$ tf plan \
   --var 'access_key=...' \
   --var 'secret_key=...' \
   --var 'host_image_list=["win2019_1809_x64_dtc_en-us_40G_container_alibase_20201120.vhd","wincore_1909_x64_dtc_en-us_40G_container_alibase_20200723.vhd","wincore_2004_x64_dtc_en-us_40G_container_alibase_20201120.vhd"]' \
   --var 'host_password=Windbag@Test' \
   --var 'image_registry_list=["registry.cn-hangzhou.aliyuncs.com", "registry.cn-hongkong.aliyuncs.com"]' \
   --var 'image_repository=foo' \
   --var 'image_name=bar' \
   --var 'image_tag=v0.0.0' \
   --var 'image_registry_username=...' \
   --var 'image_registry_password=...'

```

<!-- schema generated by tfplugindocs -->
## Schema

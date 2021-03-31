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
    "win2019_1809_x64_dtc_en-us_40G_container_alibase_20210316.vhd",
    "winsvr_64_dtcC_1903_en-us_40G_alibase_20191012.vhd",
    "wincore_1909_x64_dtc_en-us_40G_container_alibase_20200723.vhd",
    "wincore_2004_x64_dtc_en-us_40G_container_alibase_20210316.vhd"
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
Invoke-WebRequest -UseBasicParsing -Uri https://raw.githubusercontent.com/thxCode/terraform-provider-windbag/master/tools/sshd.ps1 | Invoke-Expression;
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

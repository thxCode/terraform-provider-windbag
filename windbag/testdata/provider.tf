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
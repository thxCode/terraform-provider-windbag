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

# specify the windows image to build
resource "windbag_image" "pause_window" {
  # indicate the image build context
  path = pathexpand("testdata/pause_windows")

  # indicate the image tags to build
  tag = [
    "registry.cn-hongkong.aliyuncs.com/foo/pause-windows:v1.0.0",
    "registry.cn-shenzhen.aliyuncs.com/foo/pause-windows:v1.0.0",
    "foo/pause-windows:v1.0.0"
  ]

  # indicate to push the image after build
  push = true

  # indicate the workers
  build_worker {
    address = "192.168.1.4:22"
    ssh {
      password = "Windbag@Test"
    }
  }

  build_worker {
    address = "192.168.1.3:22"
    ssh {
      password = "Windbag@Test"
    }
  }
}
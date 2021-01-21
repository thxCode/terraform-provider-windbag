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
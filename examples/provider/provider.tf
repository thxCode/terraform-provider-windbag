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
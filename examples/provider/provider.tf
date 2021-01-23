provider "windbag" {

  # specify the Docker as builder.
  docker {

    # specify the version of Docker,
    # default is "19.03".
    version = "19.03"

    # specify the URI to download the Docker ZIP archive
    download_uri = ""

  }

}

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
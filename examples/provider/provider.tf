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

  # indicate the build args release related mapper
  build_arg_release_mapper {
    release = "1809"
    build_arg = {
      "BASE_IMAGE_TAG" = "7.1.4-nanoserver-1809-20210812"
    }
  }
  build_arg_release_mapper {
    release = "1909"
    build_arg = {
      "BASE_IMAGE_TAG" = "7.1.4-nanoserver-1909-20210812"
    }
  }
  build_arg_release_mapper {
    release = "2004"
    build_arg = {
      "BASE_IMAGE_TAG" = "7.1.4-nanoserver-2004-20210812"
    }
  }

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
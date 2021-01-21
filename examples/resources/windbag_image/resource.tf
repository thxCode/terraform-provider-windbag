resource "windbag_image" "example" {

  # specify the path to build,
  # like "docker build ...",
  # default is the current directory.
  path = ""

  # specify the build-time arguments,
  # like "docker build --build-arg=...".
  build_arg = {}

  # specify the name of the building Dockerfile,
  # like "docker build --file=...",
  # default is "Dockerfile".
  file = ""

  # specify to remove intermediate containers,
  # like "docker build --force-rm"
  # default is "false".
  force_rm = false

  # specify the isolation technology of container,
  # like "docker build --isolation=...",
  # default is "process".
  isolation = ""

  # specify the metadata label,
  # like "docker build --label=...".
  label = {}

  # specify to do not use the cache,
  # like "docker build --no-cache",
  # default is "false".
  no_cache = false

  # specify to remove intermediate containers after a successful build,
  # like "docker build --rm"
  # default is "true".
  rm = true

  # specify the list of the built artifact name,
  # like "docker build --tag=...".
  tag = []

  # specify the target of build stage to build,
  # like "docker build --target=...".
  target = ""

  # specify to always push the built artifact,
  # default is "false".
  force_push = false

  # specify to push the build artifact if the digest has changed,
  # default is "true".
  push = true

  # Specify the authentication registry of registry.
  registry {

    # specify the address of registry.
    address = ""

    # specify the username of registry credential.
    username = ""

    # specify the password of registry credential.
    password = ""

  }

  # specify the workers to build image,
  # and manifest the image in the latest release worker.
  worker {

    # specify the address of worker.
    address = ""

    # specify the working directory of worker.
    work_dir = ""

    # specify to use SSH to login the worker.
    ssh {

      # specify the username for authenticating the worker,
      # default is "root".
      username = ""

      # specify the password for authenticating the worker.
      password = ""

      # specify the content of Private Key to authenticate.
      key = ""

      # specify the content of Certificate to sign the Private Key.
      cert = ""

      # specify to use ssh-agent to manage the login certificate,
      # default is "false".
      with_agent = false

    }
  }

}

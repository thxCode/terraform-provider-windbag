resource "windbag_worker" "example" {

  # specify the address of worker.
  address = ""

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

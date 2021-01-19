package windbag

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/thxcode/terraform-provider-windbag/windbag/template"
)

func TestAccResourceWindbagImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			testAccResourceWindbagImageDefault(),
		},
	})
}

func testAccResourceWindbagImageDefault() resource.TestStep {
	var (
		dockerUsername = os.Getenv("DOCKER_USERNAME")
		dockerPassword = os.Getenv("DOCKER_PASSWORD")
		address        = os.Getenv("WORKER_ADDRESS")
		password       = os.Getenv("WORKER_PASSWORD")
	)

	var configTmpl = `
data "windbag_registry" "dockerhub" {
  username = "{{ .DockerUsername }}"
  password = "{{ .DockerPassword }}"
}
resource "windbag_worker" "windows_1809" {
  address = "{{ .Address }}"
  ssh {
    password = "{{ .Password }}"
  }
}
resource "windbag_image" "flannel_windows" {
  path = pathexpand("testdata/flannel_windows")
  tag = [
    "thxcode/flannel-windows:v1.0.0"
  ]
  build_worker {
    id = windbag_worker.windows_1809.id
    os_build = windbag_worker.windows_1809.os_build
	os_release = windbag_worker.windows_1809.os_release
	os_type = windbag_worker.windows_1809.os_type
    os_arch = windbag_worker.windows_1809.os_arch
	work_dir = windbag_worker.windows_1809.work_dir
  }
}
`
	var configData = map[string]interface{}{
		"DockerUsername": dockerUsername,
		"DockerPassword": dockerPassword,
		"Address":        address,
		"Password":       password,
	}

	return resource.TestStep{
		SkipFunc: func() (bool, error) {
			return hasBlank(dockerUsername, dockerPassword, address, password), nil
		},
		Config: template.TryRender(configData, configTmpl),
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"windbag_worker.windows_1809", "id", resourceWindbagWorkerID(address),
			),
			resource.TestCheckResourceAttr(
				"windbag_image.flannel_windows", "id", "flannel-windows",
			),
		),
	}
}

package windbag

import (
	"os"
	"strings"
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
		addresses      = strings.Split(os.Getenv("WORKER_ADDRESS"), ",")
		password       = os.Getenv("WORKER_PASSWORD")
	)

	var configTmpl = `
resource "windbag_image" "pause_windows" {
  path = pathexpand("testdata/pause_windows")
  tag = [
    "thxcode/pause-windows:v1.0.0"
  ]

  registry {
    username = "{{ .DockerUsername }}"
	password = "{{ .DockerPassword }}"
  }

{{ $root := . }}
{{- range .Addresses }}

  worker {
    address = "{{ . }}"
    ssh {
      password = "{{ $root.Password }}"
    }
  }

{{- end }}
}
`
	var configData = map[string]interface{}{
		"DockerUsername": dockerUsername,
		"DockerPassword": dockerPassword,
		"Addresses":      addresses,
		"Password":       password,
	}

	return resource.TestStep{
		SkipFunc: func() (bool, error) {
			return hasBlank(append(addresses, dockerUsername, dockerPassword, password)...), nil
		},
		Config: template.TryRender(configData, configTmpl),
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"windbag_image.pause_windows", "id", "pause-windows",
			),
		),
	}
}

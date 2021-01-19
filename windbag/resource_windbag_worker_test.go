package windbag

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/thxcode/terraform-provider-windbag/windbag/template"
)

func TestAccResourceWindbagWorker(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			testAccResourceWindbagWorkerDefault(),
		},
	})
}

func testAccResourceWindbagWorkerDefault() resource.TestStep {
	var (
		address  = os.Getenv("WORKER_ADDRESS")
		password = os.Getenv("WORKER_PASSWORD")
	)

	var configTmpl = `
resource "windbag_worker" "windows_1809" {
  address = "{{ .Address }}"
  ssh {
    password = "{{ .Password }}"
  }
}`
	var configData = map[string]interface{}{
		"Address":  address,
		"Password": password,
	}

	return resource.TestStep{
		SkipFunc: func() (bool, error) {
			return hasBlank(address, password), nil
		},
		Config: template.TryRender(configData, configTmpl),
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"windbag_worker.windows_1809", "id", resourceWindbagWorkerID(address),
			),
			resource.TestCheckResourceAttr(
				"windbag_worker.windows_1809", "os_release", "1809",
			),
			resource.TestCheckResourceAttr(
				"windbag_worker.windows_1809", "os_major", "10",
			),
			resource.TestCheckResourceAttr(
				"windbag_worker.windows_1809", "os_minor", "0",
			),
			resource.TestCheckResourceAttr(
				"windbag_worker.windows_1809", "os_build", "17763",
			),
			resource.TestCheckResourceAttr(
				"windbag_worker.windows_1809", "os_type", "windows",
			),
			resource.TestCheckResourceAttr(
				"windbag_worker.windows_1809", "os_arch", "amd64",
			),
		),
	}
}

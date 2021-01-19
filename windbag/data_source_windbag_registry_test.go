package windbag

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceWindbagRegistry(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			testAccDataSourceWindbagRegistryDefault(),
			testAccDataSourceWindbagRegistryMultiple(),
		},
	})
}

var testAccDataSourceWindbagRegistryDefault = func() resource.TestStep {
	return resource.TestStep{
		Config: `
data "windbag_registry" "dockerhub" {
  address = [
    "docker.io"
  ]
  username = "Username@Test"
  password = "Password@Test"
}
`,
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.windbag_registry.dockerhub", "address.0", "docker.io",
			),
			resource.TestCheckResourceAttr(
				"data.windbag_registry.dockerhub", "username", "Username@Test",
			),
			resource.TestCheckResourceAttr(
				"data.windbag_registry.dockerhub", "password", "Password@Test",
			),
			resource.TestCheckResourceAttr(
				"data.windbag_registry.dockerhub", "id", "index.docker.io",
			),
		),
	}
}

var testAccDataSourceWindbagRegistryMultiple = func() resource.TestStep {
	return resource.TestStep{
		Config: `
data "windbag_registry" "acr" {
  address = [
    "docker.io",
    "registry.cn-hangzhou.aliyuncs.com"
  ]
  username = "Username@Test"
  password = "Password@Test"
}
`,
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.windbag_registry.acr", "address.1", "registry.cn-hangzhou.aliyuncs.com",
			),
			resource.TestCheckResourceAttr(
				"data.windbag_registry.acr", "address.0", "docker.io",
			),
			resource.TestCheckResourceAttr(
				"data.windbag_registry.acr", "username", "Username@Test",
			),
			resource.TestCheckResourceAttr(
				"data.windbag_registry.acr", "password", "Password@Test",
			),
			resource.TestCheckResourceAttr(
				"data.windbag_registry.acr", "id", "registry.cn-hangzhou.aliyuncs.com",
			),
		),
	}
}

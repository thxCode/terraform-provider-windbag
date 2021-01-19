package windbag

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// providerFactories are used to instantiate a provider during acceptance testing.
// The factory function will be invoked for every Terraform CLI command executed
// to create a provider server to which the CLI can reattach.
var providerFactories = map[string]func() (*schema.Provider, error){
	"windbag": func() (*schema.Provider, error) {
		return Provide("test")(), nil
	},
}

func TestProvider(t *testing.T) {
	// NB(thxCode): respect the Terraform Acceptance logic.
	if os.Getenv(resource.TestEnvVar) != "" {
		t.Skip(fmt.Sprintf(
			"Unit tests skipped as env '%s' set",
			resource.TestEnvVar))
		return
	}

	if err := Provide("test")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

func hasBlank(strs ...string) bool {
	for _, s := range strs {
		if s == "" {
			return true
		}
	}
	return false
}

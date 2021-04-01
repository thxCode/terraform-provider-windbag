package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/thxcode/terraform-provider-windbag/windbag"
	"github.com/thxcode/terraform-provider-windbag/windbag/log"
)

var (
	// version specifies the version of windbag provider.
	version = "dev"

	// commit specifies the commit of windbag provider.
	commit = "000000"
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	var opts = &plugin.ServeOpts{ProviderFunc: windbag.Provide(fmt.Sprintf("%s-%s", version, commit))}

	if debugMode {
		err := plugin.Debug(context.Background(), "registry.terraform.io/thxcode/windbag", opts)
		if err != nil {
			log.Fatalln(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/sirupsen/logrus"

	"github.com/thxcode/terraform-provider-windbag/windbag"
	"github.com/thxcode/terraform-provider-windbag/windbag/format"
)

var (
	// version specifies the version of windbag provider.
	version = "dev"

	// commit specifies the commit of windbag provider.
	commit = "000000"
)

func init() {
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(getLogLevel())
	logrus.SetFormatter(getLogFormatter())
}

func getLogFormatter() logrus.Formatter {
	return &format.SimpleTextFormatter{
		DisableTimestamp:       true,
		DisableSorting:         true,
		DisableLevelTruncation: true,
		QuoteEmptyFields:       true,
	}
}

func getLogLevel() logrus.Level {
	var env = os.Getenv("WINDBAG_LOG")
	if env == "" {
		env = os.Getenv(logging.EnvLog)
	}
	if env == "" {
		env = "info"
	}
	var level, err = logrus.ParseLevel(env)
	if err != nil {
		level = logrus.InfoLevel
	}
	return level
}

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	var opts = &plugin.ServeOpts{ProviderFunc: windbag.Provide(fmt.Sprintf("%s-%s", version, commit))}

	if debugMode {
		err := plugin.Debug(context.Background(), "registry.terraform.io/thxcode/windbag", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}

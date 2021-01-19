package windbag

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/thxcode/terraform-provider-windbag/windbag/worker"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
		desc := s.Description
		if s.Default != nil {
			desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
		}
		return strings.TrimSpace(desc)
	}
}

func Provide(version string) func() *schema.Provider {
	return func() *schema.Provider {
		var p = new(schema.Provider)

		registerSchema(p)
		registerDataSources(p)
		registerResources(p)
		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func registerSchema(p *schema.Provider) {
	p.Schema = map[string]*schema.Schema{}
}

func registerDataSources(p *schema.Provider) {
	p.DataSourcesMap = map[string]*schema.Resource{
		"windbag_registry": dataSourceWindbagRegistry(),
	}
}

func registerResources(p *schema.Provider) {
	p.ResourcesMap = map[string]*schema.Resource{
		"windbag_image":  resourceWindbagImage(),
		"windbag_worker": resourceWindbagWorker(),
	}
}

type provider struct {
	// data sources
	registryAuths map[string]types.AuthConfig
	workers       map[string]worker.Dialer
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var p = &provider{
			registryAuths: make(map[string]types.AuthConfig),
			workers:       make(map[string]worker.Dialer),
		}

		return p, nil
	}
}

package windbag

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/thxcode/terraform-provider-windbag/windbag/utils"
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
	p.Schema = map[string]*schema.Schema{
		"docker": {
			Description: "Specify the Docker as builder.",
			Type:        schema.TypeSet,
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"version": {
						Description: "Specify the version of Docker.",
						Type:        schema.TypeString,
						Optional:    true,
						Default:     "19.03",
					},
					"download_uri": {
						Description: "Specify the URI to download the Docker ZIP archive.",
						Type:        schema.TypeString,
						Optional:    true,
					},
				},
			},
		},
	}
}

func registerDataSources(p *schema.Provider) {
	p.DataSourcesMap = map[string]*schema.Resource{}
}

func registerResources(p *schema.Provider) {
	p.ResourcesMap = map[string]*schema.Resource{
		"windbag_image": resourceWindbagImage(),
	}
}

type provider struct {
	docker *dockerBuilder
}

type dockerBuilder struct {
	Version     string
	DownloadURI string
}

func configure(_ string, _ *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var p provider

		if v, ok := d.GetOk("docker"); ok {
			var builder dockerBuilder
			var docker = utils.ToStringInterfaceMap(v)
			if vi, ok := docker["version"]; ok {
				builder.Version = utils.ToString(vi)
			}
			if vi, ok := docker["download_uri"]; ok {
				builder.DownloadURI = utils.ToString(vi)
			}
			p.docker = &builder
		}

		return &p, nil
	}
}

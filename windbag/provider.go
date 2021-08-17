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
					"experimental": {
						Description: "Specify whether to enable experimental feature.",
						Type:        schema.TypeBool,
						Optional:    true,
						Default:     true,
					},
					"push_foreign_layers": {
						Description: "Specify where to push none distributable artifacts, like 'mcr.microsoft.com' layer.",
						Type:        schema.TypeBool,
						Optional:    true,
						Default:     false,
					},
					"max_concurrent_downloads": {
						Description: "Specify the max concurrent downloads for each pull.",
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     8,
					},
					"max_concurrent_uploads": {
						Description: "Specify the max concurrent uploads for each push.",
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     8,
					},
					"max_download_attempts": {
						Description: "Specify the max download attempts for each pull.",
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     10,
					},
					"registry_mirrors": {
						Description: "Specify the list of registry mirror.",
						Type:        schema.TypeList,
						Optional:    true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
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
	Version                       string
	DownloadURI                   string
	AllowNonDistributableArtifact []string
	Experimental                  bool
	MaxConcurrentDownloads        int
	MaxConcurrentUploads          int
	MaxDownloadAttempts           int
	RegistryMirrors               []string
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
			if vi, ok := docker["experimental"]; ok {
				builder.Experimental = utils.ToBool(vi, true)
			}
			if vi, ok := docker["push_foreign_layers"]; ok {
				if utils.ToBool(vi) {
					builder.AllowNonDistributableArtifact = make([]string, 0, 0)
				}
			}
			if vi, ok := docker["max_concurrent_downloads"]; ok {
				builder.MaxConcurrentDownloads = utils.ToInt(vi, 8)
			}
			if vi, ok := docker["max_concurrent_uploads"]; ok {
				builder.MaxConcurrentUploads = utils.ToInt(vi, 8)
			}
			if vi, ok := docker["max_download_attempts"]; ok {
				builder.MaxDownloadAttempts = utils.ToInt(vi, 10)
			}
			if vi, ok := docker["registry_mirrors"]; ok {
				builder.RegistryMirrors = utils.ToStringSlice(vi)
			}
			p.docker = &builder
		}

		return &p, nil
	}
}

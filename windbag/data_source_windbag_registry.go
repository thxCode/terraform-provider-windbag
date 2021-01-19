package windbag

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/thxcode/terraform-provider-windbag/windbag/registry"
	"github.com/thxcode/terraform-provider-windbag/windbag/utils"
)

func dataSourceWindbagRegistry() *schema.Resource {
	return &schema.Resource{
		Description: "Specify the registry to login.",

		ReadContext: dataSourceWindbagRegistryRead,

		Schema: map[string]*schema.Schema{
			"address": {
				Description: "Specify the address of the registry, and use the last item as this resource ID.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"username": {
				Description: "Specify the username of the registry credential.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"password": {
				Description: "Specify the password of the registry credential.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Sensitive:   true,
			},
		},
	}
}

func dataSourceWindbagRegistryRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var p = meta.(*provider)
	var id string

	var addresses []string
	if v, ok := d.GetOk("address"); ok {
		addresses = utils.ToStringSlice(v)
	}

	for idx := range addresses {
		var authOpts types.AuthConfig
		authOpts.ServerAddress = registry.NormalizeRegistryAddress(addresses[idx])
		authOpts.Username = utils.ToString(d.Get("username"))
		authOpts.Password = utils.ToString(d.Get("password"))

		var registryHostname = registry.ConvertToHostname(authOpts.ServerAddress)
		p.registryAuths[registryHostname] = authOpts

		id = registryHostname // use the last item as resource ID
	}

	d.SetId(id)
	return nil
}

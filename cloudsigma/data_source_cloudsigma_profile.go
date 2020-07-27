package cloudsigma

import (
	"context"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudSigmaProfile() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudSigmaProfileRead,

		Schema: map[string]*schema.Schema{
			"address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"company": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"first_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"title": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudSigmaProfileRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	profile, _, err := client.Profile.Get(ctx)
	if err != nil {
		return diag.Errorf("error getting profile: %v", err)
	}

	d.SetId(profile.UUID)
	_ = d.Set("address", profile.Address)
	_ = d.Set("company", profile.Company)
	_ = d.Set("first_name", profile.FirstName)
	_ = d.Set("last_name", profile.LastName)
	_ = d.Set("title", profile.Title)

	return nil
}

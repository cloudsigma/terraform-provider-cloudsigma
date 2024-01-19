package cloudsigma

import (
	"context"
	"errors"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudSigmaVLAN() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudSigmaVLANRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudSigmaVLANRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return diag.Errorf("issue with filters: %v", filtersOk)
	}

	vlans, _, err := client.VLANs.List(ctx)
	if err != nil {
		return diag.Errorf("error getting vlans: %v", err)
	}

	vlanList := make([]cloudsigma.VLAN, 0)

	filter := buildCloudSigmaDataSourceFilter(filters.(*schema.Set))
	for idx := range vlans {
		sm, err := structToMap(vlans[idx])
		if err != nil {
			return diag.FromErr(err)
		}

		if filterLoop(filter, sm) {
			vlanList = append(vlanList, vlans[idx])
		}
	}

	if len(vlanList) > 1 {
		return diag.FromErr(errors.New("your search returned too many results. Please refine your search to be more specific"))
	}
	if len(vlanList) < 1 {
		return diag.FromErr(errors.New("no results were found"))
	}

	d.SetId(vlanList[0].UUID)
	_ = d.Set("meta", vlanList[0].Meta)
	_ = d.Set("resource_uri", vlanList[0].ResourceURI)

	return nil
}

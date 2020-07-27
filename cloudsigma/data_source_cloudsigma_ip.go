package cloudsigma

import (
	"context"
	"errors"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCloudSigmaIP() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudSigmaIPRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"gateway": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"netmask": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudSigmaIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return diag.Errorf("issue with filters: %v", filtersOk)
	}

	ips, _, err := client.IPs.List(ctx)
	if err != nil {
		return diag.Errorf("error getting ips: %v", err)
	}

	ipList := make([]cloudsigma.IP, 0)

	f := buildCloudSigmaDataSourceFilter(filters.(*schema.Set))
	for _, ip := range ips {
		sm, err := structToMap(ip)
		if err != nil {
			return diag.FromErr(err)
		}

		if filterLoop(f, sm) {
			ipList = append(ipList, ip)
		}
	}

	if len(ipList) > 1 {
		return diag.FromErr(errors.New("your search returned too many results. Please refine your search to be more specific"))
	}
	if len(ipList) < 1 {
		return diag.FromErr(errors.New("no results were found"))
	}

	d.SetId(ipList[0].UUID)
	_ = d.Set("gateway", ipList[0].Gateway)
	_ = d.Set("netmask", ipList[0].Netmask)
	_ = d.Set("resource_uri", ipList[0].ResourceURI)

	return nil
}

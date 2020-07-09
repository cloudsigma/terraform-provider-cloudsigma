package cloudsigma

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceCloudSigmaVLAN() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCloudSigmaVLANRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"resource_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudSigmaVLANRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return fmt.Errorf("issue with filters: %v", filtersOk)
	}

	vlans, _, err := client.VLANs.List(context.Background())
	if err != nil {
		return fmt.Errorf("error getting vlans: %v", err)
	}

	vlanList := make([]cloudsigma.VLAN, 0)

	f := buildCloudSigmaDataSourceFilter(filters.(*schema.Set))
	for _, vlan := range vlans {
		sm, err := structToMap(vlan)
		if err != nil {
			return err
		}

		if filterLoop(f, sm) {
			vlanList = append(vlanList, vlan)
		}
	}

	if len(vlanList) > 1 {
		return errors.New("your search returned too many results. Please refine your search to be more specific")
	}
	if len(vlanList) < 1 {
		return errors.New("no results were found")
	}

	d.SetId(vlanList[0].UUID)
	_ = d.Set("resource_uri", vlanList[0].ResourceURI)

	return nil
}

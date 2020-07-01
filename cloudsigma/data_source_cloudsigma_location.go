package cloudsigma

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceCloudSigmaLocation() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCloudSigmaLocationRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),

			"api_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"country_code": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"display_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudSigmaLocationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudsigma.Client)

	filters, filtersOk := d.GetOk("filter")
	if !filtersOk {
		return fmt.Errorf("issue with filters: %v", filtersOk)
	}

	locations, _, err := client.Locations.List(context.Background())
	if err != nil {
		return fmt.Errorf("error getting locations: %v", err)
	}

	locationList := make([]cloudsigma.Location, 0)

	f := buildCloudSigmaDataSourceFilter(filters.(*schema.Set))
	for _, location := range locations {
		sm, err := structToMap(location)
		if err != nil {
			return err
		}

		if filterLoop(f, sm) {
			locationList = append(locationList, location)
		}
	}

	if len(locationList) > 1 {
		return errors.New("your search returned too many results. Please refine your search to be more specific")
	}
	if len(locationList) < 1 {
		return errors.New("no results were found")
	}

	d.SetId(locationList[0].ID)
	_ = d.Set("api_endpoint", locationList[0].APIEndpoint)
	_ = d.Set("country_code", locationList[0].CountryCode)
	_ = d.Set("display_name", locationList[0].DisplayName)
	_ = d.Set("id", locationList[0].ID)

	return nil
}
